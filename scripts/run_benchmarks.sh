#!/usr/bin/env bash
set -euo pipefail

# run_benchmarks.sh
# Purpose: Collect stable benchmark numbers for critical hot paths and summarize deltas.
# Usage: ./scripts/run_benchmarks.sh [OUT_DIR]
# OUT_DIR defaults to build/bench. Creates timestamped subdirectory with raw & summarized outputs.
# Requires: go, optionally benchstat (if present in PATH) for comparative stats.

OUT_ROOT=${1:-build/bench}
STAMP=$(date +%Y%m%d_%H%M%S)
OUT_DIR="$OUT_ROOT/$STAMP"
mkdir -p "$OUT_DIR"

echo "[run_benchmarks] Writing results to $OUT_DIR"

# List of benchmark packages (extend as needed)
PKGS=(
  "./pkg/agentauth"
)

# Number of repetitions for stability
COUNT=${COUNT:-5}

RAW_FILE="$OUT_DIR/raw.txt"
SUMMARY_FILE="$OUT_DIR/summary.txt"

echo "[run_benchmarks] Running benchmarks (count=$COUNT)" | tee "$SUMMARY_FILE"

for pkg in "${PKGS[@]}"; do
  echo "[run_benchmarks] Package: $pkg" | tee -a "$SUMMARY_FILE"
  go test -bench . -benchmem -run ^$ -count="$COUNT" "$pkg" | tee -a "$RAW_FILE" | grep -E 'Benchmark.+ns/op' >> "$SUMMARY_FILE" || true
  echo >> "$SUMMARY_FILE"
  echo >> "$RAW_FILE"
 done

# Optional benchstat comparison (needs at least 2 runs). We keep the last run aside if user exports BASE_FILE
if command -v benchstat >/dev/null 2>&1 && [ -n "${BASE_FILE:-}" ] && [ -f "$BASE_FILE" ]; then
  echo "[run_benchmarks] benchstat comparison vs $BASE_FILE" | tee -a "$SUMMARY_FILE"
  benchstat "$BASE_FILE" "$RAW_FILE" | tee -a "$SUMMARY_FILE"
fi

echo "[run_benchmarks] Complete. Key results:" | tee -a "$SUMMARY_FILE"
awk '/Benchmark.*ns\/op/ {print}' "$SUMMARY_FILE" | tee -a "$SUMMARY_FILE"

echo "[run_benchmarks] Done."