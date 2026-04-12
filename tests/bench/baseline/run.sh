#!/usr/bin/env bash
# Capture Go benchmark baseline for all packages.
# Usage: bash tests/bench/baseline/run.sh [output_file]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUT="${1:-${ROOT}/tests/bench/baseline/results.txt}"

echo "Running benchmarks ($(date -u +%Y-%m-%dT%H:%M:%SZ)) ..."
cd "$ROOT"
go test -bench=. -benchmem -run='^$' -count=1 ./internal/... ./tests/benchmark/... 2>&1 | tee "$OUT"
echo ""
echo "Results written to $OUT"
