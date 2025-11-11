#!/bin/bash
#
# RFC-0111 Subscription Flow Integration Test
# Tests the complete 8-step subscription process via HTTP API
#
# Usage: ./scripts/test_rfc0111_subscription_flow.sh [base_url]
# Default base_url: http://localhost:8080

# set -e  # Removed to allow better error reporting

BASE_URL="${1:-http://localhost:8080}"
API_BASE="${BASE_URL}/api/v1/rfc0111"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}RFC-0111 Subscription Flow Test${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Base URL: ${BASE_URL}"
echo -e "API Base: ${API_BASE}"
echo ""

# Helper functions
function print_step() {
    echo -e "${YELLOW}[Step $1]${NC} $2"
}

function print_success() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

function print_error() {
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

function check_response() {
    local response="$1"
    local expected_field="$2"
    local step_name="$3"
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        # Fallback: check if response contains the field name
        if echo "$response" | grep -q "$(echo "$expected_field" | tr -d '.')" 2>/dev/null; then
            print_success "$step_name completed successfully"
            return 0
        fi
    elif echo "$response" | jq -e "$expected_field" > /dev/null 2>&1; then
        print_success "$step_name completed successfully"
        return 0
    fi
    
    print_error "$step_name failed"
    echo "Response: $response"
    return 1
}

# Test server connectivity
echo -e "${BLUE}Checking server availability...${NC}"
if curl -s -o /dev/null -w "%{http_code}" "$BASE_URL" | grep -q "200\|404"; then
    print_success "Server is responding"
else
    print_error "Server is not responding at $BASE_URL"
    exit 1
fi
echo ""

# =============================================================================
# STEP I: Initiate Subscription (Owner's Authorizer Identity Proof)
# =============================================================================
print_step "I" "Initiate Subscription (Owner's Authorizer Identity Proof)"

STEP_I_REQUEST='{
  "owners_authorizer_id": "auth-12345",
  "identity_proof_request": {
    "subject_id": "auth-12345",
    "identity_type": "natural_person",
    "proof_method": "pvp_token",
    "proof_data": {
      "pvp_token": "mock-pvp-token-auth-12345",
      "timestamp": "2025-01-15T10:00:00Z"
    },
    "required_level": "substantial"
  }
}'

STEP_I_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions" \
  -H "Content-Type: application/json" \
  -d "$STEP_I_REQUEST")

echo "Step I Response: $STEP_I_RESPONSE" >&2

if echo "$STEP_I_RESPONSE" | jq -e '.subscription_id' > /dev/null 2>&1; then
    print_success "Step I completed successfully"
    SUBSCRIPTION_ID=$(echo "$STEP_I_RESPONSE" | jq -r '.subscription_id')
    echo -e "  Subscription ID: ${GREEN}$SUBSCRIPTION_ID${NC}"
else
    print_error "Step I failed"
    echo "Response: $STEP_I_RESPONSE"
    exit 1
fi
echo ""

# =============================================================================
# STEP II: Owner's Authorizer Authorization Proof
# =============================================================================
print_step "II" "Owner's Authorizer Authorization Proof (Commercial Register)"

STEP_II_REQUEST='{
  "commercial_register_ref": "CR-12345-ABC",
  "jurisdiction": "AT"
}'

STEP_II_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}/step-ii" \
  -H "Content-Type: application/json" \
  -d "$STEP_II_REQUEST")

check_response "$STEP_II_RESPONSE" '.step' "Step II"
echo ""

# =============================================================================
# STEP III: Client Owner Identity Proof
# =============================================================================
print_step "III" "Client Owner Identity Proof (via PVP)"

STEP_III_REQUEST='{
  "subject_id": "client-owner-67890",
  "identity_type": "natural_person",
  "proof_method": "pvp_token",
  "proof_data": {
    "pvp_token": "mock-pvp-token-client-owner-67890",
    "timestamp": "2025-01-15T10:05:00Z"
  },
  "required_level": "substantial"
}'

STEP_III_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}/step-iii" \
  -H "Content-Type: application/json" \
  -d "$STEP_III_REQUEST")

check_response "$STEP_III_RESPONSE" '.step' "Step III"
echo ""

# =============================================================================
# STEP IV: Client Owner Authorization Proof
# =============================================================================
print_step "IV" "Client Owner Authorization Proof (Authorization Chain)"

STEP_IV_REQUEST='{
  "authorization_chain": {
    "owners_authorizer": {
      "entity_id": "auth-12345",
      "entity_type": "natural_person",
      "entity_name": "Board Member",
      "role": "authorizer",
      "authorization_type": "statutory",
      "authorization_document": "commercial-register-12345",
      "authorization_date": "2025-01-01T00:00:00Z",
      "legal_basis": {
        "basis_type": "company_law",
        "jurisdiction": "AT",
        "legal_references": ["§78 AktG", "§15 GmbHG"],
        "registration_number": "FN123456a",
        "issuing_authority": "Handelsgericht Wien"
      },
      "identity_verified": true,
      "verification_method": "commercial_register",
      "scope_of_authority": ["manage_authorizations", "delegate_authority"],
      "valid_from": "2025-01-01T00:00:00Z",
      "valid_until": "2025-12-31T23:59:59Z",
      "status": "active",
      "statutory_authority": "Managing Director per §78 Austrian Stock Corporation Act",
      "commercial_register_ref": "FN123456a"
    },
    "client_owner": {
      "entity_id": "client-owner-67890",
      "entity_type": "natural_person",
      "entity_name": "AI System Owner",
      "role": "owner",
      "authorized_by": "auth-12345",
      "authorization_type": "delegated",
      "authorization_document": "poa-doc-789",
      "authorization_date": "2025-01-01T00:00:00Z",
      "identity_verified": true,
      "verification_method": "pip_token",
      "scope_of_authority": ["operate_ai_system", "manage_client"],
      "valid_from": "2025-01-01T00:00:00Z",
      "valid_until": "2025-12-31T23:59:59Z",
      "status": "active"
    },
    "client": {
      "entity_id": "client-app-123",
      "entity_type": "ai_system",
      "entity_name": "AI Client Application",
      "role": "client",
      "authorized_by": "client-owner-67890",
      "authorization_type": "delegated",
      "authorization_document": "client-registration-123",
      "authorization_date": "2025-01-10T00:00:00Z",
      "identity_verified": true,
      "verification_method": "client_credentials",
      "scope_of_authority": ["access_resources", "request_tokens"],
      "valid_from": "2025-01-10T00:00:00Z",
      "valid_until": "2025-12-31T23:59:59Z",
      "status": "active"
    },
    "chain_validated": true,
    "chain_depth": 3
  }
}'

STEP_IV_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}/step-iv" \
  -H "Content-Type: application/json" \
  -d "$STEP_IV_REQUEST")

check_response "$STEP_IV_RESPONSE" '.step' "Step IV"
echo ""

# =============================================================================
# STEP V: Client Authorization
# =============================================================================
print_step "V" "Client Authorization (with PoA Credential)"

STEP_V_REQUEST='{
  "client_id": "client-app-123",
  "poa_credential": "mock-poa-credential-xyz",
  "enable_identity_sharing": true,
  "enable_prompting": false
}'

STEP_V_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}/step-v" \
  -H "Content-Type: application/json" \
  -d "$STEP_V_REQUEST")

check_response "$STEP_V_RESPONSE" '.step' "Step V"
echo ""

# =============================================================================
# STEP VI: Resource Owner Identity Proof
# =============================================================================
print_step "VI" "Resource Owner Identity Proof (via PVP)"

STEP_VI_REQUEST='{
  "subject_id": "resource-owner-99999",
  "identity_type": "natural_person",
  "proof_method": "pvp_token",
  "proof_data": {
    "pvp_token": "mock-pvp-token-resource-owner-99999",
    "timestamp": "2025-01-15T10:10:00Z"
  },
  "required_level": "substantial"
}'

STEP_VI_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}/step-vi" \
  -H "Content-Type: application/json" \
  -d "$STEP_VI_REQUEST")

check_response "$STEP_VI_RESPONSE" '.step' "Step VI"
echo ""

# =============================================================================
# STEP VII: Resource Owner Authorization Proof
# =============================================================================
print_step "VII" "Resource Owner Authorization Proof (Authorization Chain)"

STEP_VII_REQUEST='{
  "authorization_chain": {
    "owners_authorizer": {
      "entity_id": "auth-12345",
      "entity_type": "natural_person",
      "entity_name": "Board Member",
      "role": "authorizer",
      "authorization_type": "statutory",
      "authorization_document": "commercial-register-12345",
      "authorization_date": "2025-01-01T00:00:00Z",
      "legal_basis": {
        "basis_type": "company_law",
        "jurisdiction": "AT",
        "legal_references": ["§78 AktG", "§15 GmbHG"],
        "registration_number": "FN123456a",
        "issuing_authority": "Handelsgericht Wien"
      },
      "identity_verified": true,
      "verification_method": "commercial_register",
      "scope_of_authority": ["manage_authorizations", "delegate_authority"],
      "valid_from": "2025-01-01T00:00:00Z",
      "valid_until": "2025-12-31T23:59:59Z",
      "status": "active",
      "statutory_authority": "Managing Director per §78 Austrian Stock Corporation Act",
      "commercial_register_ref": "FN123456a"
    },
    "client_owner": {
      "entity_id": "client-owner-67890",
      "entity_type": "natural_person",
      "entity_name": "AI System Owner",
      "role": "owner",
      "authorized_by": "auth-12345",
      "authorization_type": "delegated",
      "authorization_document": "poa-doc-789",
      "authorization_date": "2025-01-01T00:00:00Z",
      "identity_verified": true,
      "verification_method": "pip_token",
      "scope_of_authority": ["operate_ai_system", "manage_client", "manage_resources"],
      "valid_from": "2025-01-01T00:00:00Z",
      "valid_until": "2025-12-31T23:59:59Z",
      "status": "active"
    },
    "client": {
      "entity_id": "client-app-123",
      "entity_type": "ai_system",
      "entity_name": "AI Client Application",
      "role": "client",
      "authorized_by": "client-owner-67890",
      "authorization_type": "delegated",
      "authorization_document": "client-registration-123",
      "authorization_date": "2025-01-10T00:00:00Z",
      "identity_verified": true,
      "verification_method": "client_credentials",
      "scope_of_authority": ["access_resources", "request_tokens"],
      "valid_from": "2025-01-10T00:00:00Z",
      "valid_until": "2025-12-31T23:59:59Z",
      "status": "active"
    },
    "chain_validated": true,
    "chain_depth": 3
  }
}'

STEP_VII_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}/step-vii" \
  -H "Content-Type: application/json" \
  -d "$STEP_VII_REQUEST")

check_response "$STEP_VII_RESPONSE" '.step' "Step VII"
echo ""

# =============================================================================
# STEP VIII: Resource Server Authorization
# =============================================================================
print_step "VIII" "Resource Server Authorization (Complete Subscription)"

STEP_VIII_REQUEST='{
  "resource_server_id": "rs-api-server-001",
  "server_endpoint": "https://api.example.com/resources",
  "resource_types": ["document", "file", "data"],
  "allowed_operations": ["read", "write"]
}'

STEP_VIII_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}/step-viii" \
  -H "Content-Type: application/json" \
  -d "$STEP_VIII_REQUEST")

check_response "$STEP_VIII_RESPONSE" '.step' "Step VIII"
echo ""

# =============================================================================
# Verify Final Subscription State
# =============================================================================
echo -e "${BLUE}Verifying final subscription state...${NC}"

SUBSCRIPTION_STATE=$(curl -s "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}")

if echo "$SUBSCRIPTION_STATE" | jq -e '.status == "completed"' > /dev/null 2>&1; then
    print_success "Subscription is completed"
    echo -e "  Status: ${GREEN}$(echo "$SUBSCRIPTION_STATE" | jq -r '.status')${NC}"
    echo -e "  Subscription ID: ${GREEN}$(echo "$SUBSCRIPTION_STATE" | jq -r '.subscription_id')${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "Subscription is not completed"
    echo "Subscription state: $SUBSCRIPTION_STATE"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
echo ""

# =============================================================================
# Test Error Handling: Try to execute Step II again (should fail - prerequisite)
# =============================================================================
echo -e "${BLUE}Testing error handling...${NC}"
print_step "Error Test" "Attempting to re-execute Step II (should fail)"

ERROR_TEST_RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUBSCRIPTION_ID}/step-ii" \
  -H "Content-Type: application/json" \
  -d "$STEP_II_REQUEST")

if echo "$ERROR_TEST_RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    print_success "Error handling works correctly"
    echo -e "  Error: ${YELLOW}$(echo "$ERROR_TEST_RESPONSE" | jq -r '.error')${NC}"
else
    print_error "Expected error response, got success"
fi
echo ""

# =============================================================================
# Test Summary
# =============================================================================
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed ✗${NC}"
    exit 1
fi
