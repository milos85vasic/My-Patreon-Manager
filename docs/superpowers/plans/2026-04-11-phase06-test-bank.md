# Phase 6 — Test Bank Expansion Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Raise test coverage to true 100% per package for `internal/` and `cmd/`, and add every test type the spec calls for: fuzz, property-based, golden-file, mutation, contract, leak, monitoring/metrics assertions, plus expanded chaos + stress suites.

**Architecture:** The test bank is organized by type under `tests/<type>/...`. Each package also carries co-located `_test.go` unit tests. A shared `tests/testutil/` library provides fixtures, fake clocks, testcontainers helpers, vegeta wrappers, and assertion helpers reused across suites.

**Tech Stack:** `testing`, `testify`, `go.uber.org/goleak`, `pgregory.net/rapid`, `github.com/stretchr/testify`, `github.com/tsenart/vegeta/v12`, `github.com/avito-tech/go-mutesting` (wrapper), `github.com/chromedp/chromedp`, `testcontainers-go`.

**Depends on:** Phases 0, 1, 2, 3, 4, 5.

---

## File Structure

**Create:**
- `tests/testutil/fakes.go`, `testcontainers.go`, `goldens.go`, `metrics.go`, `ratelimit.go`, `fixtures.go`
- `tests/fuzz/<package>/fuzz_test.go` — one per fuzz target
- `tests/property/<package>/property_test.go`
- `tests/golden/<package>/...`
- `tests/contract/<interface>_test.go`
- `tests/mutation/run_mutation.sh`
- `tests/monitoring/metrics_assert_test.go`
- `tests/slo/slo_test.go`

**Modify:**
- Every `_test.go` under `internal/` and `cmd/` missing coverage
- `scripts/coverage.sh` — remove COVERAGE_MIN override once 100% is real

---

## Task 1: Shared `tests/testutil/` library

**Files:**
- Create: `tests/testutil/testutil.go` + sub-files.

- [ ] **Step 1: Define the helpers** as a single package:

```go
// tests/testutil/fakes.go
package testutil

import (
	"github.com/jonboulle/clockwork"
)

func NewFakeClock() clockwork.FakeClock { return clockwork.NewFakeClock() }
```

```go
// tests/testutil/goldens.go
package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func AssertGolden(t *testing.T, relPath string, got []byte) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") != "" {
		_ = os.MkdirAll(filepath.Dir(relPath), 0o755)
		_ = os.WriteFile(relPath, got, 0o644)
		return
	}
	want, err := os.ReadFile(relPath)
	if err != nil { t.Fatalf("read golden: %v", err) }
	if string(want) != string(got) {
		t.Fatalf("golden mismatch for %s", relPath)
	}
}

func Hash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
```

```go
// tests/testutil/metrics.go
package testutil

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func AssertCounterIncremented(t *testing.T, c prometheus.Counter, want float64) {
	t.Helper()
	got := testutil.ToFloat64(c)
	if got != want { t.Fatalf("counter = %v, want %v", got, want) }
}

func AssertHistogramHasObservations(t *testing.T, h prometheus.Histogram) {
	t.Helper()
	name := h.Desc().String()
	if !strings.Contains(name, "patreon") { return }
	// use collection scan:
	// left as exercise — projected via registry.Gather
}
```

```go
// tests/testutil/testcontainers.go
package testutil

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func StartPostgres(t *testing.T) (dsn string, stop func()) {
	t.Helper()
	ctx := context.Background()
	c, err := postgres.Run(ctx, "docker.io/library/postgres:16-alpine",
		postgres.WithDatabase("pm"),
		postgres.WithUsername("pm"),
		postgres.WithPassword("pm"),
	)
	if err != nil { t.Skip("testcontainers unavailable:", err) }
	dsn, _ = c.ConnectionString(ctx, "sslmode=disable")
	return dsn, func() { _ = c.Terminate(ctx) }
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "test(util): shared fakes, goldens, metrics assert, testcontainers helpers"
```

---

## Task 2: Fuzz tests

**Files:** one per target.

- [ ] **Step 1: Fuzz targets**
  - `tests/fuzz/repoignore_fuzz_test.go` — feed random glob patterns, assert parser never panics and `Match` is total.
  - `tests/fuzz/webhook_signature_fuzz_test.go` — random payloads/signatures, assert constant-time compare never panics.
  - `tests/fuzz/config_loader_fuzz_test.go` — random env strings, assert validation errors are wrapped, not raw.
  - `tests/fuzz/url_redact_fuzz_test.go` — assert `RedactURL` never returns a string containing substrings matching the redacted token.
  - `tests/fuzz/template_render_fuzz_test.go` — feed random template bodies, assert errors are wrapped.

Example:

```go
// tests/fuzz/repoignore_fuzz_test.go
package fuzz

import (
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/services/filter"
)

func FuzzRepoignoreParser(f *testing.F) {
	for _, seed := range []string{"**/*.go", "!vendor/**", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, pat string) {
		r := filter.NewRepoignore()
		_ = r.Add(pat)
		_ = r.Match("any/path")
	})
}
```

- [ ] **Step 2: Add CI job** to `.github/workflows/ci.yml`:

```yaml
  fuzz:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26.x" }
      - name: Fuzz short
        run: |
          for target in $(go test -list Fuzz ./tests/fuzz/... 2>/dev/null | grep ^Fuzz); do
            go test -run=^$ -fuzz="^$target$" -fuzztime=30s ./tests/fuzz/...
          done
```

- [ ] **Step 3: Commit**

```bash
git commit -m "test(fuzz): add fuzz targets for parsers, signatures, templates, redaction"
```

---

## Task 3: Property-based tests

**Files:** `tests/property/<package>/property_test.go`

- [ ] **Step 1: Targets**
  - `filter_property_test.go` — `Match(p)` is monotonic with respect to pattern specificity.
  - `fingerprint_property_test.go` — content fingerprint is deterministic and idempotent.
  - `tier_mapper_property_test.go` — tier mapping respects monotonic order of access levels.
  - `backoff_property_test.go` — exponential backoff stays within `[base, max]` bounds for any call count.

Example:

```go
package property

import (
	"testing"

	"pgregory.net/rapid"
)

func TestFingerprintIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		body := rapid.String().Draw(t, "body")
		a := content.Fingerprint(body)
		b := content.Fingerprint(body)
		if a != b { t.Fatalf("non-deterministic: %s != %s", a, b) }
	})
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "test(property): rapid-based invariants for filter, fingerprint, tier, backoff"
```

---

## Task 4: Golden-file tests

**Files:** `testdata/golden/...`

- [ ] **Step 1: Golden targets**
  - `testdata/golden/markdown/hello.md`
  - `testdata/golden/html/hello.html`
  - `testdata/golden/pdf/hello.pdf.hash`
  - `testdata/golden/video/slides/module1.png.hash`
  - `testdata/golden/video/waveform.png.hash`
  - `testdata/golden/openapi/routes.json` (from OpenAPI → code emitter)

- [ ] **Step 2: Add regeneration helper** `scripts/update_goldens.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
UPDATE_GOLDEN=1 go test ./internal/providers/renderer/... -run TestGolden
```

- [ ] **Step 3: Commit**

```bash
git commit -m "test(golden): goldens for markdown, html, pdf, video slides, openapi"
```

---

## Task 5: Contract tests for mocks

**Files:** `tests/contract/*_test.go`

- [ ] **Step 1: Compile-time assertions** — for every mock in the codebase, assert interface satisfaction at package init:

```go
var _ git.RepositoryProvider = (*mocks.FakeGitHubProvider)(nil)
var _ llm.LLMProvider         = (*mocks.FakeLLM)(nil)
var _ patreon.Client          = (*mocks.FakePatreon)(nil)
```

- [ ] **Step 2: Behavioral parity** — for each mock, a shared test table that both the mock and real implementation must satisfy. Real implementation tests hit containerized fixtures; mock tests hit the mock. Both must pass.

- [ ] **Step 3: Commit**

```bash
git commit -m "test(contract): compile-time and behavioral mock/real parity"
```

---

## Task 6: Leak detection

**Files:** add `TestMain` to every `_test.go` package.

- [ ] **Step 1: Generator script** — `scripts/add_goleak_main.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
for pkg in $(go list ./internal/... ./cmd/...); do
  dir=$(go list -f '{{.Dir}}' "$pkg")
  main_file="$dir/testmain_test.go"
  if [ -f "$main_file" ]; then continue; fi
  cat > "$main_file" <<'EOF'
package $PKG

import (
	"testing"

	"go.uber.org/goleak"
	"github.com/milos85vasic/My-Patreon-Manager/tests/leaks"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, leaks.Ignores()...)
}
EOF
  pkgname=$(go list -f '{{.Name}}' "$pkg")
  sed -i "s/\$PKG/$pkgname/" "$main_file"
done
```

- [ ] **Step 2: Run** `bash scripts/add_goleak_main.sh` and commit the generated files.

- [ ] **Step 3: Commit**

```bash
git commit -m "test(leaks): goleak TestMain in every internal/ and cmd/ package"
```

---

## Task 7: Monitoring/metrics assertion tests

**Files:** `tests/monitoring/metrics_assert_test.go`

- [ ] **Step 1: Failing test** — for each orchestrator path, run one iteration against a fake Prometheus registry, then assert counters/histograms present with expected labels:

```go
func TestOrchestratorEmitsMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.New(reg)
	orch := buildTestOrchestratorWithMetrics(t, m)
	_ = orch.RunSync(context.Background())

	fams, _ := reg.Gather()
	names := map[string]bool{}
	for _, f := range fams { names[f.GetName()] = true }
	for _, want := range []string{
		"patreon_sync_repos_total",
		"patreon_sync_duration_seconds",
		"patreon_webhook_received_total",
		"patreon_llm_requests_total",
		"patreon_patreon_publish_total",
	} {
		if !names[want] { t.Errorf("missing metric %s", want) }
	}
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "test(monitoring): assert every orchestrator path emits expected metrics"
```

---

## Task 8: SLO tests

**Files:** `tests/slo/slo_test.go`

- [ ] **Step 1: Failing test** — run integration scenarios and assert P99 latency thresholds per route:

```go
var SLOs = map[string]time.Duration{
	"/health":          1 * time.Millisecond,
	"/webhook/github": 50 * time.Millisecond,
	"/download":        100 * time.Millisecond,
}

func TestSLOLatencies(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	for route, budget := range SLOs {
		latencies := measureRoute(t, srv, route, 500)
		p99 := percentile(latencies, 0.99)
		if p99 > budget {
			t.Errorf("%s P99 %v > %v", route, p99, budget)
		}
	}
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "test(slo): P99 latency gates per HTTP route"
```

---

## Task 9: Mutation testing wrapper

**Files:** `tests/mutation/run_mutation.sh`, `.github/workflows/mutation.yml`

- [ ] **Step 1: Script**

```bash
#!/usr/bin/env bash
# tests/mutation/run_mutation.sh
set -euo pipefail
go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@latest
go-mutesting --disable=branch/case ./internal/... \
  --exec="go test -count=1 -timeout 60s ./..." \
  > mutation.log
survived=$(grep -c "^PASS " mutation.log || true)
total=$(grep -c "^" mutation.log || true)
if [ "$total" -gt 0 ]; then
  rate=$(( 100 * survived / total ))
  echo "survival rate: $rate%"
  if [ "$rate" -gt 5 ]; then
    echo "ERROR: mutation survival > 5%"
    exit 1
  fi
fi
```

- [ ] **Step 2: Workflow** `.github/workflows/mutation.yml`:

```yaml
name: Mutation
on:
  schedule:
    - cron: "0 4 * * 0"
  workflow_dispatch:
jobs:
  mutation:
    runs-on: ubuntu-latest
    timeout-minutes: 120
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26.x" }
      - run: bash tests/mutation/run_mutation.sh
```

- [ ] **Step 3: Commit**

```bash
git commit -m "test(mutation): nightly go-mutesting with 5% survival gate"
```

---

## Task 10: Close every coverage gap

**Files:** every package under `internal/...` / `cmd/...` below 100%.

- [ ] **Step 1: Generate the gap report**

```bash
COVERAGE_MIN=100 bash scripts/coverage.sh || true
cat coverage/coverage.func.txt | grep -v 100.0% > coverage/gaps.txt
```

- [ ] **Step 2: For every entry in `coverage/gaps.txt`, write a unit test** that exercises the missing branch. Use table-driven tests.

- [ ] **Step 3: Re-run** until empty.

- [ ] **Step 4: Flip the gate**

```bash
# scripts/coverage.sh keeps default COVERAGE_MIN=100.0 (already the default)
# remove the COVERAGE_MIN=0 overrides from .github/workflows/*.yml
```

- [ ] **Step 5: Commit**

```bash
git commit -m "test: close every coverage gap to reach true 100% per package"
```

---

## Task 11: Expanded chaos suite

**Files:** `tests/chaos/*_test.go` (new + modified)

- [ ] **Step 1: Add scenarios**
  - `chaos_network_partition_test.go` — drop 50% of packets to LLM provider via httptest handler, assert retry + fallback + breaker behavior.
  - `chaos_patreon_429_test.go` — 429 for 5 s, assert backoff, then success.
  - `chaos_db_lock_contention_test.go` — 16 writers, assert eventual convergence.
  - `chaos_disk_full_test.go` — fake filesystem with 0 free bytes, assert graceful error.
  - `chaos_fd_exhaustion_test.go` — `ulimit`-simulated FD exhaustion (via rlimit interface injection).

- [ ] **Step 2: Commit**

```bash
git commit -m "test(chaos): network partition, Patreon 429, DB contention, disk full, FD exhaust"
```

---

## Task 12: Expanded stress/load suite

**Files:** `tests/stress/*_test.go`

- [ ] **Step 1: Scenarios**
  - Orchestrator sync with N ∈ {1, 4, 16, 64, 256} concurrent repos.
  - Sustained 10k req/s webhook flood for 60 s.
  - Burst of 50k events in 5 s, assert queue drop-rate within SLA.
  - PostgreSQL with 32 concurrent transactions.

Each test asserts: no goroutine leaks, P99 latency within budget, zero data corruption.

- [ ] **Step 2: Commit**

```bash
git commit -m "test(stress): 10k req/s webhook flood + N-parallel sync scenarios"
```

---

## Task 13: Phase 6 acceptance

- [ ] `bash scripts/coverage.sh` green with `COVERAGE_MIN=100.0`.
- [ ] Every test type in the spec §8 matrix has at least one test.
- [ ] `go test -race ./...` green.
- [ ] `go.uber.org/goleak` clean on every package.
- [ ] Mutation run (nightly) shows ≤ 5% survival.
- [ ] Zero skipped tests (`grep -rn "t.Skip" internal cmd tests` returns nothing).
- [ ] Every orchestrator path has a metrics assertion.
- [ ] SLO gates green under stress.

When every box is checked, Phase 6 ships.
