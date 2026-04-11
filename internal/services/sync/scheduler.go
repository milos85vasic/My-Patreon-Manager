package sync

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// SyncRunner defines the interface for a component that can run a sync.
type SyncRunner interface {
	Run(ctx context.Context, opts SyncOptions) (*SyncResult, error)
}

type Scheduler struct {
	cron   *cron.Cron
	runner SyncRunner
	opts   SyncOptions
	alert  Alert
	logger *slog.Logger
	parent context.Context
}

func NewScheduler(runner SyncRunner, opts SyncOptions, alert Alert, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		cron:   cron.New(),
		runner: runner,
		opts:   opts,
		alert:  alert,
		logger: logger,
	}
}

// Start registers the schedule and begins the cron loop. The provided parent
// context governs all scheduled job executions: cancelling it will cancel any
// in-flight job. Passing a nil context is treated as context.Background() for
// backwards compatibility with callers that have no scoped context yet.
func (s *Scheduler) Start(ctx context.Context, schedule string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.parent = ctx
	_, err := s.cron.AddFunc(schedule, func() {
		jobCtx, cancel := context.WithTimeout(s.parent, 1*time.Hour)
		defer cancel()

		if s.logger != nil {
			s.logger.Info("scheduled sync started")
		}

		result, err := s.runner.Run(jobCtx, s.opts)
		if err != nil {
			if s.alert != nil {
				s.alert.Send("Sync Failed", err.Error())
			}
			if s.logger != nil {
				s.logger.Error("scheduled sync failed", slog.String("error", err.Error()))
			}
			return
		}

		if s.logger != nil {
			s.logger.Info("scheduled sync completed",
				slog.Int("processed", result.Processed),
				slog.Int("failed", result.Failed),
			)
		}
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}
