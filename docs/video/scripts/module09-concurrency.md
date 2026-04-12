# Module 09: Concurrency Patterns

Target length: 12 minutes
Audience: developers

## Scene list

### 00:00 — Why concurrency matters here (60s)
Narration: "The tool fans out across providers, LLMs, and renderers. Without bounds, it overwhelms upstream APIs or leaks goroutines."

### 01:00 — Lifecycle pattern (3m)
[SCENE: IDE showing internal/concurrency/lifecycle.go]
Narration: "Lifecycle owns goroutines via Go(fn). Stop(timeout) cancels the context and waits. Every long-running goroutine uses this."

### 04:00 — Semaphore pattern (2m)
[SCENE: IDE showing internal/concurrency/semaphore.go]
Narration: "Semaphore wraps x/sync/semaphore. Used for LLM concurrency cap and orchestrator fan-out."

### 06:00 — Clock injection (2m)
Narration: "The Clock interface enables deterministic time testing. FakeClock from clockwork advances time without sleeping."

### 08:00 — goleak guards (2m)
[SCENE: IDE showing internal/testhelpers/goleak.go]
Narration: "Every package has a TestMain with goleak.VerifyTestMain. Leaked goroutines fail the test."

### 10:00 — Race detector (90s)
Commands:
    go test -race ./internal/...
Narration: "Every CI run and local scripts/coverage.sh run uses -race. No race condition goes undetected."

### 11:30 — Exercise

## Exercise
1. Read internal/concurrency/ and trace how Lifecycle is used in cmd/server.
2. Write a test that would fail without the semaphore cap.
3. Run `go test -race ./internal/concurrency/...` and observe the output.

## Resources
- internal/concurrency/
- Phase 1 plan: docs/superpowers/plans/2026-04-11-phase01-concurrency.md
