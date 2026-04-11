package concurrency

import (
	"context"

	"golang.org/x/sync/semaphore"
)

type Semaphore struct{ w *semaphore.Weighted }

func NewSemaphore(n int64) *Semaphore {
	return &Semaphore{w: semaphore.NewWeighted(n)}
}

func (s *Semaphore) Acquire(ctx context.Context, n int64) error {
	return s.w.Acquire(ctx, n)
}

func (s *Semaphore) TryAcquire(n int64) bool { return s.w.TryAcquire(n) }

func (s *Semaphore) Release(n int64) { s.w.Release(n) }
