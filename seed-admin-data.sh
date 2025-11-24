#!/bin/bash

# Seed Admin Handlers with Sample Data
# This script creates sample data for all 5 admin handlers

set -e

TENANT_ID="test-tenant-1"
BASE_URL="http://localhost:8080"

echo "=========================================="
echo "Seeding Admin Handlers with Sample Data"
echo "=========================================="
echo ""

# 1. Create Circuit Breaker (Resilience Handler)
echo "1. Creating Circuit Breaker..."
curl -X POST "${BASE_URL}/api/admin/resilience/circuit-breakers?tenant_id=${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "payment-service-breaker",
    "service": "payment-api",
    "failureThreshold": 5,
    "successThreshold": 2,
    "timeout": 30000
  }' | jq '.' || echo "Circuit breaker creation failed or returned error"
echo ""

# 2. Create Rate Limiter (Resilience Handler)
echo "2. Creating Rate Limiter..."
curl -X POST "${BASE_URL}/api/admin/resilience/rate-limiters?tenant_id=${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-rate-limiter",
    "resource": "/api/v1/*",
    "algorithm": "token-bucket",
    "limit": 100,
    "window": 60,
    "burst": 20
  }' | jq '.' || echo "Rate limiter creation failed or returned error"
echo ""

# 3. Create Retry Policy (Resilience Handler)
echo "3. Creating Retry Policy..."
curl -X POST "${BASE_URL}/api/admin/resilience/retry-policies?tenant_id=${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "database-retry-policy",
    "operation": "database-query",
    "strategy": "exponential",
    "maxAttempts": 3,
    "baseDelay": 1000,
    "maxDelay": 30000,
    "jitter": true
  }' | jq '.' || echo "Retry policy creation failed or returned error"
echo ""

# 4. Create Authorization Policy (Authorization Handler)
echo "4. Creating Authorization Policy..."
curl -X POST "${BASE_URL}/api/admin/authz/policies?tenant_id=${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "admin-access-policy",
    "description": "Full admin access policy",
    "effect": "allow",
    "actions": ["*"],
    "resources": ["*"],
    "priority": 100,
    "enabled": true
  }' | jq '.' || echo "Policy creation failed or returned error"
echo ""

# 5. Create Configuration Variable (Configuration Handler)
echo "5. Creating Configuration Variable..."
curl -X POST "${BASE_URL}/api/admin/config/variables?tenant_id=${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "max_login_attempts",
    "value": "5",
    "type": "integer",
    "sensitive": false,
    "description": "Maximum number of login attempts before lockout"
  }' | jq '.' || echo "Config variable creation failed or returned error"
echo ""

# 6. Create Feature Flag (Configuration Handler)
echo "6. Creating Feature Flag..."
curl -X POST "${BASE_URL}/api/admin/config/feature-flags?tenant_id=${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "enable_2fa",
    "description": "Enable two-factor authentication",
    "enabled": true,
    "type": "boolean",
    "percentage": 100
  }' | jq '.' || echo "Feature flag creation failed or returned error"
echo ""

echo "=========================================="
echo "Sample Data Seeding Complete!"
echo "=========================================="
echo ""
echo "Now testing GET endpoints to verify data..."
echo ""

# Test all endpoints
echo "1. List Circuit Breakers:"
curl -s "${BASE_URL}/api/admin/resilience/circuit-breakers?tenant_id=${TENANT_ID}" | jq '.'
echo ""

echo "2. List Rate Limiters:"
curl -s "${BASE_URL}/api/admin/resilience/rate-limiters?tenant_id=${TENANT_ID}" | jq '.'
echo ""

echo "3. List Retry Policies:"
curl -s "${BASE_URL}/api/admin/resilience/retry-policies?tenant_id=${TENANT_ID}" | jq '.'
echo ""

echo "4. List Authorization Policies:"
curl -s "${BASE_URL}/api/admin/authz/policies?tenant_id=${TENANT_ID}" | jq '.'
echo ""

echo "5. List Configuration Variables:"
curl -s "${BASE_URL}/api/admin/config/variables?tenant_id=${TENANT_ID}" | jq '.'
echo ""

echo "6. List Feature Flags:"
curl -s "${BASE_URL}/api/admin/config/feature-flags?tenant_id=${TENANT_ID}" | jq '.'
echo ""

echo "=========================================="
echo "All handlers tested with sample data!"
echo "=========================================="
