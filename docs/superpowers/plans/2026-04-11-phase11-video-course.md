# Phase 11 — Video Course Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce full voiceover scripts for every module (8 existing + 2 new: concurrency patterns, observability), an OBS scene file, recording checklist, caption/SRT templates, an example companion repo layout, and an upload/distribution README. The agent writes everything needed to record; actual recording is the user's responsibility.

**Architecture:** Every module script is a Markdown file with timecoded sections, on-screen commands, narration, and cutaways. OBS scenes are JSON; captions are SRT templates with `{{ timecode }}` placeholders filled from the script. The companion `examples/` layout is scripted so a single helper can spin out a fresh starter repo per exercise.

**Tech Stack:** Markdown, OBS Studio, SRT, `scripts/video/spinout_example.sh`.

**Depends on:** Phases 0–10 (code and docs must reflect what's taught).

---

## File Structure

**Create:**
- `docs/video/scripts/module01-intro.md`
- `docs/video/scripts/module02-configuration.md`
- `docs/video/scripts/module03-sync.md`
- `docs/video/scripts/module04-generate.md`
- `docs/video/scripts/module05-publish.md`
- `docs/video/scripts/module06-admin.md`
- `docs/video/scripts/module07-extending.md`
- `docs/video/scripts/module08-troubleshooting.md`
- `docs/video/scripts/module09-concurrency.md`    (new)
- `docs/video/scripts/module10-observability.md`  (new)
- `docs/video/obs/scenes.json`
- `docs/video/recording-checklist.md`
- `docs/video/captions/moduleNN.srt` (10 files)
- `docs/video/distribution.md`
- `docs/video/README.md` — updated course index
- `examples/README.md` — companion repo index
- `examples/moduleNN/...` — starter files per exercise
- `scripts/video/spinout_example.sh`

**Modify:**
- `docs/video/course-outline.md` — update to reference the scripts, add modules 09 and 10.

---

## Task 1: Script template + module 01

**Files:**
- Create: `docs/video/scripts/module01-intro.md`

- [ ] **Step 1: Template** — every script follows this structure:

```markdown
# Module NN: <title>

Target length: 12 minutes
Audience: <operator|developer|admin>
Prerequisites: modules 01..N-1

## Scene list (timecoded)

### 00:00 — Intro (20s)
[SCENE: talking head]
Narration: "Welcome to module NN of My Patreon Manager..."

### 00:20 — Problem statement (60s)
[SCENE: slide 1 (docs/video/slides/moduleNN/problem.png)]
Narration: "..."

### 01:20 — Demo setup (60s)
[SCENE: terminal]
Commands to run on screen:
    cd /tmp && git clone ...
    cp .env.example .env
Narration: "..."

### 02:20 — Core demo (6m)
[SCENE: terminal + IDE split]
Commands:
    ./patreon-manager sync --dry-run
Narration: "..."

### 08:20 — Deep dive (3m)
[SCENE: IDE full]
File: internal/services/sync/orchestrator.go:120
Narration: "..."

### 11:20 — Exercise (40s)
[SCENE: talking head]
Assignment: "Open examples/moduleNN/ and ..."

### 12:00 — Outro (no spoken)
[SCENE: end card]

## Exercise
<Step-by-step exercise description with expected output.>

## Resources
- Code files shown: ...
- Related docs: ...
- GitHub: https://github.com/milos85vasic/My-Patreon-Manager
```

- [ ] **Step 2: Fill module 01** (Intro) — course overview, architecture map, what the viewer will know by the end.

- [ ] **Step 3: Commit**

```bash
git commit -m "docs(video): script template + module 01 intro"
```

---

## Task 2: Modules 02–08

**Files:** `docs/video/scripts/moduleNN-<topic>.md` × 7

- [ ] **Step 1: Author each** using the template. One module per commit.

Module 02 — Configuration: walk through `.env.example`, `internal/config/config.go`, validation, precedence.

Module 03 — Sync: `patreon-manager sync`, `.repoignore`, multi-provider discovery, mirror detection, dry-run.

Module 04 — Generate: content pipeline, LLM fallback, verifier, tiering.

Module 05 — Publish: Patreon tokens, tier gating, idempotency via fingerprints, audit log.

Module 06 — Admin & operations: `/admin/*`, Grafana, SLOs, backup/restore.

Module 07 — Extending: adding a new provider, a new renderer, a new migration, with the developer manual in hand.

Module 08 — Troubleshooting: common errors, FAQ walkthrough.

- [ ] **Step 2: Commit each** `docs(video): module 02 configuration` etc.

---

## Task 3: New Module 09 — Concurrency patterns

**Files:** `docs/video/scripts/module09-concurrency.md`

- [ ] **Step 1: Content** — Lifecycle, Semaphore, Clock, goleak, race detector, common pitfalls (referenced from Phase 1 fixes).

- [ ] **Step 2: On-screen demo** — walk through `internal/concurrency/` and one fix commit from Phase 1.

- [ ] **Step 3: Commit**

```bash
git commit -m "docs(video): module 09 concurrency patterns"
```

---

## Task 4: New Module 10 — Observability

**Files:** `docs/video/scripts/module10-observability.md`

- [ ] **Step 1: Content** — Prometheus histograms, Grafana dashboard walkthrough, pprof behind admin auth, SLOs, audit store queries.

- [ ] **Step 2: Commit**

```bash
git commit -m "docs(video): module 10 observability"
```

---

## Task 5: OBS scene file

**Files:**
- Create: `docs/video/obs/scenes.json`

- [ ] **Step 1: Author** — scenes for: talking head, terminal only, IDE only, terminal + IDE split, slide, end card.

```json
{
  "scenes": [
    {
      "name": "talking-head",
      "sources": [{"type": "dshow_input", "name": "Webcam"}]
    },
    {
      "name": "terminal",
      "sources": [{"type": "window_capture", "name": "Terminal"}]
    },
    {
      "name": "ide",
      "sources": [{"type": "window_capture", "name": "Editor"}]
    },
    {
      "name": "split-terminal-ide",
      "sources": [
        {"type": "window_capture", "name": "Terminal", "position": [0, 0], "scale": 0.5},
        {"type": "window_capture", "name": "Editor", "position": [960, 0], "scale": 0.5}
      ]
    },
    {
      "name": "slide",
      "sources": [{"type": "image_source", "name": "SlideImage"}]
    },
    {
      "name": "end-card",
      "sources": [{"type": "image_source", "name": "EndCardImage"}]
    }
  ]
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "docs(video): OBS scene collection file"
```

---

## Task 6: Recording checklist

**Files:**
- Create: `docs/video/recording-checklist.md`

- [ ] **Step 1: Author**

```markdown
# Recording Checklist

## Before
- [ ] Close all notification sources (Slack, email, Signal).
- [ ] Set system "Do Not Disturb".
- [ ] Audio gain tested against reference tone.
- [ ] Monitor resolution 1920x1080.
- [ ] Browser zoom 125%, terminal font 16pt, editor font 14pt.
- [ ] Local clock hidden (screen blocker on top bar).
- [ ] `.env` populated with **test** tokens only — never production.

## During
- [ ] Start OBS recording.
- [ ] Start screen recording backup (native tool).
- [ ] Announce module number + date at top of recording.
- [ ] Speak a 3-second pause between paragraphs for easy editing.
- [ ] Type slowly on demos; explain while typing.

## After
- [ ] Stop both recordings.
- [ ] Save raw to `raw/moduleNN-YYYYMMDD.mkv`.
- [ ] Label take number if retaking.
- [ ] Copy `raw/` to a local backup disk before editing.

## Editing
- [ ] Cut dead air > 1s.
- [ ] Normalize audio -16 LUFS.
- [ ] Add captions from `docs/video/captions/moduleNN.srt`.
- [ ] Export H.264 / AAC, 1080p30, CRF 20.
- [ ] Name `patreon-manager-moduleNN.mp4`.
```

- [ ] **Step 2: Commit**

```bash
git commit -m "docs(video): recording checklist"
```

---

## Task 7: Caption/SRT templates

**Files:** `docs/video/captions/moduleNN.srt` × 10

- [ ] **Step 1: Generate templates from scripts** — one SRT per module with timecodes from the script's scene list and narration text split into ≤42-char lines.

- [ ] **Step 2: Commit**

```bash
git commit -m "docs(video): SRT caption templates for modules 01-10"
```

---

## Task 8: Companion `examples/` repo layout

**Files:**
- Create: `examples/README.md`
- Create: `examples/moduleNN/...`

- [ ] **Step 1: Per-module directory** with starter files per the module's exercise:
  - `examples/module02/.env.example.incomplete` — fill-in exercise
  - `examples/module03/repoignore-fixture/` — sample repo tree
  - `examples/module04/llm-stub.go` — stub LLM to swap
  - `examples/module05/patreon-sandbox.env`
  - `examples/module06/grafana-dashboard.json`
  - `examples/module07/add-provider-starter/` — scaffold
  - `examples/module08/broken-config.env`
  - `examples/module09/race-starter.go` — stub with intentional race
  - `examples/module10/metrics-starter.go`

- [ ] **Step 2: `examples/README.md`** indexing every directory with a 1-line goal and expected outcome.

- [ ] **Step 3: Commit**

```bash
git commit -m "docs(video): companion examples/ with starter files per module"
```

---

## Task 9: `spinout_example.sh`

**Files:**
- Create: `scripts/video/spinout_example.sh`

- [ ] **Step 1: Script** copies `examples/moduleNN` to a fresh temp dir + initializes git, for viewers who want a clean start:

```bash
#!/usr/bin/env bash
set -euo pipefail
if [ $# -ne 1 ]; then
  echo "usage: $0 <moduleNN>"
  exit 1
fi
mod="$1"
target="$(mktemp -d)/patreon-manager-$mod"
cp -r "examples/$mod" "$target"
cd "$target"
git init -q
git add .
git commit -q -m "start: $mod"
echo "$target"
```

- [ ] **Step 2: Commit**

```bash
chmod +x scripts/video/spinout_example.sh
git commit -m "docs(video): spinout helper for companion examples"
```

---

## Task 10: Upload/distribution README

**Files:**
- Create: `docs/video/distribution.md`

- [ ] **Step 1: Author** — YouTube channel setup, video metadata template (title / description / tags / end card), Vimeo fallback, companion repo link block, accessibility checklist (captions, transcripts, color contrast).

- [ ] **Step 2: Commit**

```bash
git commit -m "docs(video): distribution README for YouTube + Vimeo + companion repo"
```

---

## Task 11: Update course outline

**Files:**
- Modify: `docs/video/course-outline.md`

- [ ] **Step 1: Reflect** the 10-module structure with links to scripts, SRTs, and examples.
- [ ] **Step 2: Commit**

```bash
git commit -m "docs(video): update course outline to 10 modules"
```

---

## Task 12: Phase 11 acceptance

- [ ] 10 module scripts committed.
- [ ] OBS scenes.json committed.
- [ ] Recording checklist committed.
- [ ] 10 SRT templates committed.
- [ ] `examples/` with 10 module starter directories committed.
- [ ] `scripts/video/spinout_example.sh` committed and executable.
- [ ] `docs/video/distribution.md` committed.
- [ ] Course outline updated to 10 modules.
- [ ] Lychee link-check green for all new files.

When every box is checked, Phase 11 ships.
