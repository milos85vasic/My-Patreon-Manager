package filter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// TestWatchSIGHUPStoppable verifies that WatchSIGHUP exits deterministically
// when its caller closes the stop channel, and that no goroutine leaks.
func TestWatchSIGHUPStoppable(t *testing.T) {
	defer goleak.VerifyNone(t)

	dir := t.TempDir()
	path := filepath.Join(dir, ".repoignore")
	if err := os.WriteFile(path, []byte("github.com/owner/repo\n"), 0o644); err != nil {
		t.Fatalf("write temp repoignore: %v", err)
	}

	r, err := ParseRepoignoreFile(path)
	if err != nil {
		t.Fatalf("ParseRepoignoreFile: %v", err)
	}

	stop := make(chan struct{})
	done := r.WatchSIGHUP(stop)
	close(stop)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("WatchSIGHUP did not exit within 500ms after stop closed")
	}
}
