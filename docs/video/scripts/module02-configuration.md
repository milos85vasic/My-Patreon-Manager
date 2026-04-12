# Module 02: Configuration

Target length: 12 minutes
Audience: operators

## Scene list

### 00:00 — Intro (20s)
[SCENE: talking head]
Narration: "In this module we walk through every configuration option, from environment variables to validation."

### 00:20 — .env.example walkthrough (4m)
[SCENE: IDE showing .env.example]
Commands: cat .env.example
Narration: Walk through each section — Server, Database, Content Generation, Git Provider Tokens, Patreon, Security scanning.

### 04:20 — Config loading order (2m)
[SCENE: IDE showing internal/config/config.go]
Narration: "Config loads from .env first, then environment variables override. Validate() checks required fields."

### 06:20 — Validation demo (2m)
[SCENE: terminal]
Commands:
    ./patreon-manager validate
    # Show error for missing PATREON_ACCESS_TOKEN
    # Fix it, re-run, show success

### 08:20 — Database options (2m)
[SCENE: slide]
Narration: "SQLite is the default — zero setup. PostgreSQL is available for multi-instance deployments."

### 10:20 — Security keys (90s)
[SCENE: terminal]
Narration: "Never commit tokens. Use .env (gitignored) or CI secrets. ADMIN_KEY protects admin endpoints. WEBHOOK_HMAC_SECRET verifies incoming webhooks."

### 11:50 — Exercise (10s)

## Exercise
1. Set up `.env` with all required fields.
2. Run `validate` and fix every error.
3. Try setting `LOG_LEVEL=debug` and observe verbose output.

## Resources
- docs/guides/configuration.md
