package concurrency

import (
	"testing"
	"time"
)

func TestRealClockNowMonotonic(t *testing.T) {
	c := RealClock{}
	t0 := c.Now()
	time.Sleep(1 * time.Millisecond)
	t1 := c.Now()
	if !t1.After(t0) {
		t.Fatalf("clock not monotonic: %v !> %v", t1, t0)
	}
}

func TestRealClockAfterFires(t *testing.T) {
	c := RealClock{}
	ch := c.After(5 * time.Millisecond)
	select {
	case <-ch:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("After did not fire")
	}
}
