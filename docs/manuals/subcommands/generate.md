# `patreon-manager generate`

## Purpose
Runs the content generation pipeline: discovers repos, generates content via the LLM with quality gates, and persists generated content to the database. Does NOT publish to Patreon.

## Usage
    patreon-manager generate [flags]

## Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --dry-run | bool | false | Show generated content without persisting |
| --repo | string | "" | Generate for a specific repository only |
| --format | string | "markdown" | Output format: markdown, html, pdf, video |
| --log-level | string | "info" | Log verbosity |

## Examples

### Generate content for all repos (dry-run)
    patreon-manager generate --dry-run

### Generate PDF for a specific repo
    patreon-manager generate --repo org/name --format=pdf

## Exit codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Configuration or LLM error |
| 2 | Quality gate rejection (all content below threshold) |
