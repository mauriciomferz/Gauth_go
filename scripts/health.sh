#!/usr/bin/env bash
set -euo pipefail

"#"" Health check for AgentAuth web demo (beta primary, educational fallback)""#
PORT="${AGENTAUTH_PORT:-8080}"
PRIMARY="http://localhost:$PORT/api/v1/beta/health"
FALLBACK="http://localhost:$PORT/api/v1/educational/health"

if command -v curl >/dev/null 2>&1; then
  echo "[health.sh] Checking $PRIMARY ..."
  if curl -fsS "$PRIMARY" | grep -q '"status":"ok"'; then
    echo "[health.sh] Web demo is healthy."
    exit 0
  else
    echo "[health.sh] Primary failed, attempting fallback..." >&2
    if curl -fsS "$FALLBACK" | grep -q '"status":"ok"'; then
      echo "[health.sh] Web demo is healthy (fallback)." >&2
      exit 0
    fi
    echo "[health.sh] Neither primary nor fallback returned ok." >&2
    exit 2
  fi
else
  echo "[health.sh] curl not found. Please install curl." >&2
  exit 3
fi
