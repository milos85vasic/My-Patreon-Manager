# Tutorial: Your First Sync (Step by Step)

This tutorial walks you through setting up My Patreon Manager from scratch and running your first repository sync. Every command shows the expected output so you can verify each step.

## Step 1: Install Go

```bash
go version
```

Expected output:
```
go version go1.26.1 linux/amd64
```

If Go is not installed, follow https://go.dev/doc/install.

## Step 2: Clone the repository

```bash
git clone https://github.com/milos85vasic/My-Patreon-Manager.git
cd My-Patreon-Manager
```

Expected: the repository is cloned into `My-Patreon-Manager/`.

## Step 3: Build both binaries

```bash
go build -o patreon-manager ./cmd/cli
go build -o patreon-manager-server ./cmd/server
```

Expected: two binaries created in the current directory. Verify:

```bash
ls -la patreon-manager patreon-manager-server
```

## Step 4: Create your configuration

```bash
cp .env.example .env
```

Open `.env` in your editor. Set these required fields:

```ini
# Server settings (defaults are fine for local testing)
PORT=8080
GIN_MODE=debug
LOG_LEVEL=info

# At minimum, set ONE git provider token:
GITHUB_TOKEN=ghp_your_personal_access_token_here

# For content generation, set an LLM provider:
LLM_PROVIDER=openai
LLM_API_KEY=sk-your-openai-key-here

# For publishing (skip if just testing sync):
# PATREON_ACCESS_TOKEN=your_patreon_token
```

### How to get a GitHub token:
1. Go to https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Select scopes: `repo` (read access to repositories)
4. Copy the token into `.env`

### How to get an OpenAI API key:
1. Go to https://platform.openai.com/api-keys
2. Create a new key
3. Copy into `.env` as `LLM_API_KEY`

## Step 5: Validate configuration

```bash
./patreon-manager validate
```

Expected output:
```
configuration valid
```

If you see errors, they tell you exactly which field is missing or invalid. Fix and re-run.

### Common validation errors:

| Error | Fix |
|-------|-----|
| `GITHUB_TOKEN required` | Set `GITHUB_TOKEN` in `.env` |
| `LLM_PROVIDER must be one of: openai, anthropic` | Set `LLM_PROVIDER=openai` |
| `invalid PORT` | Set `PORT=8080` |

## Step 6: Create a .repoignore file (optional)

```bash
cat > .repoignore << 'EOF'
# Exclude forks and archived repos
*-fork
archived-*

# Exclude specific repos
my-private-notes
old-experiments
EOF
```

This works like `.gitignore` — patterns exclude matching repos from sync.

## Step 7: Scan repositories (discovery only)

```bash
./patreon-manager scan --dry-run
```

Expected output:
```
INFO  repositories discovered  count=15
INFO  repoignore filtered      before=15 after=12
```

This shows how many repos were found and how many passed the filter. No content is generated, nothing is published.

### Scan a specific organization:

```bash
./patreon-manager scan --org your-github-username --dry-run
```

### Scan with JSON output:

```bash
./patreon-manager scan --json 2>/dev/null | jq '.[] | .full_name'
```

## Step 8: Generate content (dry-run)

```bash
./patreon-manager generate --dry-run
```

Expected output:
```
INFO  content generated  repo=your-org/your-repo  quality=0.85  tier=public
INFO  content generated  repo=your-org/another     quality=0.92  tier=bronze
```

This calls the LLM to generate content for each repo, scores it with the quality gate, and assigns tiers. In dry-run mode, nothing is saved to the database.

### Generate for a single repo:

```bash
./patreon-manager generate --dry-run --repo your-org/your-repo
```

## Step 9: Full sync (dry-run)

```bash
./patreon-manager sync --dry-run
```

Expected: the complete pipeline runs — scan, generate, tier — but nothing is written to Patreon. Review the output to verify everything looks correct.

## Step 10: Run for real

When you're satisfied with the dry-run output:

```bash
./patreon-manager sync
```

Expected:
```
INFO  sync started
INFO  repositories discovered  count=12
INFO  content generated        repo=org/repo  quality=0.88
INFO  post created             repo=org/repo  tier=public  patreon_id=12345
INFO  sync completed           processed=12  failed=0  skipped=0
```

## Step 11: Verify idempotency

Run the same command again:

```bash
./patreon-manager sync
```

Expected: no duplicate posts are created. Content fingerprinting ensures re-runs are safe.

```
INFO  sync completed  processed=12  failed=0  skipped=12
```

All 12 repos are skipped because their content hasn't changed.

## Step 12: Start the HTTP server (optional)

For webhook-driven continuous operation:

```bash
./patreon-manager-server
```

Expected:
```
INFO  server started  port=8080
```

Verify:
```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## Next steps

- Set up webhooks: [docs/manuals/admin.md](../manuals/admin.md)
- Configure scheduled sync: `./patreon-manager sync --schedule "@every 1h"`
- Monitor with Grafana: import `ops/grafana/dashboard.json`
- Read the full CLI reference: [docs/api/cli-reference.md](../api/cli-reference.md)
