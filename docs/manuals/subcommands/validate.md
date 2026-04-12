# `patreon-manager validate`

## Purpose
Validates the configuration: checks all required environment variables, verifies token formats, tests database connectivity, and reports any issues.

## Usage
    patreon-manager validate

## Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --json | bool | false | Output validation results as JSON |
| --log-level | string | "info" | Log verbosity |

## Examples

### Basic validation
    patreon-manager validate

### JSON output for CI
    patreon-manager validate --json

## Exit codes
| Code | Meaning |
|------|---------|
| 0 | Configuration valid |
| 1 | Configuration invalid — details in output |
