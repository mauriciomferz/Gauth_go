#!/usr/bin/env bash
set -euo pipefail

# bench_guardrail.sh
# Compares a new benchmark run against a stored baseline and exits non-zero if regression exceeds threshold.
# Baseline file is expected to be a prior raw.txt or summary.txt containing lines like:
# BenchmarkValidateDelegation-10    47500    1234 B/op    45 allocs/op
# Usage:
#   scripts/bench_guardrail.sh <baseline-file> <current-file> [threshold_percent]
#   GUARD_TARGETS="Regex" (optional) to restrict evaluated benchmarks (default matches all)
# Default threshold_percent = 20 (meaning any ns/op increase >20%).
# Produces a machine-readable JSON summary (guardrail.json) for CI artifact capture.

BASELINE=${1:-}
CURRENT=${2:-}
THRESH=${3:-20}
TARGET_REGEX=${GUARD_TARGETS:-".*"}

if [ -z "$BASELINE" ] || [ -z "$CURRENT" ]; then
  echo "usage: $0 <baseline-file> <current-file> [threshold_percent]" >&2
  exit 2
fi
if [ ! -f "$BASELINE" ]; then
  echo "baseline file not found: $BASELINE" >&2
  exit 2
fi
if [ ! -f "$CURRENT" ]; then
  echo "current file not found: $CURRENT" >&2
  exit 2
fi

# Extract lines containing Benchmark.*ns/op
BASE_TMP=$(mktemp)
CUR_TMP=$(mktemp)
grep -E 'Benchmark.*ns/op' "$BASELINE" > "$BASE_TMP" || true
grep -E 'Benchmark.*ns/op' "$CURRENT" > "$CUR_TMP" || true

# Map benchmarks to ns/op (average if multiple lines) using awk
bench_map() {
  awk '{
    name=$1; ns=$3; # format: name   count   ns/op ...
    # Some outputs (benchstat) differ; ensure we capture numeric field containing ns/op
    for(i=1;i<=NF;i++){ if($i ~ /ns\/op$/){ ns_prev=$(i-1); if(ns_prev ~ /^[0-9]+$/){ ns=ns_prev } }}
    count[name]++;
    sum[name]+=ns;
  } END { for(n in sum){ printf "%s %f\n", n, sum[n]/count[n]; }}' "$1"
}

BASE_DATA=$(bench_map "$BASE_TMP")
CUR_DATA=$(bench_map "$CUR_TMP")

# Join by benchmark name
REGRESSIONS=0
JSON='['
while read -r bline; do
  bname=$(echo "$bline" | awk '{print $1}')
  if ! echo "$bname" | grep -Eq "$TARGET_REGEX"; then
    continue
  fi
  bns=$(echo "$bline" | awk '{print $2}')
  cns=$(echo "$CUR_DATA" | awk -v bn="$bname" '$1==bn {print $2}') || true
  if [ -z "$cns" ]; then
    continue
  fi
  # Compute percent change (positive means slower)
  delta=$(awk -v base="$bns" -v cur="$cns" 'BEGIN { if(base==0){print 0}else{printf "%0.2f", ((cur-base)/base)*100}}')
  status="OK"
  if awk -v d="$delta" -v th="$THRESH" 'BEGIN { exit !(d > th) }'; then
    status="REGRESSION"
    REGRESSIONS=$((REGRESSIONS+1))
  fi
  JSON+="{\"benchmark\":\"$bname\",\"base_ns_per_op\":$bns,\"current_ns_per_op\":$cns,\"percent_change\":$delta,\"threshold\":$THRESH,\"status\":\"$status\"},"
  printf "%s base=%s current=%s change=%s%% status=%s\n" "$bname" "$bns" "$cns" "$delta" "$status"

done <<< "$BASE_DATA"
JSON=${JSON%,}']'

echo "$JSON" > guardrail.json

if [ $REGRESSIONS -gt 0 ]; then
  echo "Benchmark regression detected (>$THRESH% slowdown)" >&2
  exit 1
fi

echo "No benchmark regressions above $THRESH% threshold." >&2
exit 0
