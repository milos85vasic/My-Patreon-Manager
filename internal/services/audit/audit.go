// Package audit defines the audit-log Entry type and Store interface.
// Concrete stores live in ring_store.go and sqlite_store.go. Phase 3 will
// add a Postgres implementation; Phase 2 wires these into the orchestrator,
// webhook, admin, access, and content paths.
package audit

import (
	"context"
	"errors"
	"time"
)

// Entry is a single audit record. Validate() enforces required fields.
type Entry struct {
	ID        string            `json:"id"`
	Actor     string            `json:"actor"`
	Action    string            `json:"action"`
	Target    string            `json:"target,omitempty"`
	Outcome   string            `json:"outcome,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// Validate returns an error if required fields are missing.
func (e Entry) Validate() error {
	if e.Actor == "" {
		return errors.New("audit: actor required")
	}
	if e.Action == "" {
		return errors.New("audit: action required")
	}
	return nil
}

// Store persists audit entries. Implementations must be safe for concurrent
// use; Close() tears down any background goroutines or pooled connections
// the store owns.
type Store interface {
	Write(ctx context.Context, e Entry) error
	List(ctx context.Context, limit int) ([]Entry, error)
	Close() error
}
