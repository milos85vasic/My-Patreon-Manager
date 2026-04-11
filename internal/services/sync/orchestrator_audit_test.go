package sync

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/git"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/audit"
	"github.com/milos85vasic/My-Patreon-Manager/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestrator_Run_EmitsStartAndPerRepoAudit(t *testing.T) {
	gitMock := &mocks.MockRepositoryProvider{
		NameFunc: func() string { return "github" },
		ListRepositoriesFunc: func(_ context.Context, _ string, _ git.ListOptions) ([]models.Repository, error) {
			return []models.Repository{{ID: "r1", Service: "github", Owner: "o", Name: "n", HTTPSURL: "https://x"}}, nil
		},
		GetMetadataFunc: func(_ context.Context, repo models.Repository) (models.Repository, error) {
			return repo, nil
		},
	}

	db := &mocks.MockDatabase{}
	orc := NewOrchestrator(db, []git.RepositoryProvider{gitMock}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)

	_, err := orc.Run(context.Background(), SyncOptions{})
	require.NoError(t, err)

	entries, err := orc.AuditStore().List(context.Background(), 100)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 2)

	// First entry: sync.start
	assert.Equal(t, "sync.start", entries[0].Action)
	assert.Equal(t, "orchestrator", entries[0].Actor)

	// At least one sync.repo entry for the processed repo
	var sawRepo bool
	for _, e := range entries {
		if e.Action == "sync.repo" && e.Target == "o/n" {
			sawRepo = true
			break
		}
	}
	assert.True(t, sawRepo, "expected sync.repo audit entry")
}

func TestOrchestrator_Run_DryRunEmitsDryRunActions(t *testing.T) {
	gitMock := &mocks.MockRepositoryProvider{
		NameFunc: func() string { return "github" },
		ListRepositoriesFunc: func(_ context.Context, _ string, _ git.ListOptions) ([]models.Repository, error) {
			return []models.Repository{{ID: "r1", Service: "github", Owner: "o", Name: "n", HTTPSURL: "https://x"}}, nil
		},
		GetMetadataFunc: func(_ context.Context, repo models.Repository) (models.Repository, error) {
			return repo, nil
		},
	}
	db := &mocks.MockDatabase{}
	orc := NewOrchestrator(db, []git.RepositoryProvider{gitMock}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)

	_, err := orc.Run(context.Background(), SyncOptions{DryRun: true})
	require.NoError(t, err)

	entries, _ := orc.AuditStore().List(context.Background(), 100)
	require.NotEmpty(t, entries)

	var sawDryRunStart, sawDryRunRepo bool
	for _, e := range entries {
		if e.Action == "sync.dryrun.start" {
			sawDryRunStart = true
		}
		if e.Action == "sync.dryrun.repo" {
			sawDryRunRepo = true
		}
	}
	assert.True(t, sawDryRunStart, "expected sync.dryrun.start audit entry")
	assert.True(t, sawDryRunRepo, "expected sync.dryrun.repo audit entry")
}

func TestOrchestrator_Run_RepoErrorEmitsErrorAudit(t *testing.T) {
	gitMock := &mocks.MockRepositoryProvider{
		NameFunc: func() string { return "github" },
		ListRepositoriesFunc: func(_ context.Context, _ string, _ git.ListOptions) ([]models.Repository, error) {
			return []models.Repository{{ID: "r1", Service: "github", Owner: "o", Name: "n", HTTPSURL: "https://x"}}, nil
		},
		GetMetadataFunc: func(_ context.Context, repo models.Repository) (models.Repository, error) {
			return repo, errors.New("metadata blew up")
		},
	}
	db := &mocks.MockDatabase{}
	orc := NewOrchestrator(db, []git.RepositoryProvider{gitMock}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)

	_, err := orc.Run(context.Background(), SyncOptions{})
	require.NoError(t, err)

	entries, _ := orc.AuditStore().List(context.Background(), 100)
	var sawError bool
	for _, e := range entries {
		if e.Action == "sync.repo" && e.Outcome == "error" {
			sawError = true
			assert.Contains(t, e.Metadata["error"], "metadata blew up")
			break
		}
	}
	assert.True(t, sawError, "expected sync.repo audit entry with error outcome")
}

func TestOrchestrator_SetAuditStore(t *testing.T) {
	orc := NewOrchestrator(&mocks.MockDatabase{}, nil, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)
	custom := audit.NewRingStore(8)
	orc.SetAuditStore(custom)
	assert.Same(t, custom, orc.AuditStore())

	orc.SetAuditStore(nil)
	assert.NotNil(t, orc.AuditStore())
	assert.NotSame(t, custom, orc.AuditStore())
}

func TestOrchestrator_ShortErr(t *testing.T) {
	assert.Equal(t, "", shortErr(nil))
	assert.Equal(t, "boom", shortErr(errors.New("boom")))
	// Long string is truncated to 96 chars.
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	assert.Equal(t, 96, len(shortErr(errors.New(string(long)))))
	// Token is redacted.
	out := shortErr(errors.New("Bearer token=xyz failed"))
	assert.NotContains(t, out, "token")
	assert.Contains(t, out, "***")
}
