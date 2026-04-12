# Module 04: Content Generation

Target length: 12 minutes
Audience: operators, content creators

## Scene list

### 00:00 — Content pipeline overview (60s)
Narration: "Generation takes discovered repos, feeds them to an LLM with quality gates, and produces tier-gated content."

### 01:00 — LLM fallback chain (3m)
[SCENE: IDE showing internal/providers/llm/fallback.go]
Narration: "The FallbackChain tries providers in order. Circuit breakers prevent hammering a failing provider. A global semaphore caps concurrency."

### 04:00 — Quality verifier (2m)
[SCENE: IDE showing internal/providers/llm/verifier.go]
Narration: "Every generated piece is scored. Below the threshold it's rejected and retried."

### 06:00 — Tier mapping (2m)
Narration: "Content is assigned to Patreon tiers based on the mapping strategy (linear, weighted, or custom)."

### 08:00 — Template variables (2m)
[SCENE: terminal]
Commands:
    cat content-template.md
    ./patreon-manager generate --dry-run --repo org/name
Narration: "Templates use Go text/template syntax with safe functions: upper, lower, short, join, replace."

### 10:00 — Renderers (90s)
Narration: "Content renders to Markdown (default), HTML, PDF (chromedp), or video (ffmpeg pipeline)."

### 11:30 — Exercise

## Exercise
1. Run `generate --dry-run` for a specific repo.
2. Modify CONTENT_QUALITY_THRESHOLD and observe the verifier's effect.
3. Try `--format=pdf` (requires PDF_RENDERING_ENABLED=true).

## Resources
- docs/guides/content-generation.md
