// Module 09 Exercise: Find and fix the race condition
// Run with: go test -race ./examples/module09/
//
//go:build exercise

package module09

import (
	"sync"
	"testing"
)

func TestRaceExample(t *testing.T) {
	counter := 0
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// BUG: this is a data race. Fix it using sync/atomic or a mutex.
			counter++
		}()
	}
	wg.Wait()
	t.Logf("counter = %d (expected 100)", counter)
}
