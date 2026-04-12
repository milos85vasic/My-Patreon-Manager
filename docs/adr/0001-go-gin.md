# ADR 0001: Go + Gin as Primary Technology Stack

## Status

Accepted

## Context

My Patreon Manager needs a language and HTTP framework for building a CLI-first application with an optional HTTP server. The application must handle concurrent API calls to multiple Git providers, an LLM pipeline, and the Patreon API. Key requirements include:

- High concurrency for parallel repository scanning across four Git services
- Strong typing to catch integration errors at compile time
- Single-binary deployment for ease of distribution and cron scheduling
- A mature HTTP framework for the server component (health checks, Prometheus metrics, webhooks)
- Cross-platform compilation for diverse deployment environments

Alternatives considered:

- **Python + FastAPI**: Excellent ecosystem for LLM work, but GIL limits true parallelism; deployment requires runtime and dependency management.
- **Rust + Actix**: Superior performance, but higher development velocity cost for a content management tool where API call latency dominates.
- **Node.js + Express**: Strong async I/O, but TypeScript adds build complexity and lacks the single-binary deployment story.

For HTTP frameworks within Go:

- **net/http (stdlib)**: Minimal dependencies but requires manual routing, middleware chaining, and parameter binding.
- **Chi**: Lightweight, stdlib-compatible, but less middleware ecosystem than Gin.
- **Gin**: Battle-tested, high performance, rich middleware ecosystem, excellent JSON binding and validation, widespread community adoption.

## Decision

Use Go 1.26.1 as the primary language and Gin as the HTTP framework.

## Consequences

### Positive

- **Goroutines** provide lightweight concurrency for parallel Git provider scanning without thread pool management.
- **Single binary** compiles to a self-contained executable -- ideal for cron jobs, containers, and CI/CD integration.
- **Gin's middleware chain** simplifies cross-cutting concerns (logging, auth, metrics) with minimal boilerplate.
- **Strong typing** catches provider interface mismatches at compile time rather than runtime.
- **Fast compilation** keeps the development feedback loop tight.
- **Mature ecosystem**: `go-github`, `go-gitlab`, `gobreaker`, `prometheus/client_golang` are all production-grade Go libraries.

### Negative

- Go's error handling verbosity increases code volume compared to exception-based languages.
- Gin is opinionated about JSON serialization; custom content types require extra configuration.
- The Go LLM ecosystem is less mature than Python's, requiring more custom integration code for the content generation pipeline.

### Neutral

- Go's interface-based polymorphism aligns naturally with the plugin architecture (Constitution Principle I).
- The module system (`go.mod`) handles dependency management without external tools.
