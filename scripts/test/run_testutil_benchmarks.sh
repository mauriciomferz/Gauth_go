#!/usr/bin/env bash
set -euo pipefail
OUT_DIR=build/bench/testutil-$(date +%s)
mkdir -p "$OUT_DIR"
# Run only testutil benchmarks
go test -run=^$ -bench . -benchmem -count=1 ./web/testutil > "$OUT_DIR/raw.txt"
cp "$OUT_DIR/raw.txt" "$OUT_DIR/summary.txt"
