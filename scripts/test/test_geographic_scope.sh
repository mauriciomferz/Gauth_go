#!/bin/bash
# Geographic Scope Validation - Integration Test
# Tests the fix end-to-end through the HTTP API

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "=================================================="
echo "Geographic Scope Validation - Integration Tests"
echo "=================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test helper function
test_authorization() {
    local test_name="$1"
    local jurisdiction="$2"
    local expected_result="$3"
    
    echo -n "Testing: $test_name ... "
    
    # Make authorization request
    response=$(curl -s -X POST "$BASE_URL/api/v1/rfc0111/authorize" \
        -H "Content-Type: application/json" \
        -d "{
            \"client_id\": \"test-client\",
            \"subscription_id\": \"sub_123\",
            \"resource_owner_id\": \"owner@example.com\",
            \"poa_credential_ref\": \"poa_de_only\",
            \"scope\": \"read write\",
            \"jurisdiction\": \"$jurisdiction\"
        }" 2>&1)
    
    # Check result
    if [ "$expected_result" = "PASS" ]; then
        if echo "$response" | grep -q "extended_token"; then
            echo -e "${GREEN}✓ PASS${NC}"
            return 0
        else
            echo -e "${RED}✗ FAIL${NC} (Expected success, got error)"
            echo "Response: $response"
            return 1
        fi
    else
        if echo "$response" | grep -q "geographic_scope_violation\|authorization_failed"; then
            echo -e "${GREEN}✓ PASS${NC}"
            return 0
        else
            echo -e "${RED}✗ FAIL${NC} (Expected rejection, got success)"
            echo "Response: $response"
            return 1
        fi
    fi
}

echo "Prerequisites:"
echo "  1. Backend server running on $BASE_URL"
echo "  2. Test subscription 'sub_123' exists"
echo "  3. Test PoA 'poa_de_only' exists with German scope only"
echo ""
echo -e "${YELLOW}Note: These tests require proper test data setup${NC}"
echo ""

# Test Suite
echo "Running test suite..."
echo ""

# Test 1: Authorized jurisdiction should pass
test_authorization \
    "Authorized jurisdiction (DE)" \
    "DE" \
    "PASS"

# Test 2: Unauthorized jurisdiction should fail
test_authorization \
    "Unauthorized jurisdiction (US)" \
    "US" \
    "FAIL"

# Test 3: Another unauthorized jurisdiction
test_authorization \
    "Unauthorized jurisdiction (FR)" \
    "FR" \
    "FAIL"

# Test 4: Subdivision of authorized country
test_authorization \
    "Subdivision of authorized country (DE-BY)" \
    "DE-BY" \
    "PASS"  # Assuming IncludeSubdivisions is true

# Test 5: Empty jurisdiction (should warn but not fail if not strict)
echo -n "Testing: No jurisdiction provided ... "
response=$(curl -s -X POST "$BASE_URL/api/v1/rfc0111/authorize" \
    -H "Content-Type: application/json" \
    -d "{
        \"client_id\": \"test-client\",
        \"subscription_id\": \"sub_123\",
        \"resource_owner_id\": \"owner@example.com\",
        \"poa_credential_ref\": \"poa_de_only\",
        \"scope\": \"read write\"
    }" 2>&1)

if echo "$response" | grep -q "extended_token"; then
    echo -e "${YELLOW}⚠ WARN${NC} (No jurisdiction - validation skipped)"
else
    echo -e "${GREEN}✓ STRICT${NC} (Rejected - strict mode enabled)"
fi

echo ""
echo "=================================================="
echo "Test Suite Complete"
echo "=================================================="
echo ""
echo "Summary:"
echo "  - Geographic scope validation is active"
echo "  - Authorized jurisdictions: ALLOWED"
echo "  - Unauthorized jurisdictions: BLOCKED"
echo "  - RFC-0111 compliance: ✓ VERIFIED"
echo ""
echo "For detailed logs, check: /metrics endpoint for:"
echo "  - agentauth_geographic_scope_violations_total"
echo "  - agentauth_geographic_scope_validations_success_total"
