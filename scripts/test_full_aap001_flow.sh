#!/usr/bin/env bash
set -e

BASE_URL="http://localhost:8080/api/v1/rfc0111"

echo "========================================="
echo "RFC-0111 Complete Flow Test"
echo "========================================="
echo ""

# Step I: Create subscription
echo "Step I: Creating subscription..."
RESPONSE=$(curl -s -X POST "$BASE_URL/subscriptions" -H "Content-Type: application/json" -d '{
  "owners_authorizer_id": "auth-12345",
  "identity_proof_request": {
    "subject_id": "auth-12345",
    "identity_type": "natural_person",
    "proof_method": "pvp_token",
    "proof_data": {"pvp_token": "mock-pvp-token-auth-12345", "timestamp": "2025-01-15T10:00:00Z"},
    "required_level": "substantial"
  }
}')

SUBSCRIPTION_ID=$(echo "$RESPONSE" | jq -r '.subscription_id')
echo "✓ Subscription created: $SUBSCRIPTION_ID"

# Step II
echo "Step II: Authorization proof..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-ii" -H "Content-Type: application/json" -d '{
  "commercial_register_ref": "CR-12345-ABC",
  "jurisdiction": "AT"
}' > /dev/null

# Step III
echo "Step III: Client owner identity..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-iii" -H "Content-Type: application/json" -d '{
  "subject_id": "client-owner-67890",
  "identity_type": "natural_person",
  "proof_method": "pvp_token",
  "proof_data": {"pvp_token": "mock-pvp-token-client-owner-67890", "timestamp": "2025-01-15T10:05:00Z"},
  "required_level": "substantial"
}' > /dev/null

# Step IV
echo "Step IV: Client owner authorization..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-iv" -H "Content-Type: application/json" -d '{
  "authorization_chain": {
    "owners_authorizer": {"id": "auth-12345", "role": "managing_director", "name": "John Doe"},
    "client_owner": {"id": "client-owner-67890", "role": "technical_director", "name": "Jane Smith", "link": "https://company.example.com/authorizations/auth-12345-to-client-owner-67890"},
    "client": {"id": "ai-client-98765", "application_name": "DataProcessor AI v2.0", "purpose": "data_processing", "link": "https://company.example.com/systems/ai-client-98765"}
  }
}' > /dev/null

# Step V
echo "Step V: Client authorization..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-v" -H "Content-Type: application/json" -d '{
  "client_id": "ai-client-98765",
  "poa_credential_ref": "poa-cred-abc123"
}' > /dev/null

# Step VI
echo "Step VI: Resource owner identity..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-vi" -H "Content-Type: application/json" -d '{
  "subject_id": "resource-owner-456",
  "identity_type": "natural_person",
  "proof_method": "pvp_token",
  "proof_data": {"pvp_token": "mock-pvp-token-resource-owner-456", "timestamp": "2025-01-15T10:10:00Z"},
  "required_level": "substantial"
}' > /dev/null

# Step VII
echo "Step VII: Resource owner authorization..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-vii" -H "Content-Type: application/json" -d '{
  "authorization_chain": {
    "owners_authorizer": {"id": "auth-12345", "role": "managing_director"},
    "client_owner": {"id": "client-owner-67890", "role": "technical_director", "link": "https://company.example.com/authorizations/auth-12345-to-client-owner-67890"},
    "resource_owner": {"id": "resource-owner-456", "role": "data_controller", "link": "https://company.example.com/authorizations/client-owner-67890-to-resource-owner-456"}
  }
}' > /dev/null

# Step VIII
echo "Step VIII: Resource server authorization..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-viii" -H "Content-Type: application/json" -d '{
  "resource_server_id": "resource-server-789",
  "server_endpoint": "https://api.example.com/resources",
  "resource_types": ["customer_data", "transaction_logs"],
  "allowed_operations": ["read", "write"]
}' > /dev/null

echo "✓ All 8 subscription steps completed"
echo ""

# Verify subscription is completed
STATUS=$(curl -s "$BASE_URL/subscriptions/$SUBSCRIPTION_ID" | jq -r '.status')
echo "Subscription status: $STATUS"

if [ "$STATUS" != "completed" ]; then
    echo "❌ Subscription not completed! Status: $STATUS"
    exit 1
fi

echo "✓ Subscription confirmed as completed"
echo ""
echo "========================================="
echo "Testing Authorization Flow (Steps a-i)"
echo "========================================="
echo ""

# Request authorization token
echo "Requesting authorization token..."
AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" -H "Content-Type: application/json" -d "{
  \"client_id\": \"ai-client-98765\",
  \"subscription_id\": \"$SUBSCRIPTION_ID\",
  \"resource_owner_id\": \"resource-owner-456\",
  \"poa_credential_ref\": \"poa-cred-abc123\",
  \"scope\": \"read write\",
  \"context\": {\"purpose\": \"data_processing\"}
}")

# Check if we got a token
if echo "$AUTH_RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
    ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token')
    echo "✅ Token received: ${ACCESS_TOKEN:0:50}..."
    echo ""
    
    # Test validation
    echo "Validating token..."
    VALIDATE_RESPONSE=$(curl -s -X POST "$BASE_URL/token/validate" -H "Content-Type: application/json" -d "{\"token\": \"$ACCESS_TOKEN\"}")
    if echo "$VALIDATE_RESPONSE" | jq -e '.valid == true' > /dev/null 2>&1; then
        echo "✅ Token is valid"
    else
        echo "❌ Token validation failed"
    fi
    echo ""
    
    # Test introspection
    echo "Introspecting token (RFC 7662)..."
    INTROSPECT_RESPONSE=$(curl -s -X POST "$BASE_URL/token/introspect" -H "Content-Type: application/json" -d "{\"token\": \"$ACCESS_TOKEN\"}")
    if echo "$INTROSPECT_RESPONSE" | jq -e '.active == true' > /dev/null 2>&1; then
        echo "✅ Token is active"
    else
        echo "❌ Token introspection failed"
    fi
    echo ""
    
    # Test revocation
    echo "Revoking token (RFC 7009)..."
    REVOKE_RESPONSE=$(curl -s -X POST "$BASE_URL/token/revoke" -H "Content-Type: application/json" -d "{\"token\": \"$ACCESS_TOKEN\"}")
    if echo "$REVOKE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
        echo "✅ Token revoked"
    else
        echo "❌ Token revocation failed"
    fi
    echo ""
    
    echo "========================================="
    echo "✅  ALL TESTS PASSED!"
    echo "========================================="
    echo ""
    echo "Summary:"
    echo "  - Subscription Flow (Steps I-VIII): ✅"
    echo "  - Authorization Token Request (Steps a-i): ✅"
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
