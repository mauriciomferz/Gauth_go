#!/usr/bin/env bash
set -euo pipefail

PORT="8080"
if [ "${1:-}" != "" ]; then
  PORT="$1"
fi

echo "[smoke] freeing port :$PORT if occupied..."
lsof -ti:"$PORT" | xargs kill -9 2>/dev/null || true

echo "[smoke] building web-server..."
go build -o bin/web-server ./cmd/web-server

./bin/web-server &
PID=$!
echo "[smoke] started web-server PID=$PID on :$PORT"
trap 'echo "[smoke] stopping PID=$PID"; kill $PID 2>/dev/null || true' EXIT

ATTEMPTS=30
SLEEP=0.25
READY=0
for i in $(seq 1 $ATTEMPTS); do
  CODE=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/v1/beta/health" || true)
  if [ "$CODE" = "200" ]; then
    READY=1
    echo "[smoke] health endpoint 200 OK (attempt $i)"
    break
  fi
  sleep $SLEEP
done

if [ $READY -ne 1 ]; then
  SECS=$(python - <<'PY'
att=30; sl=0.25; print(f"{att*sl:.2f}")
PY
  )
  echo "[smoke] health endpoint did not reach 200 after ${SECS}s" >&2
  exit 1
fi

# Additional probes (extendable)
STATUS_CODE=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/v1/discovery" || true)
echo "[smoke] discovery status code: $STATUS_CODE"
if [ "$STATUS_CODE" != "200" ]; then
  echo "[smoke] discovery endpoint not healthy" >&2
  exit 1
fi

echo "[smoke] OK"
