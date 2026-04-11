package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	syncsvc "github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
)

// runPublish executes the "publish" subcommand: it reads previously generated
// content from the database and publishes it to Patreon with tier gating. It
// performs no repository discovery and makes no LLM calls — the pipeline
// here is strictly DB -> Patreon.
func runPublish(_ context.Context, orch orchestrator, opts syncsvc.SyncOptions, logger *slog.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("received shutdown signal")
		cancel()
	}()

	result, err := orch.PublishOnly(ctx, opts)
	if err != nil {
		logger.Error("publish failed", slog.String("error", err.Error()))
		osExit(1)
		return
	}
	logger.Info("publish result",
		slog.Int("processed", result.Processed),
		slog.Int("failed", result.Failed),
		slog.Int("skipped", result.Skipped),
	)
}
