#!/bin/bash
cd /Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/gauth-demo-app/web/backend
echo "Building server..."
go build -o server main.go
if [ $? -eq 0 ]; then
    echo "Starting server..."
    ./server &
    SERVER_PID=$!
    echo "Server started with PID: $SERVER_PID"
    sleep 3
    echo "Testing health endpoint..."
    curl http://localhost:8080/health
    echo ""
    echo "Testing RFC111 endpoint..."
    curl -X POST -H "Content-Type: application/json" -d '{"issuer": "test", "ai_system": "test"}' http://localhost:8080/api/v1/rfc111/authorize
    echo ""
else
    echo "Build failed"
fi
