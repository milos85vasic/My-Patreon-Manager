# Tutorial: Running Security Scanners Locally

This tutorial walks through every scanner in the security pipeline, what each one checks, how to read the output, and how to fix findings.

## Prerequisites

```bash
podman --version        # Podman 4+
podman-compose --version # podman-compose 1.0+
```

No sudo required. All scanners run as containers.

## Step 1: Create the coverage directory

```bash
mkdir -p coverage
```

All scanner reports are written here.

## Step 2: Run gitleaks (secret detection)

```bash
gitleaks detect --config .gitleaks.toml --source . --redact --no-git
```

Expected: `no leaks found`

If leaks ARE found:
```
Finding:     ghp_REDACTED
Secret:      ghp_REDACTED
RuleID:      github-pat
File:        path/to/file.go
Line:        42
```

**Fix:** Remove the secret from the file. If it was ever committed, rotate the token immediately and run `git-filter-repo` to purge history (see `docs/runbooks/credential-rotation.md`).

## Step 3: Run gosec (Go security analysis)

Via container:
```bash
podman-compose -f docker-compose.security.yml run --rm gosec
```

Or locally if installed:
```bash
gosec -fmt=json -out=coverage/gosec.json ./...
cat coverage/gosec.json | jq '.Issues | length'
```

Common findings and fixes:

| Rule | Description | Fix |
|------|-------------|-----|
| G104 | Unhandled error | Add `if err != nil` check |
| G304 | File path from variable | Sanitize with `filepath.Clean` |
| G401 | Weak crypto (MD5/SHA1) | Use SHA-256+ |
| G404 | Weak random (math/rand) | Use `crypto/rand` for secrets |
| G501 | Blacklisted import (crypto/md5) | Replace with crypto/sha256 |

## Step 4: Run govulncheck (vulnerability database)

Via container:
```bash
podman-compose -f docker-compose.security.yml run --rm govulncheck
```

Or locally:
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

Expected output for a clean scan:
```
No vulnerabilities found.
```

If vulnerabilities found:
```
Vulnerability #1: GO-2025-XXXX
  stdlib: net/http before 1.26.1
  Fixed in: go1.26.2
```

**Fix:** Update Go or the affected dependency: `go get -u <package>@latest && go mod tidy`

## Step 5: Run semgrep (custom rules)

```bash
semgrep --config .semgrep/rules.yml .
```

Our custom rules check:
- `context.Background()` in handlers (use request context instead)
- Missing `resp.Body.Close()` on HTTP responses
- Mutex held across I/O operations
- `panic()` in production code
- `time.After()` in loops (timer leak)

**Fix:** Each finding includes the rule ID and a fix description in the output.

## Step 6: Run trivy (filesystem vulnerability scan)

```bash
podman-compose -f docker-compose.security.yml run --rm trivy-fs
```

Or locally:
```bash
trivy fs --severity HIGH,CRITICAL .
```

Scans Go dependencies, Dockerfiles, and config files for known CVEs.

**Fix:** `go get -u <vulnerable-package>@latest && go mod tidy`

## Step 7: Run trivy on the container image

```bash
podman build -t patreon-manager:local .
trivy image --severity HIGH,CRITICAL patreon-manager:local
```

Our Dockerfile uses `scratch` base image, so findings are minimal. If the Alpine build stage has issues, pin a newer version in the `FROM` line.

## Step 8: Generate SBOM (Software Bill of Materials)

```bash
podman-compose -f docker-compose.security.yml run --rm syft
```

Output: `coverage/sbom.cdx.json` in CycloneDX format. Attach to releases for supply chain transparency.

## Step 9: Run Snyk (optional, requires token)

```bash
export SNYK_TOKEN=your-token-from-app.snyk.io
podman-compose -f docker-compose.security.yml run --rm snyk
```

Output: `coverage/snyk.json`

## Step 10: Run SonarQube (optional, requires setup)

```bash
# Start SonarQube (first run takes 2-5 minutes to initialize)
podman-compose -f docker-compose.security.yml up -d sonarqube-db sonarqube

# Wait for it to become healthy
bash scripts/security/wait_sonarqube.sh

# Open http://localhost:9000 and create a project token
# Then run the scanner:
export SONAR_TOKEN=your-project-token
podman run --rm --network host \
  -v "$PWD":/usr/src:z \
  -e SONAR_HOST_URL=http://localhost:9000 \
  -e SONAR_TOKEN=$SONAR_TOKEN \
  docker.io/sonarsource/sonar-scanner-cli
```

## Step 11: Run everything at once

```bash
bash scripts/security/run_all.sh
```

This runs every scanner sequentially and copies reports to `docs/security/baselines/`.

Expected: `=== SCAN SUITE: ALL CLEAN ===`

## Step 12: Document remediation

When you fix a finding, add an entry to `docs/security/remediation-log.md`:

```markdown
| Date | Tool | Severity | Rule | File | Root cause | Fix commit |
|------|------|----------|------|------|------------|------------|
| 2026-04-12 | gosec | HIGH | G404 | internal/utils/uuid.go:15 | Used math/rand | abc1234 |
```

## Quick reference: scanner summary

| Scanner | What it checks | Config file | Report |
|---------|---------------|-------------|--------|
| gitleaks | Secrets in code | `.gitleaks.toml` | `coverage/gitleaks.json` |
| gosec | Go security patterns | built-in rules | `coverage/gosec.json` |
| govulncheck | Known Go CVEs | go.mod/go.sum | `coverage/govulncheck.txt` |
| semgrep | Custom code patterns | `.semgrep/rules.yml` | `coverage/semgrep.json` |
| trivy | Filesystem CVEs | `.trivyignore` | `coverage/trivy.json` |
| syft | SBOM generation | none | `coverage/sbom.cdx.json` |
| snyk | Dependency analysis | `.snyk` | `coverage/snyk.json` |
| SonarQube | Code quality + SAST | `sonar-project.properties` | Web dashboard |
| golangci-lint | Go linting | `.golangci.yml` | terminal output |
