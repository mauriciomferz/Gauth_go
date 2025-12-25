#!/usr/bin/env bash
set -euo pipefail
# Simple documentation generation script (beta demonstration)
# Scans public packages and extracts top-level type and function signatures into docs/GENERATED_API.md

OUT="docs/GENERATED_API.md"
TMP="$(mktemp)"

echo "# Generated API Surface" > "$TMP"
echo "_Auto-generated on $(date -u '+%Y-%m-%dT%H:%M:%SZ') – experimental_" >> "$TMP"
echo >> "$TMP"

# Collect package list (exclude internal testdata, vendor, backup directories)
PKGS=$(go list ./... | grep -v /internal/ | grep -v /test/ | grep -v backup_ | grep -v /examples/ || true)

for p in $PKGS; do
  echo "## $p" >> "$TMP"
  echo '```go' >> "$TMP"
  # Use go doc to dump exported identifiers; suppress errors
  if ! go doc -all "$p" 2>/dev/null | sed 's/^\t\+//' >> "$TMP"; then
    echo "(no exported symbols)" >> "$TMP"
  fi
  echo '```' >> "$TMP"
  echo >> "$TMP"
 done

mv "$TMP" "$OUT"
echo "Generated $OUT"
