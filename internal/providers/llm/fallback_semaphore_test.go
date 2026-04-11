package llm

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/milos85vasic/My-Patreon-Manager/internal/concurrency"
	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
)

// TestFallbackChainCapsConcurrency asserts that concurrent GenerateContent
// calls never exceed the semaphore limit when one is configured.
func TestFallbackChainCapsConcurrency(t *testing.T) {
	var active, peak int32

	slow := func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
		n := atomic.AddInt32(&active, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		return models.Content{
			Title:        "t",
			Body:         "b",
			QualityScore: 0.99,
			ModelUsed:    "slow",
			TokenCount:   10,
		}, nil
	}

	primary := &mockProvider{generateContentFunc: slow}

	sem := concurrency.NewSemaphore(3)
	fc := NewFallbackChain([]LLMProvider{primary}, 0.8, nil, WithSemaphore(sem))

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = fc.GenerateContent(context.Background(), models.Prompt{}, models.GenerationOptions{})
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&peak); got > 3 {
		t.Fatalf("peak concurrency = %d, want <= 3", got)
	}
}

// TestFallbackChainSemaphoreContextCancel asserts Acquire returns the
// context error when the caller's context is cancelled before a slot
// becomes available.
func TestFallbackChainSemaphoreContextCancel(t *testing.T) {
	// Capacity 1, hold the slot with a blocking call, then issue a second
	// call with a cancelled context.
	block := make(chan struct{})
	release := make(chan struct{})

	hold := func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
		close(block)
		<-release
		return models.Content{QualityScore: 0.99}, nil
	}

	primary := &mockProvider{generateContentFunc: hold}

	sem := concurrency.NewSemaphore(1)
	fc := NewFallbackChain([]LLMProvider{primary}, 0.8, nil, WithSemaphore(sem))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = fc.GenerateContent(context.Background(), models.Prompt{}, models.GenerationOptions{})
	}()

	<-block // first caller is holding the slot

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := fc.GenerateContent(ctx, models.Prompt{}, models.GenerationOptions{})
	if err == nil {
		t.Fatalf("expected error from cancelled context, got nil")
	}

	close(release)
	wg.Wait()
}
