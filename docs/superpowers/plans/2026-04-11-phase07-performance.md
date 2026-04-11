# Phase 7 — Performance & Responsiveness Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every part of the system flawlessly responsive — lazy initialization everywhere feasible, bounded worker pools on every fan-out, non-blocking selects with timeouts, `sync.Pool` on hot paths, TTL caches, Prometheus histograms at every I/O boundary, and a regression-benchmark store that gates PRs.

**Architecture:** Introduce `internal/lazy` for lazy singletons, `internal/cache` for TTL LRU, `internal/obs` for Prometheus helpers, and `ops/grafana/` dashboards. All existing code paths get instrumented; hot paths get pools.

**Tech Stack:** `hashicorp/golang-lru/v2`, `prometheus/client_golang`, `golang.org/x/sync/semaphore`, `sync.Pool`, `runtime/pprof`.

**Depends on:** Phases 0, 1, 2, 3, 4, 5, 6.

---

## File Structure

**Create:**
- `internal/lazy/lazy.go` + `lazy_test.go` — `Value[T]` generic lazy singleton.
- `internal/cache/lru.go` + `lru_test.go` — TTL wrapper around golang-lru.
- `internal/obs/prometheus.go` — histogram builder helpers.
- `ops/grafana/dashboard.json` — exported Grafana 10 dashboard.
- `ops/grafana/datasource.yaml`
- `tests/bench/baseline/*.json` — perf regression baselines.
- `scripts/perf_regression.go` — compares `go test -bench` output to baseline.

**Modify:**
- `cmd/cli/main.go`, `cmd/server/main.go` — lazy-construct providers, DB, renderers, metrics.
- `internal/services/sync/orchestrator.go` — bounded worker pool, histograms.
- `internal/providers/git/*.go`, `internal/providers/llm/*.go`, `internal/providers/patreon/client.go` — histograms + pools.
- `internal/services/content/generator.go` — semaphore, histograms, TTL cache for verifier.
- `internal/services/filter/repoignore.go` — compiled-glob cache.
- `internal/handlers/*.go` — histograms.
- `internal/middleware/logger.go` — request-id propagation.

---

## Task 1: `internal/lazy/`

**Files:**
- Create: `internal/lazy/lazy.go`, `lazy_test.go`

- [ ] **Step 1: Failing test**

```go
func TestLazyValueComputesOnce(t *testing.T) {
	var calls int32
	v := lazy.New(func() (int, error) {
		atomic.AddInt32(&calls, 1)
		return 42, nil
	})
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := v.Get()
			if err != nil { t.Error(err) }
			if got != 42 { t.Errorf("got %d", got) }
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}
```

- [ ] **Step 2: Implement**

```go
package lazy

import "sync"

type Value[T any] struct {
	once sync.Once
	fn   func() (T, error)
	val  T
	err  error
}

func New[T any](fn func() (T, error)) *Value[T] { return &Value[T]{fn: fn} }

func (v *Value[T]) Get() (T, error) {
	v.once.Do(func() { v.val, v.err = v.fn() })
	return v.val, v.err
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(lazy): generic once-compute lazy singleton with -race test"
```

---

## Task 2: `internal/cache/`

**Files:**
- Create: `internal/cache/lru.go`, `lru_test.go`

- [ ] **Step 1: Failing test**

```go
func TestTTLLRUExpiresEntries(t *testing.T) {
	clock := clockwork.NewFakeClock()
	c := cache.NewTTLLRU[string, int](128, 1*time.Minute).WithClock(clock)
	c.Add("a", 1)
	if v, ok := c.Get("a"); !ok || v != 1 { t.Fatal("miss") }
	clock.Advance(2 * time.Minute)
	if _, ok := c.Get("a"); ok { t.Fatal("expected expired") }
}
```

- [ ] **Step 2: Implement** wrapping `github.com/hashicorp/golang-lru/v2` with `expiresAt` per entry + `Clock` injection.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(cache): TTL LRU wrapper around golang-lru with injectable clock"
```

---

## Task 3: Lazy construction in entrypoints

**Files:**
- Modify: `cmd/cli/main.go`, `cmd/server/main.go`

- [ ] **Step 1: Failing test** — assert that startup does not open DB connections until first use:

```go
func TestServerLazyDBInit(t *testing.T) {
	cfg := Config{DatabaseURL: "file::memory:"}
	var dbOpens int32
	newDatabase = func(cfg Config) (database.DB, error) {
		atomic.AddInt32(&dbOpens, 1)
		return &fakeDB{}, nil
	}
	srv := buildServer(cfg)
	if atomic.LoadInt32(&dbOpens) != 0 {
		t.Fatal("DB opened before first request")
	}
	// first /health request opens it
	_ = issueHealth(srv)
	if atomic.LoadInt32(&dbOpens) != 1 {
		t.Fatal("DB not opened on first request")
	}
}
```

- [ ] **Step 2: Refactor entrypoints** to wrap `newDatabase`, `newOrchestrator`, `newMetricsCollector`, PDF renderer, video pipeline, and LLM provider in `lazy.Value`.

- [ ] **Step 3: Commit**

```bash
git commit -m "perf: lazy init DB, orchestrator, renderers, LLM, metrics"
```

---

## Task 4: Bounded worker pools everywhere

**Files:**
- Modify: `internal/services/sync/orchestrator.go`, `internal/handlers/webhook.go`, `internal/providers/git/*.go`, `internal/providers/llm/fallback.go`, `internal/providers/patreon/client.go`, `internal/services/content/generator.go`.

- [ ] **Step 1: Failing test** — 256 parallel sync jobs. Assert peak concurrent git API calls ≤ `SyncGitConcurrency` (default 8). Use a counting interface spy.

- [ ] **Step 2: Pattern — acquire/release around every fan-out call:**

```go
sem := concurrency.NewSemaphore(int64(cfg.GitConcurrency))
for _, repo := range repos {
	repo := repo
	if err := sem.Acquire(ctx, 1); err != nil { return err }
	g.Go(func() error {
		defer sem.Release(1)
		return p.syncOne(ctx, repo)
	})
}
return g.Wait()
```

- [ ] **Step 3: Commit**

```bash
git commit -m "perf: bounded semaphores on every orchestrator fan-out"
```

---

## Task 5: Non-blocking selects + timeouts

**Files:**
- Modify: every file with a channel send inside a loop.

- [ ] **Step 1: Sweep**

```bash
grep -rn 'chan ' internal/ cmd/ --include='*.go' | grep -v '_test.go'
```

- [ ] **Step 2: For every channel send, add a select with timeout** or `TryEnqueue`-style non-blocking path. Document the drop policy in code comment only when the policy is subtle.

- [ ] **Step 3: Commit**

```bash
git commit -m "perf: non-blocking selects + timeouts on every channel send"
```

---

## Task 6: Context propagation audit

**Files:**
- Modify: every function doing I/O that does not accept `ctx`.

- [ ] **Step 1: Enable `contextcheck` in `.golangci.yml`** (already enabled in Phase 0).
- [ ] **Step 2: Run** `golangci-lint run` and fix every `contextcheck` finding.
- [ ] **Step 3: Commit**

```bash
git commit -m "perf: propagate context through every I/O call"
```

---

## Task 7: `sync.Pool` on hot paths

**Files:**
- Modify: `internal/providers/renderer/*.go` — buffer pools.
- Modify: `internal/handlers/webhook.go` — JSON-decoder pool.
- Modify: `internal/services/content/generator.go` — builder pool.

- [ ] **Step 1: Failing bench** — assert allocs/op drops on hot path:

```go
func BenchmarkMarkdownRenderPooled(b *testing.B) {
	r := NewMarkdownRenderer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Render(context.Background(), Content{Body: "# Hello"})
	}
}
```

Run before and after: `go test -bench=BenchmarkMarkdown -benchmem`. Assert ≥30% allocation reduction post-pool.

- [ ] **Step 2: Pattern**

```go
var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func (r *MarkdownRenderer) Render(...) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	// ...
	return append([]byte(nil), buf.Bytes()...), nil
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "perf: sync.Pool on renderer buffers, webhook decoder, content builder"
```

---

## Task 8: TTL caches

**Files:**
- Modify: `internal/services/content/generator.go` — cache verifier results by fingerprint.
- Modify: `internal/services/filter/repoignore.go` — cache compiled globs.
- Modify: `internal/services/access/tier_mapper.go` — cache tier resolution.

- [ ] **Step 1: Failing test** — assert cache hit on second identical call.
- [ ] **Step 2: Wire `cache.NewTTLLRU`** per use.
- [ ] **Step 3: Commit**

```bash
git commit -m "perf: TTL LRU caches for verifier, compiled globs, tier mapping"
```

---

## Task 9: Prometheus histograms at every I/O boundary

**Files:**
- Modify: `internal/metrics/prometheus.go` — register new histograms.
- Modify: every I/O-doing function to observe duration.

- [ ] **Step 1: Define the histograms**

```go
var (
	GitAPIDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "patreon_git_api_duration_seconds",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"provider", "op"})

	LLMCallDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "patreon_llm_call_duration_seconds",
		Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 40, 60},
	}, []string{"provider", "model"})

	PatreonMutationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "patreon_patreon_mutation_duration_seconds",
		Buckets: []float64{.01, .05, .1, .5, 1, 5, 10},
	}, []string{"op"})

	DBQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "patreon_db_query_duration_seconds",
		Buckets: []float64{.0005, .001, .005, .01, .05, .1, .5, 1},
	}, []string{"store", "op"})
)
```

- [ ] **Step 2: Instrument** every Do/Execute/Render call.

- [ ] **Step 3: Commit**

```bash
git commit -m "perf(obs): prometheus histograms at every I/O boundary"
```

---

## Task 10: Grafana dashboards

**Files:**
- Create: `ops/grafana/dashboard.json`, `ops/grafana/datasource.yaml`

- [ ] **Step 1: Author** a single Grafana 10 dashboard with:
  - Row: HTTP — RPS, error rate, P50/P90/P99 latency per route
  - Row: Sync — repos/s, success rate, queue depth
  - Row: LLM — calls/s, P99 duration, breaker state
  - Row: Patreon — mutations/s, 4xx/5xx rate, idempotency skip rate
  - Row: DB — query rate, lock contention, pool usage

- [ ] **Step 2: Commit**

```bash
git commit -m "perf(obs): Grafana dashboard covering HTTP/sync/LLM/patreon/DB"
```

---

## Task 11: `/debug/pprof/*` behind admin auth

**Files:**
- Modify: `cmd/server/main.go` (already mounted in Phase 2 admin group; verify)

- [ ] **Step 1: Failing test**

```go
func TestPprofRequiresAdminAuth(t *testing.T) {
	srv := newTestServer(t)
	res := issue(srv, "GET", "/debug/pprof/", nil)
	if res.Code != 401 { t.Fatal("expected 401") }
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "perf(obs): pprof endpoints behind middleware.Auth"
```

---

## Task 12: Regression benchmark store

**Files:**
- Create: `tests/bench/baseline/phase7.json`
- Create: `scripts/perf_regression.go`
- Modify: `.github/workflows/ci.yml` — run benchmarks, compare to baseline

- [ ] **Step 1: Snapshot baseline**

```bash
go test -bench=. -benchmem -run=^$ ./... > tests/bench/baseline/phase7.txt
go run scripts/perf_regression.go -input tests/bench/baseline/phase7.txt -out tests/bench/baseline/phase7.json
```

- [ ] **Step 2: Implement comparator** — parse new vs baseline benchmarks; fail if any regresses > 10%.

- [ ] **Step 3: Add CI job**

```yaml
  bench-regression:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26.x" }
      - run: go test -bench=. -benchmem -run=^$ ./... > bench.txt
      - run: go run scripts/perf_regression.go -input bench.txt -baseline tests/bench/baseline/phase7.json
```

- [ ] **Step 4: Commit**

```bash
git commit -m "perf: benchmark baseline + CI regression gate with ±10% budget"
```

---

## Task 13: Phase 7 acceptance

- [ ] Every I/O path has a histogram observation.
- [ ] Every fan-out is bounded by a semaphore.
- [ ] `sync.Pool` on at least renderers, webhook decoder, content builder.
- [ ] TTL caches on verifier, globs, tier mapper.
- [ ] Lazy init for DB, orchestrator, renderers, LLM, metrics verified by test.
- [ ] pprof behind admin auth.
- [ ] Grafana dashboard JSON committed.
- [ ] Regression gate green against baseline.

When every box is checked, Phase 7 ships.
