# ADR 0002: SQLite as Default Database

## Status

Accepted

## Context

My Patreon Manager requires persistent state to track repository metadata, content fingerprints, Patreon post mappings, and sync checkpoints. The database must support:

- Idempotent operations via content fingerprinting (Constitution Principle II)
- Per-repository state tracking for incremental change detection (Principle III)
- Local-to-Patreon post ID mapping for lifecycle integrity (Principle V)
- Audit log retention for 7 years (Principle V)

The application targets individual content creators running the tool on personal machines or small VPS instances, with an optional path to production-scale deployments.

Alternatives considered:

- **PostgreSQL only**: Production-grade but requires a running server, making local development and single-binary deployment more complex.
- **Embedded key-value store (BoltDB/BadgerDB)**: Simpler than SQL, but the relational queries needed for repository-to-post mapping and audit trails are awkward in key-value stores.
- **JSON file storage**: Zero dependencies, but no transactional safety, poor query performance, and no concurrent access support.

## Decision

Use SQLite as the default database with PostgreSQL as a supported alternative for production deployments. Database access is abstracted behind a common interface, allowing the choice to be made via configuration (`DB_DRIVER` environment variable).

## Consequences

### Positive

- **Zero infrastructure**: SQLite requires no server process -- the database is a single file alongside the binary, perfect for cron-based CLI usage.
- **ACID transactions**: Content fingerprint checks and post ID updates happen atomically, preventing duplicate content on crash recovery.
- **Portable backups**: Database backup is a file copy (`cp patreon-manager.db patreon-manager.db.bak`).
- **Fast local development**: No Docker compose or database server setup required.
- **PostgreSQL upgrade path**: The database interface abstraction means switching to PostgreSQL is a configuration change, not a code change.

### Negative

- SQLite does not support concurrent writers well -- the CLI's single-process model makes this acceptable, but the HTTP server under high webhook load could hit contention.
- SQLite lacks some PostgreSQL features (e.g., `LISTEN/NOTIFY`, advanced JSON operators) that could be useful for webhook-driven cache invalidation.
- The `mattn/go-sqlite3` driver requires CGo, which complicates cross-compilation. The `modernc.org/sqlite` pure-Go alternative exists but is not currently used.

### Neutral

- Both SQLite and PostgreSQL use the same SQL schema (defined in `docs/architecture/sql-schema.md`), with minor dialect differences handled in the database package.
- Migration tooling works identically across both backends.
