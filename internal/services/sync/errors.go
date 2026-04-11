package sync

import "errors"

var ErrShutdownTimeout = errors.New("sync: shutdown timed out")

// ErrWorkQueueFull is returned by Orchestrator.EnqueueRepo when the internal
// work queue has no capacity for a new repo. Callers surface this as HTTP 429
// (or equivalent) so upstream webhook senders can back off and retry.
var ErrWorkQueueFull = errors.New("sync: work queue full")
