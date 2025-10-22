#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <old_bench.txt> <new_bench.txt> [threshold_percent]" >&2
  exit 1
fi
OLD=$1
NEW=$2
THRESHOLD=${3:-10}

parse() {
  awk '/^Benchmark/ {name=$1; ops=$2; ns=$3; bytes=$4; allocs=$5; gsub(/ns\/op/,"",ns); gsub(/B\/op/,"",bytes); gsub(/allocs\/op/,"",allocs); print name,ns,bytes,allocs}' "$1"
}

# shellcheck disable=SC2034
declare -A OLD_NS OLD_BYTES OLD_ALLOCS
while read -r name ns bytes allocs; do
  OLD_NS[$name]=$ns; OLD_BYTES[$name]=$bytes; OLD_ALLOCS[$name]=$allocs;
done < <(parse "$OLD")

echo "Benchmark Regression Report (threshold ${THRESHOLD}%)"
printf "%s\t%s\t%s\t%s\t%s\n" "NAME" "Δns/op%" "ΔB/op%" "Δallocs/op%" "STATUS"

while read -r name ns bytes allocs; do
  if [ -n "${OLD_NS[$name]:-}" ]; then
    old_ns=${OLD_NS[$name]}; old_bytes=${OLD_BYTES[$name]}; old_allocs=${OLD_ALLOCS[$name]};
    pct() { awk -v o=$1 -v n=$2 'BEGIN { if (o==0) {print 0; exit}; print ((n-o)/o)*100 }'; }
    d_ns=$(pct "$old_ns" "$ns"); d_bytes=$(pct "$old_bytes" "$bytes"); d_allocs=$(pct "$old_allocs" "$allocs")
    status="OK"
    for d in $d_ns $d_bytes $d_allocs; do
      comp=$(awk -v v=$d -v t=$THRESHOLD 'BEGIN { if (v > t) print 1; else print 0 }')
      if [ "$comp" -eq 1 ]; then status="REGRESSION"; break; fi
    done
    printf "%s\t%.2f\t%.2f\t%.2f\t%s\n" "$name" "$d_ns" "$d_bytes" "$d_allocs" "$status"
  fi
done < <(parse "$NEW")

exit 0
