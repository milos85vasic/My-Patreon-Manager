# Phase 9 — Documentation Overhaul Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring every piece of documentation to nano-detail accuracy: rewrite README, fix the typo in `main_specification.md`, reframe `The_Core_Idea.md`, rewrite OpenAPI to match wired routes, regenerate SQL schema docs from Phase 3 migrations, add ADRs, runbooks, troubleshooting/FAQ, admin guide, per-package `doc.go`, architecture diagrams, and update `AGENTS.md`.

**Architecture:** Docs are tested — every code example in docs is extracted and compiled by `go test`, every link is validated by `lychee`, every markdown file lints clean under `markdownlint`. ADRs follow Nygard template. Runbooks follow Google SRE template.

**Tech Stack:** Markdown, Mermaid, drawio (export committed SVGs), `lychee`, `markdownlint-cli2`, `godoc`, `protoc-gen-doc` (not used), `redoc-cli` for OpenAPI, `mermaid-cli` for diagrams.

**Depends on:** Phases 0–8. This phase documents the post-Phase-8 reality.

---

## File Structure

**Create:**
- `docs/adr/0001-go-gin.md`
- `docs/adr/0002-sqlite-default.md`
- `docs/adr/0003-mirror-detection.md`
- `docs/adr/0004-circuit-breaker-choice.md`
- `docs/adr/0005-chromedp-for-pdf.md`
- `docs/adr/0006-ffmpeg-pipeline.md`
- `docs/adr/0007-semaphore-sizes.md`
- `docs/adr/0008-audit-store.md`
- `docs/adr/0009-migration-tool.md`
- `docs/adr/0010-observability-stack.md`
- `docs/runbooks/incident-response.md`
- `docs/runbooks/backup-recovery.md`
- `docs/runbooks/credential-rotation.md`
- `docs/runbooks/migration-rollout.md`
- `docs/runbooks/certificate-rotation.md`
- `docs/runbooks/on-call-playbook.md`
- `docs/troubleshooting/faq.md`
- `docs/troubleshooting/common-errors.md`
- `docs/admin-guide/webhook-setup.md`
- `docs/admin-guide/monitoring.md`
- `docs/admin-guide/slo-reference.md`
- `docs/architecture/diagrams/overview.mmd`
- `docs/architecture/diagrams/request-flow.mmd`
- `docs/architecture/diagrams/er.mmd`
- `docs/architecture/diagrams/deployment.mmd`
- `docs/architecture/diagrams/*.svg` — generated from .mmd
- `internal/*/doc.go` — one per package
- `docs/docs_examples_test.go` — extracts and compiles examples

**Modify:**
- `README.md` — full rewrite
- `docs/main_specification.md` → `docs/main_specification.md` (rename)
- `docs/The_Core_Idea.md` — reframed intro
- `docs/api/openapi.yaml` — regenerate from wired routes
- `docs/architecture/sql-schema.md` — regenerate from Phase 3 migrations
- `docs/guides/*.md` — align with current code
- `AGENTS.md` — regenerate

---

## Task 1: Rewrite README.md

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Full rewrite**

```markdown
# My Patreon Manager

Scans Git repositories across GitHub, GitLab, GitFlic, and GitVerse; generates content via an LLM pipeline with quality gates; publishes tier-gated posts to Patreon.

![ci](https://github.com/milos85vasic/My-Patreon-Manager/actions/workflows/ci.yml/badge.svg)
![security](https://github.com/milos85vasic/My-Patreon-Manager/actions/workflows/security.yml/badge.svg)
![coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)

## Features
- Multi-provider repository discovery (GitHub, GitLab, GitFlic, GitVerse) with `.repoignore` filtering and mirror detection.
- Pluggable LLM pipeline with fallback, verifier, circuit breaker, and bounded concurrency.
- Patreon tier-gated publishing with content fingerprinting and idempotent re-runs.
- Renderers for Markdown, HTML, PDF (chromedp), and video (ffmpeg).
- CLI (`patreon-manager`) and HTTP server (`patreon-manager-server`).
- Prometheus metrics, Grafana dashboards, pprof, structured logs.
- First-class security: gosec, govulncheck, golangci-lint, semgrep, trivy, gitleaks, snyk, SonarQube.

## Quickstart

```sh
git clone https://github.com/milos85vasic/My-Patreon-Manager.git
cd My-Patreon-Manager
cp .env.example .env
$EDITOR .env
go build ./...
./patreon-manager validate
./patreon-manager sync --dry-run
```

Full walkthrough: [docs/manuals/end-to-end.md](docs/manuals/end-to-end.md).

## Architecture

![Overview](docs/architecture/diagrams/overview.svg)

## Documentation index
- Quickstart: [docs/guides/quickstart.md](docs/guides/quickstart.md)
- Configuration: [docs/guides/configuration.md](docs/guides/configuration.md)
- Content generation: [docs/guides/content-generation.md](docs/guides/content-generation.md)
- Deployment: [docs/guides/deployment.md](docs/guides/deployment.md)
- Git providers: [docs/guides/git-providers.md](docs/guides/git-providers.md)
- CLI reference: [docs/api/cli-reference.md](docs/api/cli-reference.md)
- OpenAPI: [docs/api/openapi.yaml](docs/api/openapi.yaml)
- Architecture: [docs/architecture/overview.md](docs/architecture/overview.md)
- ADRs: [docs/adr/](docs/adr/)
- Runbooks: [docs/runbooks/](docs/runbooks/)
- Troubleshooting: [docs/troubleshooting/](docs/troubleshooting/)
- Admin guide: [docs/admin-guide/](docs/admin-guide/)
- Video course: [docs/video/course-outline.md](docs/video/course-outline.md)
- Constitution: [.specify/memory/constitution.md](.specify/memory/constitution.md)
- Security remediation log: [docs/security/remediation-log.md](docs/security/remediation-log.md)

## License
See [LICENSE](LICENSE).
```

- [ ] **Step 2: Commit**

```bash
git commit -m "docs(readme): full rewrite replacing 'Tbd' stub"
```

---

## Task 2: Fix spec filename typo

- [ ] **Step 1: Rename**

```bash
git mv docs/main_specification.md docs/main_specification.md
grep -rln 'main_specification' docs/ | xargs sed -i 's/main_specification/main_specification/g'
```

- [ ] **Step 2: Commit**

```bash
git commit -m "docs: rename main_specification.md to main_specification.md"
```

---

## Task 3: Reframe `The_Core_Idea.md`

**Files:**
- Modify: `docs/The_Core_Idea.md`

- [ ] **Step 1: Prepend a "Current reality" section** clarifying that the project is a standalone Go scanner + generator + publisher, not a Claude-Code-driven Patreon API wrapper. Keep the historical context for provenance.

- [ ] **Step 2: Commit**

```bash
git commit -m "docs: reframe The_Core_Idea with current reality section"
```

---

## Task 4: Rewrite OpenAPI to match wired routes

**Files:**
- Modify: `docs/api/openapi.yaml`

- [ ] **Step 1: Enumerate wired routes**

```bash
grep -n 'r\.\(GET\|POST\|PUT\|DELETE\)' cmd/server/main.go internal/handlers/
```

- [ ] **Step 2: Regenerate** OpenAPI 3.1 with:
  - Every route from the grep with request body schema, response schema, auth.
  - Security schemes: `adminKey` (header `X-Admin-Key`), `webhookSignature` (per-provider header).
  - Example requests for each mutation route.

- [ ] **Step 3: Validation test** — `scripts/openapi_check.go` parses the YAML and asserts every route has `summary`, `operationId`, at least one response, and an example.

```bash
go run scripts/openapi_check.go
```

- [ ] **Step 4: Commit**

```bash
git commit -m "docs(api): regenerate openapi.yaml from wired routes"
```

---

## Task 5: Regenerate SQL schema doc

**Files:**
- Modify: `docs/architecture/sql-schema.md`
- Create: `docs/architecture/diagrams/er.mmd`

- [ ] **Step 1: Rewrite** the markdown doc listing every table from Phase 3 migrations, each column, type, constraints, and references.

- [ ] **Step 2: Mermaid ER diagram**

```mermaid
erDiagram
    REPOSITORIES ||--o{ SYNC_STATES : has
    REPOSITORIES ||--o{ GENERATED_CONTENT : produces
    REPOSITORIES ||--o{ MIRROR_MAPS : "canonical"
    GENERATED_CONTENT ||--o{ POSTS : publishes
    GENERATED_CONTENT ||--o{ QUALITY_REVIEWS : reviewed_by
    CONTENT_TEMPLATES ||--o{ GENERATED_CONTENT : renders
    REPOSITORIES {
      TEXT id PK
      TEXT provider
      TEXT full_name UK
      TEXT mirror_of FK
    }
    ...
```

- [ ] **Step 3: Generate SVG** via `mmdc -i er.mmd -o er.svg`.

- [ ] **Step 4: Commit**

```bash
git commit -m "docs(db): regenerate sql-schema.md + ER diagram from Phase 3 migrations"
```

---

## Task 6: Add ADRs (10 files)

**Files:** `docs/adr/NNNN-*.md`

- [ ] **Step 1: Template (Nygard)**

```markdown
# NNNN. Title

Date: 2026-04-11

## Status
Accepted

## Context
<1–2 paragraphs: why we need a decision.>

## Decision
<What we chose.>

## Consequences
<Good / bad / neutral trade-offs.>

## Alternatives considered
<What we rejected and why.>
```

- [ ] **Step 2: Author each ADR** (one per commit):
  - 0001-go-gin.md
  - 0002-sqlite-default.md
  - 0003-mirror-detection.md
  - 0004-circuit-breaker-choice.md
  - 0005-chromedp-for-pdf.md
  - 0006-ffmpeg-pipeline.md
  - 0007-semaphore-sizes.md
  - 0008-audit-store.md
  - 0009-migration-tool.md
  - 0010-observability-stack.md

- [ ] **Step 3: Commit each**

```bash
git commit -m "docs(adr): 0001 Go + Gin"
# ... etc
```

---

## Task 7: Runbooks (6 files)

**Files:** `docs/runbooks/*.md`

- [ ] **Step 1: Template (Google SRE)**

```markdown
# Runbook: <title>

## Alert
What paged you.

## Diagnosis
Commands and dashboards to check.

## Fix
Step-by-step remediation.

## Verify
How to confirm the fix.

## Escalate
Who to wake if it's not fixed in N minutes.
```

- [ ] **Step 2: Author each**:
  - `incident-response.md`
  - `backup-recovery.md` — SQLite file copy + PostgreSQL `pg_dump`, restore verification.
  - `credential-rotation.md` — Patreon, GitHub/GitLab/GitFlic/GitVerse, admin key.
  - `migration-rollout.md` — apply, verify, roll back (down migration).
  - `certificate-rotation.md` — TLS cert refresh for HTTPS server.
  - `on-call-playbook.md` — who owns what, shift handoff.

- [ ] **Step 3: Commit each**

```bash
git commit -m "docs(runbooks): incident response"
# ...
```

---

## Task 8: Troubleshooting + FAQ

**Files:** `docs/troubleshooting/faq.md`, `docs/troubleshooting/common-errors.md`

- [ ] **Step 1: FAQ** — minimum 20 questions covering: repoignore behavior, token scopes, rate limit backoff, PostgreSQL vs SQLite, Patreon idempotency, webhook signature verification, video pipeline fps, PDF fonts, podman vs docker, admin key rotation, scheduled mode vs cron.

- [ ] **Step 2: common-errors.md** — table of error strings → root cause → fix.

- [ ] **Step 3: Commit**

```bash
git commit -m "docs(troubleshooting): FAQ + common-errors reference"
```

---

## Task 9: Admin guide

**Files:** `docs/admin-guide/*.md`

- [ ] **Step 1: Author** `webhook-setup.md`, `monitoring.md`, `slo-reference.md` covering: how to install webhook URLs on each provider, HMAC secrets rotation, dashboards to watch, SLO ownership matrix.

- [ ] **Step 2: Commit**

```bash
git commit -m "docs(admin): webhook setup, monitoring, SLO reference"
```

---

## Task 10: Per-package `doc.go`

**Files:** `internal/*/doc.go`

- [ ] **Step 1: Generator script** that writes a minimal `doc.go` in every package missing one, summarizing the package purpose in 1 paragraph. Manual review before commit.

- [ ] **Step 2: Verify** `go doc ./internal/...` produces non-empty summaries.

- [ ] **Step 3: Commit**

```bash
git commit -m "docs(go): add package-level doc.go to every internal package"
```

---

## Task 11: Architecture diagrams

**Files:** `docs/architecture/diagrams/*.mmd` + `.svg`

- [ ] **Step 1: Author mermaid sources**:
  - `overview.mmd` — component diagram (CLI, server, providers, orchestrator, DB).
  - `request-flow.mmd` — webhook → handler → queue → orchestrator → providers → audit.
  - `er.mmd` — from Task 5.
  - `deployment.mmd` — podman / k8s deployment topology.

- [ ] **Step 2: Generate SVG via `mmdc`**:

```bash
for f in docs/architecture/diagrams/*.mmd; do
  mmdc -i "$f" -o "${f%.mmd}.svg"
done
```

- [ ] **Step 3: Commit**

```bash
git commit -m "docs(arch): mermaid diagrams + SVG exports"
```

---

## Task 12: Regenerate `AGENTS.md`

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Rewrite** to reflect post-rewire reality: Phase 2 orphans wired, Phase 3 PostgreSQL parity, Phase 4 renderers complete, Phase 5 zero skips, Phase 6 100% coverage, Phase 7 lazy init + pools, Phase 8 security gates.

- [ ] **Step 2: Commit**

```bash
git commit -m "docs: regenerate AGENTS.md to reflect Phase 2-8 state"
```

---

## Task 13: Docs build/link CI green

- [ ] **Step 1: Run locally**

```bash
markdownlint-cli2 "**/*.md"
lychee --no-progress './**/*.md'
```

- [ ] **Step 2: Fix every violation** — no exemptions.
- [ ] **Step 3: Commit**

```bash
git commit -m "docs: fix lint + link-check violations across the tree"
```

---

## Task 14: Docs examples test harness

**Files:**
- Create: `docs/docs_examples_test.go`

- [ ] **Step 1: Failing test** — extract every ```go code block from docs/**/*.md and compile (not run) them inside a temp module.
- [ ] **Step 2: Fix any code blocks that don't compile.**
- [ ] **Step 3: Commit**

```bash
git commit -m "docs(test): every go code block in docs/ compiles in CI"
```

---

## Task 15: Phase 9 acceptance

- [ ] README.md has real content + badges + diagram.
- [ ] `main_specification.md` (renamed) referenced from everywhere.
- [ ] OpenAPI matches wired routes; `scripts/openapi_check.go` green.
- [ ] SQL schema doc + ER diagram regenerated from Phase 3 migrations.
- [ ] 10 ADRs committed.
- [ ] 6 runbooks committed.
- [ ] Troubleshooting + FAQ + admin guide committed.
- [ ] `doc.go` in every internal/ package.
- [ ] All diagrams rendered and committed (.mmd + .svg).
- [ ] `AGENTS.md` regenerated.
- [ ] `markdownlint`, `lychee`, docs example-compile CI green.

When every box is checked, Phase 9 ships.
