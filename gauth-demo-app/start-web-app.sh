#!/bin/bash

# GAuth+ Web Application Startup Script
# Starts both frontend and backend with proper configuration

set -e

echo "🚀 Starting GAuth+ Web Application..."

# Check if Node.js and Go are available
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js first."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

# Set working directory
cd "$(dirname "$0")"

# Start Redis if needed (in background)
echo "🔄 Checking Redis connection..."
if ! redis-cli ping &> /dev/null; then
    echo "⚠️  Redis not running. Starting Redis..."
    if command -v redis-server &> /dev/null; then
        redis-server --daemonize yes --port 6379
        sleep 2
        echo "✅ Redis started on port 6379"
    else
        echo "⚠️  Redis not found. Install Redis or use embedded memory store."
    fi
else
    echo "✅ Redis is already running"
fi

# Start backend server
echo "🏗️  Starting GAuth+ backend server..."
cd web/backend
if [ ! -f gauth-backend ]; then
    echo "📦 Building backend..."
    go build -o gauth-backend ./
fi

# Start backend in background
./gauth-backend &
BACKEND_PID=$!
echo "✅ Backend started (PID: $BACKEND_PID) on http://localhost:8080"

# Wait a moment for backend to start
sleep 3

# Start frontend
echo "🎨 Starting React frontend..."
cd ../frontend

# Start frontend in background
npm start &
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
    echo "👋 GAuth+ Web Application stopped"
    exit 0
}

# Set trap to cleanup on script exit
trap cleanup SIGINT SIGTERM EXIT

echo ""
echo "🎉 GAuth+ Web Application is running!"
echo ""
echo "📊 Available Services:"
echo "   - Backend API: http://localhost:8080"
echo "   - Frontend UI: http://localhost:3000"
echo "   - Health Check: http://localhost:8080/health"
echo "   - GAuth+ Demo: http://localhost:3000/gauth-plus"
echo ""
echo "🔧 API Endpoints:"
echo "   - Register AI: POST http://localhost:8080/api/v1/gauth-plus/authorize"
echo "   - Validate Authority: POST http://localhost:8080/api/v1/gauth-plus/validate"
echo "   - Commercial Register: GET http://localhost:8080/api/v1/gauth-plus/commercial-register"
echo ""
echo "💡 Features:"
echo "   ✅ Blockchain-based AI authorization registry"
echo "   ✅ Comprehensive power-of-attorney framework"
echo "   ✅ Dual control principle with human accountability"
echo "   ✅ Global commercial register with cryptographic verification"
echo ""
echo "Press Ctrl+C to stop all services..."

# Wait for processes
wait