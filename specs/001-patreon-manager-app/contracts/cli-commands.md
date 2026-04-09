# CLI Command Contracts

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `.env` | Path to configuration file |
| `--dry-run` | bool | false | Preview changes without side effects |
| `--log-level` | string | `info` | Log level: error, warn, info, debug, trace |
| `--json` | bool | false | Output in JSON format for scripting |

## sync

Full synchronization: discover repos → generate content → publish to Patreon.

```
patreon-manager sync [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--org` | string | Process only specified organization |
| `--repo` | string | Process single repository URL |
| `--pattern` | string | Process matching repositories (glob) |
| `--since` | string | Process repos changed since timestamp (RFC3339) |
| `--changed-only` | bool | Skip unchanged repositories |
| `--full` | bool | Force full rescan ignoring state |

**Exit codes**: 0 success, 1 partial failure, 2 configuration error, 3 lock contention.

**Output**:
```
Sync complete: 15 processed, 3 created, 8 updated, 4 unchanged, 0 failed
Duration: 4m32s | Tokens: 12,450 | Est. cost: $0.38
```

## scan

Repository discovery only — no content generation or publishing.

```
patreon-manager scan [flags]
```

Same filter flags as `sync`. Outputs discovered repository list with metadata
summary.

**Output**:
```
Discovered 23 repositories across 4 services
  GitHub:  12 (3 orgs)
  GitLab:   6 (2 orgs)
  GitFlic:  3 (1 org)
  GitVerse: 2 (1 org)
Mirrors detected: 4 groups (8 repositories)
Filtered by .repoignore: 3 excluded
```

## generate

Content generation without publishing. Writes output to local files.

```
patreon-manager generate [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--type` | string | Content type: overview, technical_doc, sponsorship, announcement |
| `--format` | string | Output format: markdown, html, pdf, video_script |
| `--output` | string | Output directory (default: `./generated/`) |
| `--all` | bool | Generate all content types for all repos |

**Output**: Files written to output directory, quality scores printed.

## validate

Validate configuration and test connectivity to all services.

```
patreon-manager validate
```

**Output**:
```
Configuration: VALID
  Patreon API:   CONNECTED (campaign: "My Campaign", 142 patrons)
  GitHub:        CONNECTED (rate limit: 4,850/5,000 remaining)
  GitLab:        CONNECTED (self-hosted: false)
  GitFlic:       CONNECTED
  GitVerse:      CONNECTED
  LLMsVerifier:  CONNECTED (3 models available)
  Database:      CONNECTED (SQLite, 23 repos tracked)
```

**Exit codes**: 0 all valid, 1 some failures, 2 config missing.

## publish

Push previously generated content to Patreon.

```
patreon-manager publish [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--input` | string | Input directory (default: `./generated/`) |
| `--draft` | bool | Publish as draft instead of immediate |
| `--schedule` | string | Schedule publication (RFC3339 timestamp) |
| `--tier` | string | Override tier association (tier ID) |
