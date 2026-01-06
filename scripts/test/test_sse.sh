#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
EXAMPLE_ID="${EXAMPLE_ID:-agentauth_protocol_basics:minimal_poa}"

echo "[SSE TEST] Starting job for example: $EXAMPLE_ID"
JOB_JSON=$(curl -s -X POST "$BASE_URL/api/v1/beta/examples/run" \
  -H 'Content-Type: application/json' \
  -d '{"id":"'"$EXAMPLE_ID"'"}') || { echo "Failed to start job"; exit 1; }
if [[ -z "$JOB_JSON" || "$JOB_JSON" == *"Not Found"* ]]; then
  echo "[SSE TEST] Primary beta endpoint failed, trying legacy educational path" >&2
  JOB_JSON=$(curl -s -X POST "$BASE_URL/api/v1/educational/examples/run" \
    -H 'Content-Type: application/json' \
    -d '{"id":"'"$EXAMPLE_ID"'"}') || { echo "Failed to start job (fallback)"; exit 1; }
fi
echo "$JOB_JSON" | sed 's/.*/[SSE TEST] Job response: &/'
JOB_ID=$(printf '%s' "$JOB_JSON" | sed -n 's/.*"job_id":"\([^"]*\)".*/\1/p')
if [[ -z "$JOB_ID" ]]; then echo "Could not extract job_id"; exit 1; fi
echo "[SSE TEST] JOB_ID=$JOB_ID"

TMP=$(mktemp -t sse-log-XXXX)
cleanup() { rm -f "$TMP"; }
trap cleanup EXIT

echo "[SSE TEST] Streaming logs (capturing first 120 lines or until done)..."
ENC_JOB_ID="$JOB_ID"
curl -s "$BASE_URL/api/v1/beta/examples/run/$ENC_JOB_ID/logs" \
  | awk '{
      print;
      if ($0 ~ /^event: done$/) { done_seen=1; next }
      if (done_seen && $0 ~ /^data:/) { exit }
      if (NR>240) exit
    }' | tee "$TMP"

if ! grep -q 'event: done' "$TMP"; then
  echo "[SSE TEST] Beta stream incomplete, retrying legacy educational path..." >&2
  curl -s "$BASE_URL/api/v1/educational/examples/run/$ENC_JOB_ID/logs" \
    | awk '{
        print;
        if ($0 ~ /^event: done$/) { done_seen=1; next }
        if (done_seen && $0 ~ /^data:/) { exit }
        if (NR>240) exit
      }' | tee "$TMP"
fi

echo "[SSE TEST] Validating stream..."
fail=0
grep -q '^: open' "$TMP" || { echo "[FAIL] Missing ': open' comment"; fail=1; }
grep -q '^retry: 3000' "$TMP" || { echo "[FAIL] Missing retry directive"; fail=1; }
grep -q 'event: open' "$TMP" || { echo "[FAIL] Missing open event"; fail=1; }
grep -q 'event: status' "$TMP" || { echo "[FAIL] Missing status event"; fail=1; }
grep -q 'event: log' "$TMP" || { echo "[FAIL] Missing log event"; fail=1; }
grep -q 'event: done' "$TMP" || { echo "[FAIL] Missing done event"; fail=1; }
grep -q '"job_id":"'$JOB_ID'"' "$TMP" || { echo "[FAIL] job_id not present in payloads"; fail=1; }
grep -q '"complete":true' "$TMP" || { echo "[FAIL] done payload missing complete:true"; fail=1; }

if [[ $fail -eq 0 ]]; then
  echo "[SSE TEST] ✅ All SSE expectations met"
else
  echo "[SSE TEST] ❌ SSE test failed"
  exit 1
fi