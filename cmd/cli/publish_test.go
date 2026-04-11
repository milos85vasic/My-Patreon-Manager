package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	syncsvc "github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
	"github.com/stretchr/testify/assert"
)

// TestRunPublish_Success verifies that runPublish invokes PublishOnly and
// does NOT call Run/ScanOnly/GenerateOnly.
func TestRunPublish_Success(t *testing.T) {
	var (
		scanCalled     bool
		runCalled      bool
		generateCalled bool
		publishCalled  bool
	)

	mockOrch := &mockOrchestrator{
		runFunc: func(ctx context.Context, opts syncsvc.SyncOptions) (*syncsvc.SyncResult, error) {
			runCalled = true
			return &syncsvc.SyncResult{}, nil
		},
		scanFunc: func(ctx context.Context, opts syncsvc.SyncOptions) ([]models.Repository, error) {
			scanCalled = true
			return nil, nil
		},
		generateFunc: func(ctx context.Context, opts syncsvc.SyncOptions) (*syncsvc.SyncResult, error) {
			generateCalled = true
			return &syncsvc.SyncResult{}, nil
		},
		publishFunc: func(ctx context.Context, opts syncsvc.SyncOptions) (*syncsvc.SyncResult, error) {
			publishCalled = true
			return &syncsvc.SyncResult{Processed: 2}, nil
		},
	}

	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))
	exited, _ := withMockExit(t, func() {
		runPublish(context.Background(), mockOrch, syncsvc.SyncOptions{}, logger)
	})
	assert.False(t, exited, "runPublish should not exit on success")
	assert.True(t, publishCalled, "PublishOnly should have been called")
	assert.False(t, runCalled, "Run should NOT have been called by publish")
	assert.False(t, scanCalled, "ScanOnly should NOT have been called by publish")
	assert.False(t, generateCalled, "GenerateOnly should NOT have been called by publish")
	assert.Contains(t, logOutput.String(), "publish result")
}

// TestRunPublish_Error verifies runPublish exits 1 on error.
func TestRunPublish_Error(t *testing.T) {
	mockOrch := &mockOrchestrator{
		publishFunc: func(ctx context.Context, opts syncsvc.SyncOptions) (*syncsvc.SyncResult, error) {
			return nil, fmt.Errorf("patreon 503")
		},
	}
	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))
	exited, code := withMockExit(t, func() {
		runPublish(context.Background(), mockOrch, syncsvc.SyncOptions{}, logger)
	})
	assert.True(t, exited)
	assert.Equal(t, 1, code)
	assert.Contains(t, logOutput.String(), "publish failed")
}
