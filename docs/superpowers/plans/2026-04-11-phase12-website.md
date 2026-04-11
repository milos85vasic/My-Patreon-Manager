# Phase 12 — Website Refresh Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the duplicate Hugo content with a build-time mount from `docs/guides/`, add a GitHub Pages deploy workflow, enable site search (lunr) and dark mode, publish runbooks + ADRs + embedded video references + Redoc-rendered OpenAPI, add a `/security.txt` and a `/status` page sourced from `/health`.

**Architecture:** Hugo mounts `docs/guides/`, `docs/adr/`, `docs/runbooks/`, `docs/manuals/`, and `docs/troubleshooting/` directly — no copies, single source of truth. A small Go helper `scripts/site/status.go` fetches `/health` at build time and writes a JSON fragment into `docs/website/data/status.json`. Redoc renders `docs/api/openapi.yaml`.

**Tech Stack:** Hugo (extended, latest), Ananke theme fork with dark mode, lunr.js, Redoc, GitHub Actions, Pages.

**Depends on:** Phases 0–11.

---

## File Structure

**Create:**
- `.github/workflows/pages.yml`
- `docs/website/layouts/shortcodes/redoc.html`
- `docs/website/static/security.txt`
- `docs/website/content/status/_index.md`
- `docs/website/data/status.json`
- `docs/website/layouts/_default/search.html`
- `docs/website/layouts/partials/dark-mode.html`
- `docs/website/assets/js/search.js`
- `docs/website/assets/css/dark-mode.css`
- `scripts/site/status.go`
- `scripts/site/sync_content.sh`

**Modify:**
- `docs/website/config.toml` — mounts, menu, theme params, baseURL
- `docs/website/content/docs/` — delete duplicates; keep index redirects
- `docs/website/themes/ananke/*` — fork dark mode addon

---

## Task 1: Hugo content mounts replace duplicates

**Files:**
- Modify: `docs/website/config.toml`
- Delete: duplicates under `docs/website/content/docs/`

- [ ] **Step 1: Config**

```toml
baseURL = "https://milos85vasic.github.io/My-Patreon-Manager/"
languageCode = "en-us"
title = "My Patreon Manager"
theme = "ananke"

[module]
[[module.mounts]]
  source = "content"
  target = "content"
[[module.mounts]]
  source = "../guides"
  target = "content/guides"
[[module.mounts]]
  source = "../adr"
  target = "content/adr"
[[module.mounts]]
  source = "../runbooks"
  target = "content/runbooks"
[[module.mounts]]
  source = "../manuals"
  target = "content/manuals"
[[module.mounts]]
  source = "../troubleshooting"
  target = "content/troubleshooting"
[[module.mounts]]
  source = "../api"
  target = "content/api"
[[module.mounts]]
  source = "../architecture"
  target = "content/architecture"
[[module.mounts]]
  source = "../video"
  target = "content/video"

[menu]
[[menu.main]]
  name = "Quickstart"
  url = "/guides/quickstart/"
  weight = 10
[[menu.main]]
  name = "Guides"
  url = "/guides/"
  weight = 20
[[menu.main]]
  name = "Manuals"
  url = "/manuals/"
  weight = 30
[[menu.main]]
  name = "Architecture"
  url = "/architecture/overview/"
  weight = 40
[[menu.main]]
  name = "ADRs"
  url = "/adr/"
  weight = 50
[[menu.main]]
  name = "Runbooks"
  url = "/runbooks/"
  weight = 60
[[menu.main]]
  name = "API"
  url = "/api/redoc/"
  weight = 70
[[menu.main]]
  name = "Status"
  url = "/status/"
  weight = 80
[[menu.main]]
  name = "GitHub"
  url = "https://github.com/milos85vasic/My-Patreon-Manager"
  weight = 90

[params]
  description = "Multi-provider Git scanner, LLM content pipeline, and Patreon publisher."
  enableDarkMode = true
  enableSearch = true
```

- [ ] **Step 2: Remove duplicates**

```bash
git rm docs/website/content/docs/cli-reference.md
git rm docs/website/content/docs/configuration.md
git rm docs/website/content/docs/getting-started.md
git rm docs/website/content/docs/_index.md
```

- [ ] **Step 3: Build locally**

```bash
cd docs/website && hugo --gc --minify
```

Expected: `public/` populated with mounted content.

- [ ] **Step 4: Commit**

```bash
git add docs/website/config.toml
git commit -m "feat(website): hugo mounts replace content duplicates"
```

---

## Task 2: GitHub Pages deploy workflow

**Files:**
- Create: `.github/workflows/pages.yml`

- [ ] **Step 1: Write**

```yaml
# Manual-only per project policy. Must be dispatched explicitly to publish.
name: Pages
on:
  workflow_dispatch:
permissions:
  contents: read
  pages: write
  id-token: write
concurrency:
  group: pages
  cancel-in-progress: true
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive
          fetch-depth: 0
      - uses: peaceiris/actions-hugo@v3
        with:
          hugo-version: latest
          extended: true
      - uses: actions/configure-pages@v5
      - name: Build
        working-directory: docs/website
        run: hugo --gc --minify --baseURL "${{ steps.pages.outputs.base_url }}/"
      - uses: actions/upload-pages-artifact@v3
        with:
          path: docs/website/public
  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

- [ ] **Step 2: Commit**

```bash
git commit -m "ci(pages): build + deploy hugo site to GitHub Pages"
```

---

## Task 3: Redoc shortcode for OpenAPI

**Files:**
- Create: `docs/website/layouts/shortcodes/redoc.html`
- Create: `docs/website/content/api/redoc.md`

- [ ] **Step 1: Shortcode**

```html
<!-- docs/website/layouts/shortcodes/redoc.html -->
<div id="redoc-container"></div>
<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
<script>
Redoc.init('{{ .Get "url" }}', {}, document.getElementById('redoc-container'));
</script>
```

- [ ] **Step 2: Page**

```markdown
---
title: API reference
---
{{< redoc url="/openapi.yaml" >}}
```

- [ ] **Step 3: Copy openapi.yaml into static/**

```bash
cp docs/api/openapi.yaml docs/website/static/openapi.yaml
```

(Alternative: a build-time mount.)

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(website): Redoc-rendered OpenAPI under /api/redoc"
```

---

## Task 4: Site search (lunr)

**Files:**
- Create: `docs/website/assets/js/search.js`
- Create: `docs/website/layouts/_default/search.html`

- [ ] **Step 1: Generate search index** via Hugo output formats:

Add to `config.toml`:

```toml
[outputFormats.SearchIndex]
  mediaType = "application/json"
  baseName  = "index"
  isPlainText = true

[outputs]
  home = ["HTML","RSS","SearchIndex"]
```

And template `layouts/_default/index.searchindex.json`:

```json
[
{{- range $i, $page := .Site.RegularPages }}
  {{- if $i }},{{ end -}}
  {
    "title": {{ $page.Title | jsonify }},
    "url":   {{ $page.Permalink | jsonify }},
    "body":  {{ $page.Plain | jsonify }}
  }
{{- end }}
]
```

- [ ] **Step 2: Client** — lunr bundle loaded in a partial; a `/search/` page renders results.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(website): lunr-based site search"
```

---

## Task 5: Dark mode

**Files:**
- Create: `docs/website/assets/css/dark-mode.css`
- Create: `docs/website/layouts/partials/dark-mode.html`

- [ ] **Step 1: CSS**

```css
:root { color-scheme: light dark; }
@media (prefers-color-scheme: dark) {
  body { background: #0b0e14; color: #e6edf3; }
  a { color: #70b8ff; }
  pre, code { background: #161b22; }
}
html[data-theme="dark"] body { background: #0b0e14; color: #e6edf3; }
html[data-theme="light"] body { background: #fff; color: #111; }
```

- [ ] **Step 2: Toggle partial** — `dark-mode.html` includes a button that flips `data-theme` on `<html>` and persists to localStorage.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(website): prefers-color-scheme dark mode + manual toggle"
```

---

## Task 6: `security.txt`

**Files:**
- Create: `docs/website/static/security.txt`

- [ ] **Step 1: Author**

```text
Contact: mailto:security@example.invalid
Preferred-Languages: en, sr
Canonical: https://milos85vasic.github.io/My-Patreon-Manager/.well-known/security.txt
Policy: https://milos85vasic.github.io/My-Patreon-Manager/security/
Expires: 2027-04-11T00:00:00.000Z
```

- [ ] **Step 2: Hugo mount /.well-known/** — add `[[module.mounts]] target = "static/.well-known/security.txt" source = "static/security.txt"` or copy.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(website): security.txt at /.well-known/"
```

---

## Task 7: Status page from /health

**Files:**
- Create: `scripts/site/status.go`
- Create: `docs/website/data/status.json`
- Create: `docs/website/content/status/_index.md`

- [ ] **Step 1: Build-time fetcher** (`scripts/site/status.go`) — queries `/health` and writes a JSON snapshot. If the server is unreachable, emits an `"unknown"` status.

- [ ] **Step 2: Status page** — `content/status/_index.md` reads `.Site.Data.status` and renders a simple up/down badge.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(website): status page from /health JSON snapshot"
```

---

## Task 8: Video embeds

**Files:**
- Create: `docs/website/layouts/shortcodes/video.html`
- Modify: `docs/website/content/video/_index.md`

- [ ] **Step 1: Shortcode**

```html
<!-- video.html -->
<div class="responsive-video">
  <iframe src="{{ .Get "url" }}" frameborder="0" allowfullscreen></iframe>
</div>
```

- [ ] **Step 2: Embed per module** placeholder (URL filled once recordings exist).

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(website): video embed shortcode and module index page"
```

---

## Task 9: Local build + deploy smoke

- [ ] **Step 1: Build**

```bash
cd docs/website && hugo --gc --minify --baseURL http://localhost:1313/
```

Expected: `public/` populated, no errors.

- [ ] **Step 2: Serve locally**

```bash
hugo server -D
```

Click through every menu item. Fix broken links.

- [ ] **Step 3: Commit**

```bash
git commit -m "docs(website): fix broken internal links discovered during local smoke" --allow-empty
```

---

## Task 10: Phase 12 acceptance

- [ ] Hugo builds without duplicate-content warnings.
- [ ] `.github/workflows/pages.yml` deploys to GitHub Pages on push to main.
- [ ] Redoc renders `openapi.yaml` at `/api/redoc/`.
- [ ] Site search returns results from every mounted directory.
- [ ] Dark mode respects `prefers-color-scheme` and manual toggle persists.
- [ ] `/.well-known/security.txt` reachable.
- [ ] `/status/` page renders `/health` snapshot.
- [ ] Video embeds present (URL-less until Phase 11 recordings uploaded).
- [ ] Lychee link-check green for the built site.

When every box is checked, Phase 12 ships.
