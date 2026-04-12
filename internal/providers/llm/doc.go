// Package llm implements the LLMProvider interface with quality-scored model
// selection, automatic fallback chains, and content quality gates. It routes
// generation requests through LLMsVerifier for provider abstraction and
// quality validation.
package llm
