#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

RED="\033[31m"; YELLOW="\033[33m"; GREEN="\033[32m"; NC="\033[0m"

warn() { echo -e "${YELLOW}[preflight] $*${NC}" >&2; }
err()  { echo -e "${RED}[preflight] ERROR: $*${NC}" >&2; }
info() { echo -e "${GREEN}[preflight] $*${NC}" >&2; }

# 1. Detect empty .go files (size 0 or only whitespace) which break 'go build'
EMPTY_FILES=()
while IFS= read -r -d '' f; do
  if [[ ! -s "$f" ]]; then
    EMPTY_FILES+=("$f")
  else
    # Strip whitespace and check again
    if ! grep -q '[^[:space:]]' "$f"; then
      EMPTY_FILES+=("$f")
    fi
  fi
done < <(find . -type f -name '*.go' -print0)

if (( ${#EMPTY_FILES[@]} > 0 )); then
  err "Detected empty Go source file(s):"
  for f in "${EMPTY_FILES[@]}"; do
    echo "  - $f" >&2
  done
  cat >&2 <<'EOF'
These empty files cause 'expected \"package\"' parse errors, blocking startup.
Resolve by either:
  * Removing unintended placeholder files, OR
  * Adding at least a 'package <name>' line and (optionally) a placeholder test.

To skip this check (NOT recommended) set: AGENTAUTH_SKIP_PREFLIGHT=1
EOF
  exit 1
fi

info "No empty Go files detected."

# Future: add port availability & module tidy checks.
exit 0
