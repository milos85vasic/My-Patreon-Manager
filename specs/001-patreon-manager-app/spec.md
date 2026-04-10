# Feature Specification: My Patreon Manager Application

**Feature Branch**: `001-patreon-manager-app`
**Created**: 2026-04-09
**Status**: Draft
**Input**: User description: "Create the My Patreon Manager application based on the main specification document. Full test coverage (unit, e2e, integration, security, stress, benchmark, DDoS, chaos). Full documentation (API docs, user guides, manuals, video courses, diagrams, SQL schemas). Enterprise-grade website."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Full Synchronization Cycle (Priority: P1)

A content creator runs a single command to scan all their Git repositories
across GitHub, GitLab, GitFlic, and GitVerse, generates promotional content
using an LLM, and publishes or updates posts on their Patreon campaign page —
all in one invocation.

**Why this priority**: This is the core value proposition. Without end-to-end
sync, no other feature matters. It validates the entire pipeline: config
loading → Git scanning → content generation → Patreon publishing.

**Independent Test**: Can be fully tested by running a sync command against a
mocked Git service (single repository) and a mocked Patreon API, then verifying
a post was created with the correct content and tier association.

**Acceptance Scenarios**:

1. **Given** valid Patreon and Git credentials are configured, **When** the
   creator triggers a full sync, **Then** all accessible repositories across
   configured services are discovered, filtered, content is generated, and
   corresponding Patreon posts are created or updated.
2. **Given** no repositories have changed since the last sync, **When** the
   creator triggers a sync, **Then** no Patreon posts are updated and a summary
   reports zero changes.
3. **Given** a repository was newly archived, **When** sync runs, **Then** the
   corresponding Patreon post is updated with "maintenance mode" messaging
   without being deleted.

---

### User Story 2 - Dry-Run Preview (Priority: P2)

A content creator wants to preview what changes would be made to their Patreon
page before committing, including which repositories would generate new posts,
which existing posts would be updated, and estimated resource consumption.

**Why this priority**: Creators need confidence before automated mutations.
Dry-run is the safety net that makes the tool trustworthy for production use.

**Independent Test**: Can be tested by running a dry-run sync and verifying
the preview report lists expected operations without any Patreon API write
calls being made.

**Acceptance Scenarios**:

1. **Given** three repositories have changed, **When** the creator runs a
   dry-run sync, **Then** a detailed report shows repository names, change
   reasons, planned content types, and estimated API calls — with zero actual
   Patreon mutations.
2. **Given** a repository would trigger post deletion, **When** dry-run
   executes, **Then** the report clearly flags the pending deletion with the
   grace period status.

---

### User Story 3 - Selective Repository Processing (Priority: P3)

A creator wants to process only a specific organization or a single repository
to conserve API quota and reduce execution time during incremental maintenance.

**Why this priority**: Large portfolios need targeted operations. Without
selective processing, every run wastes quota on unchanged repositories.

**Independent Test**: Can be tested by filtering to a single repository and
verifying only that repository's Patreon post is affected.

**Acceptance Scenarios**:

1. **Given** the creator specifies a single repository URL, **When** sync
   runs with the filter flag, **Then** only that repository is processed and
   all others are skipped.
2. **Given** the creator specifies an organization, **When** sync runs, **Then**
   only repositories under that organization are discovered and processed.

---

### User Story 4 - Content Generation Quality Control (Priority: P4)

A creator wants generated content to meet quality thresholds before it reaches
Patreon, with automatic fallback to alternative LLM models if quality is
unsatisfactory, and human review queue for content that fails all automated
checks.

**Why this priority**: Content quality directly impacts subscriber retention
and creator reputation. Without quality gates, garbage content could be
published automatically.

**Independent Test**: Can be tested by configuring a high quality threshold
and verifying that low-quality generated content is rejected, fallback models
are attempted, and failures end up in the review queue.

**Acceptance Scenarios**:

1. **Given** the primary LLM generates content scoring below the quality
   threshold, **When** the quality gate evaluates the output, **Then** a
   fallback model is automatically selected and regeneration occurs.
2. **Given** all LLM fallbacks produce sub-threshold content, **When** the
   final quality gate fails, **Then** the content is placed in a human review
   queue and no Patreon post is created.

---

### User Story 5 - Scheduled Automated Execution (Priority: P5)

A creator configures recurring synchronization on a schedule (e.g., every 6
hours, daily at 9 AM) without manual intervention, receiving alerts only when
failures occur.

**Why this priority**: Automation is what transforms a one-off tool into a
reliable system. Scheduling enables "set and forget" operation.

**Independent Test**: Can be tested by triggering the scheduler and verifying
that sync executes at the expected intervals with proper error handling and
alerting.

**Acceptance Scenarios**:

1. **Given** a cron schedule is configured for every 6 hours, **When** the
   scheduler triggers, **Then** a full sync executes and completion status is
   logged.
2. **Given** a scheduled sync fails due to a Patreon API outage, **When** the
   failure occurs, **Then** an alert is sent via the configured notification
   channel and the next scheduled run proceeds normally.

---

### User Story 6 - Webhook-Driven Real-Time Updates (Priority: P6)

A creator wants repository push events and releases to automatically trigger
content generation and Patreon updates in near-real-time, without waiting for
a scheduled sync.

**Why this priority**: Real-time responsiveness differentiates this from
batch-only tools. Webhooks enable immediate content updates on releases.

**Independent Test**: Can be tested by sending a simulated webhook payload
and verifying the repository is queued for processing and a Patreon update
occurs.

**Acceptance Scenarios**:

1. **Given** a webhook endpoint is active, **When** a push event is received
   from GitHub, **Then** the affected repository is queued for incremental
   content update.
2. **Given** rapid-fire webhook events arrive for the same repository,
   **When** deduplication processes them, **Then** only one sync operation is
   executed within a 5-minute window.

---

### User Story 7 - Multi-Platform Mirror Detection (Priority: P7)

A creator wants the system to automatically detect when the same repository
exists on multiple Git services and produce content that acknowledges all
platforms with appropriate links.

**Why this priority**: The project mirrors to four services. Mirror-aware
content helps patrons find the best platform for their needs.

**Independent Test**: Can be tested by registering the same repository on
two mocked Git services and verifying the generated content includes both URLs.

**Acceptance Scenarios**:

1. **Given** a repository exists on both GitHub and GitLab with identical
   README content, **When** sync runs, **Then** mirror detection identifies
   the relationship and generated content includes both platform links.
2. **Given** repositories have similar names but different content, **When**
   mirror detection runs, **Then** they are correctly identified as distinct
   projects and no mirror link is generated.

---

### User Story 8 - Premium Content Access Control (Priority: P8)

A creator wants to gate generated PDF documentation and video courses behind
Patreon tiers, with signed download links that expire and streaming access
that verifies active subscription.

**Why this priority**: Monetization is the business goal. Without tier-gated
access control, premium content has no revenue protection.

**Independent Test**: Can be tested by generating a premium PDF, configuring
a tier requirement, and verifying that a mocked patron at the correct tier
can download while a patron at a lower tier is denied.

**Acceptance Scenarios**:

1. **Given** a PDF is associated with a $15+ tier, **When** a patron at the
   $5 tier requests the download, **Then** access is denied with an upgrade
   prompt.
2. **Given** a signed download link was generated, **When** the link expires,
   **Then** further access is denied and a fresh link must be requested.

---

### User Story 9 - Comprehensive Documentation and Website (Priority: P9)

A creator wants a fully documented system with API reference, user guides,
architecture diagrams (in multiple formats), video course materials, SQL
schema documentation, and an enterprise-grade website for public access.

**Why this priority**: Documentation is essential for adoption, onboarding,
and maintenance. The website serves as the public face for potential users.

**Independent Test**: Can be validated by reviewing all documentation
artifacts for completeness and verifying the website renders correctly with
all sections accessible.

**Acceptance Scenarios**:

1. **Given** the documentation suite is generated, **When** a new user reads
   the quick-start guide, **Then** they can set up and run their first sync
   within 30 minutes.
2. **Given** architecture diagrams exist in SVG, PNG, and PDF formats,
   **When** rendered, **Then** all system components, data flows, and
   integration points are clearly visible and labeled.

---

### User Story 10 - Enterprise-Grade Testing Suite (Priority: P10)

The system must demonstrate resilience under all conditions through 100%
test coverage across unit, integration, end-to-end, security, stress,
benchmark, DDoS simulation, and chaos testing.

**Why this priority**: Reliability is non-negotiable for a tool that manages
a creator's revenue stream. Comprehensive testing ensures no regression.

**Independent Test**: Can be validated by running the full test suite and
verifying 100% code coverage with all test categories passing.

**Acceptance Scenarios**:

1. **Given** the test suite executes, **When** all tests complete, **Then**
   code coverage is at least 100% across all packages with zero failures.
2. **Given** a chaos test randomly kills external service connections,
   **When** the system recovers, **Then** no data is lost and all operations
   complete or fail gracefully with proper state persistence.

---

### Edge Cases

- What happens when all configured Git services are unreachable simultaneously?
- How does the system handle a repository with no README file?
- What happens when the Patreon access token expires mid-sync?
- How does the system behave when LLM generation takes longer than the configured timeout?
- What happens when a repository is renamed on the Git service?
- How does the system handle rate limit exhaustion across all four Git services at once?
- What happens when the local state database is corrupted or deleted?
- How does mirror detection handle repositories with identical names but different owners?
- What happens when a webhook payload is malformed or contains invalid signatures?
- How does the system handle concurrent sync executions (lock contention)? → Dual-layer locking: file-based PID lock with stale detection + database-level exclusive lock. Second sync attempt fails fast with a clear message indicating the active sync's PID and start time.
- What happens when a Patreon post was manually edited by the creator after automated publication?
- How does the system behave when disk space is exhausted during video generation?
- What happens when the `.repoignore` file contains invalid patterns?

## Clarifications

### Session 2026-04-09

- Q: How should the system handle concurrent sync executions (lock contention)? → A: Dual-layer locking — file-based lock with stale detection (PID file, auto-release if process died) combined with database-level lock (SQLite `BEGIN EXCLUSIVE` / PostgreSQL advisory lock).
- Q: What is the maximum portfolio size the system should support? → A: Up to 1,000 repositories (prolific creator or small organization).
- Q: What is the scope of the "enterprise-grade website"? → A: Statically generated documentation site with interactive elements (search, versioned API docs, diagrams). Deployable to any static hosting service.

## Requirements *(mandatory)*

### Functional Requirements

**Configuration and Environment**

- **FR-001**: System MUST load all configuration from `.env` files following
  twelve-factor methodology with hierarchical resolution (CLI flags override
  environment variables override `.env` file values override defaults).
- **FR-002**: System MUST validate all required credentials at startup and
  fail fast with descriptive error messages for missing or invalid
  configuration.
- **FR-003**: System MUST support Patreon OAuth2 token refresh with automatic
  detection of expired tokens and in-memory credential update.
- **FR-004**: System MUST redact all sensitive values (tokens, keys, secrets)
  from log output at INFO level and above.

**Multi-Platform Git Integration**

- **FR-005**: System MUST scan repositories across GitHub, GitLab, GitFlic,
  and GitVerse using a unified provider abstraction with per-service adapters.
- **FR-006**: System MUST support organization-level bulk enumeration with
  pagination handling appropriate to each Git service's API.
- **FR-007**: System MUST support explicit individual repository links
  (SSH, HTTPS, SCP-like formats) alongside organization enumeration.
- **FR-008**: System MUST apply `.repoignore` pattern-based filtering with
  wildcard, recursive, character class, and negation support before metadata
  extraction.
- **FR-009**: System MUST detect mirrored repositories across services using
  name matching, README content hashing, and commit SHA comparison.
- **FR-010**: System MUST extract repository metadata including README
  content, topics/tags, language statistics, activity metrics, and archive
  status.

**Content Generation**

- **FR-011**: System MUST generate Patreon-ready content using LLM providers
  routed through a quality-scored model selection service.
- **FR-012**: System MUST support multiple content types: project overviews,
  technical documentation, sponsorship appeals, and update announcements.
- **FR-013**: System MUST generate output in Markdown (primary), HTML, PDF,
  and video (MKV 1080p) formats.
- **FR-014**: System MUST enforce a configurable quality threshold (default
  0.75) for all generated content, rejecting sub-threshold output and
  triggering fallback regeneration.
- **FR-015**: System MUST implement LLM fallback chains with circuit breaker
  patterns preventing cascade failures.

**Patreon Content Lifecycle**

- **FR-016**: System MUST maintain a local state database mapping repository
  identifiers to Patreon post IDs for incremental operations.
- **FR-017**: System MUST perform idempotent operations ensuring repeated
  executions produce consistent state without duplicate content.
- **FR-018**: System MUST handle content lifecycle events: new repository
  (create post), content update (update post), archive status change (adjust
  messaging), repository removal (grace period then archival).
- **FR-019**: System MUST support tier-based content gating with configurable
  tier mapping strategies (linear, modular, exclusive).
- **FR-020**: System MUST support draft, scheduled, and immediate publication
  modes.

**Subscriber Access Control**

- **FR-021**: System MUST verify Patreon tier membership before granting
  access to premium content downloads or streaming.
- **FR-022**: System MUST generate cryptographically signed download URLs
  with expiration timestamps using HMAC-SHA256.
- **FR-023**: System MUST cache membership verification for no more than 5
  minutes with webhook-driven invalidation.

**Execution Modes**

- **FR-024**: System MUST provide CLI subcommands: `sync`, `scan`, `generate`,
  `validate`, `publish` with consistent behavior.
- **FR-025**: System MUST support dry-run mode that previews all changes
  without executing any Patreon mutations.
- **FR-026**: System MUST support selective processing via flags (`--org`,
  `--repo`, `--pattern`, `--since`, `--changed-only`).
- **FR-027**: System MUST support scheduled execution via cron, webhook-driven
  real-time updates, and containerized deployment.
- **FR-028**: System MUST implement checkpointing to enable resume after
  interruption without reprocessing completed work.
- **FR-028a**: System MUST prevent concurrent sync executions using dual-layer
  locking: a file-based PID lock with stale detection (auto-release if the
  owning process is dead) as the outer gate, and a database-level exclusive
  lock (`BEGIN EXCLUSIVE` for SQLite, advisory lock for PostgreSQL) as the
  inner gate. A second sync attempt MUST fail fast with a message indicating
  the active sync's PID and start time.
- **FR-028b**: System MUST support a maximum portfolio of 1,000 repositories
  with stable memory usage and sync completion within reasonable time bounds.
  Pagination and batch processing MUST be used to avoid loading the full
  portfolio into memory simultaneously.

**Resilience and Observability**

- **FR-029**: System MUST implement per-service rate limit handling with
  exponential backoff and service-specific strategies.
- **FR-030**: System MUST implement circuit breakers for all external service
  dependencies with configurable thresholds and cooldown periods.
- **FR-031**: System MUST emit structured metrics for sync duration, success
  rate, repository processing counts, API errors, LLM latency, and quality
  scores.
- **FR-032**: System MUST support configurable log levels (ERROR, WARN, INFO,
  DEBUG, TRACE) with credential redaction at all levels.
- **FR-033**: System MUST handle partial failures by checkpointing completed
  work and continuing with remaining operations.

**Testing**

- **FR-034**: System MUST achieve 100% test coverage across all packages.
- **FR-035**: System MUST include unit tests, integration tests, end-to-end
  tests, security tests, stress tests, benchmark tests, DDoS simulation
  tests, and chaos tests.
- **FR-036**: System MUST use mock services for all external API
  interactions in tests.

**Documentation**

- **FR-037**: System MUST include comprehensive API documentation covering
  all endpoints and CLI commands.
- **FR-038**: System MUST include user guides, quick-start manuals, and
  configuration references.
- **FR-039**: System MUST include architecture diagrams in SVG, PNG, and
  PDF formats.
- **FR-040**: System MUST include SQL schema documentation for the state
  database.
- **FR-041**: System MUST include video course materials covering setup,
  usage, and extension development.
- **FR-042**: System MUST include a statically generated documentation website
  with interactive elements: full-text search, versioned API documentation,
  interactive architecture diagrams, SQL schema browser, downloadable guides,
  and a responsive modern theme. The site MUST be deployable to any static
  hosting service without server-side runtime requirements.

### Key Entities

- **Repository**: Represents a Git repository with service-of-origin, owner,
  name, README content, topics, language stats, activity metrics, archive
  status, and mirror relationships.
- **Campaign**: Represents a Patreon campaign with tiers, benefits, and
  creator identity.
- **Post**: Represents a Patreon post with title, content body, tier
  associations, publication status, and source repository link.
- **Tier**: Represents a Patreon subscription tier with price, description,
  benefit list, and content access rules.
- **SyncState**: Tracks the synchronization state per repository including
  last commit SHA, last sync timestamp, and Patreon post ID mapping.
- **GeneratedContent**: Represents LLM-generated content with quality score,
  model used, prompt parameters, token usage, and generation timestamp.
- **ContentTemplate**: A prompt template with variable placeholders for
  content generation, including type, language, and quality tier.
- **MirrorMap**: A mapping of repositories identified as mirrors across
  services, with canonical source designation.
- **AuditEntry**: A versioned record of content lifecycle events including
  source state, generation parameters, publication metadata, and timestamps.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A creator can run their first successful sync within 30
  minutes of completing setup, generating content for at least one repository.
- **SC-002**: The system processes 100 repositories across all four Git
  services in a single sync within 30 minutes, and supports a maximum
  portfolio of up to 1,000 repositories without degradation.
- **SC-003**: Dry-run preview accurately predicts 100% of actual changes
  that a subsequent real sync would produce.
- **SC-004**: Content quality scores meet or exceed the configured threshold
  for at least 90% of generated outputs without human intervention.
- **SC-005**: The system recovers from any single external service failure
  within 60 seconds without data loss or duplicate content.
- **SC-006**: Test suite achieves 100% code coverage across all categories
  (unit, integration, e2e, security, stress, benchmark, chaos) with zero
  failures.
- **SC-007**: All API endpoints respond within performance thresholds under
  10x normal load during stress testing.
- **SC-008**: Documentation enables a new user to understand system
  architecture and configure their first sync without developer assistance.
- **SC-009**: Premium content access is denied to unauthorized patrons with
  100% accuracy across all access methods (download, streaming, web).
- **SC-010**: Mirror detection correctly identifies mirrored repositories
  with at least 95% precision and 95% recall across the four supported
  platforms.

## Assumptions

- The creator has Patreon API credentials (Client ID, Client Secret, Access
  Token) obtained through the Patreon Developer Portal.
- The creator has API tokens for each Git service they wish to scan
  (GitHub, GitLab, GitFlic, GitVerse).
- The system will initially be deployed as a single-instance CLI tool with
  optional web interface, not requiring distributed coordination.
- LLM provider access is routed through LLMsVerifier for quality-scored
  model selection; direct provider integration is out of scope for initial
  release.
- Video generation is resource-intensive and will be optional, gated behind
  a configuration flag, with script-only fallback.
- The primary deployment target is Linux servers; cross-platform support
  (macOS, Windows) is desirable but secondary.
- The state database defaults to SQLite for single-instance deployments;
  PostgreSQL support is included for scaled deployments.
- Premium content delivery (signed URLs, streaming) requires CDN
  infrastructure that the system configures but does not manage directly.
- The public website is a statically generated documentation site with
  interactive elements (search, versioned API docs, diagrams), deployable to
  any static hosting service without server-side runtime requirements.
- Creators will have basic command-line proficiency for initial setup and
  configuration.
