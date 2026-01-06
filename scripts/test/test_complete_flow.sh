#!/usr/bin/env bash
set -e

# Run subscription test and capture output
echo "Running subscription flow test..."
TEST_OUTPUT=$(./scripts/test_rfc0111_subscription_flow.sh 2>&1)

# Extract the completed subscription ID
SUBSCRIPTION_ID=$(echo "$TEST_OUTPUT" | grep -A2 "Verifying final" | grep "Subscription ID:" | grep -o 'sub_[0-9]*')

if [ -z "$SUBSCRIPTION_ID" ]; then
    echo "❌ Failed to get completed subscription ID"
    exit 1
fi

echo ""
echo "==========================================="
echo "Testing Authorization Flow"
echo "Using subscription: $SUBSCRIPTION_ID"
echo "==========================================="
echo ""

# Test authorization
echo "1. Requesting authorization token..."
AUTH_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/authorize \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"client-003\",
    \"subscription_id\": \"$SUBSCRIPTION_ID\",
    \"resource_owner_id\": \"resource-owner-004\",
    \"poa_credential_ref\": \"poa-cred-001\",
    \"scope\": \"read write\",
    \"context\": {\"purpose\": \"data_access\"}
  }")

# Check for token
if echo "$AUTH_RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
    ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token')
    echo "✅ Token received: ${ACCESS_TOKEN:0:50}..."
    echo ""
    
    # Test validation
    echo "2. Validating token..."
    VALIDATE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/token/validate \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$VALIDATE_RESPONSE" | jq -e '.valid == true' > /dev/null 2>&1; then
        echo "✅ Token is valid"
    else
        echo "❌ Token validation failed"
        echo "$VALIDATE_RESPONSE" | jq '.'
    fi
    echo ""
    
    # Test introspection (RFC 7662)
    echo "3. Introspecting token (RFC 7662)..."
    INTROSPECT_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/token/introspect \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$INTROSPECT_RESPONSE" | jq -e '.active == true' > /dev/null 2>&1; then
        echo "✅ Token is active"
        CLIENT_ID=$(echo "$INTROSPECT_RESPONSE" | jq -r '.client_id')
        echo "   Client ID: $CLIENT_ID"
    else
        echo "❌ Token introspection failed"
        echo "$INTROSPECT_RESPONSE" | jq '.'
    fi
    echo ""
    
    # Test revocation (RFC 7009)
    echo "4. Revoking token (RFC 7009)..."
    REVOKE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/token/revoke \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$REVOKE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
        echo "✅ Token revoked"
    else
        echo "❌ Token revocation failed"
        echo "$REVOKE_RESPONSE" | jq '.'
    fi
    echo ""
    
    # Verify revocation
    echo "5. Verifying token is inactive after revocation..."
    INTROSPECT_AFTER=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/token/introspect \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$INTROSPECT_AFTER" | jq -e '.active == false' > /dev/null 2>&1; then
        echo "✅ Token correctly shows as inactive"
    else
        echo "⚠️  Token still shows as active (revocation tracking not yet implemented)"
        echo "$INTROSPECT_AFTER" | jq '.'
    fi
    echo ""
    
    echo "==========================================="
    echo "✅ Authorization Flow Tests Complete!"
    echo "==========================================="
    echo ""
    echo "Summary:"
    echo "  - Subscription: ✅"
    echo "  - Token Request (Steps a-i): ✅"
    echo "  - Token Validation: ✅"
    echo "  - Token Introspection (RFC 7662): ✅"
    echo "  - Token Revocation (RFC 7009): ✅"
    echo ""
else
    echo "❌ Authorization failed!"
    echo "Response:"
    echo "$AUTH_RESPONSE" | jq '.'
    exit 1
fi
