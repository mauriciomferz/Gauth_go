---
title: Gauthplus Api Quick Start
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth+ API Quick Start Guide

**Date:** November 26, 2025  
**Status:** Production Ready  
**Base URL:** `http://localhost:8080/api/v1/gauthplus`

## Starting the Server

```bash
# Start PostgreSQL database
docker-compose -f docker-compose.database.yml up -d

# Start GAuth+ enabled server
GAUTH_DEV_INDEX=1 \
GAUTH_RFC0111_ENABLED=1 \
GAUTH_USE_JWT_LIB=1 \
GAUTH_GAUTHPLUS_ENABLED=1 \
DB_HOST=localhost \
DB_PORT=5432 \
DB_USER=gauth_app \
DB_PASSWORD=change_me_in_production \
DB_NAME=gauth \
DB_SSLMODE=disable \
GAUTH_JWT_SIGNING_KEY=dev-secret \
go run ./cmd/web-server
```

## API Endpoints Reference

### 1. Successor Management (4 endpoints)

#### Activate Successor
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/successors/activate \
  -H "Content-Type: application/json" \
  -d '{
    "poa_id": "00000000-0000-0000-0000-000000000001",
    "successor_agent_id": "successor-ai-001",
    "reason": "Primary agent unavailable",
    "activated_by": "admin-001"
  }'
```

#### Deactivate Successor
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/successors/deactivate \
  -H "Content-Type: application/json" \
  -d '{
    "poa_id": "00000000-0000-0000-0000-000000000001",
    "deactivated_by": "admin-001",
    "reason": "Primary agent recovered"
  }'
```

#### Get Active Successor
```bash
curl http://localhost:8080/api/v1/gauthplus/successors/active/00000000-0000-0000-0000-000000000001
```

#### Get Successor History
```bash
curl http://localhost:8080/api/v1/gauthplus/successors/history/00000000-0000-0000-0000-000000000001
```

---

### 2. Delegation Management (5 endpoints)

#### Create Delegation
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/delegations \
  -H "Content-Type: application/json" \
  -d '{
    "delegation": {
      "source_poa_id": "00000000-0000-0000-0000-000000000001",
      "source_agent_id": "ai-agent-001",
      "target_agent_id": "ai-agent-002",
      "delegated_scope": ["query", "read"],
      "delegation_depth": 1,
      "max_allowed_depth": 3,
      "valid_from": "2025-01-01T00:00:00Z",
      "valid_until": "2026-01-01T00:00:00Z"
    }
  }'
```

#### Get Delegation Chain
```bash
curl http://localhost:8080/api/v1/gauthplus/delegations/chain/ai-agent-001
```

#### Check Max Depth
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/delegations/check-depth \
  -H "Content-Type: application/json" \
  -d '{
    "source_agent_id": "ai-agent-001",
    "source_poa_id": "00000000-0000-0000-0000-000000000001",
    "current_depth": 2
  }'
```

#### Validate Delegation
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/delegations/validate \
  -H "Content-Type: application/json" \
  -d '{
    "source_agent_id": "ai-agent-001",
    "target_agent_id": "ai-agent-002",
    "scope": ["read"],
    "depth": 1
  }'
```

#### Revoke Delegation
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/delegations/{delegation-id}/revoke \
  -H "Content-Type: application/json" \
  -d '{
    "revoked_by": "admin-001",
    "reason": "Security policy change"
  }'
```

---

### 3. Capability Assessment (6 endpoints)

#### Create Assessment
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/capabilities/assess \
  -H "Content-Type: application/json" \
  -d '{
    "assessment": {
      "agent_id": "ai-agent-001",
      "overall_level": "high",
      "domain_scores": {
        "reasoning": 0.95,
        "safety": 0.88,
        "ethics": 0.92
      },
      "assessed_by": "evaluator-001",
      "valid_until": "2026-01-01T00:00:00Z"
    }
  }'
```

#### Get Latest Assessment
```bash
curl http://localhost:8080/api/v1/gauthplus/capabilities/agents/ai-agent-001/latest
```

#### Get Assessment History
```bash
curl http://localhost:8080/api/v1/gauthplus/capabilities/agents/ai-agent-001/history
```

#### Grant Certification (Not Implemented - Returns 501)
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/capabilities/certifications \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "ai-agent-001",
    "certification_type": "safety",
    "issued_by": "certifier-001"
  }'
```

---

### 4. Fiduciary Duty (4 endpoints)

#### Record Violation
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/fiduciary/violations \
  -H "Content-Type: application/json" \
  -d '{
    "violation": {
      "poa_id": "00000000-0000-0000-0000-000000000001",
      "agent_id": "ai-agent-001",
      "violation_type": "unauthorized_action",
      "severity": "major",
      "description": "Agent exceeded authorization scope",
      "detected_by": "monitor-001"
    }
  }'
```

#### Get Violations for PoA
```bash
curl "http://localhost:8080/api/v1/gauthplus/fiduciary/violations?poa_id=00000000-0000-0000-0000-000000000001"
```

#### Get Violations by Severity
```bash
curl "http://localhost:8080/api/v1/gauthplus/fiduciary/violations/by-severity?severity=major"
```

#### Resolve Violation
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/fiduciary/violations/{violation-id}/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "resolved_by": "admin-001",
    "resolution_notes": "Agent reauthorized after review"
  }'
```

---

### 5. Dual Control (6 endpoints)

#### Create Approval Request
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/dual-control/approvals \
  -H "Content-Type: application/json" \
  -d '{
    "approval": {
      "poa_id": "00000000-0000-0000-0000-000000000001",
      "action_type": "high_risk_transfer",
      "action_details": {"amount": 100000, "currency": "USD"},
      "requested_by": "agent-001",
      "required_approvals": 2
    }
  }'
```

#### Get Approval Status
```bash
curl http://localhost:8080/api/v1/gauthplus/dual-control/approvals/{approval-id}/status
```

#### Get Pending Approvals
```bash
curl http://localhost:8080/api/v1/gauthplus/dual-control/approvals/pending
```

#### Find Approvals by PoA and Action
```bash
curl "http://localhost:8080/api/v1/gauthplus/dual-control/approvals/query?poa_id=00000000-0000-0000-0000-000000000001&action_type=high_risk_transfer"
```

#### Approve Request
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/dual-control/approvals/{approval-id}/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approved_by": "approver-001",
    "comments": "Verified and approved"
  }'
```

#### Reject Request
```bash
curl -X POST http://localhost:8080/api/v1/gauthplus/dual-control/approvals/{approval-id}/reject \
  -H "Content-Type: application/json" \
  -d '{
    "rejected_by": "approver-002",
    "reason": "Insufficient documentation"
  }'
```

---

## Common Response Formats

### Success Response
```json
{
  "success": true,
  "data": { /* endpoint-specific data */ }
}
```

### Error Response
```json
{
  "success": false,
  "error": "error_code",
  "detail": "Human readable error message"
}
```

## HTTP Status Codes

- **200 OK** - Successful request
- **400 Bad Request** - Invalid request data or missing required fields
- **404 Not Found** - Resource not found
- **500 Internal Server Error** - Server-side error
- **501 Not Implemented** - Feature not yet implemented

## Testing

Run the complete integration test suite:

```bash
# Make executable
chmod +x test_gauthplus_api.sh

# Run all tests
./test_gauthplus_api.sh

# Expected output: 19/19 tests passing
```

## Database Requirements

Ensure these tables exist (created by migrations):
- `successor_activations`
- `ai_delegations`
- `ai_capability_assessments`
- `fiduciary_duty_violations`
- `dual_control_approvals`

Test PoA must exist:
```sql
SELECT id FROM power_of_attorney 
WHERE id = '00000000-0000-0000-0000-000000000001';
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GAUTH_GAUTHPLUS_ENABLED` | Yes | `0` | Enable GAuth+ features |
| `GAUTH_RFC0111_ENABLED` | Yes | `0` | Enable RFC-0111 support |
| `DB_HOST` | Yes | - | PostgreSQL host |
| `DB_PORT` | Yes | - | PostgreSQL port |
| `DB_USER` | Yes | - | Database user |
| `DB_PASSWORD` | Yes | - | Database password |
| `DB_NAME` | Yes | - | Database name |
| `GAUTH_JWT_SIGNING_KEY` | Yes | - | JWT signing key |

## Next Steps

1. **Read full documentation**: `GAUTHPLUS_TESTING_COMPLETE.md`
2. **Review API implementation**: `GAUTHPLUS_API_IMPLEMENTATION_COMPLETE.md`
3. **Check test results**: Run `./test_gauthplus_api.sh`
4. **Explore admin UI**: (Coming soon - see `GAUTHPLUS_NEXT_STEPS.md`)

## Support

- **Documentation**: See `GAUTHPLUS_*.md` files in project root
- **Source Code**: `web/handlers/gauthplus/*.go`
- **Tests**: `test_gauthplus_api.sh`
- **Issues**: Check server logs for detailed error messages

---

**Status**: ✅ Production Ready  
**Last Updated**: November 26, 2025
