package concurrency

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Lifecycle supervises a set of goroutines with shared stop channel +
// bounded shutdown wait.
type Lifecycle struct {
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewLifecycle() *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return &Lifecycle{ctx: ctx, cancel: cancel}
}

func (l *Lifecycle) Context() context.Context { return l.ctx }

func (l *Lifecycle) Go(fn func(ctx context.Context)) {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		fn(l.ctx)
	}()
}

func (l *Lifecycle) Stop(wait time.Duration) error {
	l.mu.Lock()
	l.cancel()
	l.mu.Unlock()

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(wait):
		return errors.New("lifecycle: goroutines did not exit within wait")
	}
}
