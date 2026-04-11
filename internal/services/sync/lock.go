package sync

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/milos85vasic/My-Patreon-Manager/internal/database"
	"github.com/milos85vasic/My-Patreon-Manager/internal/errors"
	"github.com/milos85vasic/My-Patreon-Manager/internal/utils"
)

type LockManager struct {
	db       database.Database
	lockFile string
	hostname string
	mu       sync.Mutex
	// writeFileFn is the function used to persist the on-disk lock file.
	// It is injectable so tests can swap in a slow writer to assert that
	// LockManager.mu is not held across blocking I/O.
	writeFileFn func(path string, data []byte, perm os.FileMode) error
	// held tracks whether this process currently owns the lock. It is the
	// only piece of in-memory state protected by mu.
	held bool
}

func NewLockManager(db database.Database) *LockManager {
	hostname, _ := os.Hostname()
	return &LockManager{
		db:          db,
		lockFile:    "/tmp/patreon-manager-sync.lock",
		hostname:    hostname,
		writeFileFn: os.WriteFile,
	}
}

func (lm *LockManager) AcquireLock(ctx context.Context) error {
	// Phase 1: compute desired state under the mutex. We intentionally do
	// NOT perform any blocking I/O while holding lm.mu — holding the mutex
	// across os.WriteFile would serialize every caller of LockManager and
	// block readers of the in-memory state for the duration of disk I/O.
	lm.mu.Lock()
	pid := os.Getpid()
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	lockFilePath := lm.lockFile
	hostname := lm.hostname
	writeFileFn := lm.writeFileFn
	lockInfo := database.SyncLock{
		ID:        utils.NewUUID(),
		PID:       pid,
		Hostname:  hostname,
		StartedAt: now,
		ExpiresAt: expires,
	}
	content := fmt.Sprintf("%d:%s:%s", pid, hostname, now.Format(time.RFC3339))
	lm.mu.Unlock()

	// Phase 2: perform the file-write without the mutex held.
	if err := writeFileFn(lockFilePath, []byte(content), 0644); err != nil {
		return err
	}

	// Phase 3: DB acquire (also outside the mutex — the DB has its own
	// concurrency control).
	if err := lm.db.AcquireLock(ctx, lockInfo); err != nil {
		// Best-effort cleanup of the file lock we just wrote.
		lm.releaseFileLock()
		return errors.LockContention(fmt.Sprintf("DB lock failed: %v", err))
	}

	// Phase 4: re-acquire mu briefly to record that we own the lock.
	lm.mu.Lock()
	lm.held = true
	lm.mu.Unlock()
	return nil
}

func (lm *LockManager) ReleaseLock(ctx context.Context) error {
	// Mirror AcquireLock: flip in-memory state under mu, then do I/O outside.
	lm.mu.Lock()
	lm.held = false
	lm.mu.Unlock()

	lm.releaseFileLock()
	return lm.db.ReleaseLock(ctx)
}

func (lm *LockManager) IsLocked(ctx context.Context) (bool, *database.SyncLock, error) {
	locked, err := lm.isFileLocked()
	if err != nil {
		return false, nil, err
	}
	if locked {
		lock := &database.SyncLock{PID: -1, Hostname: "unknown"}
		return true, lock, nil
	}
	return lm.db.IsLocked(ctx)
}

func (lm *LockManager) releaseFileLock() {
	os.Remove(lm.lockFile)
}

func (lm *LockManager) isFileLocked() (bool, error) {
	content, err := os.ReadFile(lm.lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	line := strings.TrimSpace(string(content))
	if line == "" {
		return false, nil
	}

	fields := strings.SplitN(line, ":", 3)
	if len(fields) < 1 {
		return false, nil
	}

	lockedPID, err := strconv.Atoi(fields[0])
	if err != nil {
		return false, nil
	}

	if lockedPID == os.Getpid() {
		return true, nil
	}

	process, err := os.FindProcess(lockedPID)
	if err != nil {
		return false, nil
	}

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}

	if pe, ok := err.(*os.PathError); ok && pe.Err == syscall.ESRCH {
		os.Remove(lm.lockFile)
		return false, nil
	}

	return true, nil
}
