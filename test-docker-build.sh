#!/bin/bash

# Docker Build Verification Script for GAuth
# This script verifies that the Docker build process works correctly

set -e

echo "🐳 GAuth Docker Build Verification"
echo "=================================="

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker first."
    exit 1
fi

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
    echo "❌ Docker daemon is not running. Please start Docker first."
    echo "   On macOS: Start Docker Desktop"
    echo "   On Linux: sudo systemctl start docker"
    exit 1
fi

echo "✅ Docker is available and running"

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "📁 Building from directory: $SCRIPT_DIR"

# Clean up any previous test images
echo "🧹 Cleaning up previous test images..."
docker rmi gauth-demo:test 2>/dev/null || true

echo "🔨 Building Docker image..."
echo "   Image: gauth-demo:test"
echo "   Context: . (excluding gauth-demo-app/ via .dockerignore)"
echo "   Strategy: Remove problematic local module dependency during build"
echo ""

# Build the Docker image
if docker build -t gauth-demo:test .; then
    echo ""
    echo "✅ Docker build completed successfully!"
    
    # Get image information
    echo ""
    echo "📊 Image Information:"
    docker images gauth-demo:test --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
    
    echo ""
    echo "🧪 Testing the built application..."
    
    # Test the application in a container
    echo "   Running: docker run --rm gauth-demo:test --help"
    if docker run --rm gauth-demo:test --help; then
        echo ""
        echo "✅ Application runs successfully in container!"
    else
        echo ""
        echo "❌ Application failed to run in container"
        exit 1
    fi
    
    echo ""
    echo "🎉 Docker build verification completed successfully!"
    echo ""
    echo "To run the container:"
    echo "   docker run -p 8080:8080 gauth-demo:test"
    echo ""
    echo "To clean up:"
    echo "   docker rmi gauth-demo:test"
    
else
    echo ""
    echo "❌ Docker build failed!"
    echo ""
    echo "Troubleshooting tips:"
    echo "1. Check if all required files are present"
    echo "2. Verify go.mod and go.sum are valid"
    echo "3. Ensure no local module dependencies are missing"
    echo "4. Check Dockerfile syntax"
    exit 1
fi