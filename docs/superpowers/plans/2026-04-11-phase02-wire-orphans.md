# Phase 2 — Wire Orphaned Features Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every orphan handler, middleware, service, renderer, and config key becomes a first-class feature reached from `cmd/cli` or `cmd/server`, with tests and docs.

**Architecture:** No new interfaces — the orphans already implement them. Work is (1) construct, (2) register/mount, (3) test end-to-end, (4) document. Respects Phase 1's `Lifecycle` and `Semaphore` primitives.

**Tech Stack:** Go 1.26.1, Gin, existing provider interfaces, `testify`, `httptest`.

**Depends on:** Phase 0, Phase 1.

---

## File Structure

**Modify:**
- `cmd/server/main.go` — wire AccessHandler, AdminHandler, middleware stack, Queue drainer, rate limiter sweeper, graceful shutdown.
- `cmd/cli/main.go` — split `scan`, `generate`, `publish` into distinct commands; construct PDF + video renderers behind flags.
- `internal/services/audit/` — previously empty; add `store.go`, `sqlite_store.go`, `postgres_store.go`, `ring_store.go`, `audit.go`, `audit_test.go`.
- `internal/handlers/access.go` — accept `SignedURLGenerator` by value via constructor.
- `internal/handlers/admin.go` — accept `AdminService` by value via constructor.
- `internal/middleware/webhook_auth.go` — accept HMAC secret by constructor; verify each provider.
- `internal/services/content/generator.go` — call audit store on every mutation.
- `internal/services/sync/orchestrator.go` — call audit store; add PublishPost, distinct from RunSync.
- `internal/config/config.go` — read + expose `PDFRenderingEnabled`, `VideoGenerationEnabled`, `AdminKey`, `WebhookHMACSecret`, `RateLimitRPS`, `RateLimitBurst`.
- `.env.example` — new keys with safe defaults.

**Create:**
- `internal/services/audit/audit.go`
- `internal/services/audit/store.go`
- `internal/services/audit/sqlite_store.go`
- `internal/services/audit/ring_store.go`
- `internal/services/audit/audit_test.go`
- `cmd/cli/scan.go`
- `cmd/cli/generate.go`
- `cmd/cli/publish.go`
- `cmd/cli/scan_test.go`
- `cmd/cli/generate_test.go`
- `cmd/cli/publish_test.go`
- `tests/integration/wiring_test.go`

---

## Task 1: Implement `internal/services/audit/` package

**Files:**
- Create: `internal/services/audit/audit.go`
- Create: `internal/services/audit/store.go`
- Create: `internal/services/audit/sqlite_store.go`
- Create: `internal/services/audit/ring_store.go`
- Create: `internal/services/audit/audit_test.go`

- [ ] **Step 1: Failing tests**

```go
// internal/services/audit/audit_test.go
package audit

import (
	"context"
	"testing"
	"time"
)

func TestRingStoreKeepsLastN(t *testing.T) {
	r := NewRingStore(3)
	for i := 0; i < 5; i++ {
		_ = r.Write(context.Background(), Entry{Actor: "cli", Action: "sync", CreatedAt: time.Now()})
	}
	entries, err := r.List(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("len = %d, want 3", len(entries))
	}
}

func TestEntryRequiresActorAndAction(t *testing.T) {
	r := NewRingStore(1)
	if err := r.Write(context.Background(), Entry{}); err == nil {
		t.Fatal("expected validation error")
	}
}
```

- [ ] **Step 2: Run** — compile error (types missing).

- [ ] **Step 3: Implement**

```go
// internal/services/audit/audit.go
package audit

import (
	"context"
	"errors"
	"time"
)

type Entry struct {
	ID        string            `json:"id"`
	Actor     string            `json:"actor"`
	Action    string            `json:"action"`
	Target    string            `json:"target,omitempty"`
	Outcome   string            `json:"outcome,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

func (e Entry) Validate() error {
	if e.Actor == "" {
		return errors.New("audit: actor required")
	}
	if e.Action == "" {
		return errors.New("audit: action required")
	}
	return nil
}

type Store interface {
	Write(ctx context.Context, e Entry) error
	List(ctx context.Context, limit int) ([]Entry, error)
	Close() error
}
```

```go
// internal/services/audit/ring_store.go
package audit

import (
	"context"
	"sync"
)

type RingStore struct {
	mu    sync.Mutex
	buf   []Entry
	size  int
	head  int
	count int
}

func NewRingStore(size int) *RingStore {
	if size < 1 {
		size = 1
	}
	return &RingStore{buf: make([]Entry, size), size: size}
}

func (r *RingStore) Write(ctx context.Context, e Entry) error {
	if err := e.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.head] = e
	r.head = (r.head + 1) % r.size
	if r.count < r.size {
		r.count++
	}
	return nil
}

func (r *RingStore) List(ctx context.Context, limit int) ([]Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.count
	if limit > 0 && limit < n {
		n = limit
	}
	out := make([]Entry, 0, n)
	start := (r.head - r.count + r.size) % r.size
	for i := 0; i < n; i++ {
		out = append(out, r.buf[(start+i)%r.size])
	}
	return out, nil
}

func (r *RingStore) Close() error { return nil }
```

```go
// internal/services/audit/sqlite_store.go
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
)

type SQLiteStore struct{ db *sql.DB }

func NewSQLiteStore(db *sql.DB) *SQLiteStore { return &SQLiteStore{db: db} }

func (s *SQLiteStore) Write(ctx context.Context, e Entry) error {
	if err := e.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(e.Metadata)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_entries(id,actor,action,target,outcome,metadata,created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		e.ID, e.Actor, e.Action, e.Target, e.Outcome, string(meta), e.CreatedAt)
	return err
}

func (s *SQLiteStore) List(ctx context.Context, limit int) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,actor,action,target,outcome,metadata,created_at
		 FROM audit_entries ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var meta string
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Target, &e.Outcome, &meta, &e.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(meta), &e.Metadata)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) Close() error { return nil }
```

- [ ] **Step 4: Run**

```bash
go test -race ./internal/services/audit/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/audit/
git commit -m "feat(audit): implement Entry, Store, RingStore, SQLiteStore

New package providing audit logging with in-memory ring fallback and
SQLite persistence. Validated Entry schema; RingStore covered by unit
tests with -race."
```

---

## Task 2: Wire audit store into orchestrator, webhook, CLI, admin, access

**Files:**
- Modify: `internal/services/sync/orchestrator.go`
- Modify: `internal/handlers/webhook.go`
- Modify: `internal/handlers/access.go`
- Modify: `internal/handlers/admin.go`
- Modify: `cmd/cli/main.go`, `cmd/server/main.go`
- Modify: `internal/services/content/generator.go`

- [ ] **Step 1: Failing tests**

```go
// tests/integration/wiring_test.go
func TestOrchestratorEmitsAuditEntryPerRepo(t *testing.T) {
	ring := audit.NewRingStore(64)
	orch := buildTestOrchestrator(t, ring)
	_ = orch.RunSync(context.Background())
	entries, _ := ring.List(context.Background(), 100)
	if len(entries) == 0 {
		t.Fatal("expected audit entries")
	}
	for _, e := range entries {
		if e.Actor == "" || e.Action == "" {
			t.Fatal("invalid audit entry emitted")
		}
	}
}
```

- [ ] **Step 2: Run** — fail (orchestrator has no audit field yet).

- [ ] **Step 3: Add `Audit audit.Store` field to `Orchestrator`** and every place it mutates state:

```go
_ = o.Audit.Write(ctx, audit.Entry{
	Actor:     "orchestrator",
	Action:    "sync.repo",
	Target:    r.FullName,
	Outcome:   "ok",
	CreatedAt: time.Now(),
})
```

Add identical emissions to: dry-run entry, webhook enqueue, publish, admin reload, access download, content generation.

- [ ] **Step 4: Run** — PASS.
- [ ] **Step 5: Commit**

```bash
git commit -m "feat(audit): emit entries from orchestrator, webhook, admin, access, content"
```

---

## Task 3: Split CLI into scan / generate / publish

**Files:**
- Create: `cmd/cli/scan.go`
- Create: `cmd/cli/generate.go`
- Create: `cmd/cli/publish.go`
- Modify: `cmd/cli/main.go`

- [ ] **Step 1: Failing tests**

```go
func TestScanOnlyDiscoversRepos(t *testing.T) {
	out := runCLI(t, "scan", "--dry-run")
	if !strings.Contains(out, "discovered") { t.Fatal("scan missing discovery output") }
	if strings.Contains(out, "publishing")  { t.Fatal("scan should not publish") }
}
func TestGenerateProducesContentWithoutPublish(t *testing.T) { ... }
func TestPublishUsesPreGeneratedContent(t *testing.T) { ... }
```

- [ ] **Step 2: Implement three commands** calling orchestrator sub-methods (add `ScanOnly`, `GenerateOnly`, `PublishOnly` to `Orchestrator`).

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(cli): split scan/generate/publish into distinct subcommands"
```

---

## Task 4: Wire middleware stack into cmd/server

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Failing test** — `tests/integration/wiring_test.go`:

```go
func TestServerMountsFullMiddlewareStack(t *testing.T) {
	srv := newTestServer(t)
	// Webhook without signature → 401
	req := httptest.NewRequest("POST", "/webhook/github", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized { t.Fatalf("got %d", w.Code) }
}
func TestAdminRoutesRequireAdminKey(t *testing.T) { ... }
func TestRateLimiterReturns429(t *testing.T)     { ... }
```

- [ ] **Step 2: Rewrite `setupRouter`**:

```go
func setupRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.Logger(deps.Logger))

	r.GET("/health", handlers.NewHealthHandler(deps.DB).Handle)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	wh := r.Group("/webhook")
	wh.Use(middleware.IPRateLimiter(deps.Limiter).Limit())
	wh.Use(middleware.WebhookAuth(deps.WebhookSecret))
	wh.POST("/github", deps.Webhook.GitHub)
	wh.POST("/gitlab", deps.Webhook.GitLab)
	wh.POST("/gitflic", deps.Webhook.GitFlic)
	wh.POST("/gitverse", deps.Webhook.GitVerse)

	admin := r.Group("/admin")
	admin.Use(middleware.Auth(deps.AdminKey))
	deps.Admin.Register(admin)

	dl := r.Group("/download")
	dl.Use(middleware.IPRateLimiter(deps.Limiter).Limit())
	dl.GET("/:content_id", deps.Access.Handle)

	// pprof behind admin auth
	pp := r.Group("/debug/pprof", middleware.Auth(deps.AdminKey))
	pprof.RouteRegister(pp)

	return r
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(server): mount recovery, logger, rate limit, auth, webhook auth

Wires previously orphaned middleware into the Gin router and moves
admin/pprof behind middleware.Auth(ADMIN_KEY)."
```

---

## Task 5: Wire PDF + Video renderers behind config flags

**Files:**
- Modify: `cmd/cli/main.go`
- Modify: `internal/config/config.go`
- Create: `cmd/cli/renderers.go`

- [ ] **Step 1: Failing test**

```go
func TestCLIRendererFlagWiresPDFWhenEnabled(t *testing.T) {
	cfg := Config{PDFRenderingEnabled: true}
	renderers := buildRenderers(cfg)
	if !containsType(renderers, "*renderer.PDFRenderer") { t.Fatal("PDFRenderer missing") }
}
func TestCLIRendererFlagOmitsVideoWhenDisabled(t *testing.T) { ... }
```

- [ ] **Step 2: Implement `buildRenderers(cfg Config) []renderer.FormatRenderer`** that conditionally appends PDF and Video renderers.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(renderers): wire PDF and video renderers behind config flags"
```

---

## Task 6: Webhook signature verification

**Files:**
- Modify: `internal/middleware/webhook_auth.go`
- Create: `internal/middleware/webhook_auth_test.go`

- [ ] **Step 1: Failing tests** — per-provider: GitHub HMAC-SHA256, GitLab `X-Gitlab-Token`, GitFlic HMAC-SHA256, GitVerse HMAC-SHA256.

- [ ] **Step 2: Implement using `crypto/hmac` + `subtle.ConstantTimeCompare`.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(webhook): verify signatures per provider with constant-time compare"
```

---

## Task 7: Phase 2 acceptance

- [ ] `go test -race ./...` green (excluding packages gated for Phase 3/4 still).
- [ ] Every previously orphan package has a reacher in `cmd/cli` or `cmd/server`.
- [ ] `tests/integration/wiring_test.go` green across all routes.
- [ ] Audit entries visible in `/admin/audit` endpoint.
- [ ] CLI `scan`, `generate`, `publish` each produce distinct output.
- [ ] `bash scripts/coverage.sh` green with `COVERAGE_MIN=0`.

When every box is checked, Phase 2 ships.
