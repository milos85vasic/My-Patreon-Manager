package sync

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/git"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/audit"
	"github.com/milos85vasic/My-Patreon-Manager/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrchestrator_EnqueueRepo_OK exercises the happy path: EnqueueRepo
// accepts a repo, emits an "ok" audit entry, and the item becomes drainable
// via DrainRepoWork.
func TestOrchestrator_EnqueueRepo_OK(t *testing.T) {
	ring := audit.NewRingStore(32)
	orc := NewOrchestrator(&mocks.MockDatabase{}, []git.RepositoryProvider{}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)
	orc.SetAuditStore(ring)

	repo := models.Repository{ID: "owner/repo", Service: "github"}
	require.NoError(t, orc.EnqueueRepo(context.Background(), repo))

	// Audit entry should record the enqueue.
	entries, err := ring.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "sync.enqueue", entries[0].Action)
	assert.Equal(t, "ok", entries[0].Outcome)
	assert.Equal(t, "owner/repo", entries[0].Target)

	// Drain should now yield the enqueued repo.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	var drained []models.Repository
	go func() {
		done <- orc.DrainRepoWork(ctx, func(_ context.Context, r models.Repository) error {
			drained = append(drained, r)
			cancel()
			return nil
		})
	}()

	select {
	case err := <-done:
		assert.True(t, errors.Is(err, context.Canceled), "drain should exit via ctx cancel, got %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("DrainRepoWork did not return")
	}
	require.Len(t, drained, 1)
	assert.Equal(t, "owner/repo", drained[0].ID)
}

// TestOrchestrator_EnqueueRepo_Full fills the internal queue and asserts
// EnqueueRepo returns ErrWorkQueueFull with a matching audit entry.
func TestOrchestrator_EnqueueRepo_Full(t *testing.T) {
	ring := audit.NewRingStore(DefaultEnqueueBufferCapacity + 8)
	orc := NewOrchestrator(&mocks.MockDatabase{}, []git.RepositoryProvider{}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)
	orc.SetAuditStore(ring)

	// Pre-fill the queue to capacity via repeated EnqueueRepo calls.
	for i := 0; i < DefaultEnqueueBufferCapacity; i++ {
		require.NoError(t, orc.EnqueueRepo(context.Background(), models.Repository{ID: "r", Service: "github"}))
	}

	// One more should fail with ErrWorkQueueFull.
	err := orc.EnqueueRepo(context.Background(), models.Repository{ID: "overflow", Service: "github"})
	assert.ErrorIs(t, err, ErrWorkQueueFull)

	// The last audit entry should be the "full" outcome.
	entries, _ := ring.List(context.Background(), DefaultEnqueueBufferCapacity+8)
	require.NotEmpty(t, entries)
	last := entries[len(entries)-1]
	assert.Equal(t, "sync.enqueue", last.Action)
	assert.Equal(t, "full", last.Outcome)
	assert.Equal(t, "overflow", last.Target)
}

// TestOrchestrator_DrainRepoWork_CtxCancel asserts DrainRepoWork exits
// cleanly when the context is cancelled while the queue is idle.
func TestOrchestrator_DrainRepoWork_CtxCancel(t *testing.T) {
	orc := NewOrchestrator(&mocks.MockDatabase{}, []git.RepositoryProvider{}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- orc.DrainRepoWork(ctx, func(context.Context, models.Repository) error { return nil })
	}()

	cancel()
	select {
	case err := <-done:
		assert.True(t, errors.Is(err, context.Canceled))
	case <-time.After(2 * time.Second):
		t.Fatal("DrainRepoWork did not exit after cancel")
	}
}

// TestOrchestrator_DrainRepoWork_FnError asserts DrainRepoWork returns the
// first non-nil error from fn.
func TestOrchestrator_DrainRepoWork_FnError(t *testing.T) {
	orc := NewOrchestrator(&mocks.MockDatabase{}, []git.RepositoryProvider{}, &mocks.PatreonClient{}, nil, nil, slog.Default(), nil)

	require.NoError(t, orc.EnqueueRepo(context.Background(), models.Repository{ID: "r", Service: "github"}))

	sentinel := errors.New("fn boom")
	err := orc.DrainRepoWork(context.Background(), func(context.Context, models.Repository) error {
		return sentinel
	})
	assert.ErrorIs(t, err, sentinel)
}
