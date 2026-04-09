# Research: My Patreon Manager Application

**Feature Branch**: `001-patreon-manager-app`
**Date**: 2026-04-09

## R1: GitHub API Client Library

**Decision**: Use `google/go-github` (official Go client for GitHub API v3).

**Rationale**: It is the officially maintained Go client with comprehensive
API coverage, automatic pagination, built-in rate limit awareness, strong
typing, and OAuth2 transport support. It is already planned in the constitution
and `go.mod` readiness.

**Alternatives considered**:
- Raw HTTP calls: No type safety, manual pagination handling, more boilerplate.
- `shurcooL/githubv4`: GraphQL-only, more complex for the read-heavy access
  patterns this system uses (list repos, get README, check archive status).

**Best practices**:
- Use `oauth2.StaticTokenSource` for authentication.
- Monitor `RateLimits` service for proactive throttling.
- Use `ListOptions` for pagination control (100 items/page max).
- Handle 403 responses as rate limit indicators with `X-RateLimit-Reset` header.

## R2: GitLab API Client Library

**Decision**: Use `xanzy/go-gitlab` (community-maintained, most comprehensive
Go client for GitLab API v4).

**Rationale**: Covers all needed endpoints (groups, projects, repositories),
supports self-hosted instances via custom base URL, handles pagination and
rate limiting, and is widely adopted in the Go ecosystem.

**Alternatives considered**:
- Raw HTTP calls: Same drawbacks as GitHub raw calls.
- `timelect/gitlab`: Less maintained, fewer contributors.

**Best practices**:
- Support `GITLAB_BASE_URL` for self-hosted instances.
- Use `ListGroupProjects` with recursive descent for nested subgroups.
- Include `statistics` parameter to get star/fork counts without extra calls.
- Handle `Retry-After` header on 429 responses.

## R3: GitFlic and GitVerse API Integration

**Decision**: Implement custom HTTP clients for both services, as no mature
Go client libraries exist for either platform.

**Rationale**: Both services have REST APIs but lack official Go SDKs.
Custom adapters implementing the `RepositoryProvider` interface provide
uniform access while encapsulating platform-specific quirks.

**Alternatives considered**:
- Generic REST client: Too much boilerplate per call; typed structs are better.
- Skip these providers: Violates the multi-platform requirement (Principle III).

**Best practices**:
- Parse API responses into common `Repository` structs.
- Implement capability detection probes for graceful degradation.
- Normalize URL formats (SSH/HTTPS) for consistent comparison.

## R4: Patreon API v2 Client

**Decision**: Implement a custom Patreon API v2 client using standard
`net/http` with OAuth2 token management.

**Rationale**: The Patreon Go ecosystem lacks a mature, maintained client
library. A custom client gives full control over OAuth2 refresh flows,
rate limiting (100 req/min), and error handling specific to Patreon's API.

**Alternatives considered**:
- `mrhavens/patreon-go`: Unmaintained, last updated years ago.
- `benbjohnson/patreon`: Minimal, doesn't cover all needed endpoints.

**Best practices**:
- Implement automatic token refresh on 401 responses.
- Use exponential backoff with jitter for rate limiting.
- Support post CRUD, tier association, and media upload.
- Validate response structures against Patreon API v2 schema.

## R5: LLMsVerifier Integration

**Decision**: Implement an HTTP client for LLMsVerifier's REST API with
quality-scored model selection and circuit breaker integration.

**Rationale**: The constitution mandates all LLM access routes through
LLMsVerifier. The integration consumes REST endpoints for model enumeration,
quality scoring, and completion generation.

**Alternatives considered**:
- Direct LLM provider integration: Prohibited by constitution (Principle IV).
- gRPC client: LLMsVerifier exposes REST, not gRPC.

**Best practices**:
- Implement retry with exponential backoff and timeout handling.
- Cache model quality scores for 5 minutes to reduce API calls.
- Track token usage per generation for cost attribution.
- Implement circuit breaker with configurable cooldown (default 300s).

## R6: State Database (SQLite / PostgreSQL)

**Decision**: Use `mattn/go-sqlite3` for SQLite and `lib/pq` for PostgreSQL
with a database interface allowing either backend.

**Rationale**: SQLite is zero-config for single-instance deployments.
PostgreSQL is needed for production deployments with concurrent access.
The interface abstraction satisfies Principle I (modular plugin architecture).

**Alternatives considered**:
- GORM ORM: Adds unnecessary abstraction for the relatively simple schema.
- sqlc: Code generation adds build complexity; not justified for this schema size.
- Bolt/BadgerDB (key-value): Relational queries needed for mirror detection
  and audit trail.

**Best practices**:
- Use migrations directory with numbered SQL files.
- Implement `BEGIN EXCLUSIVE` for SQLite locking (per FR-028a).
- Use advisory locks for PostgreSQL.
- Index on (service, owner, name) for repository lookups.

## R7: PDF Generation

**Decision**: Use `chromedp` (Go bindings for Chrome DevTools Protocol) to
render HTML to PDF.

**Rationale**: Produces high-quality PDFs with CSS support, proper typography,
and tagged PDF accessibility. The pipeline is Markdown → HTML (with print CSS)
→ chromedp → optimized PDF. Already planned in the constitution.

**Alternatives considered**:
- WeasyPrint (Python): Requires Python runtime alongside Go; deployment complexity.
- `go-pdf/fpdf`: No CSS support; manual layout; poor for complex documents.
- `jspdf` (JS): Requires Node.js runtime; not native Go.

**Best practices**:
- Use headless Chrome with `--no-sandbox` in containerized environments.
- Implement timeout (30s per PDF) and cleanup of temporary files.
- Linearize PDF output for web-optimized streaming.

## R8: Video Generation Pipeline

**Decision**: Use FFmpeg via `os/exec` with scripted assembly from generated
audio and visual content.

**Rationale**: FFmpeg is the industry standard for video encoding. Go's `os/exec`
provides process management. The pipeline generates script → audio (TTS) →
visuals → FFmpeg assembly → MKV output.

**Alternatives considered**:
- `asticode/go-astiav`: Go bindings for libavcodec; complex API, poor docs.
- External microservice: Adds deployment complexity for optional feature.

**Best practices**:
- Gate behind `VIDEO_GENERATION_ENABLED` configuration flag.
- Implement 300-second timeout per video.
- Generate multiple bitrate variants (480p, 720p, 1080p).
- Queue-based processing with concurrency limits.

## R9: .repoignore Pattern Engine

**Decision**: Implement a custom pattern matcher following `.gitignore`
semantics with SSH URL normalization.

**Rationale**: No existing Go library handles the SSH URL format matching
required by the spec. The engine supports exact matches, wildcards (`*`),
recursive wildcards (`**`), character classes (`[a-z]`), and negation (`!`).

**Alternatives considered**:
- `go-git/go-git/internal/plumbing/format/gitignore`: Internal API, not
  designed for external use; doesn't handle SSH URL normalization.
- `sabhiram/go-gitignore`: File-path oriented; doesn't match SSH URL patterns.

**Best practices**:
- Normalize all input URLs to `git@host:owner/repo.git` format before matching.
- Process patterns in declaration order with first-match semantics.
- Support dynamic reload via SIGHUP signal.

## R10: Circuit Breaker Implementation

**Decision**: Use `sony/gobreaker` library for circuit breaker pattern.

**Rationale**: Mature, well-tested library that implements the three-state
circuit breaker pattern (closed → open → half-open). Configurable thresholds,
timeouts, and ready-to-trip functions. Widely used in Go microservices.

**Alternatives considered**:
- Custom implementation: Reinventing a well-solved pattern; error-prone.
- `rakyll/go-stat`: Not a circuit breaker library.

**Best practices**:
- One circuit breaker instance per external service.
- Configure failure threshold at 5 consecutive failures.
- Set open-state cooldown to 60 seconds.
- Log state transitions for operational visibility.

## R11: Static Documentation Website

**Decision**: Use Hugo static site generator with a modern responsive theme.

**Rationale**: Hugo is the fastest static site generator, written in Go
(aligned with the project's language), supports content organization,
full-text search integration, syntax highlighting, and produces deployable
static assets. No server-side runtime required.

**Alternatives considered**:
- MkDocs (Python): Requires Python runtime; slower builds.
- Docusaurus (Node.js): Requires Node.js runtime; heavier.
- Jekyll (Ruby): Slow builds; Ruby dependency.

**Best practices**:
- Store content sources in `docs/website/`.
- Build output to `docs/website/public/` for deployment.
- Support versioned API docs via Hugo's section organization.
- Include interactive architecture diagrams via embedded SVG.
