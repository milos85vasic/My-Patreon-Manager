# Phase 1 — Concurrency Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate all 15 concurrency risks found in the audit — zero goroutine leaks, zero unbounded fan-out, zero races under `-race`, every I/O respects context.

**Architecture:** Introduce three shared primitives — `stopChan` pattern for long-running goroutines, `golang.org/x/sync/semaphore` for bounded fan-out, and a `Clock` interface for injectable time. Fix every affected file with a dedicated test that fails before the fix and passes after.

**Tech Stack:** Go 1.26.1, `golang.org/x/sync`, `go.uber.org/goleak`, `github.com/jonboulle/clockwork` (for fake clock), `httptest`.

---

## File Structure

**Create:**
- `internal/concurrency/clock.go` — `Clock` interface + `RealClock` + used-site plumbing
- `internal/concurrency/semaphore.go` — re-export of `semaphore.Weighted` with typed helpers
- `internal/concurrency/lifecycle.go` — `Lifecycle` struct helping the stop/done pattern
- `internal/concurrency/lifecycle_test.go`
- `internal/sync/export_test.go` — exposes unexported seams for `lock.go`
- `tests/leaks/leak_test.go` — `goleak` verification harness
- `internal/providers/git/breaker.go` — helper wrapping tokenManager breaker around HTTP calls (already exists as `TokenManager.cb`, but add `Execute` wrapper)

**Modify:**
- `internal/services/sync/dedup.go` — add stop channel
- `internal/handlers/webhook.go` — bounded consumer or require Queue
- `internal/services/sync/scheduler.go` — accept parent context
- `internal/middleware/ratelimit.go` — TTL eviction
- `internal/services/filter/repoignore.go` — stop channel for WatchSIGHUP
- `internal/database/sqlite.go:276` — scan error handling
- `internal/services/sync/lock.go` — release mutex before file I/O
- `internal/services/content/budget.go` — drop callbacks outside lock
- `internal/services/content/generator.go` — `time.NewTimer` with `Stop()`
- `internal/providers/git/{github,gitlab,gitflic,gitverse}.go` — wrap API calls in breaker
- `internal/providers/patreon/client.go` — add breaker
- `internal/providers/llm/fallback.go` — add global semaphore
- `cmd/cli/main.go`, `cmd/server/main.go` — construct and shut down goroutine owners

---

## Task 1: Add Clock interface and shared semaphore

**Files:**
- Create: `internal/concurrency/clock.go`
- Create: `internal/concurrency/clock_test.go`
- Create: `internal/concurrency/semaphore.go`
- Create: `internal/concurrency/semaphore_test.go`
- Create: `internal/concurrency/lifecycle.go`
- Create: `internal/concurrency/lifecycle_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/concurrency/clock_test.go
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
```

```go
// internal/concurrency/semaphore_test.go
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
```

```go
// internal/concurrency/lifecycle_test.go
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
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/concurrency/...
```

Expected: packages don't compile (types missing).

- [ ] **Step 3: Implement**

```go
// internal/concurrency/clock.go
package concurrency

import "time"

// Clock is an injectable time source. Production code uses RealClock;
// tests may inject a fake.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	NewTimer(d time.Duration) Timer
}

type Timer interface {
	C() <-chan time.Time
	Stop() bool
	Reset(d time.Duration) bool
}

type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (RealClock) NewTimer(d time.Duration) Timer         { return &realTimer{t: time.NewTimer(d)} }

type realTimer struct{ t *time.Timer }

func (r *realTimer) C() <-chan time.Time { return r.t.C }
func (r *realTimer) Stop() bool          { return r.t.Stop() }
func (r *realTimer) Reset(d time.Duration) bool {
	if !r.t.Stop() {
		select {
		case <-r.t.C:
		default:
		}
	}
	return r.t.Reset(d)
}
```

```go
// internal/concurrency/semaphore.go
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
```

```go
// internal/concurrency/lifecycle.go
package concurrency

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Lifecycle supervises a set of goroutines with shared stop channel +
// bounded shutdown wait.
type Lifecycle struct {
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewLifecycle() *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return &Lifecycle{ctx: ctx, cancel: cancel}
}

func (l *Lifecycle) Context() context.Context { return l.ctx }

func (l *Lifecycle) Go(fn func(ctx context.Context)) {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		fn(l.ctx)
	}()
}

func (l *Lifecycle) Stop(wait time.Duration) error {
	l.mu.Lock()
	l.cancel()
	l.mu.Unlock()

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(wait):
		return errors.New("lifecycle: goroutines did not exit within wait")
	}
}
```

- [ ] **Step 4: Run**

```bash
go test -race ./internal/concurrency/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/concurrency/
git commit -m "feat(concurrency): add Clock, Semaphore, Lifecycle primitives

Shared helpers backing Phase 1 concurrency fixes. Clock interface
enables injectable time; Semaphore wraps x/sync/semaphore; Lifecycle
owns supervised goroutines with bounded Stop()."
```

---

## Task 2: Add goleak harness

**Files:**
- Create: `tests/leaks/leak_test.go`
- Create: `tests/leaks/ignores.go`

- [ ] **Step 1: Write harness**

```go
// tests/leaks/leak_test.go
package leaks

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, Ignores()...)
}
```

```go
// tests/leaks/ignores.go
package leaks

import "go.uber.org/goleak"

// Ignores returns the project-wide goleak allowlist for framework-owned
// goroutines we cannot terminate (database/sql connector, gin trust proxy,
// testcontainers reaper, etc.).
func Ignores() []goleak.Option {
	return []goleak.Option{
		goleak.IgnoreTopFunction("database/sql.(*DB).connectionOpener"),
		goleak.IgnoreTopFunction("database/sql.(*DB).connectionResetter"),
		goleak.IgnoreTopFunction("github.com/testcontainers/testcontainers-go.(*Reaper).connect"),
	}
}
```

- [ ] **Step 2: Run**

```bash
go test -race ./tests/leaks/...
```

Expected: PASS (empty package, no goroutines leaked).

- [ ] **Step 3: Commit**

```bash
git add tests/leaks/
git commit -m "test(leaks): add goleak harness with framework allowlist"
```

---

## Task 3: Fix dedup goroutine leak

**Files:**
- Modify: `internal/services/sync/dedup.go`
- Create: `internal/services/sync/dedup_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/services/sync/dedup_test.go
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
```

- [ ] **Step 2: Run to verify failure**

```bash
go test -race ./internal/services/sync/ -run TestDedupCloseStopsGoroutine
```

Expected: fail because `Close()` does not exist or goroutine leaks.

- [ ] **Step 3: Implement**

Edit `internal/services/sync/dedup.go`:

```go
package sync

import (
	"sync"
	"time"
)

type EventDeduplicator struct {
	ttl   time.Duration
	mu    sync.Mutex
	seen  map[string]time.Time
	stop  chan struct{}
	done  chan struct{}
	clock func() time.Time
}

func NewEventDeduplicator(ttl time.Duration) *EventDeduplicator {
	ed := &EventDeduplicator{
		ttl:   ttl,
		seen:  make(map[string]time.Time),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
		clock: time.Now,
	}
	go ed.cleanup()
	return ed
}

func (ed *EventDeduplicator) Seen(key string) bool {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	if t, ok := ed.seen[key]; ok && ed.clock().Sub(t) < ed.ttl {
		return true
	}
	ed.seen[key] = ed.clock()
	return false
}

func (ed *EventDeduplicator) cleanup() {
	defer close(ed.done)
	ticker := time.NewTicker(ed.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ed.stop:
			return
		case now := <-ticker.C:
			ed.mu.Lock()
			for k, t := range ed.seen {
				if now.Sub(t) >= ed.ttl {
					delete(ed.seen, k)
				}
			}
			ed.mu.Unlock()
		}
	}
}

func (ed *EventDeduplicator) Close() error {
	close(ed.stop)
	select {
	case <-ed.done:
		return nil
	case <-time.After(1 * time.Second):
		return ErrShutdownTimeout
	}
}
```

Add `ErrShutdownTimeout` in `internal/services/sync/errors.go`:

```go
package sync

import "errors"

var ErrShutdownTimeout = errors.New("sync: shutdown timed out")
```

- [ ] **Step 4: Run**

```bash
go test -race ./internal/services/sync/ -run TestDedupCloseStopsGoroutine
```

Expected: PASS.

- [ ] **Step 5: Wire shutdown in cmd/server/main.go**

Modify the existing server main to call `dedup.Close()` during graceful shutdown path.

- [ ] **Step 6: Commit**

```bash
git add internal/services/sync/dedup.go internal/services/sync/dedup_test.go internal/services/sync/errors.go cmd/server/main.go
git commit -m "fix(sync): EventDeduplicator stops cleanup goroutine on Close

Adds stop/done channels, deterministic shutdown bounded by 1s, and a
goleak-verified test. Server main wires Close() into graceful shutdown."
```

---

## Task 4: Fix webhook Queue unbounded write

**Files:**
- Modify: `internal/handlers/webhook.go`
- Create: `internal/handlers/webhook_queue.go`
- Create: `internal/handlers/webhook_test.go` (new tests)

- [ ] **Step 1: Write failing test**

```go
// internal/handlers/webhook_test.go
func TestWebhookRequiresBoundedConsumer(t *testing.T) {
	h := NewWebhookHandler(nil)
	// Queue must be required, not optional
	if h.Queue == nil {
		t.Fatal("webhook handler must own a bounded queue")
	}
}

func TestWebhookRespectsQueueCapacity(t *testing.T) {
	h := NewWebhookHandler(nil)
	for i := 0; i < h.Queue.Cap(); i++ {
		if !h.Queue.TryEnqueue(models.Repo{Name: "x"}) {
			t.Fatal("unexpected full")
		}
	}
	if h.Queue.TryEnqueue(models.Repo{Name: "overflow"}) {
		t.Fatal("queue did not reject overflow")
	}
}
```

- [ ] **Step 2: Run** — fail.

- [ ] **Step 3: Implement queue**

```go
// internal/handlers/webhook_queue.go
package handlers

import (
	"sync"

	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
)

type WebhookQueue struct {
	mu    sync.Mutex
	items chan models.Repo
}

func NewWebhookQueue(cap int) *WebhookQueue { return &WebhookQueue{items: make(chan models.Repo, cap)} }

func (q *WebhookQueue) Cap() int { return cap(q.items) }

func (q *WebhookQueue) TryEnqueue(r models.Repo) bool {
	select {
	case q.items <- r:
		return true
	default:
		return false
	}
}

func (q *WebhookQueue) Drain(ctx context.Context, fn func(models.Repo) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-q.items:
			if err := fn(r); err != nil {
				return err
			}
		}
	}
}
```

Edit `webhook.go` to use `WebhookQueue` by default:

```go
type WebhookHandler struct {
	// ... existing fields
	Queue *WebhookQueue
}

func NewWebhookHandler(log *slog.Logger) *WebhookHandler {
	return &WebhookHandler{Queue: NewWebhookQueue(1024), Logger: log}
}
```

Replace every `case h.Queue <- repo:` with `h.Queue.TryEnqueue(repo)` and return HTTP 429 on overflow.

- [ ] **Step 4: Wire a consumer goroutine via Lifecycle** in `cmd/server/main.go`:

```go
lc := concurrency.NewLifecycle()
lc.Go(func(ctx context.Context) {
	_ = webhookHandler.Queue.Drain(ctx, func(r models.Repo) error {
		return orchestrator.EnqueueRepo(ctx, r)
	})
})
// on shutdown: lc.Stop(5 * time.Second)
```

- [ ] **Step 5: Run**

```bash
go test -race ./internal/handlers/... -run Webhook
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/webhook.go internal/handlers/webhook_queue.go internal/handlers/webhook_test.go cmd/server/main.go
git commit -m "fix(webhook): require bounded WebhookQueue + drain loop

Queue is no longer optional. Handler enqueues with TryEnqueue (429 on
overflow). cmd/server wires a Lifecycle-supervised Drain consumer."
```

---

## Task 5: Fix scheduler context

**Files:**
- Modify: `internal/services/sync/scheduler.go`
- Create: `internal/services/sync/scheduler_test.go`

- [ ] **Step 1: Write failing test**

```go
// scheduler_test.go
func TestSchedulerRespectsParentCancel(t *testing.T) {
	s := NewScheduler("@every 10ms", nopJob)
	parent, cancel := context.WithCancel(context.Background())
	go s.Run(parent)
	time.Sleep(15 * time.Millisecond)
	cancel()
	if !s.WaitStop(100 * time.Millisecond) {
		t.Fatal("scheduler did not stop on parent cancel")
	}
}
```

- [ ] **Step 2: Run** — fail (scheduler signature mismatch).

- [ ] **Step 3: Implement**

```go
package sync

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	c     *cron.Cron
	spec  string
	job   func(ctx context.Context) error
	wg    sync.WaitGroup
	stop  chan struct{}
}

func NewScheduler(spec string, job func(ctx context.Context) error) *Scheduler {
	return &Scheduler{c: cron.New(), spec: spec, job: job, stop: make(chan struct{})}
}

func (s *Scheduler) Run(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	_, err := s.c.AddFunc(s.spec, func() {
		jobCtx, jobCancel := context.WithTimeout(ctx, 1*time.Hour)
		defer jobCancel()
		_ = s.job(jobCtx)
	})
	if err != nil {
		return
	}
	s.c.Start()
	<-ctx.Done()
	sctx := s.c.Stop()
	<-sctx.Done()
	close(s.stop)
}

func (s *Scheduler) WaitStop(d time.Duration) bool {
	select {
	case <-s.stop:
		return true
	case <-time.After(d):
		return false
	}
}
```

- [ ] **Step 4: Run**

```bash
go test -race ./internal/services/sync/ -run TestSchedulerRespectsParentCancel
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/sync/scheduler.go internal/services/sync/scheduler_test.go
git commit -m "fix(scheduler): propagate parent context cancellation"
```

---

## Task 6: Fix rate limiter unbounded map

**Files:**
- Modify: `internal/middleware/ratelimit.go`
- Create: `internal/middleware/ratelimit_test.go`

- [ ] **Step 1: Failing test**

```go
func TestRateLimiterEvictsStaleEntries(t *testing.T) {
	clock := clockwork.NewFakeClock()
	rl := NewIPRateLimiter(2, 4, 1*time.Minute).WithClock(clock)
	for i := 0; i < 100; i++ {
		rl.Allow(fmt.Sprintf("ip-%d", i))
	}
	if rl.Len() != 100 {
		t.Fatalf("expected 100 entries, got %d", rl.Len())
	}
	clock.Advance(90 * time.Second)
	rl.Sweep()
	if rl.Len() != 0 {
		t.Fatalf("expected 0 after sweep, got %d", rl.Len())
	}
}
```

- [ ] **Step 2: Run** — fail.

- [ ] **Step 3: Implement**

```go
package middleware

import (
	"sync"
	"time"

	"github.com/jonboulle/clockwork"
	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter *rate.Limiter
	seen    time.Time
}

type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit
	burst    int
	ttl      time.Duration
	clock    clockwork.Clock
}

func NewIPRateLimiter(r rate.Limit, burst int, ttl time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		burst:    burst,
		ttl:      ttl,
		clock:    clockwork.NewRealClock(),
	}
}

func (l *IPRateLimiter) WithClock(c clockwork.Clock) *IPRateLimiter { l.clock = c; return l }

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.limiters[ip]
	if !ok {
		e = &limiterEntry{limiter: rate.NewLimiter(l.r, l.burst)}
		l.limiters[ip] = e
	}
	e.seen = l.clock.Now()
	return e.limiter.Allow()
}

func (l *IPRateLimiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.limiters)
}

func (l *IPRateLimiter) Sweep() {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := l.clock.Now().Add(-l.ttl)
	for k, e := range l.limiters {
		if e.seen.Before(cutoff) {
			delete(l.limiters, k)
		}
	}
}
```

Add a background sweeper wired via `Lifecycle` in `cmd/server/main.go`.

- [ ] **Step 4: Run**

```bash
go test -race ./internal/middleware/ -run TestRateLimiterEvictsStaleEntries
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/ratelimit.go internal/middleware/ratelimit_test.go cmd/server/main.go
git commit -m "fix(ratelimit): TTL eviction prevents unbounded map growth

Adds limiterEntry.seen timestamp, Sweep() method, injectable clock,
and a Lifecycle-supervised background sweeper in cmd/server."
```

---

## Task 7: Fix SIGHUP watcher leak

**Files:**
- Modify: `internal/services/filter/repoignore.go`
- Create: `internal/services/filter/repoignore_watch_test.go`

- [ ] **Step 1: Failing test**

```go
func TestWatchSIGHUPStoppable(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, _ := New("testdata/repoignore")
	stop := make(chan struct{})
	done := r.WatchSIGHUP(stop)
	close(stop)
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("WatchSIGHUP did not exit")
	}
}
```

- [ ] **Step 2: Fail, implement stop-channel version:**

```go
func (r *Repoignore) WatchSIGHUP(stop <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		defer close(done)
		defer signal.Stop(sig)
		for {
			select {
			case <-stop:
				return
			case <-sig:
				_ = r.Reload()
			}
		}
	}()
	return done
}
```

- [ ] **Step 3: Pass, commit:**

```bash
git commit -m "fix(repoignore): WatchSIGHUP accepts stop channel, returns done"
```

---

## Task 8: Fix sqlite.go Scan error + commit guard

**Files:**
- Modify: `internal/database/sqlite.go` (line 276 area)
- Create: `internal/database/sqlite_lock_test.go`

- [ ] **Step 1: Failing test** — simulate a query failure via sqlmock and assert `AcquireLock` returns the wrapped error.

```go
func TestAcquireLockScanErrorRolledBack(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT expires_at").WillReturnError(errors.New("boom"))
	mock.ExpectRollback()
	s := &SQLite{db: db}
	err := s.AcquireLock(context.Background(), "k", time.Minute)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected boom, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Fail, fix:**

```go
var expiresAt time.Time
if err := tx.QueryRowContext(ctx, "SELECT expires_at FROM locks WHERE key = ?", key).Scan(&expiresAt); err != nil {
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("sqlite: scan lock: %w", err)
	}
	expiresAt = time.Time{}
}
// ... rest of logic ...
if err := tx.Commit(); err != nil {
	return fmt.Errorf("sqlite: commit lock: %w", err)
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(sqlite): check Scan error before Commit in AcquireLock"
```

---

## Task 9: Release mutex before file I/O in lock.go

**Files:**
- Modify: `internal/services/sync/lock.go`
- Create: `internal/services/sync/lock_test.go` updates

- [ ] **Step 1: Failing test** — stress: spawn 50 goroutines attempting AcquireLock; assert no goroutine blocks longer than N ms.

```go
func TestLockAcquireDoesNotHoldMutexAcrossFileIO(t *testing.T) {
	// Synthetic slow filesystem via interface injection
	...
}
```

- [ ] **Step 2: Refactor `AcquireLock`** to compute the state under mutex, release, then perform `os.WriteFile`, then re-acquire to record success.

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(lock): drop mutex across os.WriteFile to unblock contention"
```

---

## Task 10: Drop budget callbacks outside lock

**Files:**
- Modify: `internal/services/content/budget.go`

- [ ] **Step 1: Failing test** — a callback that re-enters `CheckBudget` deadlocks with the current code.

```go
func TestBudgetCallbackDoesNotDeadlock(t *testing.T) {
	b := NewBudget(100)
	b.OnSoftAlert = func(pct float64) {
		// should NOT deadlock
		b.CheckBudget(1)
	}
	b.CheckBudget(80)
}
```

- [ ] **Step 2: Refactor**: capture `shouldSoft`, `shouldHard` booleans under the lock, release, then call callbacks.

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(budget): invoke alert callbacks outside the lock"
```

---

## Task 11: Replace time.After in generator retry loop

**Files:**
- Modify: `internal/services/content/generator.go`

- [ ] **Step 1: Failing test** — benchmark allocations with `-benchmem`, assert retry loop doesn't leak timers.

- [ ] **Step 2: Rewrite**:

```go
timer := time.NewTimer(delay)
select {
case <-ctx.Done():
	timer.Stop()
	return ctx.Err()
case <-timer.C:
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(generator): use time.NewTimer with Stop() to avoid timer leaks"
```

---

## Task 12: Wrap git provider API calls in circuit breaker

**Files:**
- Modify: `internal/providers/git/{github,gitlab,gitflic,gitverse}.go`
- Create: `internal/providers/git/breaker_test.go`

- [ ] **Step 1: Failing test** — record that breaker `Execute` is called for every HTTP call via interface spy.

- [ ] **Step 2: Wrap**:

```go
result, err := p.tm.cb.Execute(func() (interface{}, error) {
	return p.doGetRepos(ctx, org)
})
```

Repeat for list/get/archive/trees.

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(git): route provider calls through TokenManager circuit breaker"
```

---

## Task 13: Add circuit breaker to Patreon client

**Files:**
- Modify: `internal/providers/patreon/client.go`
- Create: `internal/providers/patreon/breaker_test.go`

- [ ] **Step 1: Failing test** — 5 consecutive failures trip the breaker; next call returns ErrOpen without hitting the network.

- [ ] **Step 2: Import `github.com/sony/gobreaker/v2`**, add `cb *gobreaker.CircuitBreaker`, wrap `CreatePost`, `UpdatePost`, `DeletePost`.

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(patreon): wrap mutations in gobreaker circuit breaker"
```

---

## Task 14: LLM global concurrency semaphore

**Files:**
- Modify: `internal/providers/llm/fallback.go`

- [ ] **Step 1: Failing test** — assert peak concurrent LLM calls ≤ configured N under 64 parallel callers.

- [ ] **Step 2: Inject `*concurrency.Semaphore` into `FallbackProvider`**; each `Generate` call `Acquire(ctx, 1)` / `Release(1)`.

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(llm): cap global concurrent LLM calls via semaphore"
```

---

## Task 15: Package goleak guards

Add a `TestMain` calling `goleak.VerifyTestMain(m, leaks.Ignores()...)` to each package touched in Phase 1:
`internal/services/sync`, `internal/handlers`, `internal/middleware`, `internal/services/filter`, `internal/services/content`, `internal/providers/git`, `internal/providers/llm`, `internal/providers/patreon`.

- [ ] **Step 1: Add a shared helper** `internal/testhelpers/goleak.go` re-exporting the ignore list.
- [ ] **Step 2: Add TestMain per package.**
- [ ] **Step 3: Commit**

```bash
git commit -m "test: add goleak.VerifyTestMain guards to concurrency-sensitive packages"
```

---

## Task 16: Phase 1 acceptance

- [ ] `go test -race ./internal/... ./cmd/... ./tests/...` green.
- [ ] `goleak` guards active on all Phase 1 packages.
- [ ] Every audit §4.3 item has a passing test asserting the fix.
- [ ] `bash scripts/coverage.sh` still green against `COVERAGE_MIN=0` (coverage not yet raised; Phase 6 handles that).
- [ ] All 14 new commits in git log, one per task.
- [ ] Final commit tagged `phase-1-concurrency-done` (local tag; do not push).

When every box is checked, Phase 1 merges to main and Phase 2 (wire orphans) may start.
