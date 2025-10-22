#!/usr/bin/env bash
# Documentation disclaimer linter (beta demonstration safeguard)
# Scans markdown for risky production-claim phrases lacking a negating qualifier.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAIL=0
SKIP_TAG="lint-disclaimer:skip"

phrases=(
  "production ready"
  "ready for production"
  "enterprise grade"
  "hardened deployment"
)

find "$ROOT_DIR" -type f -name "*.md" -print0 | while IFS= read -r -d '' file; do
  [[ "$file" == *"DISCLAIMER.md" ]] && continue
  line_no=0
  while IFS= read -r line; do
    line_no=$((line_no+1))
    lower="${line,,}"
    for p in "${phrases[@]}"; do
      if [[ "$lower" == *"$p"* ]]; then
        if [[ "$lower" =~ not\ production\ ready || "$lower" =~ not\ for\ production || "$lower" =~ legacy\ educational ]]; then
          continue
        fi
        if [[ "$line" == *"$SKIP_TAG"* ]]; then
          continue
        fi
        echo "[disclaimer-lint] $file:$line_no: '$p' without qualifying NOT/legacy educational context" >&2
        FAIL=1
      fi
    done
  done < "$file"
done

if [[ $FAIL -ne 0 ]]; then
  echo "Documentation disclaimer lint failed." >&2
  exit 1
fi
echo "Documentation disclaimer lint passed." >&2
