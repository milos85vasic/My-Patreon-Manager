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

// TestRunGenerate_Success verifies that runGenerate invokes GenerateOnly and
// does NOT call Run/ScanOnly/PublishOnly.
func TestRunGenerate_Success(t *testing.T) {
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
			return &syncsvc.SyncResult{Processed: 3, Failed: 0, Skipped: 1}, nil
		},
		publishFunc: func(ctx context.Context, opts syncsvc.SyncOptions) (*syncsvc.SyncResult, error) {
			publishCalled = true
			return &syncsvc.SyncResult{}, nil
		},
	}

	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))
	exited, _ := withMockExit(t, func() {
		runGenerate(context.Background(), mockOrch, syncsvc.SyncOptions{}, logger)
	})
	assert.False(t, exited, "runGenerate should not exit on success")
	assert.True(t, generateCalled, "GenerateOnly should have been called")
	assert.False(t, runCalled, "Run should NOT have been called by generate")
	assert.False(t, scanCalled, "ScanOnly should NOT have been called by generate")
	assert.False(t, publishCalled, "PublishOnly should NOT have been called by generate")
	assert.Contains(t, logOutput.String(), "generate result")
}

// TestRunGenerate_Error verifies runGenerate exits 1 on error.
func TestRunGenerate_Error(t *testing.T) {
	mockOrch := &mockOrchestrator{
		generateFunc: func(ctx context.Context, opts syncsvc.SyncOptions) (*syncsvc.SyncResult, error) {
			return nil, fmt.Errorf("llm outage")
		},
	}
	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))
	exited, code := withMockExit(t, func() {
		runGenerate(context.Background(), mockOrch, syncsvc.SyncOptions{}, logger)
	})
	assert.True(t, exited)
	assert.Equal(t, 1, code)
	assert.Contains(t, logOutput.String(), "generate failed")
}
