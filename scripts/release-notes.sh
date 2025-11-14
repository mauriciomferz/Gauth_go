#!/usr/bin/env bash
set -euo pipefail

# Generate release notes from git log using conventional commit style.
# Usage: scripts/release-notes.sh [SINCE_TAG] > docs/RELEASE_NOTES.auto.md

SINCE_TAG=${1:-}
DATE=$(date +%Y-%m-%d)

if [ -n "$SINCE_TAG" ]; then
  RANGE="$SINCE_TAG..HEAD"
else
  # Fallback: last 50 commits
  RANGE="HEAD~50..HEAD"
fi

echo "# Release Notes (Generated)" 
echo "> Generated: $DATE"
echo ""

GROUP_KEYS="feat fix docs chore refactor perf test ci build security"

group_title() {
  case "$1" in
    feat) echo "Features" ;;
    fix) echo "Fixes" ;;
    docs) echo "Documentation" ;;
    chore) echo "Chore" ;;
    refactor) echo "Refactoring" ;;
    perf) echo "Performance" ;;
    test) echo "Tests" ;;
    ci) echo "CI/CD" ;;
    build) echo "Build" ;;
    security) echo "Security" ;;
    *) echo "$1" ;;
  esac
}

for key in $GROUP_KEYS; do
  commits=$(git log --pretty=format:"%s" $RANGE | grep -E "^${key}(:|\()" || true)
  if [ "${commits:-}" != "" ]; then
    echo "## $(group_title "$key")"
    echo "$commits" | sed 's/^/- /'
    echo ""
  fi
done

echo "## Other"
other=$(git log --pretty=format:"%s" $RANGE | grep -Ev '^(feat|fix|docs|chore|refactor|perf|test|ci|build|security)(:|\()' || true)
if [ "${other:-}" != "" ]; then echo "$other" | sed 's/^/- /'; else echo "(none)"; fi

printf "\n---\n"
echo "_Source range: $RANGE"
