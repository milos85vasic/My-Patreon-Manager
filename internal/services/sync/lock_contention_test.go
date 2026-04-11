package sync

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/milos85vasic/My-Patreon-Manager/internal/database"
)

// contentionMockDB is a minimal database.Database stub for lock contention
// tests. Only the lock-related methods are used; everything else panics so
// any accidental call is caught loudly.
type contentionMockDB struct {
	database.Database
	acquire func(ctx context.Context, lockInfo database.SyncLock) error
	release func(ctx context.Context) error
}

func (m *contentionMockDB) AcquireLock(ctx context.Context, lockInfo database.SyncLock) error {
	if m.acquire != nil {
		return m.acquire(ctx, lockInfo)
	}
	return nil
}

func (m *contentionMockDB) ReleaseLock(ctx context.Context) error {
	if m.release != nil {
		return m.release(ctx)
	}
	return nil
}

// TestLockManagerMutexNotHeldAcrossIO asserts that while one AcquireLock call
// is inside the slow file-I/O section, another goroutine can acquire the
// LockManager mutex. If the mutex were held across os.WriteFile, the second
// goroutine's TryLock would fail — so this test directly encodes the
// invariant "mu is not held across I/O".
func TestLockManagerMutexNotHeldAcrossIO(t *testing.T) {
	dir := t.TempDir()
	db := &contentionMockDB{}
	lm := NewLockManager(db)
	lm.ExportedSetLockFile(filepath.Join(dir, "test.lock"))

	slow := 120 * time.Millisecond
	restore := installSlowWriteHook(lm, slow)
	defer restore()

	// Block the writer until the prober has had a chance to TryLock,
	// guaranteeing the probe runs *while* AcquireLock is mid-I/O.
	proberDone := make(chan struct{})

	var mutexHeldAcrossIO atomic.Bool
	var wg sync.WaitGroup
	wg.Add(2)

	// Wrap the slow hook with a gate so we observably stay inside I/O.
	origFn := lm.ExportedWriteFileFn()
	lm.ExportedSetWriteFileFn(func(path string, data []byte, perm os.FileMode) error {
		// Wait for the prober to complete its TryLock check before returning.
		<-proberDone
		return origFn(path, data, perm)
	})

	go func() {
		defer wg.Done()
		if err := lm.AcquireLock(context.Background()); err != nil {
			t.Errorf("AcquireLock: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		// Give the first goroutine time to enter the I/O phase.
		time.Sleep(20 * time.Millisecond)
		mu := lm.ExportedMutex()
		if !mu.TryLock() {
			// Mutex is still held — the mutex-across-I/O bug is present.
			mutexHeldAcrossIO.Store(true)
		} else {
			mu.Unlock()
		}
		close(proberDone)
	}()

	wg.Wait()

	if mutexHeldAcrossIO.Load() {
		t.Fatalf("LockManager.mu was held across blocking file I/O (slow=%v)", slow)
	}
	if !lm.ExportedHeld() {
		t.Fatalf("expected held=true after successful AcquireLock")
	}

	// Cleanup: release so the test leaves no file behind.
	if err := lm.ReleaseLock(context.Background()); err != nil {
		t.Fatalf("ReleaseLock: %v", err)
	}
	if lm.ExportedHeld() {
		t.Fatalf("expected held=false after ReleaseLock")
	}
}

// TestLockManagerWriteFileError ensures a failing writeFileFn propagates and
// does not leave held=true.
func TestLockManagerWriteFileError(t *testing.T) {
	dir := t.TempDir()
	db := &contentionMockDB{}
	lm := NewLockManager(db)
	lm.ExportedSetLockFile(filepath.Join(dir, "test.lock"))
	lm.ExportedSetWriteFileFn(func(path string, data []byte, perm os.FileMode) error {
		return os.ErrPermission
	})

	err := lm.AcquireLock(context.Background())
	if err == nil {
		t.Fatalf("expected error from failing writeFileFn")
	}
	if lm.ExportedHeld() {
		t.Fatalf("expected held=false after failed AcquireLock")
	}
}

// TestLockManagerDBErrorCleansUpFile verifies that when the DB rejects the
// acquire, the file-lock we optimistically wrote is cleaned up.
func TestLockManagerDBErrorCleansUpFile(t *testing.T) {
	dir := t.TempDir()
	lockFile := filepath.Join(dir, "test.lock")
	db := &contentionMockDB{
		acquire: func(ctx context.Context, _ database.SyncLock) error {
			return context.DeadlineExceeded
		},
	}
	lm := NewLockManager(db)
	lm.ExportedSetLockFile(lockFile)

	err := lm.AcquireLock(context.Background())
	if err == nil {
		t.Fatalf("expected error from DB acquire")
	}
	if _, statErr := os.Stat(lockFile); !os.IsNotExist(statErr) {
		t.Fatalf("expected lock file to be removed after DB error, stat err=%v", statErr)
	}
	if lm.ExportedHeld() {
		t.Fatalf("expected held=false after failed DB acquire")
	}
}
