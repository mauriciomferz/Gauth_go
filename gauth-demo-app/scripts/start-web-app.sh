#!/bin/bash

# GAuth Demo Application Startup Script
# Starts backend and serves static frontend

set -e

echo "🚀 Starting GAuth Demo Application..."

# Check if Go and Python are available
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is not installed. Please install Python 3 first."
    exit 1
fi

# Set working directory to the parent of scripts folder
cd "$(dirname "$0")/.."

# Start backend server
echo "🏗️  Starting Go backend server..."
cd web/backend

# Install dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

# Start backend in background
go run main.go &
BACKEND_PID=$!
echo "✅ Backend started (PID: $BACKEND_PID) on http://localhost:8080"

# Navigate back to web directory
cd ..

# Wait a moment for backend to start
sleep 3

# Start static file server for frontend
echo "🎨 Starting static file server for frontend..."
python3 -m http.server 3000 &
FRONTEND_PID=$!
echo "✅ Frontend started (PID: $FRONTEND_PID) on http://localhost:3000"

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Shutting down GAuth+ Web Application..."
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
        echo "✅ Backend stopped"
    fi
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
        echo "✅ Frontend stopped"
    fi
    echo "👋 GAuth Demo Application stopped"
    exit 0
}

# Set trap to cleanup on script exit
trap cleanup SIGINT SIGTERM EXIT

echo ""
echo "🎉 GAuth Demo Application is running!"
echo ""
echo "📊 Available Services:"
echo "   - Backend API: http://localhost:8080"
echo "   - Frontend UI: http://localhost:3000"
echo "   - Demo Interface: http://localhost:3000"
echo ""
echo "🔧 API Endpoints:"
echo "   - Get Scenarios: GET http://localhost:8080/scenarios"
echo "   - Authenticate: POST http://localhost:8080/authenticate"
echo "   - Validate: POST http://localhost:8080/validate"
echo "   - RFC-0111 Config: POST http://localhost:8080/rfc0111/config"
echo "   - RFC-0115 PoA: POST http://localhost:8080/rfc0115/poa"
echo "   - Combined Demo: POST http://localhost:8080/combined/demo"
echo ""
echo "💡 Features:"
echo "   ✅ RFC-0111 and RFC-0115 demo scenarios"
echo "   ✅ Mock authentication and validation"
echo "   ✅ Combined RFC implementation demonstration"
echo "   ❌ Educational demo only - NOT for production use"
echo ""
echo "Press Ctrl+C to stop all services..."

# Wait for processes
wait