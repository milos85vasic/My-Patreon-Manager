package main

import (
	"github.com/milos85vasic/My-Patreon-Manager/internal/concurrency"
	"github.com/milos85vasic/My-Patreon-Manager/internal/config"
	"github.com/milos85vasic/My-Patreon-Manager/internal/metrics"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/llm"
)

// buildLLMChain constructs the FallbackChain used by the content generator.
// It wraps NewFallbackChain with a concurrency.Semaphore sized from
// cfg.LLMConcurrency so in-flight LLM calls are globally capped across every
// provider in the chain. A non-positive concurrency value falls back to the
// config default (8) to avoid creating a zero-permit semaphore that would
// deadlock callers.
//
// Phase 1 Task 14 added WithSemaphore to fallback.go but left the CLI
// composition root unchanged; Phase 2 Task 5 finishes that wiring.
func buildLLMChain(cfg *config.Config, providers []llm.LLMProvider, m metrics.MetricsCollector) *llm.FallbackChain {
	if cfg == nil {
		cfg = config.NewConfig()
	}
	concurrencyLimit := cfg.LLMConcurrency
	if concurrencyLimit <= 0 {
		concurrencyLimit = 8
	}
	sem := concurrency.NewSemaphore(int64(concurrencyLimit))
	return llm.NewFallbackChain(providers, cfg.ContentQualityThreshold, m, llm.WithSemaphore(sem))
}
