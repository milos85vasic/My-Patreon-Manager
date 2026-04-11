package main

import (
	"strings"
	"testing"
)

func TestParseCoverageHundred(t *testing.T) {
	in := `github.com/milos85vasic/My-Patreon-Manager/internal/config/config.go:12:	LoadEnv		100.0%
github.com/milos85vasic/My-Patreon-Manager/internal/config/config.go:30:	Validate	100.0%
total:						(statements)	100.0%
`
	pkgs, total, err := parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if total != 100.0 {
		t.Fatalf("total = %v, want 100.0", total)
	}
	if pkgs["github.com/milos85vasic/My-Patreon-Manager/internal/config"] != 100.0 {
		t.Fatalf("pkg pct = %v", pkgs["github.com/milos85vasic/My-Patreon-Manager/internal/config"])
	}
}

func TestParseCoverageBelow(t *testing.T) {
	in := `github.com/milos85vasic/My-Patreon-Manager/internal/foo/foo.go:1:	A	50.0%
github.com/milos85vasic/My-Patreon-Manager/internal/foo/foo.go:2:	B	100.0%
total:						(statements)	75.0%
`
	pkgs, _, err := parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if pkgs["github.com/milos85vasic/My-Patreon-Manager/internal/foo"] != 75.0 {
		t.Fatalf("pkg pct = %v, want 75.0", pkgs["github.com/milos85vasic/My-Patreon-Manager/internal/foo"])
	}
}

func TestEnforceFailsOnLowPackage(t *testing.T) {
	pkgs := map[string]float64{
		"internal/a": 100.0,
		"internal/b": 99.9,
	}
	err := enforce(pkgs, 100.0, 100.0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "internal/b") {
		t.Fatalf("error does not mention offending package: %v", err)
	}
}

func TestEnforcePassesAt100(t *testing.T) {
	pkgs := map[string]float64{
		"internal/a": 100.0,
		"internal/b": 100.0,
	}
	if err := enforce(pkgs, 100.0, 100.0); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
