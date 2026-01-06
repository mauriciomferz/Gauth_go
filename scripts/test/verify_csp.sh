#!/usr/bin/env bash
set -euo pipefail
URL="${1:-http://localhost:8080/}"
DEBUG="${DEBUG_CSP:-0}"

red() { printf "\033[31m%s\033[0m\n" "$*"; }
green() { printf "\033[32m%s\033[0m\n" "$*"; }
yellow() { printf "\033[33m%s\033[0m\n" "$*"; }

echo "[CSP VERIFY] Checking Content-Security-Policy header..."

# Auto-start server if not running (simple heuristic: nothing listening on port 8080)
HOST_PORT=$(python3 - <<'PY'
import socket
s=socket.socket();
try:
    s.connect(("127.0.0.1",8080));
    print("LISTENING")
except Exception:
    print("DOWN")
finally:
    s.close()
PY
)
start_server() {
  yellow "[CSP VERIFY] Launching web server..."
  (./bin/web-server >/tmp/verify_csp_server.log 2>&1 & disown) || return 1
  sleep 1
}

build_server() {
  yellow "[CSP VERIFY] Building web server binary for host $(go env GOOS)/$(go env GOARCH)..."
  go build -o bin/web-server ./cmd/web-server/main.go
}

ensure_server() {
  # Force rebuild if binary missing or REBUILD_CSP=1
  if [[ ! -x bin/web-server || "${REBUILD_CSP:-0}" == "1" ]]; then
    if command -v go >/dev/null 2>&1; then
      build_server || { red "FAIL: build failed"; exit 1; }
    else
      red "FAIL: Go not installed and bin/web-server missing"; exit 1
    fi
  fi
  if ! start_server; then
    # Detect exec format error or permissions
    if grep -qi "cannot execute binary file" /tmp/verify_csp_server.log 2>/dev/null; then
      yellow "[CSP VERIFY] Detected exec format error. Removing and rebuilding binary..."
      rm -f bin/web-server
      build_server || { red "FAIL: rebuild failed"; exit 1; }
      start_server || { red "FAIL: server failed to start after forced rebuild"; exit 1; }
    else
      red "FAIL: Unknown server start failure (see /tmp/verify_csp_server.log)"; exit 1
    fi
  fi
}

if [[ $HOST_PORT == "DOWN" ]]; then
  yellow "[CSP VERIFY] No process on port 8080; ensuring server binary and starting..."
  ensure_server
fi
# Retry curl a few times in case server not yet ready
max_attempts=${CSP_RETRIES:-10}
attempt=1
CSP_HEADER=""
fallback_paths=("" "/index.html" "/index" )
while (( attempt <= max_attempts )); do
  target="$URL"
  # If base URL ends with '/', try appended fallback path on retries
  if [[ $attempt -gt 1 ]]; then
    fp_index=$(( (attempt - 2) % ${#fallback_paths[@]} ))
    fp="${fallback_paths[$fp_index]}"
    # only adjust if original looks like base
    if [[ $URL =~ /$ ]]; then
      target="${URL%/}$fp"
    fi
  fi
  [[ "$DEBUG" == "1" ]] && echo "[debug] attempt $attempt fetching headers from $target" >&2
  if CSP_RAW=$(curl -s -I -X GET --max-time 4 "$target" || true); then
    CSP_HEADER=$(printf '%s' "$CSP_RAW" | awk -F': ' 'tolower($1)=="content-security-policy" {print $2}' | tr -d '\r')
    if [[ -n "$CSP_HEADER" ]]; then
      break
    fi
  fi
  echo "(attempt $attempt/$max_attempts) CSP header not available yet from $target, waiting..."
  sleep 1
  ((attempt++))
done
if [[ -z "$CSP_HEADER" ]]; then
  red "FAIL: No CSP header present (checked $max_attempts attempts across variants)"
  if [[ -f /tmp/verify_csp_server.log ]]; then
    yellow "--- server log tail ---"
    tail -n 40 /tmp/verify_csp_server.log || true
    yellow "------------------------"
  fi
  exit 1
fi

echo "CSP: $CSP_HEADER"

# Basic policy assertions
fail=0
if grep -qi "unsafe-inline" <<<"$CSP_HEADER"; then red "FAIL: policy contains unsafe-inline"; fail=1; fi
if grep -qi "unsafe-eval" <<<"$CSP_HEADER"; then red "FAIL: policy contains unsafe-eval"; fail=1; fi
if ! grep -q "script-src" <<<"$CSP_HEADER"; then red "FAIL: script-src directive missing"; fail=1; fi
if ! grep -q "default-src" <<<"$CSP_HEADER"; then red "FAIL: default-src directive missing"; fail=1; fi

# Optional connect-src validation (for SSE / APIs)
SSE_ORIGIN="${CSP_SSE_ORIGIN:-}"
if grep -q "connect-src" <<<"$CSP_HEADER"; then
  CONNECT_SRC=$(grep -o 'connect-src[^;]*' <<<"$CSP_HEADER" || true)
  if ! grep -q "connect-src" <<<"$CSP_HEADER"; then
    red "FAIL: connect-src directive missing"; fail=1;
  else
    if ! grep -q "'self'" <<<"$CONNECT_SRC"; then
      red "FAIL: connect-src missing 'self' (needed for same-origin SSE)"; fail=1;
    fi
    if [[ -n "$SSE_ORIGIN" ]] && ! grep -q "$SSE_ORIGIN" <<<"$CONNECT_SRC"; then
      red "FAIL: connect-src missing required SSE origin $SSE_ORIGIN"; fail=1;
    fi
  fi
else
  yellow "WARN: No connect-src directive (browser will fallback to default-src; add connect-src for clarity)"
fi

# Fetch HTML
TMP_HTML=$(mktemp)
attempt=1
while (( attempt <= max_attempts )); do
  html_target="$URL"
  if [[ $attempt -gt 1 ]]; then
    fp_index=$(( (attempt - 2) % ${#fallback_paths[@]} ))
    fp="${fallback_paths[$fp_index]}"
    if [[ $URL =~ /$ ]]; then
      html_target="${URL%/}$fp"
    fi
  fi
  [[ "$DEBUG" == "1" ]] && echo "[debug] attempt $attempt fetching html from $html_target" >&2
  if curl -s --max-time 4 "$html_target" > "$TMP_HTML" && [[ -s "$TMP_HTML" ]]; then
    break
  fi
  echo "(attempt $attempt/$max_attempts) Failed to fetch HTML from $html_target, retrying..." >&2
  sleep 1
  ((attempt++))
done
if [[ ! -s "$TMP_HTML" ]]; then
  red "FAIL: Unable to retrieve HTML from $URL (fallbacks tried) after $max_attempts attempts"
  exit 1
fi
[[ "$DEBUG" == "1" ]] && echo "[debug] fetched HTML size $(wc -c < "$TMP_HTML") bytes" >&2

# Checks on HTML content
# Count scripts without src attribute (robust against zero matches under pipefail)
script_tags=$(grep -o "<script[^>]*>" "$TMP_HTML" || true)
if [[ -z "$script_tags" ]]; then
  scripts_without_src=0
else
  scripts_without_src=$(printf '%s\n' "$script_tags" | grep -vc "src=" || true)
fi
if (( scripts_without_src > 0 )); then
  red "FAIL: Found $scripts_without_src inline <script> tag(s) without src"
  fail=1
fi

# Look for inline event handlers
if grep -Eqi "on[a-z]+=" "$TMP_HTML"; then
  red "FAIL: Found inline event handler attributes"
  fail=1
fi

# Look for inline style attributes
if grep -qi "style=\"" "$TMP_HTML"; then
  yellow "WARN: Found inline style attributes (styles allowed but prefer external)"
fi

# Look for suspected eval usage in JS assets (download script tags) without pipefail issues
JS_DIR=$(mktemp -d)
script_srcs=$(awk 'BEGIN{IGNORECASE=1}/<script/{print}' "$TMP_HTML" | grep -o 'src="[^"]*"' | cut -d'"' -f2 || true)
if [[ -n "$script_srcs" ]]; then
  while IFS= read -r src; do
    [[ -z "$src" ]] && continue
    case "$src" in
      http*|//cdn*) curl -s "$src" -o "$JS_DIR/$(basename "$src" | cut -d'?' -f1)" || true;;
      /*) curl -s "${URL%/}$src" -o "$JS_DIR/$(basename "$src")" || true;;
    esac
  done <<< "$script_srcs"
fi
if grep -RqiE 'eval\(|new Function\(|setTimeout\(\s*"|setTimeout\(\s*'"'"'|setInterval\(\s*"|setInterval\(\s*'"'"'' "$JS_DIR"; then
  red "FAIL: Found disallowed dynamic code evaluation patterns in JS"
  fail=1
fi

if (( fail == 0 )); then
  green "PASS: CSP verification passed with no inline scripts, handlers, or unsafe eval patterns."
else
  red "CSP verification failed."; exit 1
fi

rm -rf "$TMP_HTML" "$JS_DIR" >/dev/null 2>&1 || true
