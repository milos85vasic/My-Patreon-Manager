// Package sync provides the Orchestrator, the top-level coordinator that
// wires together Git providers, content generators, the database, and metrics
// to execute the full sync pipeline. It is consumed by both the CLI and
// HTTP server entrypoints.
package sync
