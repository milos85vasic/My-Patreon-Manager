package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	syncsvc "github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
)

// runGenerate executes the "generate" subcommand: it discovers repositories
// and runs the content-generation pipeline (LLM + verifier + quality gate),
// persisting GeneratedContent records to the database. It does NOT publish
// anything to Patreon — that step is reserved for the "publish" subcommand.
func runGenerate(_ context.Context, orch orchestrator, opts syncsvc.SyncOptions, logger *slog.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("received shutdown signal")
		cancel()
	}()

	result, err := orch.GenerateOnly(ctx, opts)
	if err != nil {
		logger.Error("generate failed", slog.String("error", err.Error()))
		osExit(1)
		return
	}
	logger.Info("generate result",
		slog.Int("processed", result.Processed),
		slog.Int("failed", result.Failed),
		slog.Int("skipped", result.Skipped),
	)
}
