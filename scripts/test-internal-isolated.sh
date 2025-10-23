#!/bin/bash

# Isolated internal test runner to debug CI-specific failures
# This script mimics CI environment conditions as closely as possible

set -e

echo "🔬 Isolated Internal Test Runner"
echo "================================="

# Environment setup to match CI
export GOMAXPROCS=2
export GOMEMLIMIT=1GiB
export CGO_ENABLED=1

echo "📊 Environment:"
echo "  Go version: $(go version)"
echo "  Working dir: $(pwd)"
echo "  GOMAXPROCS: ${GOMAXPROCS}"
echo "  GOMEMLIMIT: ${GOMEMLIMIT}"
echo "  CGO_ENABLED: ${CGO_ENABLED}"
echo ""

# Clean everything like CI does
echo "🧹 Cleaning test cache and build cache..."
go clean -testcache || echo "Warning: Failed to clean test cache"
go clean -cache || echo "Warning: Failed to clean build cache"
echo ""

# List internal packages with tests
echo "📋 Internal packages with tests:"
for dir in internal/*/; do
    if [ -f "${dir}"*_test.go ] 2>/dev/null; then
        echo "  ✅ $(basename "$dir")"
    else
        echo "  ❌ $(basename "$dir") (no tests)"
    fi
done
echo ""

# Run tests one package at a time to isolate failures
echo "🧪 Running internal tests individually..."

FAILED_PACKAGES=()
TOTAL_PACKAGES=0

for pkg in internal/*/; do
    pkg_name=$(basename "$pkg")
    
    # Skip if no test files
    if ! ls "${pkg}"*_test.go >/dev/null 2>&1; then
        continue
    fi
    
    TOTAL_PACKAGES=$((TOTAL_PACKAGES + 1))
    echo "Testing: internal/$pkg_name"
    
    if go test -v -timeout=5m "./internal/$pkg_name"; then
        echo "  ✅ PASS: internal/$pkg_name"
    else
        echo "  ❌ FAIL: internal/$pkg_name"
        FAILED_PACKAGES+=("internal/$pkg_name")
    fi
    echo ""
done

# Summary
echo "📊 Test Summary:"
echo "  Total packages tested: $TOTAL_PACKAGES"
echo "  Failed packages: ${#FAILED_PACKAGES[@]}"

if [ ${#FAILED_PACKAGES[@]} -eq 0 ]; then
    echo "🎉 All internal tests passed!"
    exit 0
else
    echo "❌ Failed packages:"
    for pkg in "${FAILED_PACKAGES[@]}"; do
        echo "    $pkg"
    done
    exit 1
fi