#!/usr/bin/env bash
set -euo pipefail

IMAGE_TAG="agentauth:minimal"
PORT=8080
HEALTH_URL="http://localhost:${PORT}/api/v1/beta/health"
CONTAINER_NAME="agentauth-smoke-minimal-$$"

log() { printf "[smoke] %s\n" "$*"; }

log "Building minimal image (${IMAGE_TAG})"
docker build -f Dockerfile.minimal -t ${IMAGE_TAG} . >/dev/null

log "Starting container (${CONTAINER_NAME})"
docker run -d --rm -p ${PORT}:8080 --name ${CONTAINER_NAME} ${IMAGE_TAG} >/dev/null

cleanup() {
  docker rm -f ${CONTAINER_NAME} >/dev/null 2>&1 || true
}
trap cleanup EXIT

log "Waiting for health endpoint..."
retries=20
sleep 1
until curl -fsS "${HEALTH_URL}" >/dev/null 2>&1; do
  retries=$((retries-1))
  if [ $retries -le 0 ]; then
    log "❌ Health endpoint did not become ready: ${HEALTH_URL}" >&2
    exit 1
  fi
  sleep 1
done

status=$(curl -sS "${HEALTH_URL}")
log "✅ Health endpoint OK: $status"

log "Running embedded healthcheck flag"
if docker exec ${CONTAINER_NAME} /web-server -healthcheck >/dev/null 2>&1; then
  log "✅ In-container healthcheck flag succeeded"
else
  log "❌ In-container healthcheck flag failed" >&2
  exit 1
fi

log "All smoke tests passed"
