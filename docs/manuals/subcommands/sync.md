# `patreon-manager sync`

## Purpose
Runs the full pipeline: discover repos across all configured providers, filter via `.repoignore`, generate content via the LLM pipeline, and publish tier-gated posts to Patreon.

## Usage
    patreon-manager sync [flags]

## Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --dry-run | bool | false | Show what would happen without writing to Patreon |
| --schedule | string | "" | Cron expression for recurring runs (e.g. "@every 1h") |
| --org | string | "" | Limit to a specific organization |
| --repo | string | "" | Limit to a specific repository (owner/name) |
| --pattern | string | "" | Glob pattern to filter repos |
| --json | bool | false | Output in JSON format |
| --log-level | string | "info" | Log verbosity: debug, info, warn, error |

## Examples

### Full sync (dry-run)
    patreon-manager sync --dry-run

### Sync a single org
    patreon-manager sync --org myorg --dry-run

### Scheduled sync every 2 hours
    patreon-manager sync --schedule "@every 2h"

## Exit codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Configuration or runtime error |
| 2 | Provider connectivity error |

## Related
- [scan](scan.md) — discovery only, no generation or publishing
- [generate](generate.md) — content pipeline only, no publishing
- [publish](publish.md) — publish pre-generated content only
