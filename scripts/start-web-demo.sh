#!/usr/bin/env bash
set -euo pipefail

# AgentAuth Beta Web Interface Startup Script
# Provides optional .env loading, persistent secrets, and PID management.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_FILE="$ROOT_DIR/agentauth-web.pid"
LOG_FILE="$ROOT_DIR/agentauth-web.log"
PORT="${AGENTAUTH_PORT:-8080}"

# Preflight (empty Go file detection) unless skipped
if [[ "${AGENTAUTH_SKIP_PREFLIGHT:-0}" != "1" ]]; then
  if [[ -x "$ROOT_DIR/scripts/preflight-dev.sh" ]]; then
    if ! "$ROOT_DIR/scripts/preflight-dev.sh"; then
      echo "[start-web-demo] Preflight failed; aborting startup (set AGENTAUTH_SKIP_PREFLIGHT=1 to bypass)" >&2
      exit 1
    fi
  fi
fi

if [[ -f "$PID_FILE" ]]; then
  if kill -0 $(cat "$PID_FILE") 2>/dev/null; then
    echo "[start-web-demo] An instance is already running (PID $(cat "$PID_FILE")). Stop it first (scripts/stop-web-demo.sh)." >&2
    exit 1
  else
    echo "[start-web-demo] Removing stale PID file" >&2
    rm -f "$PID_FILE"
  fi
fi

# Load .env if present
if [[ -f "$ROOT_DIR/.env" ]]; then
  echo "[start-web-demo] Loading .env"
  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
  set +a
fi

# Ensure stable secrets if not provided
if [[ -z "${AGENTAUTH_CLIENT_SECRET:-}" ]]; then
  export AGENTAUTH_CLIENT_SECRET="demo-client-secret-change-me"
  echo "[start-web-demo] AGENTAUTH_CLIENT_SECRET not set; using demo default (beta)." >&2
fi
if [[ -z "${AGENTAUTH_SIGNING_KEY:-}" ]]; then
  export AGENTAUTH_SIGNING_KEY="demo-signing-key-change-me-32-bytes-minimum-1234"
  echo "[start-web-demo] AGENTAUTH_SIGNING_KEY not set; using demo default (beta)." >&2
fi

export AGENTAUTH_MODE="${AGENTAUTH_MODE:-development}"
export AGENTAUTH_PORT="$PORT"
export AGENTAUTH_DEV_MODULES="${AGENTAUTH_DEV_MODULES:-1}" # serve modules from disk for live reload
export AGENTAUTH_DEV_INDEX="${AGENTAUTH_DEV_INDEX:-1}" # serve index.html from disk for live reload

cd "$ROOT_DIR" || exit 1

echo "[start-web-demo] Running cmd/web-server/main.go as main server (BetaServer only)..." | tee -a "$LOG_FILE"
nohup go run ./cmd/web-server/main.go > "$LOG_FILE" 2>&1 &
PID=$!
echo $PID > "$PID_FILE"
echo "[start-web-demo] PID $PID" | tee -a "$LOG_FILE"
echo "[start-web-demo] Logs: $LOG_FILE"
echo "[start-web-demo] Visit: http://localhost:$PORT"

# Readiness wait: first wait for /ready JSON, then confirm policy tab in index.html
READY=0
for i in {1..30}; do
  if curl -fsS "http://localhost:$PORT/ready" >/dev/null; then
    if curl -fsS "http://localhost:$PORT/index.html" | grep -q 'data-testid="policy-tab"'; then
      echo "[start-web-demo] Ready (policy tab + /ready) after $i attempts" | tee -a "$LOG_FILE"
      READY=1
      break
    fi
  fi
  sleep 0.2
done
if [[ $READY -ne 1 ]]; then
  echo "[start-web-demo] WARNING: readiness criteria not fully met after attempts; continuing" | tee -a "$LOG_FILE"
fi