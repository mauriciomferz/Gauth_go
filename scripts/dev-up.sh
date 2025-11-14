#!/usr/bin/env bash
set -euo pipefail

echo "🚀 Starting unified dev environment (backend + frontend)"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BACKEND_PORT="${GAUTH_PORT:-8080}"
UI_DIR="web/ui-react"
UI_PORT="${VITE_PORT:-3000}"

echo "📁 Root: $ROOT_DIR"
echo "🌐 Backend Port: $BACKEND_PORT"
echo "🖥  UI Port: $UI_PORT"

if [ ! -f .env.backend.example ]; then
  echo "⚠️  .env.backend.example missing (expected configuration template)"
fi

echo "🧹 Tidying modules (go mod tidy)"
go mod tidy >/dev/null 2>&1 || echo "(skip tidy issues)"

echo "🧪 Quick backend health build"
go build -o bin/web-server ./cmd/web-server || { echo "❌ Backend build failed"; exit 1; }

echo "🏁 Launching backend in background"
./bin/web-server > /tmp/gauth_backend.log 2>&1 &
BACKEND_PID=$!
echo "✅ Backend PID: $BACKEND_PID (logs: /tmp/gauth_backend.log)"

echo "📦 Installing UI dependencies (if needed)"
if [ -d "$UI_DIR" ]; then
  cd "$UI_DIR"
  if [ ! -d node_modules ]; then
    npm install --no-audit --no-fund
  fi
  echo "🌐 Starting Vite dev server"
  npm run dev -- --port "$UI_PORT" > /tmp/gauth_ui.log 2>&1 &
  UI_PID=$!
  echo "✅ UI PID: $UI_PID (logs: /tmp/gauth_ui.log)"
  cd "$ROOT_DIR"
else
  echo "⚠️  UI directory missing: $UI_DIR"
fi

cat <<EOF

====================================================
GAuth Dev Environment Started
====================================================
Backend:   http://localhost:$BACKEND_PORT
UI:        http://localhost:$UI_PORT
Kill:      kill $BACKEND_PID $UI_PID
Logs:      tail -f /tmp/gauth_backend.log /tmp/gauth_ui.log
====================================================
EOF
