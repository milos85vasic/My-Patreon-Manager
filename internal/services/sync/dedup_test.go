package sync

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestDedupCloseStopsGoroutine(t *testing.T) {
	defer goleak.VerifyNone(t)
	ed := NewEventDeduplicator(10 * time.Millisecond)
	if err := ed.Close(); err != nil {
		t.Fatal(err)
	}
}
