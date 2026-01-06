#!/usr/bin/env zsh
set -euo pipefail

ROOT_DIR="${0:A:h:h}" # script resides in scripts/, go two levels up if nested
cd "$ROOT_DIR" || { echo "Failed to cd to repo root" >&2; exit 1; }

PORT="${AGENTAUTH_WEB_PORT:-8080}"
export AGENTAUTH_WEB_PORT="$PORT"

echo "[run-web] Starting web-server on :$PORT (cwd=$PWD)"
echo "[run-web] Press Ctrl+C to stop. PID will be printed below."

exec go run ./cmd/web-server