#!/usr/bin/env bash
set -euo pipefail

# docs_index.sh
# Purpose: Validate documentation metadata headers and produce summary counts.
# Usage:
#   bash scripts/docs_index.sh --validate        # exit non-zero if any file missing required header fields
#   bash scripts/docs_index.sh --summary         # print category counts
#   bash scripts/docs_index.sh --list-missing    # list files missing metadata header
#   bash scripts/docs_index.sh --json            # machine-readable JSON summary
#   bash scripts/docs_index.sh --help            # help text
#
# Required header fields (non-generated): title, category, status, lastUpdated, owners
# Generated adds: generated:true, source, refreshCadence
#
# Notes: Accepts either full YAML-ish block or commented (#) metadata lines.

ROOT_DIR="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
DOC_PATTERNS=("*.md")
REQUIRED_BASE=(title category status lastUpdated owners)
REQUIRED_GENERATED=(generated source refreshCadence)
TMP_JSON=$(mktemp)
trap 'rm -f "$TMP_JSON"' EXIT

print_help() {
  grep -E '^# ' "$0" | sed 's/^# //'
}

collect_files() {
  # Exclude vendor-like or build dirs
  find "$ROOT_DIR" -type f -name '*.md' \
    -not -path '*/node_modules/*' \
    -not -path '*/bin/*' \
    -not -path '*/build/*' \
    -not -path '*/.git/*' |
    sort
}

has_header() {
  local f="$1"
  # Look for starting metadata block delimiter or metadata comment lines.
  awk 'NR<=30' "$f" | grep -qE '^(---|# +title:)'
}

extract_field() {
  local f="$1" field="$2"
  awk 'NR<=50' "$f" | grep -E "^[# ]*${field}:" | head -1 | sed -E "s/^[# ]*${field}:[ ]*//"
}

is_generated() {
  local f="$1"; awk 'NR<=50' "$f" | grep -qi 'generated: *true'
}

validate_file() {
  local f="$1"; local missing=()
  has_header "$f" || { echo "MISSING_HEADER|$f"; return; }
  for k in "${REQUIRED_BASE[@]}"; do
    local v; v=$(extract_field "$f" "$k")
    [[ -z "$v" ]] && missing+=("$k")
  done
  if is_generated "$f"; then
    for k in "${REQUIRED_GENERATED[@]}"; do
      local v; v=$(extract_field "$f" "$k")
      [[ -z "$v" ]] && missing+=("$k")
    done
  fi
  if ((${#missing[@]})); then
    echo "INCOMPLETE|$f|${missing[*]}"
  fi
}

summary_counts() {
  local files; files=$(collect_files)
  declare -A counts
  while IFS= read -r f; do
    local c; c=$(extract_field "$f" category || true)
    [[ -z "$c" ]] && c="(uncategorized)"
    counts[$c]=$((counts[$c]+1))
  done <<<"$files"
  printf "Category Counts:\n"
  for k in "${!counts[@]}"; do
    printf "  %s: %d\n" "$k" "${counts[$k]}"
  done | sort
}

json_output() {
  local files; files=$(collect_files)
  echo '{"files":[' > "$TMP_JSON"
  local first=1
  while IFS= read -r f; do
    local title category status owners generated
    title=$(extract_field "$f" title)
    category=$(extract_field "$f" category)
    status=$(extract_field "$f" status)
    owners=$(extract_field "$f" owners)
    generated=$(is_generated "$f" && echo true || echo false)
    [[ $first -eq 0 ]] && echo ',' >> "$TMP_JSON" || first=0
    jq -n --arg path "$f" --arg title "$title" --arg category "$category" --arg status "$status" --arg owners "$owners" --argjson generated "$generated" '{path:$path,title:$title,category:$category,status:$status,owners:$owners,generated:$generated}' >> "$TMP_JSON"
  done <<<"$files"
  echo ']}' >> "$TMP_JSON"
  cat "$TMP_JSON"
}

list_missing() {
  collect_files | while IFS= read -r f; do
    has_header "$f" || echo "$f"
  done
}

main() {
  case "${1:-}" in
    --help|-h) print_help ;;
    --validate)
      local issues=0
      while IFS= read -r f; do
        out=$(validate_file "$f" || true)
        if [[ -n "$out" ]]; then
          echo "$out"; issues=1
        fi
      done < <(collect_files)
      if [[ $issues -ne 0 ]]; then
        echo "❌ Documentation validation failed" >&2
        exit 1
      else
        echo "✅ Documentation validation passed"
      fi
      ;;
    --summary) summary_counts ;;
    --json) json_output ;;
    --list-missing) list_missing ;;
    *) print_help ;;
  esac
}

main "$@"
