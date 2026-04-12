# `patreon-manager scan`

## Purpose
Discovers repositories across all configured Git providers, applies `.repoignore` filtering and mirror detection, and reports the filtered list. Does NOT generate content or publish to Patreon.

## Usage
    patreon-manager scan [flags]

## Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --dry-run | bool | false | Same as normal scan (scan is always read-only) |
| --org | string | "" | Limit to a specific organization |
| --json | bool | false | Output discovered repos as JSON |
| --log-level | string | "info" | Log verbosity |

## Examples

### Discover all repos
    patreon-manager scan

### Discover repos in a specific org, JSON output
    patreon-manager scan --org myorg --json

## Exit codes
| Code | Meaning |
|------|---------|
| 0 | Success — repos listed |
| 1 | Configuration error |
| 2 | Provider connectivity error |
