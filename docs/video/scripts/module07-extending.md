# Module 07: Extending the Tool

Target length: 12 minutes
Audience: developers

## Scene list

### 00:00 — Adding a new Git provider (5m)
[SCENE: IDE]
Narration: "Implement the RepositoryProvider interface. Register in cmd/cli/main.go. Add token failover config. Write httptest-based tests."

### 05:00 — Adding a new renderer (3m)
Narration: "Implement FormatRenderer. Add to buildRenderers() in cmd/cli/renderers.go behind a config flag."

### 08:00 — Adding a migration (2m)
Narration: "Create 000N_description.up.sql and .down.sql in internal/database/migrations/. The app runs them on startup."

### 10:00 — Writing tests (2m)
Narration: "Unit tests in the package, integration tests in tests/integration/, fuzz tests in tests/fuzz/. Run scripts/coverage.sh before committing."

### 12:00 — Exercise

## Exercise
1. Add a stub provider for Bitbucket (interface only, httptest responses).
2. Add a Markdown → plain-text renderer.
3. Run scripts/coverage.sh and verify 100%.

## Resources
- docs/manuals/developer.md
- .specify/memory/constitution.md
