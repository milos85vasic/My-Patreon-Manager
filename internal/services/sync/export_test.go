package sync

import (
	"os"
	"sync"
	"time"
)

// ExportedMutex returns the lock manager's internal mutex for tests only.
func (lm *LockManager) ExportedMutex() *sync.Mutex { return &lm.mu }

// ExportedHeld returns the in-memory held flag for tests only.
func (lm *LockManager) ExportedHeld() bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.held
}

// ExportedSetLockFile overrides the on-disk lock path for hermetic tests.
func (lm *LockManager) ExportedSetLockFile(path string) { lm.lockFile = path }

// ExportedSetWriteFileFn overrides the file-write implementation for tests.
func (lm *LockManager) ExportedSetWriteFileFn(fn func(path string, data []byte, perm os.FileMode) error) {
	lm.writeFileFn = fn
}

// ExportedWriteFileFn returns the current writeFileFn for tests.
func (lm *LockManager) ExportedWriteFileFn() func(path string, data []byte, perm os.FileMode) error {
	return lm.writeFileFn
}

// installSlowWriteHook swaps the file-I/O function with one that sleeps,
// then delegates to the original. Returns a restore func.
func installSlowWriteHook(lm *LockManager, slow time.Duration) func() {
	orig := lm.writeFileFn
	lm.writeFileFn = func(path string, data []byte, perm os.FileMode) error {
		time.Sleep(slow)
		return orig(path, data, perm)
	}
	return func() { lm.writeFileFn = orig }
}
