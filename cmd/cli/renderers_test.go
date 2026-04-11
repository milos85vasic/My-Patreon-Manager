package main

import (
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/config"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/renderer"
	"github.com/stretchr/testify/assert"
)

func TestBuildRenderers_Default(t *testing.T) {
	rs := buildRenderers(&config.Config{})
	assert.Len(t, rs, 2, "default: markdown + html only")

	formats := rendererFormats(rs)
	assert.Contains(t, formats, "markdown")
	assert.Contains(t, formats, "html")
	assert.NotContains(t, formats, "pdf")
	assert.NotContains(t, formats, "video")
}

func TestBuildRenderers_PDFEnabled(t *testing.T) {
	rs := buildRenderers(&config.Config{PDFRenderingEnabled: true})
	assert.Len(t, rs, 3)

	formats := rendererFormats(rs)
	assert.Contains(t, formats, "pdf")
	assert.NotContains(t, formats, "video")
}

func TestBuildRenderers_VideoEnabled(t *testing.T) {
	rs := buildRenderers(&config.Config{VideoGenerationEnabled: true})
	assert.Len(t, rs, 3)

	formats := rendererFormats(rs)
	assert.Contains(t, formats, "video")
	assert.NotContains(t, formats, "pdf")
}

func TestBuildRenderers_AllEnabled(t *testing.T) {
	rs := buildRenderers(&config.Config{
		PDFRenderingEnabled:    true,
		VideoGenerationEnabled: true,
	})
	assert.Len(t, rs, 4)

	formats := rendererFormats(rs)
	assert.Contains(t, formats, "markdown")
	assert.Contains(t, formats, "html")
	assert.Contains(t, formats, "pdf")
	assert.Contains(t, formats, "video")
}

func TestBuildRenderers_NilConfig(t *testing.T) {
	// Nil config must not panic — it should fall back to defaults.
	rs := buildRenderers(nil)
	assert.Len(t, rs, 2)
}

// rendererFormats extracts the Format() string from each renderer so the
// assertions above can check composition without reflecting on concrete types.
func rendererFormats(rs []renderer.FormatRenderer) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Format())
	}
	return out
}
