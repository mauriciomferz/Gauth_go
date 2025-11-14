#!/usr/bin/env bash
set -euo pipefail

# Batch prepend or update a 'Last Updated:' line in markdown files.
# Skips vendor, node_modules, build, bin directories.

DATE="$(date +%Y-%m-%d)"
NOTE="Last Updated: ${DATE}"

echo "🔄 Updating markdown timestamps -> ${NOTE}"

FILES=$(grep -RIl --exclude-dir=node_modules --exclude-dir=vendor --exclude-dir=bin --exclude-dir=build --exclude-dir=.git -e '^# ' . | grep -E '\.md$' || true)

for f in $FILES; do
  if grep -q 'Last Updated:' "$f"; then
    # Replace existing line
    sed -i '' "s/^> Last Updated:.*/> ${NOTE}/" "$f" 2>/dev/null || sed -i "s/^> Last Updated:.*/> ${NOTE}/" "$f"
  else
    # Prepend after first H1 if file starts with '#'
    if head -1 "$f" | grep -q '^#'; then
      tmp="${f}.tmp"
      { head -1 "$f"; echo "> ${NOTE}"; tail -n +2 "$f"; } > "$tmp"
      mv "$tmp" "$f"
    else
      echo "> ${NOTE}" >> "$f"
    fi
  fi
  echo "✅ $f"
done

echo "✨ Timestamp update complete"