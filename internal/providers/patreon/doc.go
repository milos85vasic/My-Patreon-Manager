// Package patreon implements the Patreon API client for managing campaigns,
// posts, and tier-gated content. It includes circuit breaker protection via
// gobreaker, automatic OAuth2 token refresh, and idempotent post operations
// using content fingerprinting.
package patreon
