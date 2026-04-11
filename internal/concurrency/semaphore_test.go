package concurrency

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSemaphoreLimitsConcurrency(t *testing.T) {
	s := NewSemaphore(2)
	var active, peak int32
	ctx := context.Background()
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			if err := s.Acquire(ctx, 1); err != nil {
				t.Error(err)
			}
			n := atomic.AddInt32(&active, 1)
			for {
				p := atomic.LoadInt32(&peak)
				if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&active, -1)
			s.Release(1)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	if atomic.LoadInt32(&peak) > 2 {
		t.Fatalf("peak=%d > 2", peak)
	}
}

func TestSemaphoreRespectsContextCancel(t *testing.T) {
	s := NewSemaphore(1)
	if err := s.Acquire(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if err := s.Acquire(ctx, 1); err == nil {
		t.Fatal("expected context deadline exceeded")
	}
}
