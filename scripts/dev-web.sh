#!/usr/bin/env bash
set -euo pipefail
PORT="${AGENTAUTH_WEB_PORT:-8080}"
LOG="${AGENTAUTH_DEV_LOG:-/tmp/agentauth_web.log}"
echo "[dev-web] starting on :$PORT (log: $LOG)"
AGENTAUTH_WEB_PORT="$PORT" go run ./cmd/web-server > "$LOG" 2>&1 &
PID=$!
echo $PID > /tmp/agentauth_web.pid
echo "[dev-web] PID $PID"
echo "tail -f $LOG"