# Module 05: Publishing to Patreon

Target length: 12 minutes
Audience: content creators

## Scene list

### 00:00 — Publishing overview (60s)
Narration: "Publish reads generated content from the database, creates tier-gated posts on Patreon, and marks them as published."

### 01:00 — Patreon API setup (3m)
[SCENE: browser showing Patreon developer portal]
Narration: "Create an OAuth client, obtain access and refresh tokens, set them in .env."

### 04:00 — Idempotency (2m)
Narration: "Content fingerprinting ensures re-running publish never creates duplicate posts. Safe to retry."

### 06:00 — Tier gating (2m)
Narration: "Each post is associated with a Patreon tier. The mapping respects the CONTENT_TIER_MAPPING_STRATEGY."

### 08:00 — Publish demo (3m)
[SCENE: terminal]
Commands:
    ./patreon-manager publish --dry-run
    ./patreon-manager publish

### 11:00 — Circuit breaker (60s)
Narration: "If Patreon returns consecutive 5xx errors, the circuit breaker trips and subsequent calls fail fast."

### 12:00 — Exercise

## Exercise
1. Run `publish --dry-run` and review the proposed posts.
2. Publish one post and verify it on Patreon.
3. Re-run publish — verify no duplicates are created.

## Resources
- docs/guides/deployment.md
