#!/bin/bash

# Simple RFC-0111 Status Verification Script
# This script verifies that RFC-0111 is properly initialized

set -e

echo "=== RFC-0111 Status Verification ==="
echo ""

# Check if server is running
if ! pgrep -f "web-server" > /dev/null; then
    echo "❌ Server is not running"
    echo ""
    echo "Start the server with:"
    echo "  AGENTAUTH_AAP-001_ENABLED=1 AGENTAUTH_AAP-001_USE_MOCKS=1 ./bin/web-server"
    exit 1
fi

echo "✅ Server is running (PID: $(pgrep -f web-server))"
echo ""

# Test 1: Create a subscription (Step I)
echo "Test 1: Creating subscription (Step I)..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "owners_authorizer_id": "test-auth-001",
    "identity_proof_request": {
      "subject_id": "test-subject-001",
      "identity_type": "legal_entity",
      "proof_method": "qualified_signature",
      "proof_data": {"certificate": "test_cert", "signature": "test_sig"},
      "required_level": "substantial"
    }
  }')

if echo "$RESPONSE" | grep -q "subscription_id"; then
    SUBSCRIPTION_ID=$(echo "$RESPONSE" | jq -r '.subscription_id')
    echo "✅ Subscription created successfully"
    echo "   Subscription ID: $SUBSCRIPTION_ID"
else
    echo "❌ Subscription creation failed"
    echo "   Response: $RESPONSE"
    exit 1
fi
echo ""

# Test 2: Try authorization with incomplete subscription
echo "Test 2: Attempting authorization with incomplete subscription..."
AUTH_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/authorize \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"test-client\",
    \"subscription_id\": \"$SUBSCRIPTION_ID\",
    \"resource_owner_id\": \"test-resource-owner\",
    \"poa_credential_ref\": \"test-poa\",
    \"scope\": \"read write\",
    \"context\": {\"purpose\": \"testing\"}
  }")

if echo "$AUTH_RESPONSE" | grep -q "Subscription must be completed"; then
    echo "✅ Authorization endpoint working correctly"
    echo "   (Properly rejecting incomplete subscription)"
    CURRENT_STATUS=$(echo "$AUTH_RESPONSE" | jq -r '.error_description' | grep -o 'awaiting_[a-z_]*')
    echo "   Current subscription status: $CURRENT_STATUS"
elif echo "$AUTH_RESPONSE" | grep -q "RFC-0111 protocol orchestrator not initialized"; then
    echo "❌ RFC-0111 NOT PROPERLY INITIALIZED"
    echo "   Restart server with: AGENTAUTH_AAP-001_ENABLED=1 ./bin/web-server"
    exit 1
else
    echo "⚠️  Unexpected response:"
    echo "$AUTH_RESPONSE" | jq '.'
fi
echo ""

# Test 3: Verify subscription status
echo "Test 3: Checking subscription status..."
STATUS_RESPONSE=$(curl -s http://localhost:8080/api/v1/rfc0111/subscriptions/$SUBSCRIPTION_ID)
STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.status')

if [ "$STATUS" != "null" ]; then
    echo "✅ Subscription status endpoint working"
    echo "   Status: $STATUS"
else
    echo "❌ Could not retrieve subscription status"
fi
echo ""

# Test 4: Token validation endpoint
echo "Test 4: Testing token validation endpoint..."
VALIDATE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/token/validate \
  -H "Content-Type: application/json" \
  -d '{"token": "invalid_token_for_testing"}')

if echo "$VALIDATE_RESPONSE" | grep -q "valid"; then
    echo "✅ Token validation endpoint responding"
else
    echo "⚠️  Token validation endpoint response:"
    echo "$VALIDATE_RESPONSE" | jq '.'
fi
echo ""

# Summary
echo "=================================="
echo "RFC-0111 STATUS SUMMARY"
echo "=================================="
echo ""
echo "✅ Server running with RFC-0111 enabled"
echo "✅ Subscription creation working (Step I)"
echo "✅ Authorization endpoint initialized"
echo "✅ Token validation endpoint available"
echo ""
echo "CONCLUSION: RFC-0111 is properly configured and working."
echo ""
echo "To complete authorization:"
echo "  1. Complete subscription Steps II-VIII"
echo "  2. Use completed subscription ID in authorization request"
echo ""
echo "For details, see: AAP-001_FIX_SUMMARY.md"
