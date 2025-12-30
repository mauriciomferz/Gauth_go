#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_FILE="$ROOT_DIR/agentauth-web.pid"

if [[ ! -f "$PID_FILE" ]]; then
  echo "[stop-web-demo] No PID file found ($PID_FILE). Is the server running?" >&2
  exit 1
fi
PID=$(cat "$PID_FILE")
if kill -0 "$PID" 2>/dev/null; then
  echo "[stop-web-demo] Stopping PID $PID"
  kill "$PID"
  # Wait up to 5s
  for _ in {1..10}; do
    if kill -0 "$PID" 2>/dev/null; then
      sleep 0.5
    else
      break
    fi
  done
  if kill -0 "$PID" 2>/dev/null; then
    echo "[stop-web-demo] Force killing PID $PID" >&2
    kill -9 "$PID" || true
  fi
  echo "[stop-web-demo] Stopped"
else
  echo "[stop-web-demo] Process $PID not running (stale PID file)" >&2
fi
rm -f "$PID_FILE"