#!/bin/bash
#
# RFC-0111 Quick Demo
# Demonstrates Steps I-III of the subscription flow
#

set -e

API_BASE="http://localhost:8080/api/v1/rfc0111"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}RFC-0111 API Quick Demo${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step I: Create Subscription
echo -e "${BLUE}Step I: Owner's Authorizer Identity Proof${NC}"
RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions" \
  -H "Content-Type: application/json" \
  -d '{
    "owners_authorizer_id": "auth-12345",
    "identity_proof_request": {
      "subject_id": "auth-12345",
      "identity_type": "natural_person",
      "proof_method": "pvp_token",
      "proof_data": {
        "pvp_token": "mock-pvp-token",
        "timestamp": "2025-11-11T10:00:00Z"
      },
      "required_level": "substantial"
    }
  }')

SUB_ID=$(echo "$RESPONSE" | jq -r '.subscription_id')
echo -e "${GREEN}✓${NC} Created subscription: ${GREEN}${SUB_ID}${NC}"
echo "$RESPONSE" | jq
echo ""

# Step II: Authorization Proof
echo -e "${BLUE}Step II: Owner's Authorizer Authorization Proof${NC}"
RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUB_ID}/step-ii" \
  -H "Content-Type: application/json" \
  -d '{
    "commercial_register_ref": "CR-12345-ABC",
    "jurisdiction": "AT"
  }')
echo -e "${GREEN}✓${NC} Step II completed"
echo "$RESPONSE" | jq
echo ""

# Step III: Client Owner Identity
echo -e "${BLUE}Step III: Client Owner Identity Proof${NC}"
RESPONSE=$(curl -s -X POST "${API_BASE}/subscriptions/${SUB_ID}/step-iii" \
  -H "Content-Type: application/json" \
  -d '{
    "subject_id": "client-owner-001",
    "identity_type": "natural_person",
    "proof_method": "pvp_token",
    "proof_data": {
      "pvp_token": "mock-client-owner-token",
      "timestamp": "2025-11-11T10:05:00Z"
    },
    "required_level": "substantial"
  }')
echo -e "${GREEN}✓${NC} Step III completed"
echo "$RESPONSE" | jq
echo ""

# Check Status
echo -e "${BLUE}Final Subscription Status:${NC}"
curl -s "${API_BASE}/subscriptions/${SUB_ID}" | jq
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Demo completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Next steps:"
echo "  - Complete Steps IV-VIII to finish subscription"
echo "  - Use subscription for authorization requests"
echo "  - See AAP-001_API_GUIDE.md for full documentation"
