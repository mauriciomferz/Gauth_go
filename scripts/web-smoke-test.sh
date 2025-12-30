#!/usr/bin/env bash
set -euo pipefail

echo "[web-smoke-test] Starting AgentAuth beta web smoke test (legacy educational fallback enabled)" >&2

PORT=${1:-}
if [[ -z "${PORT}" ]]; then
  PORT=$(( ( RANDOM % 1000 ) + 9000 ))
fi

BIN_CMD=(go run cmd/web-server/main.go "${PORT}")

LOG_FILE="$(mktemp -t gauth-web-smoke-XXXX.log)"
echo "[web-smoke-test] Using port: ${PORT}" >&2

"${BIN_CMD[@]}" >"${LOG_FILE}" 2>&1 &
PID=$!
trap 'echo "[web-smoke-test] Stopping server" >&2; kill ${PID} 2>/dev/null || true' EXIT

ATTEMPTS=0
until grep -q "Server starting on:" "${LOG_FILE}"; do
  sleep 0.5
  ATTEMPTS=$((ATTEMPTS+1))
  if (( ATTEMPTS > 20 )); then
    echo "[web-smoke-test] ERROR: Server did not start within timeout" >&2
    cat "${LOG_FILE}" >&2 || true
    exit 1
  fi
done

BASE="http://localhost:${PORT}"
echo "[web-smoke-test] Server started: ${BASE}" >&2

fail() { echo "[web-smoke-test] FAIL: $*" >&2; exit 1; }
check() { echo "[web-smoke-test] ✓ $*" >&2; }

curl_silent() { curl -fsS "$@"; }

HTML=$(curl_silent "${BASE}/") || fail "Root page fetch failed"
[[ "${HTML}" == *"AgentAuth Beta"* || "${HTML}" == *"AgentAuth Educational Demo"* ]] || fail "Root page missing expected beta or legacy text"
check "Root page OK"

HEALTH=$(curl_silent "${BASE}/api/v1/beta/health" || curl_silent "${BASE}/api/v1/educational/health") || fail "Health endpoint failed"
echo "${HEALTH}" | grep -q '"success":true' || fail "Health JSON missing success:true"
check "Health endpoint OK"

INFO=$(curl_silent "${BASE}/api/v1/beta/info" || curl_silent "${BASE}/api/v1/educational/info") || fail "Info endpoint failed"
echo "${INFO}" | grep -q '"disclaimer"' || fail "Info JSON missing disclaimer"
check "Info endpoint OK"

CSP=$(curl -I -s "${BASE}/" | grep -i '^Content-Security-Policy:') || fail "Missing CSP header"
echo "${CSP}" | grep -q "default-src 'self'" || fail "CSP missing default-src"
check "CSP header present"

echo "[web-smoke-test] All checks passed." >&2
exit 0