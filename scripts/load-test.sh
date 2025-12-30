#!/bin/bash
# Load test using the Go-based runner (requires kubectl context to be set)

NAMESPACE="agentauth-staging"
SERVICE="agentauth-service"
LOCAL_PORT=8085
TARGET_URL="http://localhost:$LOCAL_PORT/api/v1/beta/authz/evaluate"

echo "🚧 Building loadtest runner..."
if ! go build -o loadtest-runner ./cmd/loadtest; then
    echo "❌ Build failed"
    exit 1
fi

echo "🔌 Establishing port-forward to $SERVICE in $NAMESPACE..."
# Find first pod for the service to port-forward (more reliable than svc forward sometimes)
POD=$(kubectl get pod -n $NAMESPACE -l app=agentauth -o jsonpath="{.items[0].metadata.name}" 2>/dev/null)
if [ -z "$POD" ]; then
    echo "⚠️  No pods found for app=agentauth. Trying service forward..."
    kubectl port-forward -n $NAMESPACE svc/$SERVICE $LOCAL_PORT:8080 > /dev/null 2>&1 &
else
    echo "Forwarding to pod: $POD"
    kubectl port-forward -n $NAMESPACE $POD $LOCAL_PORT:8080 > /dev/null 2>&1 &
fi
PF_PID=$!

# Wait for forward to be ready
echo "⏳ Waiting for connection..."
sleep 3
if ! curl -s "http://localhost:$LOCAL_PORT/api/v1/beta/health" > /dev/null; then
    echo "❌ Failed to connect to service via port-forward"
    kill $PF_PID
    exit 1
fi

echo "🚀 Running Load Test (Standard Scenario)..."
echo "Target: $TARGET_URL"
./loadtest-runner --target "$TARGET_URL" --scenario auth-std --duration 30s --users 20

TEST_EXIT_CODE=$?

echo "🧹 Cleaning up..."
kill $PF_PID
rm loadtest-runner

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ Load Test Completed Successfully"
else
    echo "❌ Load Test Failed"
    exit $TEST_EXIT_CODE
fi
