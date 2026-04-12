# AGENTS.md

## Project

Go 1.26.1 application that scans Git repositories across GitHub, GitLab, GitFlic, and GitVerse, generates content via an LLM pipeline with quality gates, and publishes tier-gated posts to Patreon.

- **Language:** Go 1.26.1
- **Framework:** Gin (`github.com/gin-gonic/gin`)
- **Module:** `github.com/milos85vasic/My-Patreon-Manager`
- **Entrypoints:**
  - `cmd/cli/main.go` (`patreon-manager`) -- CLI with subcommands `sync`, `scan`, `generate`, `validate`, `publish`
  - `cmd/server/main.go` -- Gin HTTP server on `:8080` with health, metrics (Prometheus), and webhook handlers

## Commands

```sh
go build ./...                                  # build all packages
go run ./cmd/cli sync --dry-run                 # dry-run a sync
go run ./cmd/cli validate                       # validate config/env
go run ./cmd/server                             # run HTTP server
go test ./internal/... ./cmd/... ./tests/...    # run full test suite
go test -race ./...                             # race detector
go vet ./...                                    # static analysis
bash scripts/coverage.sh                        # full coverage run (gates at 100%)
```

## Layout

### Entrypoints

- `cmd/cli/` -- CLI application with subcommands; uses dependency-injection via package-level function variables (`newConfig`, `newDatabase`, `newOrchestrator`, `newMetricsCollector`, `osExit`, etc.)
- `cmd/server/` -- HTTP server; same DI pattern as CLI

### Internal Packages

- `internal/config/` -- environment and file-based configuration loading with validation
- `internal/database/` -- SQLite (default) and PostgreSQL database layer
- `internal/handlers/` -- HTTP request handlers and webhook processors
- `internal/middleware/` -- Gin middleware (logging, auth, etc.)
- `internal/models/` -- data structures (`Campaign`, `Post`, `Tier`, `Repository`, etc.)
- `internal/errors/` -- domain-specific error types
- `internal/metrics/` -- Prometheus metrics collector interface and circuit breaker metrics
- `internal/utils/` -- shared utilities
- `internal/concurrency/` -- concurrency primitives and helpers
- `internal/cache/` -- LRU cache implementation
- `internal/lazy/` -- lazy initialization wrapper
- `internal/testhelpers/` -- shared test utilities

### Providers (`internal/providers/`)

Pluggable external integrations behind Go interfaces:

- `git/` -- `RepositoryProvider` implementations for GitHub, GitLab, GitFlic, GitVerse with per-service auth, pagination, rate limiting, mirror detection, and `.repoignore` filtering
- `llm/` -- `LLMProvider` with fallback chains and quality-scored model selection via LLMsVerifier
- `patreon/` -- Patreon API client with tier gating and circuit breakers (gobreaker)
- `renderer/` -- `FormatRenderer` for Markdown, HTML, PDF, and video output

### Services (`internal/services/`)

Orchestration layered on top of providers:

- `sync/` -- `Orchestrator` is the top-level coordinator wiring providers + generator + db + metrics; consumed by both `cmd/cli` and `cmd/server`
- `content/` -- content `Generator` and `TierMapper`
- `filter/` -- repository selection and `.repoignore` filtering
- `access/` -- tier access control
- `audit/` -- audit logging

### Documentation

- `docs/` -- guides, architecture docs, API reference, ADRs, runbooks, troubleshooting
- `docs/main_specification.md` -- full system specification
- `docs/The_Core_Idea.md` -- high-level concept and Patreon API setup guide
- `.specify/memory/constitution.md` -- architectural principles (I-VII), authoritative

### Other

- `scripts/` -- build and coverage scripts (`coverage.sh` enforces 100% per-package coverage)
- `Upstreams/` -- push helper scripts for GitHub, GitLab, GitFlic, GitVerse mirrors
- `tests/` -- additional test suites (unit, integration)

## Current State

Phases 0-6 are complete. The application is functional with:

- Full CLI with all subcommands (`sync`, `scan`, `generate`, `validate`, `publish`) including `--dry-run`, `--schedule` (cron), `--org`, `--repo`, `--pattern`, `--json`, `--log-level`
- HTTP server with health checks, Prometheus metrics, and webhook handlers
- Four Git provider adapters (GitHub, GitLab, GitFlic, GitVerse) with mirror detection
- LLM content generation with quality gates and fallback chains
- Patreon API integration with tier gating
- SQLite and PostgreSQL database support
- Circuit breakers, rate limiting, exponential backoff
- Content fingerprinting and checkpointing for idempotent operations
- 100% test coverage enforced per package

## Environment

Copy `.env.example` to `.env` and fill in API credentials. Required variables include Patreon OAuth2 tokens, Git provider tokens, and LLM provider configuration.

## Security

**CRITICAL**: No token, API key, password, or secret of any kind may ever be committed to version control. This includes test files, documentation, configuration examples, or any tracked file. All such values must be redacted or replaced with placeholders (e.g., `***`, `your_client_id_here`). Use `.env.example` for placeholder examples; real credentials must be stored only in `.env` (gitignored) or environment variables.

Any accidental exposure must be treated as a security incident: rotate credentials immediately, use `git-filter-repo` to purge from history, and force-push to all remotes.

## Dependency-Injection Pattern

Both `cmd/cli/main.go` and `cmd/server/main.go` expose package-level function variables that tests swap out. When editing these entrypoints, preserve that indirection -- tests hit 100% coverage by overriding those variables.

## Key Dependencies

- `github.com/gin-gonic/gin` -- HTTP framework
- `github.com/google/go-github/v69` -- GitHub API client
- `github.com/xanzy/go-gitlab` -- GitLab API client
- `github.com/sony/gobreaker` -- circuit breaker
- `github.com/mattn/go-sqlite3` -- SQLite driver
- `github.com/lib/pq` -- PostgreSQL driver
- `github.com/prometheus/client_golang` -- Prometheus metrics
- `github.com/robfig/cron/v3` -- cron scheduling
- `github.com/joho/godotenv` -- .env loading
- `github.com/stretchr/testify` -- test assertions and mocks

## Mirrors

The repo mirrors to four Git hosting services. Push scripts live in `Upstreams/` (`GitHub.sh`, `GitLab.sh`, `GitFlic.sh`, `GitVerse.sh`).

## Authoritative References

- `.specify/memory/constitution.md` -- architectural principles (I-VII). Read before non-trivial changes; these are enforced, not aspirational.
- `specs/001-patreon-manager-app/tasks.md` -- active implementation tasks and user stories
- `CLAUDE.md` -- companion reference with build commands and architecture overview
