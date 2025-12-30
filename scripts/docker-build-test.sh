#!/bin/bash

# Docker Build Test Script
# This script tests the Docker build process and validates the container

set -e

echo "🐳 Testing Docker Build Process..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${2}${1}${NC}"
}

# Check if Docker is running
if ! docker info &> /dev/null; then
    print_status "❌ Docker daemon is not running. Please start Docker first." "$RED"
    exit 1
fi

print_status "✅ Docker daemon is running" "$GREEN"

# Clean up any existing test containers/images
echo "🧹 Cleaning up previous test artifacts..."
docker rm -f agentauth-test-container 2>/dev/null || true
docker rmi -f agentauth-test 2>/dev/null || true

# Build the Docker image
print_status "🔨 Building Docker image..." "$YELLOW"
if docker build -t agentauth-test .; then
    print_status "✅ Docker build successful!" "$GREEN"
else
    print_status "❌ Docker build failed!" "$RED"
    exit 1
fi

# Test running the container (without starting the server)
print_status "🧪 Testing container startup..." "$YELLOW"
if docker run --name agentauth-test-container --rm -d -p 8080:8080 agentauth-test; then
    print_status "✅ Container started successfully!" "$GREEN"
    
    # Wait a moment for startup
    sleep 3
    
    # Check if container is still running
    if docker ps | grep -q agentauth-test-container; then
        print_status "✅ Container is running!" "$GREEN"
        
        # Test health check endpoint (if available)
        if command -v curl &> /dev/null; then
            if curl -f http://localhost:8080/health &> /dev/null; then
                print_status "✅ Health check passed!" "$GREEN"
            else
                print_status "⚠️  Health check not responding (this may be expected)" "$YELLOW"
            fi
        fi
        
        # Stop the container
        docker stop agentauth-test-container &> /dev/null
        print_status "✅ Container stopped cleanly" "$GREEN"
    else
        print_status "❌ Container failed to stay running" "$RED"
        docker logs agentauth-test-container
        exit 1
    fi
else
    print_status "❌ Container failed to start!" "$RED"
    exit 1
fi

# Clean up
print_status "🧹 Cleaning up test artifacts..." "$YELLOW"
docker rmi agentauth-test &> /dev/null || true

print_status "🎉 Docker build test completed successfully!" "$GREEN"
print_status "⚠️  Dockerfile build verified - FOR BETA DEMONSTRATION / NON-PRODUCTION USE ONLY" "$YELLOW"

echo ""
echo "To build and run manually:"
echo "  docker build -t agentauth-server ."
echo "  docker run -d -p 8080:8080 --name agentauth agentauth-server"