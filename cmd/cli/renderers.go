package main

import (
	"github.com/milos85vasic/My-Patreon-Manager/internal/config"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/renderer"
)

// buildRenderers returns the list of renderers to register with the content
// generator based on the given config. Markdown + HTML are always enabled;
// PDF and Video are opt-in via the PDF_RENDERING_ENABLED and
// VIDEO_GENERATION_ENABLED env keys.
//
// Phase 2 Task 5 wires this into the CLI composition root, replacing the
// hard-coded [Markdown, HTML] slice that lived in main.go. Phase 4 will flesh
// out the PDF / Video implementations behind these same flags.
func buildRenderers(cfg *config.Config) []renderer.FormatRenderer {
	if cfg == nil {
		cfg = config.NewConfig()
	}
	rs := []renderer.FormatRenderer{
		renderer.NewMarkdownRenderer(),
		renderer.NewHTMLRenderer(),
	}
	if cfg.PDFRenderingEnabled {
		rs = append(rs, renderer.NewPDFRenderer())
	}
	if cfg.VideoGenerationEnabled {
		rs = append(rs, renderer.NewVideoPipeline(true))
	}
	return rs
}
