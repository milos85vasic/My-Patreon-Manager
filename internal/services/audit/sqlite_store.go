package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// marshalMetadata is swapped in tests to exercise the marshal-error branch.
var marshalMetadata = json.Marshal

// SQLiteStore persists audit entries to an `audit_entries` table. The table
// is created by the Phase 3 migration `0002_audit.up.sql`; this store
// assumes it already exists.
type SQLiteStore struct{ db *sql.DB }

// NewSQLiteStore wraps the given *sql.DB.
func NewSQLiteStore(db *sql.DB) *SQLiteStore { return &SQLiteStore{db: db} }

// Write inserts a validated entry into audit_entries.
func (s *SQLiteStore) Write(ctx context.Context, e Entry) error {
	if err := e.Validate(); err != nil {
		return err
	}
	meta, err := marshalMetadata(e.Metadata)
	if err != nil {
		return fmt.Errorf("audit: marshal metadata: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO audit_entries(id, actor, action, target, outcome, metadata, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Actor, e.Action, e.Target, e.Outcome, string(meta), e.CreatedAt)
	if err != nil {
		return fmt.Errorf("audit: insert: %w", err)
	}
	return nil
}

// List returns up to `limit` most-recent entries ordered by created_at
// descending. A limit <= 0 defaults to 100.
func (s *SQLiteStore) List(ctx context.Context, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, actor, action, target, outcome, metadata, created_at
		 FROM audit_entries ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("audit: query: %w", err)
	}
	defer rows.Close()

	var out []Entry
	for rows.Next() {
		var e Entry
		var meta string
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Target, &e.Outcome, &meta, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("audit: scan: %w", err)
		}
		if meta != "" {
			_ = json.Unmarshal([]byte(meta), &e.Metadata)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit: rows: %w", err)
	}
	return out, nil
}

// Close is a no-op; the store does not own the *sql.DB lifecycle.
func (s *SQLiteStore) Close() error { return nil }
