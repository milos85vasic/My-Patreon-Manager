package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	syncsvc "github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
)

// runScan executes the "scan" subcommand: it discovers every repository
// visible to the configured providers, applies the .repoignore filter, and
// logs the result. It never generates LLM content and never publishes to
// Patreon — that is the whole point of having a dedicated discovery mode.
func runScan(_ context.Context, orch orchestrator, opts syncsvc.SyncOptions, logger *slog.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("received shutdown signal")
		cancel()
	}()

	repos, err := orch.ScanOnly(ctx, opts)
	if err != nil {
		logger.Error("scan failed", slog.String("error", err.Error()))
		osExit(1)
		return
	}
	logger.Info("scan discovered repositories", slog.Int("count", len(repos)))
	for _, r := range repos {
		logger.Info("discovered",
			slog.String("service", r.Service),
			slog.String("owner", r.Owner),
			slog.String("repo", r.Name),
			slog.String("url", r.URL),
		)
	}
}
