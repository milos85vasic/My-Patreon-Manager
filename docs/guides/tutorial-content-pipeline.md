# Tutorial: Content Generation Pipeline

This tutorial explains how the LLM content pipeline works end-to-end: from repository discovery through quality gating to tier assignment and rendering.

## Pipeline overview

```
Repositories → Filter → LLM Generate → Quality Gate → Tier Assign → Render → Store
```

## Step 1: Understand the configuration

Key settings in `.env`:

```ini
# LLM provider (openai or anthropic)
LLM_PROVIDER=openai
LLM_API_KEY=sk-your-key

# Quality gate threshold (0.0 to 1.0)
# Content below this score is rejected and retried
CONTENT_QUALITY_THRESHOLD=0.75

# Daily token budget (prevents runaway LLM costs)
LLM_DAILY_TOKEN_BUDGET=100000

# Max concurrent LLM calls (prevents API rate limits)
LLM_CONCURRENCY=8

# Tier mapping: how content is assigned to Patreon tiers
CONTENT_TIER_MAPPING_STRATEGY=linear
```

## Step 2: How content is generated

For each discovered repository, the generator:

1. **Builds a prompt** from the repo metadata (name, description, language, recent commits)
2. **Calls the LLM** via the FallbackChain (tries primary provider, falls back to secondary on failure)
3. **Receives content** with a quality score from the LLM
4. **Runs the quality gate** — if `qualityScore < CONTENT_QUALITY_THRESHOLD`, the content is rejected and retried (up to 3 times with exponential backoff)
5. **Assigns a tier** based on the mapping strategy
6. **Renders** the content in the requested format (markdown, HTML, PDF, video)
7. **Stores** the content with a fingerprint hash for idempotency

## Step 3: Inspect generated content

After running `./patreon-manager generate`:

```bash
# View what was generated (requires sqlite3)
sqlite3 patreon_manager.db "SELECT id, title, tier, quality_score, format FROM generated_content ORDER BY created_at DESC LIMIT 10"
```

Output:
```
id-1|Update on org/my-project|public|0.88|markdown
id-2|Weekly from org/another|bronze|0.92|markdown
```

## Step 4: Template variables

Content templates support Go `text/template` syntax with safe functions:

```markdown
# {{ .RepoName | upper }}

Last commit: {{ .Commit | short }}
Updated: {{ .UpdatedAt | date }}
Language: {{ .Language | default "Unknown" }}
```

Available functions:

| Function | Example | Result |
|----------|---------|--------|
| `upper` | `{{ "hello" \| upper }}` | `HELLO` |
| `lower` | `{{ "HELLO" \| lower }}` | `hello` |
| `trim` | `{{ " hi " \| trim }}` | `hi` |
| `short` | `{{ "deadbeefcafe" \| short }}` | `deadbee` |
| `date` | `{{ .Time \| date }}` | `2026-04-11` |
| `join` | `{{ .Tags \| join ", " }}` | `go, cli` |
| `replace` | `{{ .Name \| replace "-" " " }}` | `my project` |
| `contains` | `{{ if contains .Name "test" }}...{{ end }}` | conditional |
| `default` | `{{ .Lang \| default "Unknown" }}` | fallback value |

## Step 5: Quality gate tuning

```bash
# Generate with a low threshold (accepts most content)
CONTENT_QUALITY_THRESHOLD=0.5 ./patreon-manager generate --dry-run

# Generate with a high threshold (rejects low-quality content)
CONTENT_QUALITY_THRESHOLD=0.95 ./patreon-manager generate --dry-run
```

The quality gate log shows:
```
INFO  content generated  repo=org/repo  quality=0.88  passed=true
INFO  content rejected   repo=org/other quality=0.62  passed=false  retrying=true
```

## Step 6: Tier mapping

The `CONTENT_TIER_MAPPING_STRATEGY` determines how content maps to Patreon tiers:

- **linear**: quality 0.0–0.33 → free, 0.33–0.66 → bronze, 0.66–1.0 → gold
- **weighted**: uses repo activity metrics to weight the tier assignment
- **custom**: define your own mapping in a template

```bash
# See which tier each repo's content would get
./patreon-manager generate --dry-run 2>&1 | grep "tier="
```

## Step 7: Output formats

```bash
# Markdown (default)
./patreon-manager generate --dry-run --format=markdown

# HTML
./patreon-manager generate --dry-run --format=html

# PDF (requires PDF_RENDERING_ENABLED=true)
PDF_RENDERING_ENABLED=true ./patreon-manager generate --dry-run --format=pdf

# Video (requires VIDEO_GENERATION_ENABLED=true + ffmpeg)
VIDEO_GENERATION_ENABLED=true ./patreon-manager generate --dry-run --format=video
```

## Step 8: Token budget management

The daily token budget prevents runaway costs:

```bash
# Set a conservative budget
LLM_DAILY_TOKEN_BUDGET=10000 ./patreon-manager generate --dry-run
```

When the budget is exhausted:
```
WARN  token budget soft alert  utilization=75%
ERROR token budget exhausted   used=10000  limit=10000
```

The soft alert fires at 75% utilization. The hard stop fires at 100%.

## Step 9: Concurrency control

The LLM semaphore prevents overwhelming your API provider:

```bash
# Default: 8 concurrent calls
LLM_CONCURRENCY=8 ./patreon-manager generate

# Conservative: 2 concurrent calls (slower but gentler on the API)
LLM_CONCURRENCY=2 ./patreon-manager generate
```

## Step 10: Circuit breaker behavior

If the LLM provider returns consecutive errors, the circuit breaker trips:

```
ERROR  LLM call failed     provider=openai  error="rate limited"
ERROR  LLM call failed     provider=openai  error="rate limited"
ERROR  LLM call failed     provider=openai  error="rate limited"
WARN   circuit breaker open provider=openai  falling back to secondary
INFO   LLM call succeeded  provider=anthropic
```

The breaker auto-recovers after 30 seconds.

## Step 11: Content fingerprinting

Every piece of generated content gets a fingerprint hash. On re-run:

```bash
./patreon-manager generate   # first run: generates content
./patreon-manager generate   # second run: skips unchanged repos
```

```
INFO  content unchanged  repo=org/repo  fingerprint=abc123  skipping
```

Content is only regenerated when the repo has new commits since the last sync.
