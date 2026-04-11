# Phase 13 — Final Verification Gate Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove every Phase 0–12 deliverable is green simultaneously — not just individually. Produce a single evidence bundle that documents coverage, race, leaks, mutation, perf regression, scanner state, docs state, website state, and video course state. Cut the release tag only if every gate is green.

**Architecture:** A single `scripts/release/verify_all.sh` orchestrates every gate, aggregates outputs into `docs/releases/<version>/evidence/`, and fails loudly if anything regressed. `goreleaser` (already wired in Phase 0) produces the release artifacts only after the script passes.

**Tech Stack:** bash, Go, all scanners from Phase 8, Hugo, `goreleaser`, `cosign`.

**Depends on:** Phases 0–12 merged to main.

---

## File Structure

**Create:**
- `scripts/release/verify_all.sh` — single verifier
- `scripts/release/write_evidence.go` — structured evidence writer
- `docs/releases/<version>/evidence/*`
- `docs/releases/<version>/RELEASE_NOTES.md`
- `docs/releases/<version>/CHANGELOG-delta.md`
- `.github/workflows/release-gate.yml` — on-tag verification workflow

**Modify:**
- `specs/001-patreon-manager-app/tasks.md` — mark every user story accepted
- `CHANGELOG.md` — prepend Phase 0–13 delta
- `VERSION` — bump to `0.2.0`

---

## Task 1: Evidence writer

**Files:**
- Create: `scripts/release/write_evidence.go`

- [ ] **Step 1: Failing test** — feed sample inputs; assert JSON schema.

- [ ] **Step 2: Implement** — accepts flags for every gate (`-coverage`, `-race`, `-leaks`, `-mutation`, `-perf`, `-scan`, `-docs`, `-website`, `-video`, `-out`). Emits `evidence.json` with pass/fail per gate and absolute paths to the raw artifacts.

```go
// scripts/release/write_evidence.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

type Gate struct {
	Name    string `json:"name"`
	Status  string `json:"status"`   // "pass" | "fail" | "skip"
	Artifact string `json:"artifact"` // relative path
	Summary string `json:"summary"`
}

type Evidence struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Gates     []Gate    `json:"gates"`
}

func main() {
	version := flag.String("version", "", "release version")
	out := flag.String("out", "evidence.json", "output path")
	flag.Parse()
	// ... parse remaining positional gate=status pairs
	ev := Evidence{Version: *version, Timestamp: time.Now().UTC()}
	for _, arg := range flag.Args() {
		// parse name=status=artifact=summary
	}
	f, _ := os.Create(*out)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(ev); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(release): evidence JSON writer"
```

---

## Task 2: Master verification script

**Files:**
- Create: `scripts/release/verify_all.sh`

- [ ] **Step 1: Script**

```bash
#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:?usage: $0 <version>}"
EV_DIR="docs/releases/$VERSION/evidence"
mkdir -p "$EV_DIR"

FAIL=0
record() {
  local name="$1" status="$2" artifact="$3" summary="$4"
  echo "[$status] $name — $summary"
  printf '%s=%s=%s=%s\n' "$name" "$status" "$artifact" "$summary" >> "$EV_DIR/_raw.tsv"
  if [ "$status" = "fail" ]; then FAIL=1; fi
}

echo "=== coverage (100%) ==="
if bash scripts/coverage.sh > "$EV_DIR/coverage.log" 2>&1; then
  record coverage pass "$EV_DIR/coverage.log" "100% per package"
else
  record coverage fail "$EV_DIR/coverage.log" "coverage gate failed"
fi

echo "=== race ==="
if go test -race -timeout 20m ./... > "$EV_DIR/race.log" 2>&1; then
  record race pass "$EV_DIR/race.log" "all races clean"
else
  record race fail "$EV_DIR/race.log" "race detector found failures"
fi

echo "=== goleak ==="
if go test -run TestMain -count=1 ./... > "$EV_DIR/goleak.log" 2>&1; then
  record goleak pass "$EV_DIR/goleak.log" "no goroutine leaks"
else
  record goleak fail "$EV_DIR/goleak.log" "goroutine leak detected"
fi

echo "=== mutation ==="
if bash tests/mutation/run_mutation.sh > "$EV_DIR/mutation.log" 2>&1; then
  record mutation pass "$EV_DIR/mutation.log" "≤5% survival"
else
  record mutation fail "$EV_DIR/mutation.log" "survival rate exceeded"
fi

echo "=== perf regression ==="
go test -bench=. -benchmem -run=^$ ./... > "$EV_DIR/bench.txt" 2>&1 || true
if go run scripts/perf_regression.go -input "$EV_DIR/bench.txt" -baseline tests/bench/baseline/phase7.json > "$EV_DIR/perf.log" 2>&1; then
  record perf pass "$EV_DIR/perf.log" "within ±10% baseline"
else
  record perf fail "$EV_DIR/perf.log" "regression > 10%"
fi

echo "=== security scans ==="
if bash scripts/security/run_all.sh > "$EV_DIR/security.log" 2>&1; then
  record security pass "$EV_DIR/security.log" "no HIGH/CRITICAL"
else
  record security fail "$EV_DIR/security.log" "scanner findings"
fi

echo "=== docs lint + link check ==="
if markdownlint-cli2 "**/*.md" > "$EV_DIR/markdownlint.log" 2>&1 && \
   lychee --no-progress './**/*.md' > "$EV_DIR/lychee.log" 2>&1; then
  record docs pass "$EV_DIR/markdownlint.log" "lint + links clean"
else
  record docs fail "$EV_DIR/markdownlint.log" "docs violations"
fi

echo "=== website build ==="
if (cd docs/website && hugo --gc --minify) > "$EV_DIR/hugo.log" 2>&1; then
  record website pass "$EV_DIR/hugo.log" "hugo build clean"
else
  record website fail "$EV_DIR/hugo.log" "hugo build failed"
fi

echo "=== video artifacts inventory ==="
missing=0
for n in 01 02 03 04 05 06 07 08 09 10; do
  [ -f "docs/video/scripts/module$n"*.md ] || { echo "missing script $n"; missing=1; }
  [ -f "docs/video/captions/module$n.srt" ] || { echo "missing srt $n"; missing=1; }
done
if [ "$missing" -eq 0 ]; then
  record video pass "docs/video/" "scripts + SRTs present"
else
  record video fail "docs/video/" "missing module artifacts"
fi

echo "=== writing evidence.json ==="
go run scripts/release/write_evidence.go -version "$VERSION" -out "$EV_DIR/evidence.json" \
  $(awk -F= '{printf "%s=%s=%s=%s\n", $1, $2, $3, $4}' "$EV_DIR/_raw.tsv")

if [ "$FAIL" -ne 0 ]; then
  echo "=== RELEASE GATE FAILED ==="
  exit 1
fi
echo "=== RELEASE GATE PASSED ==="
```

- [ ] **Step 2: Commit**

```bash
chmod +x scripts/release/verify_all.sh
git add scripts/release/
git commit -m "feat(release): master verification script for Phase 13 gate"
```

---

## Task 3: On-tag verification workflow

**Files:**
- Create: `.github/workflows/release-gate.yml`

- [ ] **Step 1: Write**

```yaml
# Manual-only per project policy. Dispatched explicitly with a version input.
name: Release Gate
on:
  workflow_dispatch:
    inputs:
      version:
        description: "Release version to verify (e.g. v0.2.0)"
        required: true
jobs:
  gate:
    runs-on: ubuntu-latest
    timeout-minutes: 90
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: "1.26.x" }
      - name: verify_all
        run: bash scripts/release/verify_all.sh "${{ inputs.version }}"
      - uses: actions/upload-artifact@v4
        with:
          name: evidence-${{ inputs.version }}
          path: docs/releases/${{ inputs.version }}/evidence/
```

- [ ] **Step 2: Commit**

```bash
git commit -m "ci(release): on-tag verify_all gate"
```

---

## Task 4: Release notes + changelog delta

**Files:**
- Create: `docs/releases/v0.2.0/RELEASE_NOTES.md`
- Create: `docs/releases/v0.2.0/CHANGELOG-delta.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Author release notes** — a user-facing summary of Phase 0–12 deliverables:
  - CI + security scanning (Phase 0, 8)
  - Concurrency hardening (Phase 1)
  - Orphans wired (Phase 2)
  - PostgreSQL parity (Phase 3)
  - Full renderers (Phase 4)
  - Zero skipped tests (Phase 5)
  - 100% coverage + all test types (Phase 6)
  - Lazy init, pools, caches, observability (Phase 7)
  - Zero HIGH/CRITICAL scan findings (Phase 8)
  - Full docs overhaul (Phase 9)
  - User manuals (Phase 10)
  - Full video course scripts (Phase 11)
  - Website refresh (Phase 12)

- [ ] **Step 2: CHANGELOG delta** in keep-a-changelog format.
- [ ] **Step 3: Prepend to CHANGELOG.md.**
- [ ] **Step 4: Commit**

```bash
git commit -m "docs(release): v0.2.0 release notes + changelog delta"
```

---

## Task 5: Mark every user story accepted

**Files:**
- Modify: `specs/001-patreon-manager-app/tasks.md`

- [ ] **Step 1: Walk every story** and mark `[X]` or add a new sub-story for each Phase 0–12 deliverable that maps to it.
- [ ] **Step 2: Commit**

```bash
git commit -m "docs(spec): accept every user story across Phase 0-12"
```

---

## Task 6: Run the gate

- [ ] **Step 1: Run locally**

```bash
bash scripts/release/verify_all.sh v0.2.0
```

Expected: `=== RELEASE GATE PASSED ===`.

- [ ] **Step 2: Commit evidence**

```bash
git add docs/releases/v0.2.0/
git commit -m "chore(release): v0.2.0 gate evidence"
```

---

## Task 7: Cut the tag

- [ ] **Step 1: Bump VERSION**

```bash
echo "0.2.0" > VERSION
git add VERSION
git commit -m "chore(release): bump VERSION to 0.2.0"
```

- [ ] **Step 2: Tag**

```bash
git tag -s v0.2.0 -m "v0.2.0 — Phase 0-13 complete"
```

- [ ] **Step 3: Confirm before pushing** (explicit user approval required — Phase 13 is the first time Phase work touches public state on mirrors).

- [ ] **Step 4: Push to all four mirrors** — once user approves:

```bash
bash Upstreams/push-github.sh v0.2.0
bash Upstreams/push-gitlab.sh v0.2.0
bash Upstreams/push-gitflic.sh v0.2.0
bash Upstreams/push-gitverse.sh v0.2.0
```

---

## Task 8: Post-release verification

- [ ] **Step 1: Assert** `.github/workflows/release-gate.yml` ran on the tag and passed.
- [ ] **Step 2: Assert** GoReleaser produced signed artifacts.
- [ ] **Step 3: Assert** SBOM + SLSA attestation uploaded to the GitHub release.
- [ ] **Step 4: Assert** GitHub Pages site now shows the release notes and video page.

---

## Task 9: Phase 13 acceptance

- [ ] `scripts/release/verify_all.sh v0.2.0` passes.
- [ ] `docs/releases/v0.2.0/evidence/evidence.json` committed with every gate marked `pass`.
- [ ] `specs/001-patreon-manager-app/tasks.md` has every user story accepted.
- [ ] `CHANGELOG.md` has the v0.2.0 delta.
- [ ] `VERSION` bumped.
- [ ] Tag `v0.2.0` signed and pushed to all four mirrors (after user approval).
- [ ] Release-gate workflow green.
- [ ] GoReleaser artifacts cosigned and attached to release.
- [ ] Website reflects the release.

When every box is checked, the completion program is done: nothing is unfinished, nothing is broken, nothing is undocumented, nothing is uncovered. v0.2.0 ships.
