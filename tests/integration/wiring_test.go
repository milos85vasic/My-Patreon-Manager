package integration

import (
	"context"
	"log/slog"
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/git"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/audit"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
	"github.com/milos85vasic/My-Patreon-Manager/tests/mocks"
)

// TestOrchestratorEmitsAuditEntryPerRepo asserts that RunSync emits at least
// one audit entry per repository it processes. The orchestrator is wired with
// a fake provider returning a fixed list and a RingStore as the audit sink;
// the test then drains the store and verifies an entry per repo plus the
// sync.start envelope.
func TestOrchestratorEmitsAuditEntryPerRepo(t *testing.T) {
	ring := audit.NewRingStore(128)

	repos := []models.Repository{
		{ID: "r1", Service: "github", Owner: "o", Name: "one", HTTPSURL: "https://x/one"},
		{ID: "r2", Service: "github", Owner: "o", Name: "two", HTTPSURL: "https://x/two"},
	}
	provider := &mocks.MockRepositoryProvider{
		NameFunc: func() string { return "github" },
		ListRepositoriesFunc: func(_ context.Context, _ string, _ git.ListOptions) ([]models.Repository, error) {
			return repos, nil
		},
		GetMetadataFunc: func(_ context.Context, repo models.Repository) (models.Repository, error) {
			return repo, nil
		},
	}

	db := &mocks.MockDatabase{}
	orc := sync.NewOrchestrator(db, []git.RepositoryProvider{provider}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)
	orc.SetAuditStore(ring)

	if _, err := orc.Run(context.Background(), sync.SyncOptions{}); err != nil {
		t.Fatalf("RunSync: %v", err)
	}

	entries, err := ring.List(context.Background(), 100)
	if err != nil {
		t.Fatalf("List audits: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected audit entries, got none")
	}

	var sawStart int
	repoTargets := map[string]bool{}
	for _, e := range entries {
		if e.Actor == "" || e.Action == "" {
			t.Fatalf("invalid entry: %+v", e)
		}
		if e.CreatedAt.IsZero() {
			t.Fatalf("entry missing CreatedAt: %+v", e)
		}
		if e.Action == "sync.start" {
			sawStart++
		}
		if e.Action == "sync.repo" {
			repoTargets[e.Target] = true
		}
	}
	if sawStart != 1 {
		t.Fatalf("expected exactly one sync.start entry, got %d", sawStart)
	}
	if len(repoTargets) != len(repos) {
		t.Fatalf("expected one sync.repo entry per repo, got %d (entries=%+v)", len(repoTargets), entries)
	}
}

// TestDryRunEmitsAuditEntries asserts the dry-run path produces dryrun-flavored
// audit entries.
func TestDryRunEmitsAuditEntries(t *testing.T) {
	ring := audit.NewRingStore(128)
	provider := &mocks.MockRepositoryProvider{
		NameFunc: func() string { return "github" },
		ListRepositoriesFunc: func(_ context.Context, _ string, _ git.ListOptions) ([]models.Repository, error) {
			return []models.Repository{{ID: "r1", Service: "github", Owner: "o", Name: "one", HTTPSURL: "https://x/one"}}, nil
		},
		GetMetadataFunc: func(_ context.Context, repo models.Repository) (models.Repository, error) {
			return repo, nil
		},
	}
	db := &mocks.MockDatabase{}
	orc := sync.NewOrchestrator(db, []git.RepositoryProvider{provider}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)
	orc.SetAuditStore(ring)

	if _, err := orc.Run(context.Background(), sync.SyncOptions{DryRun: true}); err != nil {
		t.Fatalf("dry run: %v", err)
	}

	entries, _ := ring.List(context.Background(), 100)
	var sawDryRunStart, sawDryRunRepo bool
	for _, e := range entries {
		if e.Action == "sync.dryrun.start" {
			sawDryRunStart = true
		}
		if e.Action == "sync.dryrun.repo" {
			sawDryRunRepo = true
		}
	}
	if !sawDryRunStart || !sawDryRunRepo {
		t.Fatalf("expected dry-run audit entries, got: %+v", entries)
	}
}
