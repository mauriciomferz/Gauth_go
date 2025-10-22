#!/usr/bin/env bash
set -euo pipefail
PORT="${GAUTH_WEB_PORT:-8080}"
LOG="${GAUTH_DEV_LOG:-/tmp/gauth_web.log}"
echo "[dev-web] starting on :$PORT (log: $LOG)"
GAUTH_WEB_PORT="$PORT" go run ./cmd/web-server > "$LOG" 2>&1 &
PID=$!
echo $PID > /tmp/gauth_web.pid
echo "[dev-web] PID $PID"
echo "tail -f $LOG"