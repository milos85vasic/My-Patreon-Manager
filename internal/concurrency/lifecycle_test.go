package concurrency

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestLifecycleStopClosesDone(t *testing.T) {
	l := NewLifecycle()
	var ran int32
	l.Go(func(ctx context.Context) {
		atomic.StoreInt32(&ran, 1)
		<-ctx.Done()
	})
	if err := l.Stop(100 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&ran) != 1 {
		t.Fatal("goroutine did not run")
	}
}

func TestLifecycleStopTimesOut(t *testing.T) {
	l := NewLifecycle()
	l.Go(func(ctx context.Context) {
		time.Sleep(200 * time.Millisecond) // ignores ctx intentionally
	})
	if err := l.Stop(10 * time.Millisecond); err == nil {
		t.Fatal("expected timeout")
	}
}
