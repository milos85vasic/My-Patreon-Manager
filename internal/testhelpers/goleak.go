// Package testhelpers exposes test-only helpers usable from package _test.go
// files. It has no production callers.
package testhelpers

import "go.uber.org/goleak"

// GoleakIgnores returns the project-wide goleak allowlist. Keep in sync with
// tests/leaks/ignores.go.
func GoleakIgnores() []goleak.Option {
	return []goleak.Option{
		goleak.IgnoreTopFunction("database/sql.(*DB).connectionOpener"),
		goleak.IgnoreTopFunction("database/sql.(*DB).connectionResetter"),
		goleak.IgnoreTopFunction("github.com/testcontainers/testcontainers-go.(*Reaper).connect"),
	}
}
