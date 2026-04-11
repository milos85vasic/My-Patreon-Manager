package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/milos85vasic/My-Patreon-Manager/internal/config"
	"github.com/milos85vasic/My-Patreon-Manager/internal/handlers"
	"github.com/milos85vasic/My-Patreon-Manager/internal/metrics"
	"github.com/milos85vasic/My-Patreon-Manager/internal/middleware"
	syncsvc "github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
)

var (
	osExit                                              = os.Exit
	godotenvLoad                                        = godotenv.Load
	loadFromEnv                                         = (*config.Config).LoadFromEnv
	newConfig                                           = config.NewConfig
	newMetricsCollector func() metrics.MetricsCollector = func() metrics.MetricsCollector { return metrics.NewPrometheusCollector() }
	setupRouterFn                                       = setupRouter
	runServerFn                                         = runServer
	signalNotifyContext                                 = signal.NotifyContext
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := newConfig()
	godotenvLoad()
	loadFromEnv(cfg)

	addr := fmt.Sprintf(":%d", cfg.Port)
	ctx, stop := signalNotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runServerFn(ctx, cfg, addr, logger); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		osExit(1)
	}
}

func runServer(ctx context.Context, cfg *config.Config, addr string, logger *slog.Logger) error {
	metricsCollector := newMetricsCollector()
	r := setupRouterFn(cfg, metricsCollector)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Info("server starting", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server listen failed", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Info("server shutting down")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	return nil
}

func setupRouter(cfg *config.Config, metricsCollector metrics.MetricsCollector) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()

	r.Use(middleware.Logger())
	r.Use(gin.Recovery())

	// Create deduplicator for webhooks
	dedup := syncsvc.NewEventDeduplicator(5 * time.Minute)
	webhookHandler := handlers.NewWebhookHandler(dedup, metricsCollector, slog.Default())

	r.GET("/health", handlers.HealthCheck)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/webhook/github", webhookHandler.GitHubWebhook)
	r.POST("/webhook/gitlab", webhookHandler.GitLabWebhook)
	r.POST("/webhook/:service", webhookHandler.GenericWebhook)

	return r
}
