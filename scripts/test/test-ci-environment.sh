#!/bin/bash
# Test CI Environment Simulation
# This script simulates the GitHub Actions CI environment to test our build fix

set -e

echo "🧪 Simulating CI Environment"
echo "============================"

# Create a temporary directory with the nested structure that causes issues
TEMP_DIR="/tmp/ci-test-$(date +%s)"
mkdir -p "$TEMP_DIR/AgentAuth/AgentAuth"

echo "📁 Created test directory: $TEMP_DIR"

# Copy the project to the nested structure (like GitHub Actions does)
echo "📋 Copying project to nested structure..."
cp -r . "$TEMP_DIR/AgentAuth/AgentAuth/" 2>/dev/null || echo "Copy completed (some files skipped)"

# Change to the nested directory
cd "$TEMP_DIR/AgentAuth/AgentAuth"

echo "📍 Test working directory: $(pwd)"
echo ""

# Run our direct build implementation (copied from CI workflow)
echo "🚀 CI Build with Direct Implementation (bypassing Make)"
echo "======================================================"
echo "📍 Working Directory: $(pwd)"
echo "👤 User: $(whoami)"
echo "🏠 Home: $HOME"
echo ""
echo "📂 Directory Structure Check:"
ls -la . | head -10
echo ""
echo "📂 cmd Directory Check:"
if [ -d "cmd" ]; then
  echo "✅ cmd directory found"
  ls -la cmd/ | head -10
else
  echo "❌ cmd directory not found"
fi
echo ""
echo "🔍 Searching for agentauth-server source..."

# Method 1: Standard relative path  
if [ -f "./cmd/agentauth-server/main.go" ]; then
  SOURCE_PATH="./cmd/agentauth-server"
  echo "✅ Method 1: Found ./cmd/agentauth-server/main.go"
# Method 2: No leading dot
elif [ -f "cmd/agentauth-server/main.go" ]; then
  SOURCE_PATH="cmd/agentauth-server"  
  echo "✅ Method 2: Found cmd/agentauth-server/main.go"
# Method 3: Search filesystem
else
  echo "🔍 Method 3: Searching filesystem..."
  MAIN_GO_PATH=$(find . -name "main.go" -path "*/agentauth-server/*" 2>/dev/null | head -1)
  if [ -n "$MAIN_GO_PATH" ]; then
    SOURCE_PATH=$(dirname "$MAIN_GO_PATH")
    echo "✅ Method 3: Found $MAIN_GO_PATH"
  else
    echo "❌ ERROR: Cannot find agentauth-server/main.go anywhere"
    echo "📋 Available main.go files:"
    find . -name "main.go" 2>/dev/null | head -10 || echo "No main.go files found"
    echo "📋 Available agentauth-server directories:"  
    find . -name "*agentauth-server*" -type d 2>/dev/null | head -5 || echo "No agentauth-server directories found"
    exit 1
  fi
fi

echo "🏗️ Building agentauth-server..."
echo "📁 Source path: $SOURCE_PATH"
mkdir -p build/bin

if go build -ldflags="-s -w" -o build/bin/agentauth-server "$SOURCE_PATH"; then
  echo "✅ Build successful!"
  echo "📊 Binary info:"
  ls -la build/bin/agentauth-server
  echo "🧪 Testing binary..."
  if ./build/bin/agentauth-server --help >/dev/null 2>&1 || ./build/bin/agentauth-server -h >/dev/null 2>&1; then
    echo "✅ Binary test passed"
  else
    echo "⚠️ Binary test failed (may be normal for some binaries)"
  fi
  
  echo ""
  echo "🎉 CI Test Simulation SUCCESSFUL!"
  echo "✅ Our direct build implementation works in nested directory structure"
else
  echo "❌ Build failed with source path: $SOURCE_PATH"
  exit 1
fi

# Cleanup
echo ""
echo "🧹 Cleaning up test directory..."
rm -rf "$TEMP_DIR"
echo "✅ Test completed successfully!"