# Phase 4 — Renderer Completion Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace every renderer stub with a production-quality implementation: full markdown template variable substitution, real PDF via headless Chromium, complete video pipeline with slide + waveform + ffmpeg assembly. Each renderer has golden-file tests and CLI wiring.

**Architecture:** Markdown renderer uses `text/template` with a vetted function set. PDF renderer uses `chromedp` against a containerized headless Chromium (no sudo, no system Chromium). Video pipeline uses bounded worker pool + cancellable ffmpeg subprocesses; slide generator builds PNGs from template, waveform generator from narration audio.

**Tech Stack:** Go 1.26.1, `text/template`, `chromedp`, `ffmpeg` (subprocess, containerized), `golang.org/x/image`, `hashicorp/go-multierror`, Phase-1 `Semaphore`.

**Depends on:** Phase 0, Phase 1. Can run in parallel with Phases 1 and 3.

---

## File Structure

**Modify:**
- `internal/providers/renderer/markdown.go` — real template substitution
- `internal/providers/renderer/markdown_test.go` — add template tests
- `internal/providers/renderer/pdf.go` — real chromedp rendering
- `internal/providers/renderer/pdf_test.go` — golden-file tests
- `internal/providers/renderer/video.go` — real script generation
- `internal/providers/renderer/video_pipeline.go` — real pipeline
- `internal/providers/renderer/video_pipeline_test.go`

**Create:**
- `internal/providers/renderer/template_funcs.go` — sprig-like safe function set
- `internal/providers/renderer/template_funcs_test.go`
- `internal/providers/renderer/pdf_chromedp.go` — Chromium driver
- `internal/providers/renderer/slide_generator.go`
- `internal/providers/renderer/waveform_generator.go`
- `internal/providers/renderer/ffmpeg_driver.go`
- `testdata/golden/markdown/*.md`
- `testdata/golden/pdf/*.pdf.hash`
- `testdata/golden/video/slides/*.png`
- `testdata/golden/video/waveforms/*.png`

---

## Task 1: Markdown template variable substitution

**Files:**
- Modify: `internal/providers/renderer/markdown.go`
- Create: `internal/providers/renderer/template_funcs.go`
- Create: `internal/providers/renderer/template_funcs_test.go`

- [ ] **Step 1: Failing tests**

```go
func TestMarkdownAppliesTemplateVariables(t *testing.T) {
	r := NewMarkdownRenderer()
	out, err := r.Render(context.Background(), Content{
		Body: "Hello {{ .RepoName }} at {{ .Commit | short }}!",
		Vars: map[string]any{"RepoName": "foo/bar", "Commit": "deadbeefcafe"},
	})
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(out), "Hello foo/bar at deadbee!") {
		t.Fatalf("got %q", out)
	}
}

func TestMarkdownRejectsUnsafeFunction(t *testing.T) {
	r := NewMarkdownRenderer()
	_, err := r.Render(context.Background(), Content{Body: `{{ exec "ls" }}`})
	if err == nil {
		t.Fatal("expected error; exec must not be registered")
	}
}
```

- [ ] **Step 2: Implement `template_funcs.go`**

```go
package renderer

import (
	"strings"
	"text/template"
	"time"
)

func SafeFuncs() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,
		"short": func(s string) string {
			if len(s) > 7 { return s[:7] }
			return s
		},
		"now":      func() time.Time { return time.Now().UTC() },
		"date":     func(t time.Time) string { return t.Format("2006-01-02") },
		"rfc3339":  func(t time.Time) string { return t.Format(time.RFC3339) },
		"join":     strings.Join,
		"replace":  strings.ReplaceAll,
		"contains": strings.Contains,
		"default":  func(d, v string) string { if v == "" { return d }; return v },
	}
}
```

- [ ] **Step 3: Implement `applyTemplateVariables`**

```go
func (m *MarkdownRenderer) applyTemplateVariables(body string, vars map[string]any) (string, error) {
	tmpl, err := template.New("md").Option("missingkey=error").Funcs(SafeFuncs()).Parse(body)
	if err != nil {
		return "", fmt.Errorf("markdown: parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("markdown: execute template: %w", err)
	}
	return buf.String(), nil
}
```

- [ ] **Step 4: Run + commit**

```bash
git commit -m "feat(renderer): real markdown template substitution with safe funcs"
```

---

## Task 2: PDF via chromedp

**Files:**
- Modify: `internal/providers/renderer/pdf.go`
- Create: `internal/providers/renderer/pdf_chromedp.go`
- Create: `internal/providers/renderer/pdf_test.go` (rewrite)
- Modify: `docker-compose.yml` — add `headless-chrome` service (optional for local)

- [ ] **Step 1: Failing test**

```go
func TestPDFRendererProducesRealPDF(t *testing.T) {
	if testing.Short() { t.Skip("requires chromedp") }
	r := NewPDFRenderer(PDFOptions{ExecPath: os.Getenv("CHROMEDP_EXEC")})
	out, err := r.Render(context.Background(), Content{Body: "# Hello"})
	if err != nil { t.Fatal(err) }
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
	// Golden hash — stable modulo timestamps
	h := hashModuloTimestamps(out)
	golden := string(mustReadFile(t, "testdata/golden/pdf/hello.pdf.hash"))
	if h != golden { t.Fatalf("hash mismatch: %s vs %s", h, golden) }
}
```

- [ ] **Step 2: Implement chromedp driver**

```go
// internal/providers/renderer/pdf_chromedp.go
package renderer

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type PDFRenderer struct{ opts PDFOptions }

type PDFOptions struct {
	ExecPath  string
	Timeout   time.Duration
	PaperSize string // "A4", "Letter"
}

func NewPDFRenderer(opts PDFOptions) *PDFRenderer {
	if opts.Timeout == 0 { opts.Timeout = 30 * time.Second }
	return &PDFRenderer{opts: opts}
}

func (r *PDFRenderer) Render(ctx context.Context, c Content) ([]byte, error) {
	html, err := MarkdownToHTML(c.Body)
	if err != nil { return nil, err }

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
	)
	if r.opts.ExecPath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(r.opts.ExecPath))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer cancelAlloc()
	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, r.opts.Timeout)
	defer cancelTimeout()

	var pdf []byte
	err = chromedp.Run(taskCtx,
		chromedp.Navigate("data:text/html," + urlEncode(html)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var e error
			pdf, _, e = page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			return e
		}),
	)
	if err != nil { return nil, fmt.Errorf("pdf: %w", err) }
	return pdf, nil
}
```

- [ ] **Step 3: Generate golden hash** — helper script at `scripts/gen_golden_pdf.go` that renders `testdata/input/hello.md` and writes the hash to `testdata/golden/pdf/hello.pdf.hash`.

- [ ] **Step 4: Document** Chromium container in `docker-compose.yml` — optional `headless-chrome` service on port 9222 using `chromedp/headless-shell:latest`.

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(renderer): real PDF via chromedp headless chromium

Containerized Chromium avoids system dependency. Golden-file test
asserts PDF byte structure modulo timestamps."
```

---

## Task 3: Slide generator

**Files:**
- Create: `internal/providers/renderer/slide_generator.go`
- Create: `internal/providers/renderer/slide_generator_test.go`

- [ ] **Step 1: Failing test**

```go
func TestSlideGeneratorProducesStablePNG(t *testing.T) {
	g := NewSlideGenerator(SlideOptions{Width: 1920, Height: 1080})
	png, err := g.Generate(context.Background(), Slide{
		Title:    "Module 1",
		Subtitle: "Intro",
		Body:     "Welcome",
	})
	if err != nil { t.Fatal(err) }
	goldenHash := mustReadString(t, "testdata/golden/video/slides/module1.hash")
	if sha256String(png) != goldenHash {
		t.Fatalf("slide hash mismatch")
	}
}
```

- [ ] **Step 2: Implement** using `golang.org/x/image/font` + `image/png` — fixed fonts, fixed layout, deterministic output.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(renderer): deterministic slide generator with golden-file tests"
```

---

## Task 4: Waveform generator

**Files:**
- Create: `internal/providers/renderer/waveform_generator.go`
- Create: `internal/providers/renderer/waveform_generator_test.go`

- [ ] **Step 1: Failing test** — feed a deterministic sine-wave WAV (`testdata/input/sine440.wav`), assert generated PNG matches golden hash.

- [ ] **Step 2: Implement** — parse WAV, downsample to bar count, draw bars to image/png.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(renderer): waveform generator with golden-file tests"
```

---

## Task 5: ffmpeg driver

**Files:**
- Create: `internal/providers/renderer/ffmpeg_driver.go`
- Create: `internal/providers/renderer/ffmpeg_driver_test.go`

- [ ] **Step 1: Failing test** — assert command composition, not actual execution, via injected `Runner` interface.

```go
func TestFFmpegAssembleCommandComposition(t *testing.T) {
	var captured []string
	d := &FFmpegDriver{Runner: func(ctx context.Context, args ...string) error {
		captured = args; return nil
	}}
	_ = d.Assemble(ctx, AssembleRequest{
		SlidePaths: []string{"s1.png","s2.png"},
		AudioPath:  "a.wav",
		OutPath:    "out.mp4",
	})
	want := []string{"-y","-loop","1","-i","s1.png",...} // full expected
	if !reflect.DeepEqual(captured, want) { t.Fatalf("cmd mismatch") }
}
```

- [ ] **Step 2: Implement**

```go
type FFmpegDriver struct{
	Runner func(ctx context.Context, args ...string) error
}

func (d *FFmpegDriver) Assemble(ctx context.Context, req AssembleRequest) error {
	args := []string{"-y"}
	for _, s := range req.SlidePaths {
		args = append(args, "-loop", "1", "-t", "5", "-i", s)
	}
	args = append(args, "-i", req.AudioPath)
	// filter_complex: concat slides, overlay audio
	args = append(args, "-filter_complex",
		fmt.Sprintf("concat=n=%d:v=1:a=0[v]", len(req.SlidePaths)),
		"-map", "[v]", "-map", fmt.Sprintf("%d:a", len(req.SlidePaths)),
		"-shortest", "-pix_fmt", "yuv420p",
		req.OutPath)
	return d.Runner(ctx, args...)
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(renderer): ffmpeg driver with injected Runner for testability"
```

---

## Task 6: Complete video_pipeline.go

**Files:**
- Modify: `internal/providers/renderer/video_pipeline.go`
- Modify: `internal/providers/renderer/video_pipeline_test.go`

- [ ] **Step 1: Failing integration test** — end-to-end slide + waveform + ffmpeg composition (Runner captured, no real subprocess).

- [ ] **Step 2: Implement** `VideoPipeline.Render` as:

```go
func (p *VideoPipeline) Render(ctx context.Context, script VideoScript) ([]byte, error) {
	if err := p.sem.Acquire(ctx, 1); err != nil { return nil, err }
	defer p.sem.Release(1)

	tmp := t.TempDir() // via injected dir factory in prod
	var slidePaths []string
	for i, slide := range script.Slides {
		png, err := p.slideGen.Generate(ctx, slide)
		if err != nil { return nil, err }
		path := filepath.Join(tmp, fmt.Sprintf("s%03d.png", i))
		if err := os.WriteFile(path, png, 0o600); err != nil { return nil, err }
		slidePaths = append(slidePaths, path)
	}

	waveform, err := p.waveGen.Generate(ctx, script.Audio)
	if err != nil { return nil, err }
	_ = waveform // overlay later

	outPath := filepath.Join(tmp, "out.mp4")
	if err := p.ffmpeg.Assemble(ctx, AssembleRequest{
		SlidePaths: slidePaths,
		AudioPath:  script.AudioPath,
		OutPath:    outPath,
	}); err != nil { return nil, err }

	return os.ReadFile(outPath)
}
```

- [ ] **Step 3: Wire Semaphore from Phase 1** — `p.sem = concurrency.NewSemaphore(int64(cfg.VideoConcurrency))`.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(renderer): full video pipeline with bounded semaphore"
```

---

## Task 7: Wire renderers into CLI and generator

**Files:**
- Modify: `cmd/cli/renderers.go` (from Phase 2)
- Modify: `internal/services/content/generator.go`

- [ ] **Step 1: Failing test**

```go
func TestGenerateVideoFormatWired(t *testing.T) {
	out := runCLI(t, "generate", "--format=video", "--dry-run", "--repo=foo/bar")
	if !strings.Contains(out, "video pipeline") { t.Fatal(out) }
}
```

- [ ] **Step 2: Update `buildRenderers`**:

```go
if cfg.PDFRenderingEnabled {
	rs = append(rs, renderer.NewPDFRenderer(renderer.PDFOptions{ExecPath: cfg.ChromiumExecPath}))
}
if cfg.VideoGenerationEnabled {
	rs = append(rs, renderer.NewVideoPipeline(cfg.VideoConcurrency, slideGen, waveGen, ffmpeg))
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(cli): wire PDF and video renderers behind config flags"
```

---

## Task 8: Phase 4 acceptance

- [ ] Markdown template substitution with golden tests.
- [ ] PDF renderer produces real PDFs (containerized chromedp).
- [ ] Slide + waveform + ffmpeg drivers have deterministic golden tests.
- [ ] `VideoPipeline.Render` end-to-end composition test green.
- [ ] CLI `generate --format=pdf` and `generate --format=video` both work.
- [ ] `internal/providers/renderer/...` at 100% coverage.
- [ ] All new code passes `-race` and goleak.

When every box is checked, Phase 4 ships.
