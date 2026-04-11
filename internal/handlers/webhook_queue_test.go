package handlers

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWebhookQueueCapLen(t *testing.T) {
	q := NewWebhookQueue[int](4)
	if q.Cap() != 4 {
		t.Fatalf("cap = %d", q.Cap())
	}
	if q.Len() != 0 {
		t.Fatalf("len = %d", q.Len())
	}
	if !q.TryEnqueue(1) {
		t.Fatal("enqueue failed")
	}
	if q.Len() != 1 {
		t.Fatalf("len after enqueue = %d", q.Len())
	}
}

func TestWebhookQueueClampsCapacity(t *testing.T) {
	q := NewWebhookQueue[int](0)
	if q.Cap() != 1 {
		t.Fatalf("cap = %d, want clamp to 1", q.Cap())
	}
	q2 := NewWebhookQueue[int](-5)
	if q2.Cap() != 1 {
		t.Fatalf("negative cap = %d, want clamp to 1", q2.Cap())
	}
}

func TestWebhookQueueRejectsOverflow(t *testing.T) {
	q := NewWebhookQueue[int](2)
	if !q.TryEnqueue(1) {
		t.Fatal("first enqueue failed")
	}
	if !q.TryEnqueue(2) {
		t.Fatal("second enqueue failed")
	}
	if q.TryEnqueue(3) {
		t.Fatal("third enqueue should have been rejected")
	}
}

func TestWebhookQueueDrainsUntilCancel(t *testing.T) {
	q := NewWebhookQueue[int](8)
	for i := 0; i < 5; i++ {
		if !q.TryEnqueue(i) {
			t.Fatalf("enqueue %d failed", i)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	var seen []int
	err := q.Drain(ctx, func(v int) error {
		seen = append(seen, v)
		if len(seen) == 5 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx error, got %v", err)
	}
	if len(seen) != 5 {
		t.Fatalf("seen = %v", seen)
	}
}

func TestWebhookQueueDrainPropagatesError(t *testing.T) {
	q := NewWebhookQueue[int](4)
	_ = q.TryEnqueue(1)
	sentinel := errors.New("boom")
	err := q.Drain(context.Background(), func(v int) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel err, got %v", err)
	}
}

func TestWebhookQueueDrainCtxAlreadyCancelled(t *testing.T) {
	q := NewWebhookQueue[int](2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := q.Drain(ctx, func(v int) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected ctx.Canceled, got %v", err)
	}
}
