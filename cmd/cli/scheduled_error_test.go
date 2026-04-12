package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	syncsvc "github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
	"github.com/stretchr/testify/assert"
)

func TestRunScheduled_StartError(t *testing.T) {
	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mockOrch := &mockOrchestrator{}

	// Use an invalid cron expression to trigger scheduler.Start error
	exited, code := withMockExit(t, func() {
		runScheduled(context.Background(), mockOrch, syncsvc.SyncOptions{}, "invalid-cron", logger)
	})

	assert.True(t, exited, "should exit on scheduler start error")
	assert.Equal(t, 1, code)
	assert.Contains(t, logOutput.String(), "failed to start scheduler")
}
