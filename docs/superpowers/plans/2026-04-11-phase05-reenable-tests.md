# Phase 5 — Re-enable Every Disabled Test Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete every `//go:build disabled` tag and every `t.Skip` in the codebase, replacing each skipped test with a real implementation that matches the intent of the original test. Target zero skipped tests at the end of this phase.

**Architecture:** Apply the remediation pattern recommended per finding in `docs/superpowers/specs/2026-04-11-project-completion-design.md` §4.1. Introduce test-only seams (`export_test.go`) for unexported fields, inject a `Clock` interface for time-dependent tests, use `httptest` for hardcoded-URL tests, and use `t.TempDir() + os.Chmod` for OS-specific directory permission tests.

**Tech Stack:** `testing`, `httptest`, `testify`, Phase-1 `Clock` interface, `github.com/jonboulle/clockwork`, `go.uber.org/goleak`.

**Depends on:** Phases 0, 1, 2, 3, 4.

---

## File Structure

**Modify:**
- `tests/chaos/service_failure_test.go:209`
- `tests/ddos/webhook_flood_test.go:111`
- `tests/integration/dryrun_test.go:231`
- `tests/unit/filter/repoignore_test.go:410`
- `tests/unit/metrics/metrics_test.go:120,143`
- `tests/unit/sync/lock_test.go:80`
- `tests/unit/config/env_test.go:66`
- `tests/unit/database/recovery_test.go:69`
- `tests/unit/providers/llm/fallback_test.go:232`
- `tests/benchmark/sync_bench_test.go:93`

**Move (rename directory):**
- `disabled_tests/security/` → `tests/security/`
  - `webhook_signature_test.go`
  - `access_control_test.go`
  - `credential_redaction_test.go`

**Create:**
- `internal/services/sync/export_test.go` — test-only seams
- `internal/services/filter/export_test.go`
- `internal/providers/llm/clock.go` — injectable clock
- `internal/providers/llm/clock_test.go`

---

## Task 1: Move disabled security tests into active suite

**Files:**
- Move: `disabled_tests/security/*.go` → `tests/security/*.go`
- Modify: all three to remove `//go:build disabled`

- [ ] **Step 1: Verify current state**

```bash
ls disabled_tests/security/
```

Expected: 3 files.

- [ ] **Step 2: Move files**

```bash
mkdir -p tests/security
git mv disabled_tests/security/webhook_signature_test.go tests/security/webhook_signature_test.go
git mv disabled_tests/security/access_control_test.go   tests/security/access_control_test.go
git mv disabled_tests/security/credential_redaction_test.go tests/security/credential_redaction_test.go
rmdir disabled_tests/security
rmdir disabled_tests 2>/dev/null || true
```

- [ ] **Step 3: Remove `//go:build disabled` from each file.**

```bash
for f in tests/security/*.go; do
  sed -i '/^\/\/go:build disabled/d' "$f"
  sed -i '/^\/\/ +build disabled/d' "$f"
done
```

- [ ] **Step 4: Refresh imports** to match current API (package move from `disabled_tests/security` to `tests/security`).

- [ ] **Step 5: Run**

```bash
go test -race ./tests/security/...
```

Expected: many tests will fail because they reference APIs that changed. Fix each test minimally — do not disable or skip. Each fix must be an actual remediation, not a stub.

- [ ] **Step 6: Commit**

```bash
git add tests/security/ disabled_tests
git commit -m "test(security): un-disable webhook signature, access control, redaction

Moved disabled_tests/security to tests/security. All three suites
now compile against current APIs and run in CI."
```

---

## Task 2: Fix `tests/unit/metrics/metrics_test.go` skips (×2)

**Files:**
- Modify: `tests/unit/metrics/metrics_test.go`
- Modify: `internal/metrics/circuitbreaker.go` (add test-only state accessor)

- [ ] **Step 1: Failing test** — replace `t.Skip("cannot locate state field in gobreaker.CircuitBreaker")` with a behavior-level assertion.

The original skips were trying to reflect into gobreaker internals. Replace with behavior observation:

```go
func TestCircuitBreakerExposesOpenState(t *testing.T) {
	cb := metrics.NewCircuitBreaker("test", metrics.CBConfig{
		MaxFailures: 2,
		Interval:    100 * time.Millisecond,
	})
	// Trip it
	for i := 0; i < 3; i++ {
		_, _ = cb.Execute(func() (any, error) { return nil, errors.New("boom") })
	}
	if cb.State() != metrics.StateOpen {
		t.Fatalf("state = %v, want Open", cb.State())
	}
}
```

- [ ] **Step 2: Add `State()` method to wrapper** in `internal/metrics/circuitbreaker.go`:

```go
func (c *CircuitBreaker) State() State {
	switch c.inner.State() {
	case gobreaker.StateOpen:    return StateOpen
	case gobreaker.StateHalfOpen: return StateHalfOpen
	default:                     return StateClosed
	}
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(metrics): expose circuit breaker state via wrapper, remove skips"
```

---

## Task 3: Fix `tests/unit/sync/lock_test.go:80` stale detection

**Files:**
- Create: `internal/services/sync/export_test.go`
- Modify: `tests/unit/sync/lock_test.go`

- [ ] **Step 1: Create seam**

```go
// internal/services/sync/export_test.go
package sync

import "os"

// ExportedLockFile returns the on-disk lock path for tests only.
func (lm *LockManager) ExportedLockFile() string { return lm.lockFile }

// ExportedSetLockFile allows tests to inject a synthetic stale lock.
func (lm *LockManager) ExportedSetLockFile(path string) { lm.lockFile = path }

// ExportedStaleCutoff exposes the stale-lock cutoff used in tests.
var ExportedStaleCutoff = staleCutoff

var _ = os.Chmod // keep import if unused in some variants
```

- [ ] **Step 2: Replace the skipped test** with one that seeds a stale lock file and verifies detection.

```go
func TestLockManagerDetectsStaleLock(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "sync.lock")
	os.WriteFile(stale, []byte("0000"), 0o600)
	os.Chtimes(stale, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))
	lm := NewLockManager(dir)
	lm.ExportedSetLockFile(stale)
	if !lm.IsStale() { t.Fatal("expected stale") }
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(sync): expose lockFile via export_test.go, remove skip"
```

---

## Task 4: Fix `tests/unit/config/env_test.go:66` godotenv error simulation

**Files:**
- Modify: `internal/config/env.go` — introduce loader interface
- Modify: `tests/unit/config/env_test.go`

- [ ] **Step 1: Define loader interface**

```go
type envLoader interface {
	Load(paths ...string) error
}

var defaultLoader envLoader = godotenvAdapter{}
```

- [ ] **Step 2: Test injects a fake loader that returns a synthetic error**

```go
type fakeLoader struct{ err error }
func (f fakeLoader) Load(...string) error { return f.err }

func TestLoadEnvHandlesNonPathError(t *testing.T) {
	oldLoader := defaultLoader
	defer func() { defaultLoader = oldLoader }()
	defaultLoader = fakeLoader{err: errors.New("custom error")}
	_, err := LoadEnv(".")
	if err == nil || !strings.Contains(err.Error(), "custom error") {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(config): dependency-inject godotenv loader, remove skip"
```

---

## Task 5: Fix `tests/unit/database/recovery_test.go:69` read-only directory

**Files:**
- Modify: `tests/unit/database/recovery_test.go`

- [ ] **Step 1: Replace skip with real test** using `t.TempDir()` + `os.Chmod(dir, 0o500)`. If chmod is unsupported on a platform, use `t.Skipf` only on that platform (this is acceptable).

```go
func TestDatabaseRecoveryReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skipf("read-only dir semantics differ") }
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil { t.Fatal(err) }
	defer os.Chmod(dir, 0o700)
	db, err := OpenSQLite(filepath.Join(dir, "test.db"))
	if err == nil { t.Fatal("expected error on read-only dir") }
	_ = db
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "fix(database): read-only recovery test uses TempDir+Chmod"
```

---

## Task 6: Fix `tests/unit/providers/llm/fallback_test.go:232` timing skip

**Files:**
- Create: `internal/providers/llm/clock.go`
- Modify: `internal/providers/llm/fallback.go`
- Modify: `tests/unit/providers/llm/fallback_test.go`

- [ ] **Step 1: Add clock**

```go
// internal/providers/llm/clock.go
package llm

import (
	"github.com/jonboulle/clockwork"
)

type Clock = clockwork.Clock
var DefaultClock = clockwork.NewRealClock()
```

- [ ] **Step 2: Inject into `FallbackProvider`** via `WithClock(c Clock)` functional option.

- [ ] **Step 3: Rewrite skipped test** to use `fakeClock := clockwork.NewFakeClock()` and `fakeClock.Advance(cooldown)`.

- [ ] **Step 4: Commit**

```bash
git commit -m "fix(llm): inject clockwork.Clock into FallbackProvider, remove timing skip"
```

---

## Task 7: Fix `tests/unit/filter/repoignore_test.go:410` reflection panic

**Files:**
- Create: `internal/services/filter/export_test.go`
- Modify: `tests/unit/filter/repoignore_test.go`

- [ ] **Step 1: Expose a test-only seam** instead of reflecting:

```go
// internal/services/filter/export_test.go
package filter
func (r *Repoignore) ExportedPatterns() []string { return r.patterns }
```

- [ ] **Step 2: Replace reflection-based assertion** with `r.ExportedPatterns()` check.

- [ ] **Step 3: Commit**

```bash
git commit -m "fix(filter): replace reflection with export_test.go seam"
```

---

## Task 8: Fix `tests/unit/providers/git/token_failover_test.go` hardcoded URL skips

**Files:**
- Modify: `internal/providers/git/*.go` (already accept base URL)
- Modify: `tests/unit/providers/git/token_failover_test.go`

- [ ] **Step 1: Replace each `t.Skip` with an `httptest.Server` fixture** that returns the status codes needed (401 / 403 / 200) to exercise the failover state machine.

```go
func TestTokenFailoverRotatesOn403(t *testing.T) {
	mu := sync.Mutex{}; calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock(); defer mu.Unlock()
		calls++
		if calls < 2 { w.WriteHeader(403); return }
		w.WriteHeader(200); w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	tm := NewTokenManager([]string{"t1","t2"})
	p := NewGitHubProvider(Config{BaseURL: srv.URL}, tm)
	if _, err := p.ListRepos(context.Background(), "org"); err != nil { t.Fatal(err) }
	if tm.Current() != "t2" { t.Fatal("did not rotate") }
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "fix(git): replace hardcoded URL skips with httptest server fixtures"
```

---

## Task 9: Fix `tests/chaos/service_failure_test.go:209` breaker trip skip

**Files:**
- Modify: `tests/chaos/service_failure_test.go`

- [ ] **Step 1: Replace skip** with a real assertion using the Phase-5 Task-2 `State()` wrapper.

```go
func TestCircuitBreakerTripsOnConsecutiveFailures(t *testing.T) {
	cb := metrics.NewCircuitBreaker("chaos", metrics.CBConfig{MaxFailures: 3})
	for i := 0; i < 4; i++ {
		_, _ = cb.Execute(func() (any, error) { return nil, errors.New("outage") })
	}
	if cb.State() != metrics.StateOpen { t.Fatal("expected Open") }
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "fix(chaos): real circuit-breaker trip assertion, remove skip"
```

---

## Task 10: Fix `tests/ddos/webhook_flood_test.go:111` server-responsiveness skip

**Files:**
- Modify: `tests/ddos/webhook_flood_test.go`

- [ ] **Step 1: Replace skip** with a real `httptest.NewServer` + vegeta flood at 1k req/s for 2 s, assert P99 latency < threshold and 0% 5xx.

```go
func TestWebhookServerResponsivenessUnderFlood(t *testing.T) {
	srv := newTestGinServer(t)
	defer srv.Close()
	rate := vegeta.Rate{Freq: 1000, Per: time.Second}
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "POST",
		URL:    srv.URL + "/webhook/github",
		Header: signedHeaders(t, "{}"),
		Body:   []byte("{}"),
	})
	attacker := vegeta.NewAttacker()
	var m vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, 2*time.Second, "flood") {
		m.Add(res)
	}
	m.Close()
	if m.Latencies.P99 > 100*time.Millisecond {
		t.Fatalf("P99 %v > 100ms", m.Latencies.P99)
	}
	if m.StatusCodes["5"] > 0 {
		t.Fatalf("5xx count=%d", m.StatusCodes["5"])
	}
}
```

- [ ] **Step 2: Add vegeta import** in `go.mod`.
- [ ] **Step 3: Commit**

```bash
git commit -m "fix(ddos): real vegeta flood asserting P99 and 0% 5xx"
```

---

## Task 11: Fix `tests/integration/dryrun_test.go:231` audit entries TODO

**Files:**
- Modify: `tests/integration/dryrun_test.go`

- [ ] **Step 1: Replace TODO comment** with a real assertion that Phase-2 wiring writes audit entries from the dry-run path:

```go
func TestDryRunEmitsAuditEntries(t *testing.T) {
	ring := audit.NewRingStore(64)
	orch := buildTestOrchestrator(t, ring)
	_ = orch.RunDryRun(context.Background())
	entries, _ := ring.List(context.Background(), 100)
	if len(entries) == 0 { t.Fatal("dry-run should emit audit entries") }
	for _, e := range entries {
		if e.Action != "sync.dryrun.repo" { t.Fatalf("unexpected action %q", e.Action) }
	}
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "fix(dryrun): assert audit entries are emitted, remove TODO"
```

---

## Task 12: Un-skip `tests/benchmark/sync_bench_test.go:93`

**Files:**
- Modify: `tests/benchmark/sync_bench_test.go`

- [ ] **Step 1: Replace skip** with an actual benchmark body measuring orchestrator end-to-end run time per repo count.

- [ ] **Step 2: Commit**

```bash
git commit -m "bench(sync): real orchestrator benchmark, remove skip"
```

---

## Task 13: Sweep for any remaining skips

- [ ] **Step 1: Verification command**

```bash
grep -rn "t.Skip" internal/ cmd/ tests/ disabled_tests 2>/dev/null
```

Expected: empty output.

```bash
grep -rn "//go:build disabled\|// +build disabled" internal/ cmd/ tests/ disabled_tests 2>/dev/null
```

Expected: empty output.

- [ ] **Step 2: If anything remains**, fix it with the same patterns.

- [ ] **Step 3: Commit**

```bash
git commit -m "test: confirm zero skipped tests across entire repo" --allow-empty
```

---

## Task 14: Phase 5 acceptance

- [ ] Zero `t.Skip` in any `*_test.go`.
- [ ] Zero `//go:build disabled` tags.
- [ ] `disabled_tests/` directory removed.
- [ ] `go test -race ./...` green for all suites.
- [ ] `tests/security/` runs by default in CI.
- [ ] Every new test has an assertion (no empty bodies).

When every box is checked, Phase 5 ships.
