package handlers

import (
	"context"
)

// WebhookQueue is a bounded, non-blocking queue that drops on overflow.
// Callers use TryEnqueue and the drain goroutine consumes via Drain. The
// queue owns its backing channel so there are no silent-drop paths created
// by callers forgetting to wire a consumer.
type WebhookQueue[T any] struct {
	items chan T
}

// NewWebhookQueue creates a new bounded queue with the given capacity.
// A capacity < 1 is clamped to 1 so the queue is always usable.
func NewWebhookQueue[T any](capacity int) *WebhookQueue[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &WebhookQueue[T]{items: make(chan T, capacity)}
}

// Cap returns the configured capacity of the queue.
func (q *WebhookQueue[T]) Cap() int { return cap(q.items) }

// Len returns the current number of buffered items.
func (q *WebhookQueue[T]) Len() int { return len(q.items) }

// TryEnqueue attempts a non-blocking send. Returns false if the queue is
// full so the caller can surface backpressure (e.g. HTTP 429).
func (q *WebhookQueue[T]) TryEnqueue(v T) bool {
	select {
	case q.items <- v:
		return true
	default:
		return false
	}
}

// Drain consumes items and passes each to fn until ctx is cancelled or fn
// returns an error. It is intended to be run from a Lifecycle-supervised
// goroutine so shutdown is guaranteed to be observed.
func (q *WebhookQueue[T]) Drain(ctx context.Context, fn func(T) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v := <-q.items:
			if err := fn(v); err != nil {
				return err
			}
		}
	}
}
