#!/usr/bin/env bash
set -euo pipefail

# Tail logs for AgentAuth web demo
LOG_FILE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/agentauth-web.log"

if [[ ! -f "$LOG_FILE" ]]; then
  echo "[tail-logs.sh] Log file not found: $LOG_FILE" >&2
  exit 1
fi

echo "[tail-logs.sh] Tailing $LOG_FILE (Ctrl+C to stop)..."
tail -f "$LOG_FILE"
