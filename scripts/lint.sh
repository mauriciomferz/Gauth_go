#!/bin/bash
# Reliable golangci-lint runner for Go 1.24.0 projects
# This script ensures proper environment variables are set for version compatibility

set -e

echo "🔍 Running golangci-lint with Go 1.24.0 compatibility..."

GOROOT=$(go env GOROOT) \
GOVERSION=1.24.0 \
$(go env GOPATH)/bin/golangci-lint run --timeout=10m "$@"

echo "✅ Linting completed!"