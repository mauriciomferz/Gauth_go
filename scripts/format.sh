#!/usr/bin/env bash
set -euo pipefail

# format.sh - Opinionated formatting helper for GAuth
# Usage:
#   ./scripts/format.sh            # apply formatting & imports
#   DRY=1 ./scripts/format.sh       # show what would change
#   VERBOSE=1 ./scripts/format.sh   # extra logging
#
# Provides:
# - gofmt (canonical formatting)
# - goimports (import grouping + removal of unused imports)
# - Optional dry-run to preview changes
# - Summary of changed files
# - Installs goimports if missing
# - Respects go.work (format all referenced modules)

DRY=${DRY:-0}
VERBOSE=${VERBOSE:-0}
ROOT=$(pwd)

log() { echo "[format] $*"; }

if [[ $VERBOSE -eq 1 ]]; then
  set -x
fi

if ! command -v go >/dev/null 2>&1; then
  log "ERROR: go toolchain not found in PATH"; exit 1
fi

if ! command -v goimports >/dev/null 2>&1; then
  log "Installing goimports (one-time)";
  go install golang.org/x/tools/cmd/goimports@latest
fi

log "Starting formatting (DRY=$DRY VERBOSE=$VERBOSE)"

# Collect file list (exclude vendor, node_modules, generated artifacts)
readarray -t GO_FILES < <(find . -type f -name '*.go' \
  -not -path '*/vendor/*' \
  -not -path '*/node_modules/*' \
  -not -path '*/coveragebadges/*' \
  -not -path '*/embedded/*')

if [[ $DRY -eq 1 ]]; then
  CHANGED=()
  for f in "${GO_FILES[@]}"; do
    BEFORE=$(cat "$f")
    TMP=$(mktemp)
    goimports "$f" > "$TMP" || cp "$f" "$TMP"
    gofmt "$TMP" > "$TMP.fmt"
    if ! diff -q "$f" "$TMP.fmt" >/dev/null 2>&1; then
      CHANGED+=("$f")
    fi
    rm -f "$TMP" "$TMP.fmt"
  done
  if ((${#CHANGED[@]})); then
    log "Files needing formatting ("${#CHANGED[@]}"):";
    for c in "${CHANGED[@]}"; do echo "  • $c"; done
    log "Run without DRY=1 to apply.";
  else
    log "No formatting changes required.";
  fi
  exit 0
fi

log "Applying goimports..."
# Use -w to write changes; run twice to stabilize import grouping
goimports -w "${GO_FILES[@]}"

goimports -w "${GO_FILES[@]}" 2>/dev/null || true

log "Applying gofmt..."
gofmt -w "${GO_FILES[@]}"

# Show changed files in this run (git required)
if command -v git >/dev/null 2>&1; then
  MODIFIED=$(git ls-files -m '*.go' || true)
  if [[ -n "$MODIFIED" ]]; then
    log "Modified files:";
    echo "$MODIFIED" | sed 's/^/  • /'
  else
    log "No files modified by formatting step."
  fi
else
  log "git not found; skipping change summary"
fi

# Optional module tidy (fast)
if [[ ${SKIP_TIDY:-0} -eq 0 ]]; then
  log "Running go mod tidy (root)"
  go mod tidy
  if [[ -f go.work ]]; then
    awk '/use / {print $2}' go.work | while read -r m; do
      log "→ tidy $m"
      (cd "$m" && go mod tidy)
    done
  fi
fi

log "Format complete ✅"