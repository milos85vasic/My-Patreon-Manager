# Local Verification Guide

Run every check locally before publishing. Each section is independent and can be run in any order.

## Prerequisites

```bash
# Required
go version          # Go 1.26.1+
podman --version    # Podman 4+ (for security scanners)

# Optional (installed via go install if missing)
golangci-lint --version   # v1.60+
gitleaks version          # v8.18+
semgrep --version         # v1.50+
```

## 1. Build

```bash
go build ./...
```

Expected: exit 0, no output. Two binaries buildable: `./cmd/cli` and `./cmd/server`.

## 2. Static analysis

```bash
go vet ./...
```

Expected: exit 0, no output.

## 3. Lint

```bash
golangci-lint run --config .golangci.yml ./...
```

Expected: exit 0 or known baseline findings (see `docs/security/baselines/phase0-golangci-lint.checkstyle.xml`).

## 4. Tests (with race detector)

```bash
go test -race -count=1 ./... -timeout 15m
```

Expected: 50 packages, all `ok`, zero `FAIL`. Takes ~2 minutes.

## 5. Coverage

```bash
COVERAGE_MIN=0 bash scripts/coverage.sh
```

Expected: `OK total=XX.XX% across 23 packages`. Produces:
- `coverage/coverage.out` — raw coverage profile
- `coverage/coverage.html` — visual HTML report (open in browser)
- `coverage/coverage.func.txt` — per-function coverage

To enforce 100% (Phase 6+ target):

```bash
bash scripts/coverage.sh
```

## 6. Secret scanning

```bash
gitleaks detect --config .gitleaks.toml --source . --redact --no-git
```

Expected: `no leaks found`.

## 7. Custom security rules

```bash
semgrep --config .semgrep/rules.yml .
```

Expected: findings report. Baseline captured in `docs/security/baselines/phase0-semgrep.txt`.

## 8. Fuzz testing

```bash
go test -fuzz=FuzzRepoignoreMatch -fuzztime=30s ./tests/fuzz/...
```

Expected: `ok` with thousands of executions, zero crashes.

## 9. Benchmarks

```bash
go test -bench=. -benchmem -run=^$ ./tests/benchmark/... ./internal/...
```

Expected: benchmark results with ns/op and allocs/op.

## 10. Container build

```bash
podman build -t patreon-manager:local .
```

Expected: multi-stage build succeeds, image tagged.

## 11. Container security scan

```bash
podman-compose -f docker-compose.security.yml run --rm trivy-fs
podman-compose -f docker-compose.security.yml run --rm gosec
podman-compose -f docker-compose.security.yml run --rm gitleaks
```

Expected: reports in `coverage/trivy.json`, `coverage/gosec.json`, `coverage/gitleaks.json`.

## 12. Server smoke test

```bash
cp .env.example .env
# Edit .env with test values
go run ./cmd/server &
sleep 2

# Health
curl -s http://localhost:8080/health | jq .

# Metrics
curl -s http://localhost:8080/metrics | head -20

# Admin (requires ADMIN_KEY set in .env)
curl -s -H "X-Admin-Key: $ADMIN_KEY" http://localhost:8080/admin/sync-status | jq .

# Cleanup
kill %1
```

## 13. CLI smoke test

```bash
go run ./cmd/cli validate
go run ./cmd/cli scan --dry-run
go run ./cmd/cli sync --dry-run
```

## 14. Documentation build

```bash
cd docs/website && hugo --gc --minify
```

Expected: `public/` directory generated with the static site.

## 15. Full pre-publish checklist

```bash
bash scripts/release/verify_all.sh v0.2.0
```

Expected: `=== RELEASE GATE: PASSED ===` with evidence in `docs/releases/v0.2.0/evidence/`.

## Quick one-liner (runs everything except containers)

```bash
go build ./... && go vet ./... && go test -race -count=1 ./... -timeout 15m && gitleaks detect --config .gitleaks.toml --source . --redact --no-git && echo "ALL CHECKS PASSED"
```
