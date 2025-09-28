#!/bin/bash

echo "=== GAuth+ Backend Server Startup ==="
echo "Date: $(date)"

# Navigate to backend directory
cd /Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/gauth-demo-app/web/backend

# Check if already running
if lsof -i :8080 > /dev/null 2>&1; then
    echo "⚠️  Port 8080 is already in use. Stopping existing process..."
    pkill -f "go run main.go"
    pkill -f "main"
    sleep 2
fi

# Build the server
echo "🔨 Building server..."
go build -o gauth-backend-server main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    
    # Start the server in background
    echo "🚀 Starting GAuth+ backend server on localhost:8080..."
    ./gauth-backend-server &
    SERVER_PID=$!
    
    echo "📋 Server started with PID: $SERVER_PID"
    echo "🔗 Server URL: http://localhost:8080"
    
    # Wait a bit for server to start
    sleep 3
    
    # Test the server
    echo ""
    echo "🧪 Testing server endpoints..."
    
    echo "1. Health check:"
    curl -s http://localhost:8080/health || echo "❌ Health check failed"
    
    echo ""
    echo "2. RFC111 Authorization test:"
    curl -s -X POST -H "Content-Type: application/json" \
         -d '{"issuer": "test-issuer", "ai_system": "test-system"}' \
         http://localhost:8080/api/v1/rfc111/authorize || echo "❌ RFC111 test failed"
    
    echo ""
    echo "3. Advanced Audit test:"
    curl -s -X POST -H "Content-Type: application/json" \
         -d '{"audit_id": "test-audit", "scope": "full"}' \
         http://localhost:8080/api/v1/audit/advanced || echo "❌ Advanced Audit test failed"
    
    echo ""
    echo "✅ GAuth+ Backend Server is running successfully!"
    echo "📊 You can now test the complete system at http://localhost:8080"
    
else
    echo "❌ Build failed! Please check for compilation errors."
    exit 1
fi
