# Phase 8 — Security Scanning & Remediation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring up `docker-compose.security.yml` under podman, run every scanner (gosec, govulncheck, golangci-lint, semgrep, trivy fs, trivy image, gitleaks, syft, snyk test, snyk container test, SonarQube), triage every finding, fix every HIGH/CRITICAL, and freeze the scan pipeline into CI with hard-fail gates.

**Architecture:** Scans are orchestrated from a single helper script `scripts/security/run_all.sh` that runs every one-shot runner from `docker-compose.security.yml`, waits for SonarQube to come up, runs the SonarQube scanner, and writes a unified report to `docs/security/remediation-log.md`. Every finding tracked through to closure.

**Tech Stack:** podman-compose, gosec, govulncheck, golangci-lint, semgrep, trivy, gitleaks, syft, snyk, sonarqube.

**Depends on:** Phase 0 (scanner files exist), Phases 1–7 (code is in the state we want to scan).

---

## File Structure

**Create:**
- `scripts/security/run_all.sh` — single-command scan orchestrator
- `scripts/security/triage.go` — parse scan outputs, emit CSV for triage
- `scripts/security/wait_sonarqube.sh` — readiness poll
- `docs/security/remediation-log.md` — running log of fixed findings
- `docs/security/baselines/` — initial scan outputs (frozen)

**Modify:**
- `.github/workflows/security.yml` — hard-fail thresholds
- `Dockerfile` — defensive hardening if findings require
- Application code — as dictated by scan findings

---

## Task 1: Scan orchestration script

**Files:**
- Create: `scripts/security/run_all.sh`
- Create: `scripts/security/wait_sonarqube.sh`

- [ ] **Step 1: `wait_sonarqube.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail
deadline=$((SECONDS + 300))
until curl -sf http://localhost:9000/api/system/status | grep -q '"status":"UP"'; do
  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "sonarqube did not come up within 5m" >&2
    exit 1
  fi
  sleep 5
done
echo "sonarqube UP"
```

- [ ] **Step 2: `run_all.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail
mkdir -p coverage docs/security/baselines

echo "== gosec =="
podman-compose -f docker-compose.security.yml run --rm gosec

echo "== govulncheck =="
podman-compose -f docker-compose.security.yml run --rm govulncheck

echo "== gitleaks =="
podman-compose -f docker-compose.security.yml run --rm gitleaks

echo "== trivy fs =="
podman-compose -f docker-compose.security.yml run --rm trivy-fs

echo "== semgrep =="
podman-compose -f docker-compose.security.yml run --rm semgrep

echo "== syft =="
podman-compose -f docker-compose.security.yml run --rm syft

echo "== snyk =="
if [ -n "${SNYK_TOKEN:-}" ]; then
  podman-compose -f docker-compose.security.yml run --rm snyk
else
  echo "SNYK_TOKEN unset, skipping snyk"
fi

echo "== sonarqube =="
podman-compose -f docker-compose.security.yml up -d sonarqube-db sonarqube
bash scripts/security/wait_sonarqube.sh
podman run --rm --network host \
  -v "$PWD:/usr/src:z" \
  -e SONAR_HOST_URL="${SONAR_HOST_URL:-http://localhost:9000}" \
  -e SONAR_TOKEN="${SONAR_TOKEN:-}" \
  docker.io/sonarsource/sonar-scanner-cli

echo "== copy baselines =="
cp coverage/gosec.json       docs/security/baselines/phase8-gosec.json       2>/dev/null || true
cp coverage/govulncheck.txt  docs/security/baselines/phase8-govulncheck.txt  2>/dev/null || true
cp coverage/gitleaks.json    docs/security/baselines/phase8-gitleaks.json    2>/dev/null || true
cp coverage/trivy.json       docs/security/baselines/phase8-trivy.json       2>/dev/null || true
cp coverage/semgrep.json     docs/security/baselines/phase8-semgrep.json     2>/dev/null || true
cp coverage/sbom.cdx.json    docs/security/baselines/phase8-sbom.cdx.json    2>/dev/null || true
cp coverage/snyk.json        docs/security/baselines/phase8-snyk.json        2>/dev/null || true
```

- [ ] **Step 3: Commit**

```bash
chmod +x scripts/security/*.sh
git add scripts/security/
git commit -m "feat(security): single-command run_all.sh orchestrator for all scanners"
```

---

## Task 2: Triage helper

**Files:**
- Create: `scripts/security/triage.go`

- [ ] **Step 1: Failing test** — given sample scan outputs under `testdata/`, emit a unified CSV with columns: `tool,severity,rule_id,file,line,message,status`.

- [ ] **Step 2: Implement** — parse gosec, trivy, snyk JSON; concatenate govulncheck/gitleaks; sort by severity.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(security): triage tool unifies all scanner outputs into one CSV"
```

---

## Task 3: Run scans, capture baseline

- [ ] **Step 1: Execute**

```bash
bash scripts/security/run_all.sh
go run scripts/security/triage.go > docs/security/baselines/phase8-triage.csv
```

- [ ] **Step 2: Commit baselines**

```bash
git add docs/security/baselines/
git commit -m "chore(security): Phase 8 baseline scan outputs"
```

---

## Task 4: Remediate findings

**Process** (iterate until `triage.go` reports 0 HIGH/CRITICAL):

- [ ] **Step 1: For each HIGH/CRITICAL finding:**
  - Open `docs/security/baselines/phase8-triage.csv`, pick the top entry.
  - Write a failing test that reproduces the issue (if possible — for SCA vulns the test is `go run ./...` against a newer dep version; for gosec code smells the test asserts the fixed code path).
  - Implement the fix.
  - Re-run the relevant scanner.
  - Move the entry to `docs/security/remediation-log.md` with: tool, severity, rule, file:line, root cause, fix commit SHA, date.

- [ ] **Step 2: Examples of expected fixes** (not a complete list — actual list comes from the baseline):
  - `go mod tidy` + `go get -u` to absorb govulncheck-reported CVEs, then re-run `govulncheck`.
  - gosec G404 (weak random) → switch to `crypto/rand` where used for tokens / signatures.
  - gosec G304 (file inclusion via variable) → sanitize / constrain path.
  - trivy HIGH in base image → pin to patched Alpine version in `Dockerfile`.
  - semgrep `no-context-background-in-handler` → pass `ctx` through.
  - semgrep `missing-body-close` → add `defer resp.Body.Close()`.
  - snyk license issue → replace or add an ADR documenting the acceptance.

- [ ] **Step 3: Commit each fix** with message `fix(security): <rule> in <file>:<line>`.

---

## Task 5: Harden Dockerfile

**Files:**
- Modify: `Dockerfile`

- [ ] **Step 1: Failing test** — `trivy image` on the built image returns 0 HIGH/CRITICAL.

```bash
podman build -t patreon-manager:local .
trivy image --severity HIGH,CRITICAL --exit-code 1 patreon-manager:local
```

Expected before: may fail.

- [ ] **Step 2: Apply hardening**

```dockerfile
# syntax=docker/dockerfile:1.7
FROM docker.io/library/golang:1.26-alpine AS build
ENV CGO_ENABLED=0 GOFLAGS=-mod=readonly
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/patreon-manager ./cmd/cli
RUN go build -trimpath -ldflags="-s -w" -o /out/patreon-manager-server ./cmd/server

FROM docker.io/library/alpine:3.20 AS rootfs
RUN apk add --no-cache ca-certificates tini && \
    addgroup -S app && adduser -S -G app app && \
    mkdir -p /data && chown app:app /data

FROM scratch
COPY --from=rootfs /etc/passwd /etc/group /etc/
COPY --from=rootfs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=rootfs /sbin/tini /sbin/tini
COPY --from=rootfs --chown=app:app /data /data
COPY --from=build --chown=app:app /out/patreon-manager /usr/local/bin/patreon-manager
COPY --from=build --chown=app:app /out/patreon-manager-server /usr/local/bin/patreon-manager-server
USER app
WORKDIR /data
ENTRYPOINT ["/sbin/tini","--","/usr/local/bin/patreon-manager-server"]
```

- [ ] **Step 3: Re-run trivy**

```bash
podman build -t patreon-manager:local .
trivy image --severity HIGH,CRITICAL --exit-code 1 patreon-manager:local
```

- [ ] **Step 4: Commit**

```bash
git commit -m "chore(dockerfile): scratch-based, non-root, tini, CA certs only"
```

---

## Task 6: Freeze CI hard-fail thresholds

**Files:**
- Modify: `.github/workflows/security.yml`

- [ ] **Step 1: Change** continue-on-error to false; gate on HIGH/CRITICAL:

```yaml
  snyk:
    steps:
      - uses: snyk/actions/golang@master
        env: { SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }} }
        with: { args: --severity-threshold=high --fail-on=all }

  trivy-fs:
    steps:
      - uses: aquasecurity/trivy-action@0.24.0
        with:
          scan-type: fs
          severity: HIGH,CRITICAL
          exit-code: "1"
```

- [ ] **Step 2: Commit**

```bash
git commit -m "ci(security): hard-fail on HIGH/CRITICAL across all scanners"
```

---

## Task 7: Remediation log

**Files:**
- Create: `docs/security/remediation-log.md`

- [ ] **Step 1: Template**

```markdown
# Security Remediation Log

## Phase 8 — 2026-04-11

| Tool | Severity | Rule | File | Root cause | Commit | Date |
|------|----------|------|------|------------|--------|------|
| govulncheck | HIGH | GO-2025-NNNN | go.mod | upstream fixed in v1.2.3 | <sha> | 2026-04-11 |
| gosec | HIGH | G404 | internal/foo/bar.go:42 | used math/rand for token | <sha> | 2026-04-11 |
| ... | ... | ... | ... | ... | ... | ... |
```

- [ ] **Step 2: Commit**

```bash
git commit -m "docs(security): establish remediation log schema"
```

---

## Task 8: Phase 8 acceptance

- [ ] `bash scripts/security/run_all.sh` green end-to-end.
- [ ] `docs/security/baselines/phase8-triage.csv` reports 0 HIGH/CRITICAL.
- [ ] `docs/security/remediation-log.md` records every fixed finding with a commit SHA.
- [ ] `.github/workflows/security.yml` hard-fails on HIGH/CRITICAL.
- [ ] `trivy image` on the release container is clean.
- [ ] SBOM committed at `docs/security/baselines/phase8-sbom.cdx.json`.

When every box is checked, Phase 8 ships.
