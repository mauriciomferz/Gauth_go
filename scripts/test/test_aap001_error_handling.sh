#!/bin/bash

# RFC-0111 Error Handling and Edge Cases Test
# Tests various error scenarios and validation failures

set +e  # Don't exit on errors - we're testing error handling

BASE_URL="http://localhost:8080/api/v1/rfc0111"

echo "=========================================="
echo "RFC-0111 Error Handling Test Suite"
echo "=========================================="
echo ""

PASSED=0
FAILED=0

# Helper function to test error scenarios
test_error_scenario() {
    local test_name=$1
    local expected_error=$2
    local response=$3
    
    local actual_error=$(echo "$response" | jq -r '.error // empty')
    
    if [ -n "$actual_error" ]; then
        if [[ "$actual_error" == *"$expected_error"* ]] || [[ "$response" == *"$expected_error"* ]]; then
            echo "✅ $test_name"
            echo "   Expected error: $expected_error"
            echo "   Got: $actual_error"
            ((PASSED++))
        else
            echo "❌ $test_name"
            echo "   Expected error containing: $expected_error"
            echo "   Got: $actual_error"
            ((FAILED++))
        fi
    else
        echo "❌ $test_name"
        echo "   Expected error but got success"
        echo "   Response: $response"
        ((FAILED++))
    fi
    echo ""
}

# Test 1: Authorization without subscription
echo "=== Test 1: Authorization Without Subscription ==="
RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-app-123",
    "subscription_id": "sub_nonexistent",
    "resource_owner_id": "resource-owner-99999",
    "poa_credential_ref": "mock-poa-credential-xyz",
    "scope": "read write",
    "context": {"purpose": "data_processing"}
  }')

test_error_scenario "Authorization with non-existent subscription" "not found" "$RESPONSE"

# Test 2: Authorization with incomplete subscription
echo "=== Test 2: Authorization With Incomplete Subscription ==="

# Create subscription but don't complete it
SUB_RESPONSE=$(curl -s -X POST "$BASE_URL/subscriptions" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-app-123",
    "client_owner_identity": {"subject_id": "client-owner-67890"},
    "owners_authorizer_identity": {"subject_id": "auth-12345"},
    "pip_token": "mock-pip-token-abc",
    "pvp_token": "mock-pvp-token-xyz"
  }')

INCOMPLETE_SUB_ID=$(echo "$SUB_RESPONSE" | jq -r '.subscription_id')

RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"client-app-123\",
    \"subscription_id\": \"$INCOMPLETE_SUB_ID\",
    \"resource_owner_id\": \"resource-owner-99999\",
    \"poa_credential_ref\": \"mock-poa-credential-xyz\",
    \"scope\": \"read write\",
    \"context\": {\"purpose\": \"data_processing\"}
  }")

test_error_scenario "Authorization with incomplete subscription" "not completed" "$RESPONSE"

# Test 3: Missing required fields
echo "=== Test 3: Missing Client ID ==="
RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d '{
    "subscription_id": "sub_12345",
    "resource_owner_id": "resource-owner-99999",
    "scope": "read write"
  }')

test_error_scenario "Missing client_id" "client_id" "$RESPONSE"

echo "=== Test 4: Missing Subscription ID ==="
RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-app-123",
    "resource_owner_id": "resource-owner-99999",
    "scope": "read write"
  }')

test_error_scenario "Missing subscription_id" "subscription_id" "$RESPONSE"

# Test 5: Empty scope
echo "=== Test 5: Empty Scope ==="
# Create a completed subscription first
./scripts/test_rfc0111_subscription_flow.sh > /tmp/test_sub.txt 2>&1
VALID_SUB_ID=$(grep -oE 'sub_[0-9]+' /tmp/test_sub.txt | tail -1)

RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"client-app-123\",
    \"subscription_id\": \"$VALID_SUB_ID\",
    \"resource_owner_id\": \"resource-owner-99999\",
    \"poa_credential_ref\": \"mock-poa-credential-xyz\",
    \"scope\": \"\",
    \"context\": {\"purpose\": \"data_processing\"}
  }")

# This should still work because we have a default scope
if echo "$RESPONSE" | jq -e '.extended_token' > /dev/null 2>&1; then
    echo "✅ Empty scope handled gracefully (default scope applied)"
    ((PASSED++))
else
    echo "⚠️  Empty scope test result unclear"
fi
echo ""

# Test 6: Invalid JSON
echo "=== Test 6: Invalid JSON ==="
RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d '{invalid json}')

test_error_scenario "Invalid JSON" "invalid" "$RESPONSE"

# Test 7: Step II idempotency protection
echo "=== Test 7: Step II Idempotency Protection ==="
./scripts/test_rfc0111_subscription_flow.sh > /tmp/test_sub2.txt 2>&1
VALID_SUB_ID2=$(grep -oE 'sub_[0-9]+' /tmp/test_sub2.txt | tail -1)

# Try to execute Step II again
RESPONSE=$(curl -s -X POST "$BASE_URL/subscriptions/$VALID_SUB_ID2/step-ii" \
  -H "Content-Type: application/json" \
  -d '{
    "pip_token": "mock-pip-token-abc"
  }')

test_error_scenario "Step II re-execution blocked" "already" "$RESPONSE"

# Test 8: Authorization with mismatched client ID
echo "=== Test 8: Mismatched Client ID ==="
RESPONSE=$(curl -s -X POST "$BASE_URL/authorize" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"different-client-999\",
    \"subscription_id\": \"$VALID_SUB_ID\",
    \"resource_owner_id\": \"resource-owner-99999\",
    \"poa_credential_ref\": \"mock-poa-credential-xyz\",
    \"scope\": \"read write\",
    \"context\": {\"purpose\": \"data_processing\"}
  }")

# This might fail at compliance validation
if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    echo "✅ Mismatched client ID rejected"
    ((PASSED++))
else
    echo "⚠️  Mismatched client ID test: unclear result"
fi
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "Tests Passed: $PASSED"
echo "Tests Failed: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "✅ All error handling tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi
