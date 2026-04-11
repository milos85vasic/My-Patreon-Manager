package sync

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// cancelAwareRunner is a minimal SyncRunner that records invocations and
// blocks on ctx.Done(), so a test can verify parent-context cancellation
// reaches in-flight scheduled jobs.
type cancelAwareRunner struct {
	calls   int32
	started chan struct{}
}

func (r *cancelAwareRunner) Run(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	atomic.AddInt32(&r.calls, 1)
	select {
	case r.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestSchedulerRespectsParentCancel verifies that cancelling the parent
// context passed to Start unblocks an in-flight scheduled job. Before the
// fix, the per-job context was derived from context.Background(), so the
// job would run for up to an hour regardless of the caller's scope.
func TestSchedulerRespectsParentCancel(t *testing.T) {
	defer func() {
		// Earlier orchestrator tests in this package may have started the
		// package-level repoignore SIGHUP watcher via loadRepoignore. Stop
		// it deterministically before the leak check so we don't need any
		// IgnoreTopFunction escape hatches.
		StopRepoignoreWatch()
		goleak.VerifyNone(t)
	}()

	runner := &cancelAwareRunner{started: make(chan struct{}, 1)}
	s := NewScheduler(runner, SyncOptions{}, nil, nil)

	parent, cancel := context.WithCancel(context.Background())
	if err := s.Start(parent, "@every 50ms"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for the first job to start.
	select {
	case <-runner.started:
	case <-time.After(2 * time.Second):
		cancel()
		s.Stop()
		t.Fatal("scheduled job never started")
	}

	// Cancel the parent. The in-flight job's context must become Done.
	cancel()

	// Stop must return promptly: the cron library waits for running jobs,
	// and the running job is only unblocked via parent cancellation.
	stopDone := make(chan struct{})
	go func() {
		s.Stop()
		close(stopDone)
	}()

	select {
	case <-stopDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return after parent cancel (leak: job blocked on Background context)")
	}

	if atomic.LoadInt32(&runner.calls) == 0 {
		t.Fatal("job never fired")
	}
}

// TestSchedulerNilParentContext verifies the nil-context defensive branch:
// passing a nil parent is treated as context.Background() so legacy call
// sites continue to work.
func TestSchedulerNilParentContext(t *testing.T) {
	defer func() {
		// Earlier orchestrator tests in this package may have started the
		// package-level repoignore SIGHUP watcher via loadRepoignore. Stop
		// it deterministically before the leak check so we don't need any
		// IgnoreTopFunction escape hatches.
		StopRepoignoreWatch()
		goleak.VerifyNone(t)
	}()

	fired := make(chan struct{}, 1)
	runner := &fireOnceRunner{fired: fired}
	s := NewScheduler(runner, SyncOptions{}, nil, nil)

	//nolint:staticcheck // intentionally testing the nil-context fallback
	if err := s.Start(nil, "@every 50ms"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not fire under nil parent context")
	}
}

type fireOnceRunner struct {
	once  atomic.Bool
	fired chan struct{}
}

func (r *fireOnceRunner) Run(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	if r.once.CompareAndSwap(false, true) {
		r.fired <- struct{}{}
	}
	return &SyncResult{}, nil
}

// TestSchedulerAlertOnFailure keeps the in-package test file self-sufficient
// by exercising the failure path against the real struct (not via the
// external test package), which also ensures the error branch is counted
// against the package's own coverage profile.
func TestSchedulerAlertOnFailure(t *testing.T) {
	defer func() {
		// Earlier orchestrator tests in this package may have started the
		// package-level repoignore SIGHUP watcher via loadRepoignore. Stop
		// it deterministically before the leak check so we don't need any
		// IgnoreTopFunction escape hatches.
		StopRepoignoreWatch()
		goleak.VerifyNone(t)
	}()

	alertCalled := make(chan struct{}, 1)
	alert := &recordingAlert{ch: alertCalled}
	runner := &errRunner{err: errors.New("boom")}
	s := NewScheduler(runner, SyncOptions{}, alert, nil)

	if err := s.Start(context.Background(), "@every 50ms"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	select {
	case <-alertCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("alert not invoked on runner failure")
	}
}

type errRunner struct{ err error }

func (r *errRunner) Run(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	return nil, r.err
}

type recordingAlert struct {
	ch chan struct{}
}

func (a *recordingAlert) Send(subject, body string) {
	select {
	case a.ch <- struct{}{}:
	default:
	}
}
