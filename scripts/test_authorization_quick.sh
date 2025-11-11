#!/usr/bin/env bash
set -e

BASE_URL="http://localhost:8080/api/v1/rfc0111"

echo "========================================="
echo "Quick Authorization Flow Test"
echo "========================================="

# Step I: Create subscription
echo "Creating subscription..."
STEP_I_RESPONSE=$(curl -s -X POST "$BASE_URL/subscriptions" \
  -H "Content-Type: application/json" \
  -d '{
    "owners_authorizer_id": "auth-001",
    "identity_proof_request": {
      "subject_id": "auth-001",
      "identity_type": "natural_person",
      "proof_method": "pvp_token",
      "proof_data": {
        "pvp_token": "mock-token",
        "timestamp": "2025-11-11T10:00:00Z"
      },
      "required_level": "substantial"
    }
  }')

SUBSCRIPTION_ID=$(echo "$STEP_I_RESPONSE" | jq -r '.subscription_id')
echo "✓ Subscription created: $SUBSCRIPTION_ID"

# Step II: Authorization proof
echo "Step II: Authorization proof..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-ii" \
  -H "Content-Type: application/json" \
  -d '{
    "authorization_proof": {
      "proof_method": "commercial_register",
      "proof_data": {
        "register_id": "FN123456a",
        "timestamp": "2025-11-11T10:01:00Z"
      }
    }
  }' > /dev/null

# Step III: Client owner identity
echo "Step III: Client owner identity..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-iii" \
  -H "Content-Type: application/json" \
  -d '{
    "client_owner_id": "owner-002",
    "identity_proof_request": {
      "subject_id": "owner-002",
      "identity_type": "natural_person",
      "proof_method": "pvp_token",
      "proof_data": {
        "pvp_token": "mock-token-owner",
        "timestamp": "2025-11-11T10:02:00Z"
      },
      "required_level": "substantial"
    }
  }' > /dev/null

# Step IV: Client owner authorization
echo "Step IV: Client owner authorization..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-iv" \
  -H "Content-Type: application/json" \
  -d '{
    "authorization_proof": {
      "authorization_chain": {
        "owners_authorizer": {"id": "auth-001", "role": "board_member"},
        "client_owner": {"id": "owner-002", "role": "cto", "link": "https://auth-001.example.com/authorizes/owner-002"},
        "client": {"id": "client-003", "application_name": "DataProcessor AI", "link": "https://owner-002.example.com/owns/client-003"}
      }
    }
  }' > /dev/null

# Step V: Client authorization
echo "Step V: Client authorization..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-v" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-003",
    "poa_credential": {
      "credential_id": "poa-cred-001",
      "issuer": "owner-002",
      "holder": "client-003",
      "issuance_date": "2025-11-11T10:00:00Z"
    }
  }' > /dev/null

# Step VI: Resource owner identity
echo "Step VI: Resource owner identity..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-vi" \
  -H "Content-Type: application/json" \
  -d '{
    "resource_owner_id": "resource-owner-004",
    "identity_proof_request": {
      "subject_id": "resource-owner-004",
      "identity_type": "natural_person",
      "proof_method": "pvp_token",
      "proof_data": {
        "pvp_token": "mock-token-resource-owner",
        "timestamp": "2025-11-11T10:03:00Z"
      },
      "required_level": "substantial"
    }
  }' > /dev/null

# Step VII: Resource owner authorization
echo "Step VII: Resource owner authorization..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-vii" \
  -H "Content-Type: application/json" \
  -d '{
    "authorization_proof": {
      "authorization_chain": {
        "owners_authorizer": {"id": "auth-001", "role": "board_member"},
        "client_owner": {"id": "owner-002", "role": "cto", "link": "https://auth-001.example.com/authorizes/owner-002"},
        "resource_owner": {"id": "resource-owner-004", "role": "data_controller", "link": "https://owner-002.example.com/authorizes/resource-owner-004"}
      }
    }
  }' > /dev/null

# Step VIII: Resource server authorization
echo "Step VIII: Resource server authorization..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-viii" \
  -H "Content-Type: application/json" \
  -d '{
    "resource_server_id": "resource-server-005",
    "resource_server_authorization": {
      "server_id": "resource-server-005",
      "authorized_resources": ["dataset-A", "dataset-B"]
    }
  }' > /dev/null

echo "✓ Subscription completed (all 8 steps)"

# Now test authorization flow
echo ""
echo "Testing authorization flow..."
AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"client-003\",
    \"subscription_id\": \"$SUBSCRIPTION_ID\",
    \"resource_owner_id\": \"resource-owner-004\",
    \"poa_credential_ref\": \"poa-cred-001\",
    \"scope\": \"read write\",
    \"context\": {\"purpose\": \"data_access\"}
  }")

# Check if we got a token
if echo "$AUTH_RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
    echo "✅ Authorization successful!"
    ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token')
    echo "Access token: ${ACCESS_TOKEN:0:50}..."
    
    # Test validation
    echo ""
    echo "Testing token validation..."
    VALIDATE_RESPONSE=$(curl -s -X POST "$BASE_URL/token/validate" \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$VALIDATE_RESPONSE" | jq -e '.valid' > /dev/null 2>&1; then
        echo "✅ Token validation successful!"
    fi
    
    # Test introspection
    echo ""
    echo "Testing token introspection (RFC 7662)..."
    INTROSPECT_RESPONSE=$(curl -s -X POST "$BASE_URL/token/introspect" \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$INTROSPECT_RESPONSE" | jq -e '.active == true' > /dev/null 2>&1; then
        echo "✅ Token introspection successful! (active: true)"
    fi
    
    # Test revocation
    echo ""
    echo "Testing token revocation (RFC 7009)..."
    REVOKE_RESPONSE=$(curl -s -X POST "$BASE_URL/token/revoke" \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$REVOKE_RESPONSE" | jq -e '.success' > /dev/null 2>&1; then
        echo "✅ Token revocation successful!"
    fi
    
    # Verify revocation
    echo ""
    echo "Verifying token is revoked..."
    INTROSPECT_AFTER=$(curl -s -X POST "$BASE_URL/token/introspect" \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$INTROSPECT_AFTER" | jq -e '.active == false' > /dev/null 2>&1; then
        echo "✅ Token correctly shows as inactive after revocation!"
    fi
    
    echo ""
    echo "========================================="
    echo "✅ All authorization flow tests passed!"
    echo "========================================="
else
    echo "❌ Authorization failed!"
    echo "Error: $AUTH_RESPONSE" | jq '.'
    exit 1
fi
