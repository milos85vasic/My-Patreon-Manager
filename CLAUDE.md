# CLAUDE.md - Project Guidelines for AI Assistants

## Security First

**NO TOKENS IN VERSION CONTROL**: Under no circumstances may any token, API key, password, or secret be committed to git. This includes:
- Real credentials in test files, configuration examples, or documentation
- Partial or masked tokens that could be reconstructed
- Placeholder values that resemble real tokens (e.g., `ghp_1234567890...`)

**Redaction Rules**:
- All test files must use placeholder values like `***`, `your_client_id_here`, `test-access-token`
- Documentation examples must use `your_client_id_here`, `your_client_secret_here`
- `.env.example` is the only tracked file with placeholders
- Real credentials belong only in `.env` (gitignored) or environment variables

**Incident Response**:
If a token is accidentally committed:
1. Rotate the exposed credential immediately
2. Use `git-filter-repo` with replace-text rules to purge from history
3. Force-push to all remotes (GitHub, GitLab, GitFlic, GitVerse)
4. Update any exposed test files with redacted values

## Project Context

My Patreon Manager is a Go application that:
- Scans Git repositories across GitHub, GitLab, GitFlic, GitVerse
- Generates content via LLMsVerifier with quality gates  
- Publishes to Patreon with tier-gated access
- Runs as CLI-first with idempotent operations

## Key Files

- `.specify/memory/constitution.md` - Architectural principles (MUST follow)
- `specs/001-patreon-manager-app/tasks.md` - Implementation tasks
- `AGENTS.md` - Project reference for AI assistants
- `scripts/coverage.sh` - Test coverage checker (requires 100% per package)

## Development Standards

- **100% test coverage** for all `internal/` and `cmd/` packages (enforced by coverage script)
- Follow Go conventions, Gin framework patterns
- Modular plugin architecture for providers
- Idempotent operations, CLI-first design
- Resilience patterns: circuit breakers, rate limiting, exponential backoff

## Git Workflow

- Repository mirrors to four upstreams (GitHub, GitLab, GitFlic, GitVerse)
- Use `Upstreams/` scripts for multi-platform pushes
- Branch protection may be enabled on some remotes
- Prefer merge requests over force-pushing to protected branches

## When Implementing Features

1. Check tasks.md for relevant user story
2. Follow constitution principles
3. Write tests first (TDD)
4. Ensure 100% coverage for affected packages
5. Run `bash scripts/coverage.sh` before committing
6. Never commit credentials