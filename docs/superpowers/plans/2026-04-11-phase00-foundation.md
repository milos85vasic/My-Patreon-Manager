# Phase 0 — Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the coverage gate honest, stand up CI, and scaffold the complete security-scanning pipeline so every subsequent phase ships against a real safety net.

**Architecture:** Replace the bash-floating-point coverage averaging with a tiny Go helper; add four GitHub Actions workflows (ci/security/docs/release); add all linter/scanner config files; build `docker-compose.security.yml` for podman-compatible local scans. No application code changes in this phase — only tooling.

**Tech Stack:** Go 1.26.1, GitHub Actions, golangci-lint, gosec, govulncheck, gitleaks, Semgrep, Snyk, SonarQube, Trivy, syft, Hugo, chromedp (not yet), podman-compose, cosign.

---

## File Structure

**Create:**
- `scripts/coverdiff/main.go` — Go helper replacing bash averaging
- `scripts/coverdiff/main_test.go` — unit tests for helper
- `scripts/coverdiff/go.mod` (optional; may live in repo module)
- `scripts/coverage.sh` — rewritten to call coverdiff
- `.github/workflows/ci.yml`
- `.github/workflows/security.yml`
- `.github/workflows/docs.yml`
- `.github/workflows/release.yml`
- `.golangci.yml`
- `.gitleaks.toml`
- `.pre-commit-config.yaml`
- `.semgrep/rules.yml`
- `sonar-project.properties`
- `.snyk`
- `.trivyignore`
- `docker-compose.security.yml`
- `Dockerfile.security` (multi-runner image if needed) — **optional**, prefer upstream images
- `docs/security/README.md` — one-page scanner usage

**Modify:**
- `.env.example` — add SNYK_TOKEN, SONAR_TOKEN
- `CLAUDE.md` — fix the "100% coverage" claim to describe the new enforcement
- `scripts/coverage.sh` — replace content (see rewrite above)

---

## Task 1: Scaffold the coverdiff Go helper

**Files:**
- Create: `scripts/coverdiff/main.go`
- Create: `scripts/coverdiff/main_test.go`

- [ ] **Step 1: Write the failing test**

```go
// scripts/coverdiff/main_test.go
package main

import (
	"strings"
	"testing"
)

func TestParseCoverageHundred(t *testing.T) {
	in := `github.com/milos85vasic/My-Patreon-Manager/internal/config/config.go:12:	LoadEnv		100.0%
github.com/milos85vasic/My-Patreon-Manager/internal/config/config.go:30:	Validate	100.0%
total:						(statements)	100.0%
`
	pkgs, total, err := parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if total != 100.0 {
		t.Fatalf("total = %v, want 100.0", total)
	}
	if pkgs["github.com/milos85vasic/My-Patreon-Manager/internal/config"] != 100.0 {
		t.Fatalf("pkg pct = %v", pkgs["github.com/milos85vasic/My-Patreon-Manager/internal/config"])
	}
}

func TestParseCoverageBelow(t *testing.T) {
	in := `github.com/milos85vasic/My-Patreon-Manager/internal/foo/foo.go:1:	A	50.0%
github.com/milos85vasic/My-Patreon-Manager/internal/foo/foo.go:2:	B	100.0%
total:						(statements)	75.0%
`
	pkgs, _, err := parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if pkgs["github.com/milos85vasic/My-Patreon-Manager/internal/foo"] != 75.0 {
		t.Fatalf("pkg pct = %v, want 75.0", pkgs["github.com/milos85vasic/My-Patreon-Manager/internal/foo"])
	}
}

func TestEnforceFailsOnLowPackage(t *testing.T) {
	pkgs := map[string]float64{
		"internal/a": 100.0,
		"internal/b": 99.9,
	}
	err := enforce(pkgs, 100.0, 100.0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "internal/b") {
		t.Fatalf("error does not mention offending package: %v", err)
	}
}

func TestEnforcePassesAt100(t *testing.T) {
	pkgs := map[string]float64{
		"internal/a": 100.0,
		"internal/b": 100.0,
	}
	if err := enforce(pkgs, 100.0, 100.0); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd scripts/coverdiff && go test ./...
```

Expected: `main.go` does not exist, `parse` and `enforce` undefined. Compile error.

- [ ] **Step 3: Write minimal implementation**

```go
// scripts/coverdiff/main.go
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

// parse reads `go tool cover -func` output and returns per-package average coverage
// and the reported total.
func parse(r io.Reader) (map[string]float64, float64, error) {
	pkgSum := map[string]float64{}
	pkgCount := map[string]int{}
	var total float64
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return nil, 0, fmt.Errorf("malformed total line: %q", line)
			}
			pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
			v, err := strconv.ParseFloat(pctStr, 64)
			if err != nil {
				return nil, 0, err
			}
			total = v
			continue
		}
		// Example: github.com/x/y/internal/foo/file.go:12: Func  100.0%
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		filePath := line[:colon]
		pkg := path.Dir(filePath)
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
		v, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			continue
		}
		pkgSum[pkg] += v
		pkgCount[pkg]++
	}
	if err := s.Err(); err != nil {
		return nil, 0, err
	}
	out := map[string]float64{}
	for k, sum := range pkgSum {
		out[k] = sum / float64(pkgCount[k])
	}
	return out, total, nil
}

func enforce(pkgs map[string]float64, totalPct, minPct float64) error {
	var bad []string
	for k, v := range pkgs {
		if v+1e-9 < minPct {
			bad = append(bad, fmt.Sprintf("%s: %.2f%%", k, v))
		}
	}
	sort.Strings(bad)
	if totalPct+1e-9 < minPct {
		bad = append(bad, fmt.Sprintf("TOTAL: %.2f%%", totalPct))
	}
	if len(bad) > 0 {
		return errors.New("packages below threshold:\n  " + strings.Join(bad, "\n  "))
	}
	return nil
}

func main() {
	minPct := flag.Float64("min", 100.0, "minimum per-package and total coverage percent")
	flag.Parse()
	pkgs, total, err := parse(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse error:", err)
		os.Exit(2)
	}
	if err := enforce(pkgs, total, *minPct); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("OK total=%.2f%% across %d packages (min=%.1f%%)\n", total, len(pkgs), *minPct)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd scripts/coverdiff && go test ./...
```

Expected: `PASS` — all 4 tests.

- [ ] **Step 5: Commit**

```bash
git add scripts/coverdiff/
git commit -m "feat(coverage): add coverdiff Go helper with unit tests

Replaces the bash floating-point averaging in scripts/coverage.sh
with a deterministic Go tool. Tested with PASS/FAIL fixtures."
```

---

## Task 2: Replace scripts/coverage.sh with coverdiff-based version

**Files:**
- Modify: `scripts/coverage.sh`

- [ ] **Step 1: Write the failing test** — shell test script

Create `scripts/coverage_test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
# smoke test: script runs, produces coverage/coverage.out, exits nonzero on <100
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
cp -r . "$tmpdir/repo"
cd "$tmpdir/repo"
# Should fail loudly because coverage is currently < 100
if bash scripts/coverage.sh > "$tmpdir/out" 2>&1; then
  echo "expected coverage.sh to fail on current 82.7% coverage"
  cat "$tmpdir/out"
  exit 1
fi
grep -q "packages below threshold" "$tmpdir/out" || { echo "missing enforcement message"; cat "$tmpdir/out"; exit 1; }
echo "coverage_test.sh: PASS"
```

- [ ] **Step 2: Run test to verify it fails**

```bash
bash scripts/coverage_test.sh
```

Expected: FAIL (current `coverage.sh` reports success or emits wrong message).

- [ ] **Step 3: Rewrite coverage.sh**

```bash
#!/usr/bin/env bash
# scripts/coverage.sh
# Runs the full test suite under -race with coverage, then enforces 100% per-package
# and 100% total via scripts/coverdiff.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p coverage
OUT="coverage/coverage.out"
MIN="${COVERAGE_MIN:-100.0}"

# Build coverdiff helper (fast; no external deps).
go build -o coverage/coverdiff ./scripts/coverdiff

# Run full test matrix with race detector + coverage across internal/ and cmd/.
CGO_ENABLED=1 go test -race -timeout 10m \
  -covermode=atomic \
  -coverpkg=./internal/...,./cmd/... \
  -coverprofile="$OUT" \
  ./internal/... ./cmd/... ./tests/...

go tool cover -html="$OUT" -o coverage/coverage.html

# Enforce.
go tool cover -func="$OUT" | tee coverage/coverage.func.txt | \
  ./coverage/coverdiff -min "$MIN"
```

- [ ] **Step 4: Run test to verify it passes (once Phase 1–6 raise coverage)**

```bash
bash scripts/coverage_test.sh
```

Expected: this will still fail today (intentional — it is the new gate). Add `COVERAGE_MIN=0` override for Phase 0 smoke so the workflow completes:

```bash
COVERAGE_MIN=0 bash scripts/coverage.sh
```

Expected: exit 0, coverage/coverage.out populated.

- [ ] **Step 5: Commit**

```bash
git add scripts/coverage.sh scripts/coverage_test.sh
git commit -m "feat(coverage): enforce per-package coverage via coverdiff

Rewrites coverage.sh to run under -race, cover internal/+cmd/+tests/,
and hard-fail via coverdiff. Defaults to min=100.0 but honors
COVERAGE_MIN env override for phased ramp-up."
```

---

## Task 3: Add .golangci.yml

**Files:**
- Create: `.golangci.yml`

- [ ] **Step 1: Verification command (this replaces TDD for lint config)**

```bash
golangci-lint run ./... --out-format=colored-line-number || true
```

Expected before: lint runs against defaults (will produce noise).

- [ ] **Step 2: Write the config**

```yaml
# .golangci.yml
run:
  timeout: 5m
  build-tags:
    - integration
    - e2e
  modules-download-mode: readonly

linters:
  disable-all: true
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - typecheck
    - bodyclose
    - contextcheck
    - errorlint
    - exhaustive
    - gocritic
    - gocyclo
    - gofmt
    - gofumpt
    - goimports
    - gosec
    - misspell
    - nakedret
    - nilerr
    - noctx
    - nolintlint
    - prealloc
    - revive
    - unconvert
    - unparam
    - wastedassign

linters-settings:
  gocyclo:
    min-complexity: 15
  govet:
    enable-all: true
    disable:
      - fieldalignment
  revive:
    rules:
      - name: exported
        disabled: false
  gosec:
    excludes:
      - G104 # covered by errcheck
  errorlint:
    errorf: true
    asserts: true
    comparison: true

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
  exclude-rules:
    - path: _test\.go
      linters: [gosec, gocyclo, errcheck, dupl]
```

- [ ] **Step 3: Verify linter accepts config**

```bash
golangci-lint config -c .golangci.yml
```

Expected: no error, lists enabled linters.

- [ ] **Step 4: Commit**

```bash
git add .golangci.yml
git commit -m "chore(lint): add golangci-lint configuration

Enables errcheck, gosec, contextcheck, errorlint, gocritic, gofumpt,
revive and more. Excludes _test.go from gosec/gocyclo/errcheck/dupl."
```

---

## Task 4: Add .gitleaks.toml

**Files:**
- Create: `.gitleaks.toml`

- [ ] **Step 1: Verification command**

```bash
gitleaks detect --config .gitleaks.toml --source . --redact --no-git || true
```

- [ ] **Step 2: Write the config**

```toml
# .gitleaks.toml — tailored to this repo's redaction rules from CLAUDE.md
title = "My Patreon Manager gitleaks config"

[extend]
useDefault = true

[[rules]]
id = "patreon-access-token"
description = "Patreon access token"
regex = '''(?i)patreon.{0,30}?['"][a-zA-Z0-9]{32,}['"]'''
tags = ["key", "patreon"]

[[rules]]
id = "github-pat"
description = "GitHub personal access token"
regex = '''ghp_[0-9A-Za-z]{36}'''
tags = ["key", "github"]

[[rules]]
id = "gitlab-pat"
description = "GitLab personal access token"
regex = '''glpat-[0-9A-Za-z\-_]{20}'''
tags = ["key", "gitlab"]

[[rules]]
id = "sonar-token"
description = "SonarQube token"
regex = '''(?i)sonar.{0,30}?['"][0-9a-f]{40}['"]'''
tags = ["key", "sonar"]

[[rules]]
id = "snyk-token"
description = "Snyk token"
regex = '''(?i)snyk.{0,30}?['"][0-9a-f\-]{36}['"]'''
tags = ["key", "snyk"]

[allowlist]
description = "Placeholders and test fixtures"
regexes = [
  '''your_client_id_here''',
  '''your_client_secret_here''',
  '''test-access-token''',
  '''\*\*\*''',
]
paths = [
  '''\.env\.example$''',
  '''docs/.*\.md$''',
  '''.*_test\.go$''',
]
```

- [ ] **Step 3: Verify config loads**

```bash
gitleaks detect --config .gitleaks.toml --source . --redact --no-git
```

Expected: exit 0 and "no leaks found" — or if findings exist, fix them; they represent a real problem.

- [ ] **Step 4: Commit**

```bash
git add .gitleaks.toml
git commit -m "chore(security): add gitleaks config with project-specific rules

Detects Patreon, GitHub, GitLab, SonarQube, and Snyk tokens. Allowlists
placeholders documented in CLAUDE.md."
```

---

## Task 5: Add .pre-commit-config.yaml

**Files:**
- Create: `.pre-commit-config.yaml`

- [ ] **Step 1: Verification command**

```bash
pre-commit install --install-hooks
pre-commit run --all-files || true
```

- [ ] **Step 2: Write the config**

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-added-large-files
      - id: check-merge-conflict
      - id: check-yaml
      - id: check-json
      - id: mixed-line-ending
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.4
    hooks:
      - id: gitleaks
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.60.1
    hooks:
      - id: golangci-lint
        args: ["--config=.golangci.yml"]
  - repo: https://github.com/returntocorp/semgrep
    rev: v1.86.0
    hooks:
      - id: semgrep
        args: ["--config=.semgrep/rules.yml", "--error"]
  - repo: local
    hooks:
      - id: go-vet
        name: go vet
        entry: go vet ./...
        language: system
        pass_filenames: false
      - id: go-fmt
        name: gofumpt
        entry: gofumpt -l -w .
        language: system
        pass_filenames: false
```

- [ ] **Step 3: Verify**

```bash
pre-commit run --all-files
```

Expected: either PASS or a specific list of fixable formatting issues. Commit with fixes applied.

- [ ] **Step 4: Commit**

```bash
git add .pre-commit-config.yaml
git commit -m "chore(pre-commit): add local + remote hooks

Runs gitleaks, golangci-lint, semgrep, go vet, gofumpt, and stdlib
pre-commit hooks (trailing-whitespace, eof-fixer, large files, merge
conflict, yaml/json)."
```

---

## Task 6: Add .semgrep/rules.yml

**Files:**
- Create: `.semgrep/rules.yml`

- [ ] **Step 1: Verification command**

```bash
semgrep --config .semgrep/rules.yml --error .
```

- [ ] **Step 2: Write the rules**

```yaml
# .semgrep/rules.yml
rules:
  - id: no-context-background-in-handler
    pattern: context.Background()
    paths:
      include:
        - "internal/handlers/**/*.go"
        - "internal/services/sync/**/*.go"
    message: "Use the request/parent context; context.Background() hides cancellation."
    severity: ERROR
    languages: [go]

  - id: missing-body-close
    pattern: |
      $RESP, $ERR := $C.Do($REQ)
      ...
    pattern-not: |
      $RESP, $ERR := $C.Do($REQ)
      ...
      defer $RESP.Body.Close()
      ...
    message: "HTTP response body must be closed via defer."
    severity: ERROR
    languages: [go]

  - id: mutex-across-io
    patterns:
      - pattern-either:
          - pattern: |
              $MU.Lock()
              ...
              os.WriteFile(...)
              ...
              $MU.Unlock()
          - pattern: |
              $MU.Lock()
              ...
              io.Copy(...)
              ...
              $MU.Unlock()
    message: "Do not hold a mutex across I/O."
    severity: ERROR
    languages: [go]

  - id: panic-in-production
    patterns:
      - pattern: panic($X)
    paths:
      include:
        - "internal/**/*.go"
      exclude:
        - "**/*_test.go"
    message: "No panics in production code; return an error."
    severity: ERROR
    languages: [go]

  - id: time-after-in-loop
    pattern: |
      for {
        ...
        <-time.After(...)
        ...
      }
    message: "time.After in loops leaks timers; use time.NewTimer with Stop()."
    severity: WARNING
    languages: [go]
```

- [ ] **Step 3: Run**

```bash
semgrep --config .semgrep/rules.yml --error .
```

Expected: may emit findings that Phase 1 will fix. For Phase 0 acceptance, capture them to `docs/security/semgrep-phase0-baseline.txt`.

```bash
semgrep --config .semgrep/rules.yml . > docs/security/semgrep-phase0-baseline.txt || true
```

- [ ] **Step 4: Commit**

```bash
mkdir -p docs/security
git add .semgrep docs/security/semgrep-phase0-baseline.txt
git commit -m "chore(security): add semgrep custom rules + phase-0 baseline

Rules: no context.Background in handlers, missing response-body close,
mutex-across-IO, panic-in-production, time.After in loops."
```

---

## Task 7: Add sonar-project.properties

**Files:**
- Create: `sonar-project.properties`

- [ ] **Step 1: Write the file**

```properties
sonar.projectKey=milos85vasic_My-Patreon-Manager
sonar.organization=milos85vasic
sonar.projectName=My Patreon Manager
sonar.projectVersion=0.1.0

sonar.sources=cmd,internal
sonar.tests=tests,internal
sonar.test.inclusions=**/*_test.go
sonar.exclusions=**/testdata/**,**/*.pb.go,**/mocks/**,scripts/coverdiff/**

sonar.go.coverage.reportPaths=coverage/coverage.out
sonar.go.tests.reportPaths=coverage/test-report.out
sonar.go.golangci-lint.reportPaths=coverage/golangci-lint.out

sonar.sourceEncoding=UTF-8
```

- [ ] **Step 2: Verify (dry-run)**

```bash
grep -q '^sonar.projectKey=' sonar-project.properties
```

Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
git add sonar-project.properties
git commit -m "chore(sonar): add sonar-project.properties"
```

---

## Task 8: Add .snyk, .trivyignore, .env.example updates

**Files:**
- Create: `.snyk`
- Create: `.trivyignore`
- Modify: `.env.example`

- [ ] **Step 1: Write .snyk**

```yaml
# .snyk
version: v1.25.0
ignore: {}
patch: {}
```

- [ ] **Step 2: Write .trivyignore**

```
# .trivyignore — CVE suppressions require an ADR in docs/adr/
# Format: CVE-YYYY-NNNN (one per line). None as of Phase 0.
```

- [ ] **Step 3: Append to .env.example**

```bash
cat >> .env.example <<'ENV'

# --- Security scanning (Phase 0) ---
# Obtain from https://app.snyk.io/account and https://sonarcloud.io/account/security/
SNYK_TOKEN=your_snyk_token_here
SONAR_TOKEN=your_sonar_token_here
SONAR_HOST_URL=http://localhost:9000
ENV
```

- [ ] **Step 4: Verify**

```bash
grep -q SNYK_TOKEN .env.example
grep -q SONAR_TOKEN .env.example
```

Expected: exit 0 for both.

- [ ] **Step 5: Commit**

```bash
git add .snyk .trivyignore .env.example
git commit -m "chore(security): add .snyk, .trivyignore, env placeholders"
```

---

## Task 9: Add docker-compose.security.yml (podman-compatible)

**Files:**
- Create: `docker-compose.security.yml`

- [ ] **Step 1: Write the compose file**

```yaml
# docker-compose.security.yml — podman-compose compatible
# Usage:
#   podman-compose -f docker-compose.security.yml up -d sonarqube sonarqube-db
#   podman-compose -f docker-compose.security.yml run --rm gosec
#   podman-compose -f docker-compose.security.yml run --rm govulncheck
#   podman-compose -f docker-compose.security.yml run --rm gitleaks
#   podman-compose -f docker-compose.security.yml run --rm trivy-fs
#   podman-compose -f docker-compose.security.yml run --rm semgrep
#   podman-compose -f docker-compose.security.yml run --rm snyk
# All one-shots read the repo via /workspace bind mount.

version: "3.8"

services:
  sonarqube-db:
    image: docker.io/library/postgres:16-alpine
    environment:
      POSTGRES_DB: sonarqube
      POSTGRES_USER: sonar
      POSTGRES_PASSWORD: ${SONARQUBE_DB_PASSWORD:-sonarpw}
    volumes:
      - sonarqube_db:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sonar"]
      interval: 10s
      timeout: 5s
      retries: 10

  sonarqube:
    image: docker.io/library/sonarqube:10-community
    depends_on:
      sonarqube-db:
        condition: service_healthy
    environment:
      SONAR_JDBC_URL: jdbc:postgresql://sonarqube-db:5432/sonarqube
      SONAR_JDBC_USERNAME: sonar
      SONAR_JDBC_PASSWORD: ${SONARQUBE_DB_PASSWORD:-sonarpw}
    ports:
      - "9000:9000"
    volumes:
      - sonarqube_data:/opt/sonarqube/data
      - sonarqube_logs:/opt/sonarqube/logs
      - sonarqube_extensions:/opt/sonarqube/extensions
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:9000/api/system/status | grep -q UP"]
      interval: 30s
      timeout: 10s
      retries: 20

  gosec:
    image: docker.io/securego/gosec:latest
    working_dir: /workspace
    volumes:
      - .:/workspace:z
    entrypoint: ["/bin/gosec", "-fmt=json", "-out=/workspace/coverage/gosec.json", "./..."]

  govulncheck:
    image: docker.io/golang:1.26-alpine
    working_dir: /workspace
    volumes:
      - .:/workspace:z
    entrypoint: ["/bin/sh", "-c", "apk add --no-cache git && go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./... > /workspace/coverage/govulncheck.txt"]

  gitleaks:
    image: docker.io/zricethezav/gitleaks:v8.18.4
    working_dir: /workspace
    volumes:
      - .:/workspace:z
    entrypoint: ["gitleaks", "detect", "--source=/workspace", "--redact", "--report-path=/workspace/coverage/gitleaks.json"]

  trivy-fs:
    image: docker.io/aquasec/trivy:latest
    working_dir: /workspace
    volumes:
      - .:/workspace:z
      - trivy_cache:/root/.cache/trivy
    entrypoint: ["trivy", "fs", "--severity=HIGH,CRITICAL", "--format=json", "-o", "/workspace/coverage/trivy.json", "/workspace"]

  semgrep:
    image: docker.io/returntocorp/semgrep:latest
    working_dir: /workspace
    volumes:
      - .:/workspace:z
    entrypoint: ["semgrep", "--config=/workspace/.semgrep/rules.yml", "--json", "--output=/workspace/coverage/semgrep.json", "/workspace"]

  snyk:
    image: docker.io/snyk/snyk:golang
    working_dir: /workspace
    volumes:
      - .:/workspace:z
      - snyk_cache:/root/.snyk
    environment:
      SNYK_TOKEN: ${SNYK_TOKEN}
    entrypoint: ["snyk", "test", "--json-file-output=/workspace/coverage/snyk.json"]

  syft:
    image: docker.io/anchore/syft:latest
    working_dir: /workspace
    volumes:
      - .:/workspace:z
    entrypoint: ["syft", "dir:/workspace", "-o", "cyclonedx-json=/workspace/coverage/sbom.cdx.json"]

volumes:
  sonarqube_db:
  sonarqube_data:
  sonarqube_logs:
  sonarqube_extensions:
  trivy_cache:
  snyk_cache:
```

- [ ] **Step 2: Verify podman-compose parses it**

```bash
podman-compose -f docker-compose.security.yml config >/dev/null
```

Expected: no error (exit 0). If podman-compose is not installed, this task documents installation in `docs/security/README.md` below.

- [ ] **Step 3: Write docs/security/README.md**

```markdown
# Security scanning

All scans run via `docker-compose.security.yml` against podman-compose. No interactive auth.

## Bring up SonarQube (persistent)

    podman-compose -f docker-compose.security.yml up -d sonarqube sonarqube-db
    # wait for http://localhost:9000/api/system/status == UP

## One-shot scanners

    mkdir -p coverage
    podman-compose -f docker-compose.security.yml run --rm gosec
    podman-compose -f docker-compose.security.yml run --rm govulncheck
    podman-compose -f docker-compose.security.yml run --rm gitleaks
    podman-compose -f docker-compose.security.yml run --rm trivy-fs
    podman-compose -f docker-compose.security.yml run --rm semgrep
    podman-compose -f docker-compose.security.yml run --rm syft
    SNYK_TOKEN=<env> podman-compose -f docker-compose.security.yml run --rm snyk

Reports land in `coverage/` (gosec.json, govulncheck.txt, gitleaks.json, trivy.json, semgrep.json, sbom.cdx.json, snyk.json).

## SonarQube scan

    podman run --rm -v "$PWD":/usr/src sonarsource/sonar-scanner-cli \
      -Dsonar.host.url=http://localhost:9000 \
      -Dsonar.login=$SONAR_TOKEN
```

- [ ] **Step 4: Commit**

```bash
git add docker-compose.security.yml docs/security/README.md
git commit -m "feat(security): scaffold podman-compatible scan pipeline

Adds docker-compose.security.yml with SonarQube (persistent), and
one-shot runners for gosec, govulncheck, gitleaks, trivy, semgrep,
syft, snyk. Documented in docs/security/README.md."
```

---

## Task 10: Add .github/workflows/ci.yml

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Write the workflow**

```yaml
# .github/workflows/ci.yml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read
  checks: write

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"
          cache: true
      - name: go build
        run: go build ./...
      - name: go vet
        run: go vet ./...
      - name: race tests + coverage
        run: bash scripts/coverage.sh
        env:
          COVERAGE_MIN: "0"
      - uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage/

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"
      - uses: golangci/golangci-lint-action@v6
        with:
          version: v1.60.1
          args: --config=.golangci.yml --out-format=github-actions

  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - run: govulncheck ./...

  gosec:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: securego/gosec@master
        with:
          args: "./..."

  gitleaks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  semgrep:
    runs-on: ubuntu-latest
    container: returntocorp/semgrep
    steps:
      - uses: actions/checkout@v4
      - run: semgrep --config=.semgrep/rules.yml --error .
```

- [ ] **Step 2: Verify YAML parses**

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml'))"
```

Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
mkdir -p .github/workflows
git add .github/workflows/ci.yml
git commit -m "ci: add main CI workflow

Jobs: build, lint (golangci-lint), govulncheck, gosec, gitleaks,
semgrep. Coverage is run via scripts/coverage.sh with
COVERAGE_MIN=0 until Phase 6 raises coverage to 100%."
```

---

## Task 11: Add .github/workflows/security.yml

**Files:**
- Create: `.github/workflows/security.yml`

- [ ] **Step 1: Write the workflow**

```yaml
# .github/workflows/security.yml
name: Security
on:
  push:
    branches: [main]
  schedule:
    - cron: "0 3 * * *"
  workflow_dispatch:

permissions:
  contents: read
  security-events: write

jobs:
  snyk:
    runs-on: ubuntu-latest
    if: ${{ github.event_name == 'push' || github.event_name == 'schedule' }}
    steps:
      - uses: actions/checkout@v4
      - uses: snyk/actions/golang@master
        continue-on-error: true
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
        with:
          args: --sarif-file-output=snyk.sarif
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: snyk.sarif

  sonarqube:
    runs-on: ubuntu-latest
    if: ${{ github.event_name == 'push' }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"
      - run: bash scripts/coverage.sh
        env:
          COVERAGE_MIN: "0"
      - uses: SonarSource/sonarqube-scan-action@v2
        env:
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
          SONAR_HOST_URL: ${{ secrets.SONAR_HOST_URL }}

  trivy-fs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aquasecurity/trivy-action@0.24.0
        with:
          scan-type: fs
          severity: HIGH,CRITICAL
          format: sarif
          output: trivy.sarif
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: trivy.sarif

  syft-sbom:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: anchore/sbom-action@v0
        with:
          format: cyclonedx-json
          output-file: sbom.cdx.json
      - uses: actions/upload-artifact@v4
        with:
          name: sbom
          path: sbom.cdx.json

  codeql:
    runs-on: ubuntu-latest
    permissions:
      security-events: write
    strategy:
      matrix:
        language: [go]
    steps:
      - uses: actions/checkout@v4
      - uses: github/codeql-action/init@v3
        with:
          languages: ${{ matrix.language }}
      - uses: github/codeql-action/autobuild@v3
      - uses: github/codeql-action/analyze@v3
```

- [ ] **Step 2: Verify parse**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/security.yml'))"
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/security.yml
git commit -m "ci(security): add Snyk, SonarQube, Trivy, syft, CodeQL workflow"
```

---

## Task 12: Add .github/workflows/docs.yml

**Files:**
- Create: `.github/workflows/docs.yml`

- [ ] **Step 1: Write the workflow**

```yaml
# .github/workflows/docs.yml
name: Docs
on:
  push:
    branches: [main]
    paths:
      - "docs/**"
      - "*.md"
      - ".github/workflows/docs.yml"
  pull_request:
    paths:
      - "docs/**"
      - "*.md"

jobs:
  markdownlint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: DavidAnson/markdownlint-cli2-action@v16
        with:
          globs: "**/*.md"
          config: ".markdownlint.jsonc"

  link-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: lycheeverse/lychee-action@v1
        with:
          args: --no-progress --verbose --exclude-mail './**/*.md'

  hugo:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive
      - uses: peaceiris/actions-hugo@v3
        with:
          hugo-version: "latest"
          extended: true
      - name: build
        working-directory: docs/website
        run: hugo --gc --minify --baseURL "https://milos85vasic.github.io/My-Patreon-Manager/"
      - uses: actions/upload-artifact@v4
        with:
          name: site
          path: docs/website/public
```

- [ ] **Step 2: Add `.markdownlint.jsonc`**

```jsonc
{
  "default": true,
  "MD013": false,
  "MD024": { "siblings_only": true },
  "MD033": false,
  "MD041": false
}
```

- [ ] **Step 3: Verify parse**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/docs.yml'))"
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/docs.yml .markdownlint.jsonc
git commit -m "ci(docs): add markdownlint, lychee link-check, hugo build"
```

---

## Task 13: Add .github/workflows/release.yml

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Write the workflow**

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: ["v*"]

permissions:
  contents: write
  packages: write
  id-token: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"
      - uses: sigstore/cosign-installer@v3
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          COSIGN_EXPERIMENTAL: "1"
```

- [ ] **Step 2: Add `.goreleaser.yaml`**

```yaml
# .goreleaser.yaml
version: 2
before:
  hooks:
    - go mod tidy
builds:
  - id: patreon-manager
    main: ./cmd/cli
    binary: patreon-manager
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
  - id: patreon-manager-server
    main: ./cmd/server
    binary: patreon-manager-server
    env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]
archives:
  - id: default
    formats: [tar.gz]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
checksum:
  name_template: "checksums.txt"
signs:
  - artifacts: checksum
    cmd: cosign
    args: ["sign-blob", "--yes", "--output-signature=${signature}", "${artifact}"]
release:
  draft: false
  prerelease: auto
```

- [ ] **Step 3: Verify parse**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml')); yaml.safe_load(open('.goreleaser.yaml'))"
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/release.yml .goreleaser.yaml
git commit -m "ci(release): add goreleaser + cosign signed release workflow"
```

---

## Task 14: Fix the CLAUDE.md coverage claim

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Show the current line**

```bash
grep -n "100%" CLAUDE.md
```

Expected: line referring to `scripts/coverage.sh` enforcement.

- [ ] **Step 2: Edit the claim**

Replace the paragraph starting `scripts/coverage.sh enforces 100% per-package coverage...` with:

```markdown
`scripts/coverage.sh` runs `go test -race` with `-coverpkg=./internal/...,./cmd/...`, writes HTML + func coverage reports to `coverage/`, and hard-fails via `scripts/coverdiff` if any package or the total drops below `COVERAGE_MIN` (default **100.0**, lowerable during phased ramp-up with `COVERAGE_MIN=<n>`). Run it before committing.
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs(claude): reflect actual coverage enforcement semantics"
```

---

## Task 15: Phase 0 smoke — run the whole pipeline once

- [ ] **Step 1: Run coverage with ramp-down**

```bash
COVERAGE_MIN=0 bash scripts/coverage.sh
```

Expected: green, `coverage/coverage.out` populated, `coverage/coverage.html` generated.

- [ ] **Step 2: Run linters**

```bash
golangci-lint run --config .golangci.yml ./... || true
```

Expected: may emit findings — capture to `coverage/golangci-lint.out` for SonarQube ingestion and Phase 1–6 remediation.

```bash
golangci-lint run --config .golangci.yml ./... --out-format=checkstyle > coverage/golangci-lint.out || true
```

- [ ] **Step 3: Run semgrep**

```bash
semgrep --config .semgrep/rules.yml . > coverage/semgrep.txt || true
```

- [ ] **Step 4: Run gitleaks**

```bash
gitleaks detect --config .gitleaks.toml --source . --redact --no-git --report-path coverage/gitleaks.json
```

Expected: exit 0 (repo clean after CLAUDE.md redaction rules).

- [ ] **Step 5: Commit smoke artifacts**

```bash
git add coverage/.gitkeep 2>/dev/null || true
git commit -m "chore(phase0): smoke pipeline complete" --allow-empty
```

---

## Task 16: Phase 0 acceptance

- [ ] `scripts/coverage.sh` exits non-zero when any package is under threshold (default 100).
- [ ] `go build ./...` and `go vet ./...` green.
- [ ] `golangci-lint run` parses `.golangci.yml`.
- [ ] `.github/workflows/{ci,security,docs,release}.yml` parse as valid YAML.
- [ ] `docker-compose.security.yml` parses under `podman-compose -f ... config`.
- [ ] `docs/security/README.md` describes every scanner.
- [ ] `CLAUDE.md` reflects actual behaviour.
- [ ] `.env.example` has `SNYK_TOKEN`, `SONAR_TOKEN` placeholders.
- [ ] Phase-0 baselines captured under `coverage/` and `docs/security/semgrep-phase0-baseline.txt`.

When every box is checked, Phase 0 is done and Phases 1, 3, 4 may start in parallel worktrees.
