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
echo "🔍 Repository status check..."
if command -v git >/dev/null 2>&1; then
    echo "📊 Git branch: $(git branch --show-current 2>/dev/null || echo 'detached HEAD')"
    echo "📊 Git commit: $(git rev-parse --short HEAD 2>/dev/null || echo 'no commit')"
    echo "📊 Tracked files: $(git ls-files | wc -l 2>/dev/null || echo 'N/A')"
fi

# Find the source path with enhanced error handling
echo "🔍 Searching for gauth-server source..."

# First, let's ensure we can see what's happening
echo "🔍 Pre-search verification:"
echo "  Current directory: $(pwd)"
echo "  Directory contents:"
ls -la . | head -10
echo "  cmd directory check:"
if [ -d "cmd" ]; then
    echo "  ✅ cmd directory exists"
    ls -la cmd/ | head -10
else
    echo "  ❌ cmd directory missing"
fi

# Try each method individually with detailed logging
echo ""
echo "🔍 Method-by-method search:"

# Method 1
echo "  🔍 Method 1: Testing ./cmd/gauth-server/main.go"
if [ -f "./cmd/gauth-server/main.go" ]; then
    SOURCE_PATH="./cmd/gauth-server"
    echo "  ✅ Method 1: Found ./cmd/gauth-server/main.go"
else
    echo "  ❌ Method 1: ./cmd/gauth-server/main.go not found"
    
    # Method 2
    echo "  🔍 Method 2: Testing cmd/gauth-server/main.go"
    if [ -f "cmd/gauth-server/main.go" ]; then
        SOURCE_PATH="cmd/gauth-server"
        echo "  ✅ Method 2: Found cmd/gauth-server/main.go"
    else
        echo "  ❌ Method 2: cmd/gauth-server/main.go not found"
        
        # Method 3
        echo "  🔍 Method 3: Searching filesystem..."
        MAIN_GO_PATH=$(find . -name "main.go" -path "*/cmd/gauth-server/*" 2>/dev/null | head -1)
        if [ -n "$MAIN_GO_PATH" ]; then
            SOURCE_PATH=$(dirname "$MAIN_GO_PATH")
            echo "  ✅ Method 3: Found $MAIN_GO_PATH"
        else
            echo "  ❌ Method 3: No main.go found in */cmd/gauth-server/* pattern"
            
            # Method 4
            echo "  🔍 Method 4: Directory search..."
            GAUTH_DIR=$(find . -name "gauth-server" -type d -path "*/cmd/*" 2>/dev/null | head -1)
            if [ -n "$GAUTH_DIR" ] && [ -f "$GAUTH_DIR/main.go" ]; then
                SOURCE_PATH="$GAUTH_DIR"
                echo "  ✅ Method 4: Found directory $GAUTH_DIR with main.go"
            else
                echo "  ❌ Method 4: No gauth-server directory with main.go found"
                SOURCE_PATH=""
            fi
        fi
    fi
fi

# Check if we found anything
if [ -z "$SOURCE_PATH" ]; then
    echo ""
    echo "❌ ERROR: Could not find cmd/gauth-server/main.go using any method"
    echo ""
    echo "🔍 Enhanced Search Analysis:"
    echo "============================"
    echo "📂 All Go files containing 'gauth':"
    find . -name "*.go" 2>/dev/null | grep -i gauth | head -10 || echo "None found"
    echo ""
    echo "📂 All main.go files:"
    find . -name "main.go" 2>/dev/null | head -15 || echo "No main.go files found"
    echo ""
    echo "📂 All directories named gauth-server:"
    find . -name "*gauth-server*" -type d 2>/dev/null || echo "No gauth-server directories found"
    echo ""
    echo "📂 Complete cmd directory structure:"
    if [ -d "cmd" ]; then
        echo "cmd directory contents:"
        find cmd -type f -name "*.go" 2>/dev/null | head -15 || echo "No Go files in cmd"
        echo ""
        echo "cmd subdirectories:"
        find cmd -type d 2>/dev/null | head -10 || echo "No subdirectories in cmd"
    else
        echo "cmd directory does not exist"
    fi
    echo ""
    echo "📂 File system analysis:"
    echo "Total files in workspace: $(find . -type f 2>/dev/null | wc -l)"
    echo "Total Go files: $(find . -name "*.go" 2>/dev/null | wc -l)"
    echo "Files in root: $(ls -1 . 2>/dev/null | wc -l)"
    echo ""
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