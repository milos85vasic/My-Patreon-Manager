# Tutorial: Running and Writing Tests

This tutorial covers every test type in the project, how to run each one, how to write new tests, and how to read coverage reports.

## Test Types at a Glance

| Type | Location | Command | Purpose |
|------|----------|---------|---------|
| Unit | `internal/*_test.go` | `go test ./internal/...` | Function-level correctness |
| Race | same files | `go test -race ./...` | Data race detection |
| Integration | `tests/integration/` | `go test ./tests/integration/...` | Cross-component workflows |
| E2E | `tests/e2e/` | `go test ./tests/e2e/...` | Full CLI pipeline |
| Fuzz | `tests/fuzz/` | `go test -fuzz=... ./tests/fuzz/...` | Input fuzzing |
| Benchmark | `tests/benchmark/` | `go test -bench=. ./tests/benchmark/...` | Performance measurement |
| Chaos | `tests/chaos/` | `go test ./tests/chaos/...` | Failure resilience |
| Stress | `tests/stress/` | `go test ./tests/stress/...` | Load handling |
| DDoS | `tests/ddos/` | `go test ./tests/ddos/...` | Flood resilience |
| Contract | `tests/contract/` | `go test ./tests/contract/...` | Mock-interface parity |
| Monitoring | `tests/monitoring/` | `go test ./tests/monitoring/...` | Metrics emission |
| Security | `tests/security/` | `go test ./tests/security/...` | Auth + signature |
| Leak | `internal/*/testmain_test.go` | automatic via TestMain | Goroutine leak detection |

## Step 1: Run all tests

```bash
go test -race -count=1 ./... -timeout 15m
```

Expected: 50 packages, all `ok`. Takes ~2 minutes.

## Step 2: Run a single package

```bash
go test -race -v ./internal/services/sync/... -timeout 2m
```

The `-v` flag shows each test name and result.

## Step 3: Run a single test

```bash
go test -race -v ./internal/services/sync/ -run TestDedupCloseStopsGoroutine
```

## Step 4: Check coverage for a package

```bash
go test -coverprofile=tmp.cov ./internal/handlers/...
go tool cover -func=tmp.cov | grep -v 100.0%
```

This shows every function below 100% coverage. To see the HTML report:

```bash
go tool cover -html=tmp.cov -o tmp.html
# Open tmp.html in your browser — green=covered, red=uncovered
```

## Step 5: Run the full coverage suite

```bash
COVERAGE_MIN=0 bash scripts/coverage.sh
```

Outputs:
- `coverage/coverage.out` — raw profile
- `coverage/coverage.html` — visual report
- `coverage/coverage.func.txt` — per-function percentages

Open `coverage/coverage.html` in your browser to see exactly which lines are covered.

## Step 6: Run fuzz tests

```bash
# 30-second fuzz run
go test -fuzz=FuzzRepoignoreMatch -fuzztime=30s ./tests/fuzz/...
```

Expected:
```
fuzz: elapsed: 30s, execs: 64448 (2148/sec), new interesting: 12
PASS
```

If a crash is found, the failing input is saved to `testdata/fuzz/` for regression testing.

## Step 7: Run benchmarks

```bash
go test -bench=. -benchmem -run=^$ ./tests/benchmark/... ./internal/...
```

Expected output:
```
BenchmarkFullSync-8          1000    1234567 ns/op    45678 B/op    123 allocs/op
BenchmarkMarkdownRender-8   50000     23456 ns/op     4567 B/op     12 allocs/op
```

Compare against baseline:
```bash
# Save current results
go test -bench=. -benchmem -run=^$ ./... > bench_current.txt

# Compare (requires benchstat)
go install golang.org/x/perf/cmd/benchstat@latest
benchstat tests/bench/baseline/phase7.txt bench_current.txt
```

## Writing New Tests

### Unit test pattern

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "hello", "HELLO", false},
        {"empty input", "", "", false},
        {"error case", "bad", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### Testing HTTP handlers

```go
func TestHealthEndpoint(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.GET("/health", handler.Handle)

    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "ok")
}
```

### Testing with sqlmock

```go
func TestStoreCreate(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    mock.ExpectExec("INSERT INTO").
        WithArgs("id-1", "value").
        WillReturnResult(sqlmock.NewResult(1, 1))

    store := NewStore(db)
    err := store.Create(context.Background(), "id-1", "value")
    assert.NoError(t, err)
    assert.NoError(t, mock.ExpectationsWereMet())
}
```

### Testing with httptest (for providers)

```go
func TestProviderHandles500(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(500)
    }))
    defer srv.Close()

    provider := NewProvider(Config{BaseURL: srv.URL})
    _, err := provider.ListRepos(context.Background(), "org")
    assert.Error(t, err)
}
```

### Adding goleak to a new package

Every package must have a `testmain_test.go`:

```go
package mypackage

import (
    "testing"
    "go.uber.org/goleak"
    "github.com/milos85vasic/My-Patreon-Manager/internal/testhelpers"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m, testhelpers.GoleakIgnores()...)
}
```

### Writing a fuzz test

```go
func FuzzMyParser(f *testing.F) {
    f.Add("valid input")
    f.Add("")
    f.Add("edge\x00case")
    f.Fuzz(func(t *testing.T, input string) {
        // Must not panic
        _ = MyParser(input)
    })
}
```

## Troubleshooting test failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| `race detected` | Concurrent access without sync | Add mutex or use atomic |
| `goleak: found unexpected goroutines` | Goroutine not stopped | Add `Close()` or `defer cancel()` |
| `timeout` | Test takes too long | Check for infinite loops or missing context cancellation |
| `sqlmock: expectations were not met` | Missing mock setup | Add the expected SQL call to mock |
| `FAIL` with no output | Panic in test setup | Run with `-v` to see the panic |
