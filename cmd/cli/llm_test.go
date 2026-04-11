package main

import (
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/config"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/llm"
	"github.com/stretchr/testify/assert"
)

func TestBuildLLMChain_WithSemaphore(t *testing.T) {
	chain := buildLLMChain(&config.Config{LLMConcurrency: 4, ContentQualityThreshold: 0.75}, nil, nil)
	assert.NotNil(t, chain)
}

func TestBuildLLMChain_DefaultsWhenZero(t *testing.T) {
	// LLMConcurrency == 0 must not create a zero-permit semaphore that would
	// deadlock callers; the helper falls back to 8.
	chain := buildLLMChain(&config.Config{}, nil, nil)
	assert.NotNil(t, chain)
}

func TestBuildLLMChain_NegativeConcurrency(t *testing.T) {
	chain := buildLLMChain(&config.Config{LLMConcurrency: -1}, nil, nil)
	assert.NotNil(t, chain)
}

func TestBuildLLMChain_NilConfig(t *testing.T) {
	chain := buildLLMChain(nil, nil, nil)
	assert.NotNil(t, chain)
}

func TestBuildLLMChain_WithProviders(t *testing.T) {
	providers := []llm.LLMProvider{llm.NewVerifierClient("", "secret", nil)}
	chain := buildLLMChain(&config.Config{LLMConcurrency: 2, ContentQualityThreshold: 0.5}, providers, nil)
	assert.NotNil(t, chain)
}
