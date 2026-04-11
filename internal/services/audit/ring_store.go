package audit

import (
	"context"
	"sync"
)

// RingStore is a bounded in-memory audit store implemented as a ring buffer.
// Used as a fallback when no persistent store is configured, and as the
// backing store for goleak-safe unit tests.
type RingStore struct {
	mu    sync.Mutex
	buf   []Entry
	size  int
	head  int
	count int
}

// NewRingStore returns a RingStore with the given capacity. Capacities < 1
// are clamped to 1.
func NewRingStore(size int) *RingStore {
	if size < 1 {
		size = 1
	}
	return &RingStore{buf: make([]Entry, size), size: size}
}

// Write appends an entry to the ring, overwriting the oldest entry once
// full. The entry is validated before being stored.
func (r *RingStore) Write(_ context.Context, e Entry) error {
	if err := e.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.head] = e
	r.head = (r.head + 1) % r.size
	if r.count < r.size {
		r.count++
	}
	return nil
}

// List returns up to `limit` most-recent entries in insertion order (oldest
// first). A limit <= 0 returns all stored entries.
func (r *RingStore) List(_ context.Context, limit int) ([]Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.count
	if limit > 0 && limit < n {
		n = limit
	}
	// Oldest entry sits at (head - count) mod size. When limiting, start at
	// (head - n) mod size so we return the n most-recent entries.
	start := (r.head - n + r.size) % r.size
	out := make([]Entry, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, r.buf[(start+i)%r.size])
	}
	return out, nil
}

// Close is a no-op; the ring store holds no external resources.
func (r *RingStore) Close() error { return nil }
