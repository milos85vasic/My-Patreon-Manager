# ADR 0003: gobreaker for Circuit Breaker Implementation

## Status

Accepted

## Context

My Patreon Manager interacts with multiple external APIs (GitHub, GitLab, GitFlic, GitVerse, Patreon, LLM providers), each with distinct failure modes and rate limits. Constitution Principle VI mandates circuit breaker patterns for all external service interactions:

> Every external service interaction MUST implement rate limiting, exponential backoff, and circuit breaker patterns.

The circuit breaker must support the standard state machine: closed (normal operation) -> open (on threshold breach, fail fast) -> half-open (after cooldown, probe with limited requests) -> closed (on probe success).

Requirements:

- Thread-safe for concurrent goroutine access during parallel repository scanning
- Configurable failure thresholds, cooldown periods, and success thresholds for half-open state
- Per-provider circuit breaker instances (each Git service and the Patreon API gets its own breaker)
- Observable state transitions for Prometheus metrics integration

Alternatives considered:

- **Custom implementation**: Full control but requires careful testing of edge cases (concurrent state transitions, timer management, reset logic). Maintenance burden for a well-understood pattern.
- **go-kit circuit breaker**: Part of the go-kit microservices toolkit. Pulls in a large dependency tree for a single pattern. The project does not use go-kit for anything else.
- **hystrix-go**: Port of Netflix Hystrix. Feature-rich but the upstream Java project is in maintenance mode. Heavier than needed (includes metrics, dashboard integration, request collapsing).
- **gobreaker (sony/gobreaker)**: Purpose-built circuit breaker library from Sony. Minimal API surface, well-tested, no transitive dependencies, active maintenance.

## Decision

Use `github.com/sony/gobreaker` (v1.0.0) as the circuit breaker implementation for all external service interactions.

## Consequences

### Positive

- **Minimal API**: `gobreaker.NewCircuitBreaker(settings)` + `cb.Execute(func)` -- easy to wrap around any provider call.
- **Zero transitive dependencies**: Does not pull in additional packages, keeping the dependency tree lean.
- **Thread-safe**: Safe for concurrent use across goroutines scanning multiple repositories in parallel.
- **Configurable callbacks**: `OnStateChange` callback integrates directly with the Prometheus metrics collector to track `circuit_breaker_state_changes_total`.
- **Battle-tested**: Used in production by Sony and widely adopted in the Go ecosystem.
- **Settings per instance**: Each provider gets its own `CircuitBreaker` with tuned thresholds (e.g., GitHub's 5,000/hr vs. Patreon's 100/min).

### Negative

- gobreaker does not include built-in rate limiting or exponential backoff -- these are implemented separately using `golang.org/x/time/rate` and custom retry logic.
- The v1.0.0 API lacks some features available in newer versions (e.g., two-step execution for streaming responses). This has not been a practical limitation so far.
- No built-in metrics export -- state change metrics are wired manually via the `OnStateChange` callback.

### Neutral

- Circuit breaker instances are created during provider initialization and passed to the provider constructors via dependency injection, consistent with the project's DI pattern.
- The `internal/metrics/circuitbreaker.go` module bridges gobreaker state changes to Prometheus counters.
