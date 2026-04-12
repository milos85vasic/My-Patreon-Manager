# Module 03: Repository Sync

Target length: 12 minutes
Audience: operators

## Scene list

### 00:00 — What sync does (60s)
Narration: "Sync discovers repos across all configured providers, filters them, generates content, and publishes to Patreon."

### 01:00 — Provider discovery (3m)
[SCENE: IDE showing internal/providers/git/]
Narration: "Each provider implements RepositoryProvider. GitHub uses the REST API with token failover. GitLab uses go-gitlab. GitFlic and GitVerse use raw HTTP."

### 04:00 — .repoignore (2m)
[SCENE: terminal]
Commands:
    cat .repoignore
    ./patreon-manager scan --dry-run
Narration: "Patterns work like .gitignore. Prefix ! to un-ignore. Mirror detection groups duplicates automatically."

### 06:00 — Dry-run vs real sync (2m)
Commands:
    ./patreon-manager sync --dry-run
    ./patreon-manager sync

### 08:00 — Scheduled mode (2m)
Commands:
    ./patreon-manager sync --schedule "@every 1h"
Narration: "The scheduler accepts cron expressions and respects context cancellation."

### 10:00 — Audit trail (90s)
Narration: "Every sync emits audit entries — viewable via /admin/audit."

### 11:30 — Exercise

## Exercise
1. Create a `.repoignore` excluding forks and archived repos.
2. Run `scan --dry-run` and verify only desired repos appear.
3. Run `sync --dry-run` and review the generated content preview.

## Resources
- docs/guides/git-providers.md
