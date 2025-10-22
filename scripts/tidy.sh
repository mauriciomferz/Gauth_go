#!/usr/bin/env bash
set -euo pipefail

DRY=${DRY:-0}
FAST_LINT=${FAST_LINT:-1}

echo "[tidy] Starting hygiene (DRY=$DRY FAST_LINT=$FAST_LINT)"

if [[ $DRY -eq 1 ]]; then
  echo "[tidy] (dry) Listing files with formatting issues:";
  gofmt -l . | sed 's/^/  • /' || true
else
  echo "[tidy] Formatting Go sources..."
  go fmt ./...
fi

if [[ $DRY -eq 1 ]]; then
  echo "[tidy] (dry) Skipping go mod tidy"
else
  echo "[tidy] Running go mod tidy (root + go.work modules if present)..."
  go mod tidy
  if [[ -f go.work ]]; then
    awk '/use / {print $2}' go.work | while read -r m; do
      echo "[tidy] → module $m"
      (cd "$m" && go mod tidy)
    done
  fi
fi

echo "[tidy] Running go vet..."
go vet ./...

if command -v golangci-lint >/dev/null 2>&1; then
  if [[ $FAST_LINT -eq 1 ]]; then
    echo "[tidy] Running fast golangci-lint subset (ineffassign, errcheck)..."
    golangci-lint run --disable-all --enable=ineffassign --enable=errcheck || { echo "[tidy] ❌ fast lint issues"; exit 1; }
  else
    echo "[tidy] Running full golangci-lint..."
    golangci-lint run ./... || { echo "[tidy] ❌ full lint issues"; exit 1; }
  fi
else
  echo "[tidy] golangci-lint not installed (skipping lint subset). Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

echo "[tidy] Running tests (race, short)..."
go test -race -count=1 ./... >/dev/null || { echo "[tidy] ❌ tests failed"; exit 1; }

echo "[tidy] Generating TODO report (make todo-report)..."
make todo-report >/dev/null || { echo "[tidy] ❌ todo-report failed"; exit 1; }

echo "[tidy] ✅ Hygiene complete"