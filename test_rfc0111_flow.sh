#!/bin/bash

# RFC-0111 Complete Flow Test Script
# This script demonstrates the full RFC-0111 subscription and authorization flow

set -e

BASE_URL="http://localhost:8080/api/v1/rfc0111"

echo "=== RFC-0111 Complete Flow Test ==="
echo ""

# Step I: Create Subscription with Owner's Authorizer Identity Proof
echo "Step I: Initiating subscription with owner's authorizer identity proof..."
SUBSCRIPTION_RESPONSE=$(curl -s -X POST "$BASE_URL/subscriptions" \
  -H "Content-Type: application/json" \
  -d '{
    "owners_authorizer_id": "authorizer-001",
    "identity_proof_request": {
      "subject_id": "resource-owner-004",
      "identity_type": "legal_entity",
      "proof_method": "qualified_signature",
      "proof_data": {
        "certificate": "mock_cert_data",
        "signature": "mock_signature"
      },
      "required_level": "substantial"
    }
  }')

echo "Response: $SUBSCRIPTION_RESPONSE"
SUBSCRIPTION_ID=$(echo $SUBSCRIPTION_RESPONSE | jq -r '.subscription_id')

if [ "$SUBSCRIPTION_ID" == "null" ] || [ -z "$SUBSCRIPTION_ID" ]; then
    echo "ERROR: Failed to create subscription"
    exit 1
fi

echo "✓ Subscription created: $SUBSCRIPTION_ID"
echo ""

# Step II: Authorizer Authentication Proof
echo "Step II: Authorizer authentication proof..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-ii" \
  -H "Content-Type: application/json" \
  -d '{
    "auth_credential": {
      "credential_type": "qualified_certificate",
      "credential_data": {
        "certificate": "mock_cert",
        "signature": "mock_sig"
      }
    }
  }' | jq '.'
echo ""

# Step III: Client Owner Identity Proof
echo "Step III: Client owner identity proof..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-iii" \
  -H "Content-Type: application/json" \
  -d '{
    "identity_proof_request": {
      "subject_id": "client-owner-001",
      "identity_type": "legal_entity",
      "proof_method": "qualified_signature",
      "proof_data": {
        "certificate": "mock_cert",
        "signature": "mock_sig"
      },
      "required_level": "substantial"
    }
  }' | jq '.'
echo ""

# Step IV: Client Owner Authentication
echo "Step IV: Client owner authentication..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-iv" \
  -H "Content-Type: application/json" \
  -d '{
    "auth_credential": {
      "credential_type": "qualified_certificate",
      "credential_data": {
        "certificate": "mock_cert",
        "signature": "mock_sig"
      }
    }
  }' | jq '.'
echo ""

# Step V: Client Authorization (PoA)
echo "Step V: Client authorization (PoA)..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-v" \
  -H "Content-Type: application/json" \
  -d '{
    "poa_credential": {
      "credential_type": "power_of_attorney",
      "authorization_scope": {
        "authorization_type": {
          "representation_type": "direct",
          "restrictions": [],
          "sub_proxy_authority": false,
          "signature_type": "qualified"
        },
        "applicable_sectors": [],
        "applicable_regions": [],
        "authorized_actions": {
          "transactions": [],
          "decisions": [],
          "physical_actions": [],
          "non_physical_actions": ["analyzing", "documenting"]
        }
      },
      "credential_data": {
        "poa_id": "poa-cred-001",
        "signature": "mock_poa_signature"
      }
    }
  }' | jq '.'
echo ""

# Step VI: Resource Owner Identity Proof
echo "Step VI: Resource owner identity proof..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-vi" \
  -H "Content-Type: application/json" \
  -d '{
    "identity_proof_request": {
      "subject_id": "resource-owner-004",
      "identity_type": "legal_entity",
      "proof_method": "qualified_signature",
      "proof_data": {
        "certificate": "mock_cert",
        "signature": "mock_sig"
      },
      "required_level": "substantial"
    }
  }' | jq '.'
echo ""

# Step VII: Resource Owner Authentication
echo "Step VII: Resource owner authentication..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-vii" \
  -H "Content-Type: application/json" \
  -d '{
    "auth_credential": {
      "credential_type": "qualified_certificate",
      "credential_data": {
        "certificate": "mock_cert",
        "signature": "mock_sig"
      }
    }
  }' | jq '.'
echo ""

# Step VIII: Resource Server Authentication
echo "Step VIII: Resource server authentication..."
curl -s -X POST "$BASE_URL/subscriptions/$SUBSCRIPTION_ID/step-viii" \
  -H "Content-Type: application/json" \
  -d '{
    "auth_credential": {
      "credential_type": "api_key",
      "credential_data": {
        "api_key": "mock_api_key",
        "signature": "mock_sig"
      }
    }
  }' | jq '.'
echo ""

# Verify subscription is complete
echo "Verifying subscription status..."
curl -s -X GET "$BASE_URL/subscriptions/$SUBSCRIPTION_ID" | jq '.'
echo ""

# Now attempt authorization with completed subscription (Steps a-i)
echo "=== Authorization Flow (Steps a-i) ==="
echo ""
echo "Requesting token with RFC-0111 compliant authorization..."
TOKEN_RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"client-003\",
    \"subscription_id\": \"$SUBSCRIPTION_ID\",
    \"resource_owner_id\": \"resource-owner-004\",
    \"poa_credential_ref\": \"poa-cred-001\",
    \"scope\": \"read write\",
    \"context\": {\"purpose\": \"data_access\"}
  }")

echo "Token Response:"
echo "$TOKEN_RESPONSE" | jq '.'

# Check if token was successfully issued
if echo "$TOKEN_RESPONSE" | jq -e '.extended_token' > /dev/null 2>&1; then
    echo ""
    echo "✓ SUCCESS: RFC-0111 compliant token issued!"
    
    # Extract and validate token
    TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.extended_token.access_token')
    echo ""
    echo "Validating token..."
    curl -s -X POST "$BASE_URL/token/validate" \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$TOKEN\"}" | jq '.'
else
    echo ""
    echo "✗ FAILED: Token issuance failed"
    exit 1
fi

echo ""
echo "=== RFC-0111 Flow Test Complete ==="
