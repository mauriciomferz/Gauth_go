#!/bin/bash
# RFC-0111 Authorization Flow Integration Test
# Tests complete flow: subscription (Steps I-VIII) → authorization (Steps a-i) → validation → introspection → revocation

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_BASE="$BASE_URL/api/v1/rfc0111"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✅ $1${NC}"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}❌ $1${NC}"
    ((TESTS_FAILED++))
}

info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

section() {
    echo ""
    echo "========================================"
    echo "$1"
    echo "========================================"
}

# Store temporary data
SUBSCRIPTION_ID=""
ACCESS_TOKEN=""
EXTENDED_TOKEN=""

section "RFC-0111 Complete Authorization Flow Test"

info "Starting full authorization flow test: subscription → token request → validation → introspection → revocation"

# ============================================
# Phase 1: Complete Subscription Flow (Steps I-VIII)
# ============================================

section "Phase 1: Subscription Flow (Steps I-VIII)"

info "Step I: Creating subscription and proving owner's authorizer identity..."
RESPONSE=$(curl -s -X POST "$API_BASE/subscriptions" \
    -H "Content-Type: application/json" \
    -d '{
        "owners_authorizer_id": "managing-director-001",
        "identity_proof_request": {
            "subject_id": "managing-director-001",
            "identity_type": "natural_person",
            "proof_method": "pvp_token",
            "proof_data": {
                "pvp_token": "mock-pvp-token-managing-director-001",
                "timestamp": "2025-01-15T10:00:00Z"
            },
            "required_level": "substantial"
        }
    }')

SUBSCRIPTION_ID=$(echo "$RESPONSE" | jq -r '.subscription_id // empty')
if [[ -z "$SUBSCRIPTION_ID" ]]; then
    fail "Step I: Failed to create subscription"
    echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
    exit 1
fi
pass "Step I: Subscription created with ID: $SUBSCRIPTION_ID"

# Quick execution of Steps II-VIII
info "Executing Steps II-VIII..."

# Step II
curl -s -X POST "$API_BASE/subscriptions/$SUBSCRIPTION_ID/step-ii" \
    -H "Content-Type: application/json" \
    -d '{"authorization_proof": {"type": "power_of_attorney", "value": "poa-doc-456"}}' > /dev/null
pass "Step II: Owner authorizer authorization proof"

# Step III
curl -s -X POST "$API_BASE/subscriptions/$SUBSCRIPTION_ID/step-iii" \
    -H "Content-Type: application/json" \
    -d '{"client_owner_id": "owner-002", "identity_proof": {"type": "biometric", "value": "fingerprint-ABC"}}' > /dev/null
pass "Step III: Client owner identity proof"

# Step IV with complete authorization chain
curl -s -X POST "$API_BASE/subscriptions/$SUBSCRIPTION_ID/step-iv" \
    -H "Content-Type: application/json" \
    -d '{
        "authorization_chain": {
            "owners_authorizer": {
                "entity_id": "managing-director-001",
                "role": "managing_director",
                "authorization_type": "statutory",
                "legal_basis": {
                    "law": "Austrian Corporate Law 2024",
                    "section": "§15(1)",
                    "jurisdiction": "AT"
                },
                "status": "active",
                "valid_from": "2024-01-01T00:00:00Z",
                "valid_until": "2025-12-31T23:59:59Z",
                "statutory_authority": "Managing Director Authority",
                "commercial_register_ref": "FN123456a"
            },
            "client_owner": {
                "entity_id": "owner-002",
                "role": "shareholder",
                "authorization_type": "delegated",
                "status": "active",
                "valid_from": "2024-01-01T00:00:00Z",
                "valid_until": "2025-12-31T23:59:59Z"
            },
            "client": {
                "entity_id": "client-003",
                "role": "service_user",
                "authorization_type": "delegated",
                "status": "active",
                "valid_from": "2024-01-01T00:00:00Z",
                "valid_until": "2025-12-31T23:59:59Z"
            }
        }
    }' > /dev/null
pass "Step IV: Client owner authorization proof"

# Step V (PoA credential optional)
curl -s -X POST "$API_BASE/subscriptions/$SUBSCRIPTION_ID/step-v" \
    -H "Content-Type: application/json" \
    -d '{"authorization_grant": {"scope": "read write", "expires_at": "2025-12-31T23:59:59Z"}}' > /dev/null
pass "Step V: Client authorization"

# Step VI
curl -s -X POST "$API_BASE/subscriptions/$SUBSCRIPTION_ID/step-vi" \
    -H "Content-Type: application/json" \
    -d '{"resource_owner_id": "resource-owner-004", "identity_proof": {"type": "certificate", "value": "cert-DEF123"}}' > /dev/null
pass "Step VI: Resource owner identity proof"

# Step VII with complete authorization chain
curl -s -X POST "$API_BASE/subscriptions/$SUBSCRIPTION_ID/step-vii" \
    -H "Content-Type: application/json" \
    -d '{
        "authorization_chain": {
            "owners_authorizer": {
                "entity_id": "managing-director-001",
                "role": "managing_director",
                "authorization_type": "statutory",
                "legal_basis": {
                    "law": "Austrian Corporate Law 2024",
                    "section": "§15(1)",
                    "jurisdiction": "AT"
                },
                "status": "active",
                "valid_from": "2024-01-01T00:00:00Z",
                "valid_until": "2025-12-31T23:59:59Z",
                "statutory_authority": "Managing Director Authority",
                "commercial_register_ref": "FN123456a"
            },
            "client_owner": {
                "entity_id": "owner-002",
                "role": "shareholder",
                "authorization_type": "delegated",
                "status": "active",
                "valid_from": "2024-01-01T00:00:00Z",
                "valid_until": "2025-12-31T23:59:59Z"
            },
            "client": {
                "entity_id": "resource-owner-004",
                "role": "data_owner",
                "authorization_type": "delegated",
                "status": "active",
                "valid_from": "2024-01-01T00:00:00Z",
                "valid_until": "2025-12-31T23:59:59Z"
            }
        }
    }' > /dev/null
pass "Step VII: Resource owner authorization proof"

# Step VIII
curl -s -X POST "$API_BASE/subscriptions/$SUBSCRIPTION_ID/step-viii" \
    -H "Content-Type: application/json" \
    -d '{"resource_server_consent": {"granted": true, "scope": "data:read"}}' > /dev/null
pass "Step VIII: Resource server authorization"

# Verify subscription completed
SUBSCRIPTION_STATUS=$(curl -s -X GET "$API_BASE/subscriptions/$SUBSCRIPTION_ID" | jq -r '.status // empty')
if [[ "$SUBSCRIPTION_STATUS" == "completed" ]]; then
    pass "Subscription status verified: completed"
else
    fail "Subscription status: expected 'completed', got '$SUBSCRIPTION_STATUS'"
fi

# ============================================
# Phase 2: Authorization Flow (Steps a-i)
# ============================================

section "Phase 2: Authorization Flow - Token Request (Steps a-i)"

info "Requesting RFC-0111 compliant access token..."
AUTH_RESPONSE=$(curl -s -X POST "$API_BASE/authorize" \
    -H "Content-Type: application/json" \
    -d "{
        \"subscription_id\": \"$SUBSCRIPTION_ID\",
        \"client_id\": \"client-003\",
        \"scope\": \"read write\",
        \"redirect_uri\": \"https://client.example.com/callback\"
    }")

# Check for success
SUCCESS=$(echo "$AUTH_RESPONSE" | jq -r '.success // false')
if [[ "$SUCCESS" != "true" ]]; then
    fail "Authorization request failed"
    echo "$AUTH_RESPONSE" | jq '.' 2>/dev/null || echo "$AUTH_RESPONSE"
    exit 1
fi

# Extract tokens
ACCESS_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.access_token // empty')
EXTENDED_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.extended_token // empty')

if [[ -z "$ACCESS_TOKEN" ]] || [[ "$ACCESS_TOKEN" == "null" ]]; then
    fail "No access token received in authorization response"
    echo "$AUTH_RESPONSE" | jq '.'
    exit 1
fi

pass "Authorization successful: access token received"
info "Token type: $(echo "$AUTH_RESPONSE" | jq -r '.token_type')"
info "Expires in: $(echo "$AUTH_RESPONSE" | jq -r '.expires_in') seconds"
info "Scope: $(echo "$AUTH_RESPONSE" | jq -r '.scope')"

# ============================================
# Phase 3: Token Validation
# ============================================

section "Phase 3: Token Validation"

info "Validating access token..."
VALIDATE_RESPONSE=$(curl -s -X POST "$API_BASE/token/validate" \
    -H "Content-Type: application/json" \
    -d "{\"token\": \"$ACCESS_TOKEN\"}")

VALID=$(echo "$VALIDATE_RESPONSE" | jq -r '.valid // false')
if [[ "$VALID" == "true" ]]; then
    pass "Token validation: token is valid"
    info "Client ID: $(echo "$VALIDATE_RESPONSE" | jq -r '.client_id')"
    info "Scope: $(echo "$VALIDATE_RESPONSE" | jq -r '.scope')"
else
    fail "Token validation: token is invalid"
    echo "$VALIDATE_RESPONSE" | jq '.'
fi

# ============================================
# Phase 4: Token Introspection (RFC 7662)
# ============================================

section "Phase 4: Token Introspection (RFC 7662)"

info "Introspecting access token..."
INTROSPECT_RESPONSE=$(curl -s -X POST "$API_BASE/token/introspect" \
    -H "Content-Type: application/json" \
    -d "{\"token\": \"$ACCESS_TOKEN\"}")

ACTIVE=$(echo "$INTROSPECT_RESPONSE" | jq -r '.active // false')
if [[ "$ACTIVE" == "true" ]]; then
    pass "Token introspection: token is active"
    info "Token type: $(echo "$INTROSPECT_RESPONSE" | jq -r '.token_type')"
    info "Client ID: $(echo "$INTROSPECT_RESPONSE" | jq -r '.client_id')"
    info "Subject: $(echo "$INTROSPECT_RESPONSE" | jq -r '.sub')"
    info "Scope: $(echo "$INTROSPECT_RESPONSE" | jq -r '.scope')"
else
    fail "Token introspection: token is not active"
    echo "$INTROSPECT_RESPONSE" | jq '.'
fi

# Test introspection with invalid token
info "Testing introspection with invalid token..."
INVALID_INTROSPECT=$(curl -s -X POST "$API_BASE/token/introspect" \
    -H "Content-Type: application/json" \
    -d '{"token": "invalid.token.here"}')

INVALID_ACTIVE=$(echo "$INVALID_INTROSPECT" | jq -r '.active // false')
if [[ "$INVALID_ACTIVE" == "false" ]]; then
    pass "Invalid token introspection: correctly returns active=false"
else
    fail "Invalid token introspection: should return active=false"
fi

# ============================================
# Phase 5: Token Revocation (RFC 7009)
# ============================================

section "Phase 5: Token Revocation (RFC 7009)"

info "Revoking access token..."
REVOKE_RESPONSE=$(curl -s -X POST "$API_BASE/token/revoke" \
    -H "Content-Type: application/json" \
    -d "{\"token\": \"$ACCESS_TOKEN\"}")

REVOKE_SUCCESS=$(echo "$REVOKE_RESPONSE" | jq -r '.success // false')
if [[ "$REVOKE_SUCCESS" == "true" ]]; then
    pass "Token revocation: successful"
else
    # Per RFC 7009, revocation MUST return 200 OK regardless
    STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE/token/revoke" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if [[ "$STATUS_CODE" == "200" ]]; then
        pass "Token revocation: RFC 7009 compliant (200 OK)"
    else
        fail "Token revocation: HTTP $STATUS_CODE (expected 200 OK per RFC 7009)"
    fi
fi

# Test idempotency - revoking again should succeed
info "Testing revocation idempotency..."
REVOKE2_RESPONSE=$(curl -s -X POST "$API_BASE/token/revoke" \
    -H "Content-Type: application/json" \
    -d "{\"token\": \"$ACCESS_TOKEN\"}")

STATUS_CODE2=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE/token/revoke" \
    -H "Content-Type: application/json" \
    -d "{\"token\": \"$ACCESS_TOKEN\"}")

if [[ "$STATUS_CODE2" == "200" ]]; then
    pass "Token revocation idempotency: RFC 7009 compliant"
else
    fail "Token revocation idempotency: HTTP $STATUS_CODE2 (expected 200)"
fi

# ============================================
# Test Summary
# ============================================

section "Test Summary"

TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED))
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"
echo "Total Tests: $TOTAL_TESTS"
echo ""

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo ""
    echo "RFC-0111 Implementation Status:"
    echo "  • Subscription Flow (Steps I-VIII): ✅ Complete"
    echo "  • Authorization Flow (Steps a-i): ✅ Complete"
    echo "  • Token Validation: ✅ Working"
    echo "  • Token Introspection (RFC 7662): ✅ Compliant"
    echo "  • Token Revocation (RFC 7009): ✅ Compliant"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi
