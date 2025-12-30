#!/bin/bash

# AgentAuth Beta Web Interface Startup Script
# ⚠️ Beta Implementation - Not for Production Use

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}�️ AgentAuth Beta Web Interface${NC}"
echo -e "${YELLOW}⚠️  BETA IMPLEMENTATION ONLY${NC}"
echo -e "${BLUE}📚 RFC-0150 Go Learning Environment${NC}"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
    echo "Please install Go 1.21 or higher to run the beta demo"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | grep -oE '[0-9]+\.[0-9]+')
echo -e "${GREEN}✓ Go version: ${GO_VERSION}${NC}"

# Ensure we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}❌ Please run this script from the project root directory${NC}"
    exit 1
fi

# Check if web server exists, build if not
# Always rebuild the web server from the proper main package (cmd/web-server)
echo -e "${BLUE}🔨 Building beta web server (cmd/web-server)...${NC}"
if ! go build -o web-server ./cmd/web-server; then
    echo -e "${RED}❌ Build failed${NC}"; exit 1
fi
echo -e "${GREEN}✓ Web server built successfully${NC}"

# Default port
PORT=${1:-8080}

echo ""
echo -e "${BLUE}🚀 Starting Beta Demo Server...${NC}"
echo -e "${GREEN}🌐 Server will start on: http://localhost:${PORT}${NC}"
echo -e "${GREEN}📖 Documentation: http://localhost:${PORT}/docs/${NC}"
echo -e "${GREEN}🔧 Health Check: http://localhost:${PORT}/api/v1/beta/health${NC}"
echo ""
echo -e "${YELLOW}⚠️  Beta Notice:${NC}"
echo "   This is a learning implementation for RFC-0150 concepts"
echo "   NOT intended for production use or real security applications"
echo ""
echo -e "${BLUE}Press Ctrl+C to stop the beta demo server${NC}"
echo ""

# Start the server
./web-server "$PORT"