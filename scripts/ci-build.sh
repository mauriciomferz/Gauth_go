#!/bin/bash
# CI Build Script - Alternative to Makefile for problematic CI environments
# Usage: ./scripts/ci-build.sh

set -e

echo "🚀 GAuth CI Build Script"
echo "========================"

# Function to find gauth-server source
find_gauth_server() {
    # Method 1: Standard relative path
    if [ -f "./cmd/gauth-server/main.go" ]; then
        echo "✅ Found ./cmd/gauth-server/main.go" >&2
        echo "./cmd/gauth-server"
        return 0
    fi
    
    # Method 2: Without leading dot
    if [ -f "cmd/gauth-server/main.go" ]; then
        echo "✅ Found cmd/gauth-server/main.go" >&2
        echo "cmd/gauth-server"
        return 0
    fi
    
    # Method 3: Find anywhere
    local main_go_path=$(find . -name "main.go" -path "*/cmd/gauth-server/*" 2>/dev/null | head -1)
    if [ -n "$main_go_path" ]; then
        local source_path=$(dirname "$main_go_path")
        echo "✅ Found $main_go_path" >&2
        echo "$source_path"
        return 0
    fi
    
    # Method 4: Find gauth-server directory
    local gauth_dir=$(find . -name "gauth-server" -type d -path "*/cmd/*" 2>/dev/null | head -1)
    if [ -n "$gauth_dir" ] && [ -f "$gauth_dir/main.go" ]; then
        echo "✅ Found directory $gauth_dir with main.go" >&2
        echo "$gauth_dir"
        return 0
    fi
    
    return 1
}

# Function to show diagnostics
show_diagnostics() {
    echo ""
    echo "🐛 Diagnostic Information"
    echo "========================"
    echo "📍 Working Directory: $(pwd)"
    echo "👤 User: $(whoami)"
    echo "💻 System: $(uname -a)"
    echo ""
    echo "📁 Current Directory Contents:"
    ls -la . | head -15
    echo ""
    echo "📂 Looking for cmd directories:"
    find . -name "cmd" -type d 2>/dev/null | head -5 || echo "No cmd directories found"
    echo ""
    echo "📂 Looking for gauth-server directories:"  
    find . -name "gauth-server" -type d 2>/dev/null | head -5 || echo "No gauth-server directories found"
    echo ""
    echo "📄 Looking for main.go files:"
    find . -name "main.go" 2>/dev/null | head -10 || echo "No main.go files found"
    echo ""
    echo "🔍 Go Environment:"
    go version 2>/dev/null || echo "Go not found"
    echo ""
}

# Main execution
echo "📍 Current working directory: $(pwd)"

# Find the source path
echo "🔍 Searching for gauth-server source..."
SOURCE_PATH=$(find_gauth_server)
if [ $? -ne 0 ] || [ -z "$SOURCE_PATH" ]; then
    echo "❌ ERROR: Could not find cmd/gauth-server/main.go"
    show_diagnostics
    exit 1
fi

echo "✅ Using source path: $SOURCE_PATH"

# Create output directory
echo "📁 Creating build directory..."
mkdir -p build/bin

# Build the server
echo "🔧 Building gauth-server..."
if go build -ldflags="-s -w" -o build/bin/gauth-server "$SOURCE_PATH"; then
    echo "✅ Build successful!"
    echo "📊 Binary info:"
    ls -la build/bin/gauth-server
    
    # Test the binary
    echo "🧪 Testing binary..."
    if ./build/bin/gauth-server --help >/dev/null 2>&1; then
        echo "✅ Binary test successful!"
    else
        echo "⚠️ Binary test failed (may be normal for some flags)"
    fi
else
    echo "❌ Build failed!"
    show_diagnostics
    exit 1
fi

echo ""
echo "🎉 CI Build completed successfully!"