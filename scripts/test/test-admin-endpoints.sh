#!/bin/bash
# Test script for admin endpoints

echo "========================================="
echo "Testing Admin Handler Endpoints"
echo "========================================="
echo ""

TENANT_ID="test-tenant-1"
BASE_URL="http://localhost:8080/api/admin"

echo "1. Testing Power of Attorney - List PoAs"
curl -s "${BASE_URL}/poa?tenant_id=${TENANT_ID}" | jq .
echo ""

echo "2. Testing Resilience - List Circuit Breakers"
curl -s "${BASE_URL}/resilience/circuit-breakers?tenant_id=${TENANT_ID}" | jq .
echo ""

echo "3. Testing Events - List Events"
curl -s "${BASE_URL}/events?tenant_id=${TENANT_ID}" | jq .
echo ""

echo "4. Testing Authorization - List Policies"
curl -s "${BASE_URL}/authz/policies?tenant_id=${TENANT_ID}" | jq .
echo ""

echo "5. Testing Configuration - List Variables"
curl -s "${BASE_URL}/config/variables?tenant_id=${TENANT_ID}" | jq .
echo ""

echo "========================================="
echo "Admin Endpoint Tests Complete"
echo "========================================="
