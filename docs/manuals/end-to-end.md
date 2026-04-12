# End-to-End Walkthrough: Zero to First Published Patreon Post

## Prerequisites
- Go 1.26.1+ (`go version`)
- Podman 4+ (`podman --version`)
- A Patreon creator account with API access
- A GitHub account (and optionally GitLab/GitFlic/GitVerse) with a personal access token

## Step 1: Clone and build

```bash
git clone https://github.com/milos85vasic/My-Patreon-Manager.git
cd My-Patreon-Manager
go build ./...
```

Expected: two binaries `patreon-manager` (CLI) and `patreon-manager-server` (HTTP).

## Step 2: Configure

```bash
cp .env.example .env
$EDITOR .env
```

Set at minimum:
- `PATREON_ACCESS_TOKEN` — from Patreon developer portal
- `GITHUB_TOKEN` — from GitHub Settings > Developer settings > Personal access tokens
- `LLM_API_KEY` and `LLM_PROVIDER` — your LLM provider credentials

## Step 3: Validate

```bash
./patreon-manager validate
```

Expected: `configuration valid`. If errors appear, check docs/troubleshooting/faq.md.

## Step 4: Scan (discovery only)

```bash
./patreon-manager scan --dry-run
```

Expected: a list of repositories that would be processed, filtered by `.repoignore`. No content generated, no Patreon posts created.

## Step 5: Generate content (no publish)

```bash
./patreon-manager generate --dry-run
```

Expected: content generated for each discovered repository, quality-scored by the verifier, but NOT published to Patreon. Review the output to ensure content quality meets your standards.

## Step 6: Full sync (dry-run)

```bash
./patreon-manager sync --dry-run
```

Expected: the complete pipeline runs — scan, generate, publish — but in dry-run mode. No actual Patreon posts are created.

## Step 7: Publish for real

```bash
./patreon-manager publish
```

Expected: pre-generated content is published as tier-gated Patreon posts. The content fingerprint prevents duplicates on re-run.

## Step 8: Verify

1. Check your Patreon page — the post should be visible.
2. Re-run `publish` — no duplicate should be created (idempotent).
3. Check metrics: `curl http://localhost:8080/metrics | grep patreon_`

## Step 9: Monitor (optional)

Start the HTTP server for continuous operation:

```bash
./patreon-manager-server
```

Endpoints:
- `GET /health` — service health
- `GET /metrics` — Prometheus metrics
- `POST /webhook/github` — webhook receiver (configure in GitHub repo settings)
- `GET /admin/audit` — audit log (requires `X-Admin-Key` header)

## Troubleshooting

See docs/troubleshooting/faq.md for common issues.
