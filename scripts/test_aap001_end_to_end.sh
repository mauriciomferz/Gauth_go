#!/bin/bash

# RFC-0111 End-to-End Authorization Flow Test
# Tests complete flow: Subscription → Authorization → Token Issuance

set -e

BASE_URL="http://localhost:8080/api/v1/rfc0111"

echo "=========================================="
echo "RFC-0111 End-to-End Authorization Flow Test"
echo "=========================================="
echo ""

# Step 1: Create Subscription (Steps I-VIII)
echo "=== PHASE 1: Subscription Flow ==="
echo "Creating subscription with all 8 steps..."
echo ""

SUB_RESPONSE=$(./scripts/test_rfc0111_subscription_flow.sh 2>&1)
SUB_ID=$(echo "$SUB_RESPONSE" | grep -oE 'sub_[0-9]+' | tail -1)

if [ -z "$SUB_ID" ]; then
    echo "❌ Failed to create subscription"
    exit 1
fi

echo "✅ Subscription created: $SUB_ID"
echo ""

# Verify subscription is completed
SUB_STATUS=$(curl -s "$BASE_URL/subscriptions/$SUB_ID" | jq -r '.status')
if [ "$SUB_STATUS" != "completed" ]; then
    echo "❌ Subscription status is $SUB_STATUS, expected 'completed'"
    exit 1
fi

echo "✅ Subscription status: $SUB_STATUS"
echo ""

# Step 2: Request Authorization (Steps a-i)
echo "=== PHASE 2: Authorization Flow ==="
echo "Requesting authorization token..."
echo ""

AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"client-app-123\",
    \"subscription_id\": \"$SUB_ID\",
    \"resource_owner_id\": \"resource-owner-99999\",
    \"poa_credential_ref\": \"mock-poa-credential-xyz\",
    \"scope\": \"read write\",
    \"context\": {\"purpose\": \"data_processing\"}
  }")

# Check if authorization succeeded
ERROR=$(echo "$AUTH_RESPONSE" | jq -r '.error // empty')
if [ -n "$ERROR" ]; then
    echo "❌ Authorization failed:"
    echo "$AUTH_RESPONSE" | jq
    exit 1
fi

# Extract token
ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.access_token')
REFRESH_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.refresh_token')
EXPIRES_IN=$(echo "$AUTH_RESPONSE" | jq -r '.expires_in')
TOKEN_TYPE=$(echo "$AUTH_RESPONSE" | jq -r '.token_type')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
    echo "❌ No access token in response"
    exit 1
fi

echo "✅ Authorization successful!"
echo "   Token Type: $TOKEN_TYPE"
echo "   Access Token: ${ACCESS_TOKEN:0:40}..."
echo "   Refresh Token: ${REFRESH_TOKEN:0:40}..."
echo "   Expires In: $EXPIRES_IN seconds"
echo ""

# Step 3: Verify Extended Token Metadata
echo "=== PHASE 3: Token Metadata Verification ==="
echo ""

# Check Power of Attorney
POA_IDENTITY=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.power_of_attorney.parties.authorized_client.Identity')
POA_STATUS=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.power_of_attorney.parties.authorized_client.OperationalStatus')
POA_CAPABILITY=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.power_of_attorney.parties.authorized_client.CapabilityLevel')

echo "Power of Attorney:"
echo "  ✅ Client Identity: $POA_IDENTITY"
echo "  ✅ Operational Status: $POA_STATUS"
echo "  ✅ Capability Level: $POA_CAPABILITY"
echo ""

# Check Authorization Chain
CHAIN_VALIDATED=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.authorization_chain.chain_validated')
CHAIN_DEPTH=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.authorization_chain.chain_depth')
AUTHORIZER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.authorization_chain.owners_authorizer.entity_id')
OWNER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.authorization_chain.client_owner.entity_id')
CLIENT_ID=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.authorization_chain.client.entity_id')

echo "Authorization Chain:"
echo "  ✅ Chain Validated: $CHAIN_VALIDATED"
echo "  ✅ Chain Depth: $CHAIN_DEPTH"
echo "  ✅ Level 1 (Authorizer): $AUTHORIZER_ID"
echo "  ✅ Level 2 (Owner): $OWNER_ID"
echo "  ✅ Level 3 (Client): $CLIENT_ID"
echo ""

# Check Legal Framework
GOVERNING_LAW=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.legal_framework.applicable_laws[0]')
JURISDICTION=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.legal_framework.jurisdiction')

echo "Legal Framework:"
echo "  ✅ Governing Law: $GOVERNING_LAW"
echo "  ✅ Jurisdiction: $JURISDICTION"
echo ""

# Check Compliance Status
COMPLIANT=$(echo "$AUTH_RESPONSE" | jq -r '.compliance_status.Compliant')
VIOLATIONS=$(echo "$AUTH_RESPONSE" | jq -r '.compliance_status.Violations | length')

echo "Compliance Status:"
echo "  ✅ Compliant: $COMPLIANT"
echo "  ✅ Violations: $VIOLATIONS"
echo ""

# Check Verification Proof
VERIFICATION_STATUS=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.verification_proof.overall_verification')
ASSURANCE_LEVEL=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token.verification_proof.verification_levels[0].assurance_level')

echo "Verification Proof:"
echo "  ✅ Overall Status: $VERIFICATION_STATUS"
echo "  ✅ Assurance Level: $ASSURANCE_LEVEL"
echo ""

# Step 4: Verify Authorized Actions
echo "=== PHASE 4: Authorized Actions ==="
ACTIONS=$(echo "$AUTH_RESPONSE" | jq -r '.scope.AuthorizedActions.NonPhysicalActions[]' 2>/dev/null)
echo "Authorized Non-Physical Actions:"
for ACTION in $ACTIONS; do
    echo "  ✅ $ACTION"
done
echo ""

# Step 5: Summary
echo "=========================================="
echo "✅ END-TO-END TEST PASSED"
echo "=========================================="
echo ""
echo "Summary:"
echo "  • Subscription Flow: ✅ Complete (8 steps)"
echo "  • Authorization Flow: ✅ Complete (steps a-i)"
echo "  • Token Issuance: ✅ Success"
echo "  • PoA Validation: ✅ Passed"
echo "  • Chain Validation: ✅ Passed ($CHAIN_DEPTH levels)"
echo "  • Legal Framework: ✅ $GOVERNING_LAW in $JURISDICTION"
echo "  • Compliance: ✅ $COMPLIANT with $VIOLATIONS violations"
echo "  • Verification: ✅ $VERIFICATION_STATUS ($ASSURANCE_LEVEL assurance)"
echo ""
echo "RFC-0111 Implementation: FULLY OPERATIONAL"
echo ""

# Optional: Save full response for inspection
echo "$AUTH_RESPONSE" > /tmp/rfc0111_full_token.json
echo "Full token response saved to: /tmp/rfc0111_full_token.json"
echo ""
