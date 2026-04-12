# `patreon-manager publish`

## Purpose
Reads previously generated content from the database and publishes it as tier-gated posts to Patreon. Does NOT discover repos or generate new content. Content fingerprinting ensures idempotent re-runs (no duplicates).

## Usage
    patreon-manager publish [flags]

## Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --dry-run | bool | false | Show what would be published without calling Patreon API |
| --tier | string | "" | Publish only content for a specific tier |
| --log-level | string | "info" | Log verbosity |

## Examples

### Publish all pending content (dry-run)
    patreon-manager publish --dry-run

### Publish only public-tier content
    patreon-manager publish --tier=public

### Verify idempotency
    patreon-manager publish  # first run: creates posts
    patreon-manager publish  # second run: no duplicates

## Exit codes
| Code | Meaning |
|------|---------|
| 0 | Success — all content published or already published |
| 1 | Configuration or Patreon API error |
| 2 | Circuit breaker open (Patreon is down) |
