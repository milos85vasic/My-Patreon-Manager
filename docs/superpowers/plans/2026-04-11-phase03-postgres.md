# Phase 3 — PostgreSQL Backend Completion Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement every stub in `internal/database/postgres.go:323–410` to full parity with the SQLite backend, populate `internal/database/migrations/` with versioned DDL, and verify with a shared interface test suite under both backends via `testcontainers-go`.

**Architecture:** Replace the nil-returning methods with `pgx/v5`-backed prepared statements. Introduce `golang-migrate` as the migration runner, wrapped behind an advisory-lock gate for multi-instance safety. A shared parity test harness runs the same matrix against SQLite and PostgreSQL.

**Tech Stack:** Go 1.26.1, `github.com/jackc/pgx/v5`, `github.com/golang-migrate/migrate/v4`, `github.com/testcontainers/testcontainers-go`, `github.com/testcontainers/testcontainers-go/modules/postgres`.

**Depends on:** Phase 0 (CI + scan), Phase 1 (Lifecycle, concurrency primitives). Can run in parallel with Phases 1 and 4.

---

## File Structure

**Create:**
- `internal/database/migrations/0001_init.up.sql`
- `internal/database/migrations/0001_init.down.sql`
- `internal/database/migrations/0002_audit.up.sql`
- `internal/database/migrations/0002_audit.down.sql`
- `internal/database/migrate.go` — runner with advisory lock
- `internal/database/migrate_test.go`
- `internal/database/postgres_stores.go` — concrete implementations
- `tests/integration/database/parity_test.go`
- `tests/integration/database/helpers.go` — testcontainers bootstrap

**Modify:**
- `internal/database/postgres.go` — replace every stub with real implementation
- `internal/database/postgres_test.go` — replace `// stub returns nil` comments with actual assertions

---

## Task 1: Write migration files

**Files:**
- Create: `internal/database/migrations/0001_init.up.sql`
- Create: `internal/database/migrations/0001_init.down.sql`
- Create: `internal/database/migrations/0002_audit.up.sql`
- Create: `internal/database/migrations/0002_audit.down.sql`

- [ ] **Step 1: Write 0001_init.up.sql**

```sql
-- 0001_init.up.sql
BEGIN;

CREATE TABLE IF NOT EXISTS repositories (
    id            TEXT PRIMARY KEY,
    provider      TEXT NOT NULL,
    owner         TEXT NOT NULL,
    name          TEXT NOT NULL,
    full_name     TEXT NOT NULL UNIQUE,
    default_branch TEXT,
    visibility    TEXT,
    mirror_of     TEXT,
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata      JSONB NOT NULL DEFAULT '{}'::JSONB
);
CREATE INDEX IF NOT EXISTS repositories_provider_idx ON repositories(provider);
CREATE INDEX IF NOT EXISTS repositories_mirror_of_idx ON repositories(mirror_of);

CREATE TABLE IF NOT EXISTS sync_states (
    repo_id       TEXT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    last_commit   TEXT,
    checkpoint    TEXT,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mirror_maps (
    canonical_id  TEXT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    mirror_id     TEXT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    PRIMARY KEY (canonical_id, mirror_id)
);

CREATE TABLE IF NOT EXISTS content_templates (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL UNIQUE,
    body      TEXT NOT NULL,
    version   INT  NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS generated_content (
    id          TEXT PRIMARY KEY,
    repo_id     TEXT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    template_id TEXT REFERENCES content_templates(id),
    fingerprint TEXT NOT NULL,
    body        TEXT NOT NULL,
    tier        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (repo_id, fingerprint)
);
CREATE INDEX IF NOT EXISTS generated_content_tier_idx ON generated_content(tier);

CREATE TABLE IF NOT EXISTS posts (
    id          TEXT PRIMARY KEY,
    content_id  TEXT NOT NULL REFERENCES generated_content(id) ON DELETE CASCADE,
    patreon_id  TEXT UNIQUE,
    published   BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    tier        TEXT NOT NULL,
    url         TEXT
);

CREATE TABLE IF NOT EXISTS quality_reviews (
    id           TEXT PRIMARY KEY,
    content_id   TEXT NOT NULL REFERENCES generated_content(id) ON DELETE CASCADE,
    verifier     TEXT NOT NULL,
    score        REAL NOT NULL,
    passed       BOOLEAN NOT NULL,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS locks (
    key         TEXT PRIMARY KEY,
    holder      TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);

COMMIT;
```

- [ ] **Step 2: Write 0001_init.down.sql**

```sql
BEGIN;
DROP TABLE IF EXISTS locks;
DROP TABLE IF EXISTS quality_reviews;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS generated_content;
DROP TABLE IF EXISTS content_templates;
DROP TABLE IF EXISTS mirror_maps;
DROP TABLE IF EXISTS sync_states;
DROP TABLE IF EXISTS repositories;
COMMIT;
```

- [ ] **Step 3: Write 0002_audit.up.sql**

```sql
BEGIN;
CREATE TABLE IF NOT EXISTS audit_entries (
    id          TEXT PRIMARY KEY,
    actor       TEXT NOT NULL,
    action      TEXT NOT NULL,
    target      TEXT,
    outcome     TEXT,
    metadata    JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS audit_entries_created_at_idx ON audit_entries(created_at DESC);
CREATE INDEX IF NOT EXISTS audit_entries_action_idx ON audit_entries(action);
COMMIT;
```

- [ ] **Step 4: Write 0002_audit.down.sql**

```sql
BEGIN;
DROP TABLE IF EXISTS audit_entries;
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git add internal/database/migrations/
git commit -m "feat(db): versioned golang-migrate SQL migrations

0001_init: 8 core tables with FKs and indexes.
0002_audit: audit_entries table from Phase 2."
```

---

## Task 2: Migration runner with advisory lock

**Files:**
- Create: `internal/database/migrate.go`
- Create: `internal/database/migrate_test.go`

- [ ] **Step 1: Failing test**

```go
// internal/database/migrate_test.go
package database

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestMigrateUpPostgres(t *testing.T) {
	ctx := context.Background()
	pgC, err := postgres.Run(ctx, "docker.io/library/postgres:16-alpine",
		postgres.WithDatabase("pm"),
		postgres.WithUsername("pm"),
		postgres.WithPassword("pm"),
	)
	if err != nil { t.Skip("testcontainers unavailable:", err) }
	defer pgC.Terminate(ctx)
	dsn, _ := pgC.ConnectionString(ctx, "sslmode=disable")
	if err := Migrate(ctx, dsn, "file://migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/database/migrate.go
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate(ctx context.Context, dsn, source string) error {
	m, err := migrate.New(source, dsn)
	if err != nil {
		return fmt.Errorf("migrate: open: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Run**

```bash
go test -race ./internal/database/ -run TestMigrateUpPostgres
```

Expected: PASS (or skip if the container cannot start; the CI matrix will run it).

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(db): golang-migrate runner with postgres + sqlite drivers"
```

---

## Task 3: Implement PostgresRepositoryStore

**Files:**
- Modify: `internal/database/postgres.go`
- Create: `internal/database/postgres_repositories_test.go`

- [ ] **Step 1: Failing tests** — CRUD + list-by-provider + mirror detection round-trip.

```go
func TestPostgresRepositoryStoreCRUD(t *testing.T) {
	s := newPGStore(t)
	r := models.Repo{ID: "r1", Provider: "github", Owner: "o", Name: "n", FullName: "o/n"}
	if err := s.Create(ctx, r); err != nil { t.Fatal(err) }
	got, err := s.Get(ctx, r.ID)
	if err != nil || got.FullName != r.FullName { t.Fatalf("%v %v", got, err) }
	r.DefaultBranch = "main"
	if err := s.Update(ctx, r); err != nil { t.Fatal(err) }
	list, _ := s.ListByProvider(ctx, "github")
	if len(list) != 1 { t.Fatal("list") }
	if err := s.Delete(ctx, r.ID); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 2: Implement**

```go
func (s *PostgresRepositoryStore) Create(ctx context.Context, r models.Repo) error {
	meta, _ := json.Marshal(r.Metadata)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO repositories(id,provider,owner,name,full_name,default_branch,visibility,mirror_of,last_seen_at,metadata)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,NOW(),$9)`,
		r.ID, r.Provider, r.Owner, r.Name, r.FullName, r.DefaultBranch, r.Visibility, r.MirrorOf, meta)
	return err
}

func (s *PostgresRepositoryStore) Get(ctx context.Context, id string) (models.Repo, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id,provider,owner,name,full_name,default_branch,visibility,mirror_of,last_seen_at,metadata
		 FROM repositories WHERE id=$1`, id)
	var r models.Repo
	var meta []byte
	err := row.Scan(&r.ID, &r.Provider, &r.Owner, &r.Name, &r.FullName,
		&r.DefaultBranch, &r.Visibility, &r.MirrorOf, &r.LastSeenAt, &meta)
	if err != nil {
		return models.Repo{}, err
	}
	_ = json.Unmarshal(meta, &r.Metadata)
	return r, nil
}

func (s *PostgresRepositoryStore) Update(ctx context.Context, r models.Repo) error {
	meta, _ := json.Marshal(r.Metadata)
	_, err := s.pool.Exec(ctx,
		`UPDATE repositories SET provider=$2, owner=$3, name=$4, full_name=$5,
		 default_branch=$6, visibility=$7, mirror_of=$8, last_seen_at=NOW(), metadata=$9
		 WHERE id=$1`,
		r.ID, r.Provider, r.Owner, r.Name, r.FullName, r.DefaultBranch, r.Visibility, r.MirrorOf, meta)
	return err
}

func (s *PostgresRepositoryStore) Delete(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM repositories WHERE id=$1`, id)
	return err
}

func (s *PostgresRepositoryStore) ListByProvider(ctx context.Context, provider string) ([]models.Repo, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,provider,owner,name,full_name,default_branch,visibility,mirror_of,last_seen_at,metadata
		 FROM repositories WHERE provider=$1 ORDER BY full_name`, provider)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Repo
	for rows.Next() {
		var r models.Repo
		var meta []byte
		if err := rows.Scan(&r.ID, &r.Provider, &r.Owner, &r.Name, &r.FullName,
			&r.DefaultBranch, &r.Visibility, &r.MirrorOf, &r.LastSeenAt, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &r.Metadata)
		out = append(out, r)
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: Run + commit**

```bash
git commit -m "feat(db): implement PostgresRepositoryStore with pgx/v5"
```

---

## Task 4: Implement remaining stores (SyncState, MirrorMap, GeneratedContent, Post, ContentTemplate, QualityReview, AuditEntry, Lock)

Follow the Task-3 pattern. One commit per store:

- [ ] **Task 4a:** `PostgresSyncStateStore.{Create,Get,Update,Delete,UpdateCheckpoint}`
- [ ] **Task 4b:** `PostgresMirrorMapStore.{Create,GetCanonical,GetMirrors,Delete,List}`
- [ ] **Task 4c:** `PostgresGeneratedContentStore.{Create,Get,GetByFingerprint,List,Delete}`
- [ ] **Task 4d:** `PostgresPostStore.{Create,MarkPublished,Get,ListByTier}`
- [ ] **Task 4e:** `PostgresContentTemplateStore.{Create,Get,Update,Delete,List}`
- [ ] **Task 4f:** `PostgresQualityReviewStore.{Create,ListByContent,LatestByContent}`
- [ ] **Task 4g:** `PostgresAuditEntryStore.{Write,List}` (aligned with `audit.Entry` from Phase 2)
- [ ] **Task 4h:** `PostgresLockStore.{Acquire,Release,IsLocked}` using `pg_advisory_try_lock` + `expires_at` row

Each store gets:
- Unit tests using `testcontainers` in `internal/database/postgres_<store>_test.go`
- A commit `feat(db): implement Postgres<Store>Store`

---

## Task 5: Parity interface test suite

**Files:**
- Create: `tests/integration/database/parity_test.go`
- Create: `tests/integration/database/helpers.go`

- [ ] **Step 1: Failing test** — a single `TestParity` table iterating over both backends using a shared `[]func(t *testing.T, factory StoreFactory)` scenario list.

```go
// tests/integration/database/parity_test.go
package database

import "testing"

func TestParity(t *testing.T) {
	backends := map[string]func(*testing.T) StoreFactory{
		"sqlite":   newSQLiteFactory,
		"postgres": newPostgresFactory,
	}
	scenarios := []struct {
		name string
		run  func(*testing.T, StoreFactory)
	}{
		{"repo-crud", testRepoCRUD},
		{"sync-state", testSyncState},
		{"mirror-map", testMirrorMap},
		{"generated-content-fingerprint", testGeneratedContentFingerprint},
		{"post-publish", testPostPublish},
		{"quality-reviews", testQualityReviews},
		{"audit-write-list", testAuditWriteList},
		{"lock-advisory", testLockAdvisory},
	}
	for backendName, factory := range backends {
		for _, sc := range scenarios {
			t.Run(backendName+"/"+sc.name, func(t *testing.T) { sc.run(t, factory(t)) })
		}
	}
}
```

- [ ] **Step 2: Implement `helpers.go`** — `newPostgresFactory` uses `testcontainers`, `newSQLiteFactory` uses a temp file.

- [ ] **Step 3: Run both**

```bash
go test -race ./tests/integration/database/...
```

- [ ] **Step 4: Commit**

```bash
git commit -m "test(db): shared parity suite running 8 scenarios against sqlite + postgres"
```

---

## Task 6: Advisory-lock contention test

**Files:**
- Create: `tests/integration/database/lock_contention_test.go`

- [ ] **Step 1: Failing test** — spawn 16 goroutines all calling `Acquire("k")`; exactly one succeeds; the rest fail with `ErrLocked`; the holder releases; a second round grants a new holder.

- [ ] **Step 2: Commit**

```bash
git commit -m "test(db): 16-way lock contention scenario"
```

---

## Task 7: Startup migration gate

**Files:**
- Modify: `cmd/cli/main.go`, `cmd/server/main.go`

- [ ] **Step 1: Failing test** — `TestStartupRunsMigrations`.
- [ ] **Step 2: Wire `database.Migrate(ctx, dsn, "file://internal/database/migrations")` into both entrypoints behind config `DATABASE_AUTO_MIGRATE=true` (default true).
- [ ] **Step 3: Commit**

```bash
git commit -m "feat(db): run migrations on startup behind auto-migrate flag"
```

---

## Task 8: Phase 3 acceptance

- [ ] Every method in `internal/database/postgres.go` is non-stub.
- [ ] `tests/integration/database/parity_test.go` green on both backends.
- [ ] Lock contention test green.
- [ ] `bash scripts/coverage.sh` reaches 100% for `internal/database/...`.
- [ ] `go test -race ./internal/database/...` green.
- [ ] New migrations files round-trip (up + down) under SQLite and PostgreSQL.
- [ ] CLI startup auto-runs migrations.

When every box is checked, Phase 3 ships.
