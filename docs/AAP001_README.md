---
title: Rfc0111 Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AAP-001 Authorization Protocol Implementation

## Overview

This directory contains the complete implementation of the AAP-001 Authorization Protocol for AI systems, including multi-party authorization chains, Power of Attorney integration, and comprehensive compliance validation.

## Quick Start

### Prerequisites

- Go 1.21 or later
- Server running on `http://localhost:8080`
- AAP-001 feature enabled: `AGENTAUTH_AAP-001_ENABLED=1`

### Running Tests

```bash
# Test complete end-to-end flow
./scripts/test_aap001_end_to_end.sh

# Test subscription flow only (Steps I-VIII)
./scripts/test_aap001_subscription_flow.sh

# Test error handling
./scripts/test_aap001_error_handling.sh

# Test performance
./scripts/test_aap001_performance.sh
```

## Architecture

### Subscription Flow (Steps I-VIII)

The subscription flow establishes the foundation for authorization by creating a multi-party authorization chain:

```
Step I   → Subscription Initiation
Step II  → Identity Verification (PIP Token)
Step III → Subscriber Confirmation
Step IV  → Authorization Chain Building
Step V   → PoA Credential Submission
Step VI  → PoA Validation
Step VII → Subscription Finalization
Step VIII → Subscription Confirmation
```

**Key Components**:
- `SubscriptionManager` - Orchestrates subscription lifecycle
- `MemorySubscriptionStore` - In-memory subscription persistence
- Authorization chain with 3 levels:
  1. **Owner's Authorizer** - Statutory authority (e.g., board member)
  2. **Client Owner** - AI system owner
  3. **Client** - AI system itself

### Authorization Flow (Steps a-i)

The authorization flow issues AAP-001 compliant tokens with embedded compliance metadata:

```
Step (a) → Client Authorization Request
Step (b) → Request Compliance Validation
Step (c) → Authorization Grant Issuance
Step (d) → Extended Token Request
Step (e) → Extended Token Issuance
Step (f) → Grant Compliance Validation
Step (g) → Transaction/Decision/Action Request
Step (h) → Token Validation & Request Fulfillment
Step (i) → Compliance Tracking
```

**Key Components**:
- `ProtocolOrchestrator` - Orchestrates authorization protocol steps
- `ComplianceValidator` - Validates AAP-001 compliance requirements
- `ExtendedTokenService` - Creates extended tokens with full metadata
- `AuthorizationChainValidator` - Validates multi-party authorization chains

## API Endpoints

### Subscription Endpoints

```bash
# Create subscription
POST /api/v1/aap001/subscriptions
{
  "client_id": "client-app-123",
  "client_owner_identity": {"subject_id": "client-owner-67890"},
  "owners_authorizer_identity": {"subject_id": "auth-12345"},
  "pip_token": "mock-pip-token-abc",
  "pvp_token": "mock-pvp-token-xyz"
}

# Get subscription
GET /api/v1/aap001/subscriptions/{id}

# Execute subscription steps
POST /api/v1/aap001/subscriptions/{id}/step-ii
POST /api/v1/aap001/subscriptions/{id}/step-iii
POST /api/v1/aap001/subscriptions/{id}/step-iv
POST /api/v1/aap001/subscriptions/{id}/step-v
POST /api/v1/aap001/subscriptions/{id}/step-vi
POST /api/v1/aap001/subscriptions/{id}/step-vii
POST /api/v1/aap001/subscriptions/{id}/step-viii
```

### Authorization Endpoints

```bash
# Request authorization token
POST /api/v1/aap001/authorize
{
  "client_id": "client-app-123",
  "subscription_id": "sub_1234567890",
  "resource_owner_id": "resource-owner-99999",
  "poa_credential_ref": "mock-poa-credential-xyz",
  "scope": "read write",
  "context": {"purpose": "data_processing"}
}

# Response includes:
# - extended_token: Full AAP-001 compliant token
# - compliance_status: Compliance validation result
# - scope: Authorized actions
```

### Token Lifecycle Endpoints

```bash
# Validate token (Note: Currently limited - see Known Limitations)
POST /api/v1/aap001/token/validate
{"token": "agentauth_at_..."}

# Introspect token (RFC 7662)
POST /api/v1/aap001/token/introspect
{"token": "agentauth_at_..."}

# Revoke token (RFC 7009)
POST /api/v1/aap001/token/revoke
{"token": "agentauth_at_..."}
```

## Extended Token Structure

AAP-001 extended tokens include comprehensive metadata:

```json
{
  "access_token": "agentauth_at_...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "agentauth_rt_...",
  
  "power_of_attorney": {
    "parties": {
      "principal": {"type": "organization", "identity": "..."},
      "authorized_client": {
        "Identity": "client-app-123",
        "OperationalStatus": "active",
        "CapabilityLevel": "L3_conditional_automation"
      }
    },
    "authorization": {
      "AuthorizedActions": {
        "NonPhysicalActions": ["Analyzing", "Documenting"]
      }
    },
    "requirements": {
      "JurisdictionLaw": {
        "GoverningLaw": "EU-GDPR",
        "PlaceOfJurisdiction": "Germany"
      }
    }
  },
  
  "authorization_chain": {
    "owners_authorizer": {...},
    "client_owner": {...},
    "client": {...},
    "chain_validated": true,
    "chain_depth": 3
  },
  
  "legal_framework": {
    "applicable_laws": ["EU-GDPR"],
    "jurisdiction": "Germany"
  },
  
  "verification_proof": {
    "overall_verification": "verified",
    "verification_levels": [...]
  },
  
  "compliance_level": "rfc-0111-compliant",
  "audit_trail": [...]
}
```

## Compliance Validation

The implementation validates multiple AAP-001 compliance requirements:

### Request Compliance (Step b)
- ✅ Request structure (client ID, scopes)
- ✅ Client identification via PIP
- ✅ Authorization chain validation
- ✅ Power of Attorney validation
- ✅ Legal framework requirements

### Grant Compliance (Step f)
- ✅ Grant structure and validity
- ✅ Issuer authority
- ✅ Resource owner authorization
- ✅ Legal framework compliance

### Token Compliance (Step e)
- ✅ Authorization chain integrity
- ✅ Power of Attorney validity
- ✅ Multi-party verification
- ✅ Legal framework information

## OAuth Scope Mapping

OAuth 2.0 scopes are automatically mapped to AAP-001 action types:

| OAuth Scope | AAP-001 Action |
|-------------|-----------------|
| `read`      | `ActionNonPhysicalAnalyzing` |
| `write`     | `ActionNonPhysicalDocumenting` |
| `delete`    | `ActionNonPhysicalApproving` |
| `admin`     | `ActionNonPhysicalApproving` |

Example:
```json
{
  "scope": "read write"
}
```
Maps to:
```json
{
  "AuthorizedActions": {
    "NonPhysicalActions": ["Analyzing", "Documenting"]
  }
}
```

## Known Limitations

### Token Validation
Extended tokens are **not persisted** after creation. The token validation, introspection, and revocation endpoints exist but don't work with AAP-001 extended tokens.

**Workaround**: Resource servers can validate tokens using embedded metadata:
- Authorization chain validation
- PoA structure validation
- Legal framework compliance
- Timestamp and expiry checks

**Future Enhancement**: Implement `ExtendedTokenStore` for token persistence.

### Power of Attorney
Currently uses **mock PoA** creation in Step V. Production implementation should:
- Parse full PoADefinition from request body
- Validate PoA structure completely
- Store PoA credentials in database
- Support PoA references and retrieval

### Storage
Uses **in-memory storage** for subscriptions and tokens. Data is lost on server restart.

**Production Requirement**: Implement PostgreSQL storage:
- `PostgresSubscriptionStore`
- `PostgresTokenStore`
- `PostgresPoAStore`

## Testing

### Test Scripts

1. **`test_aap001_end_to_end.sh`**
   - Complete subscription + authorization flow
   - Verifies all metadata fields
   - Checks compliance status
   - **Status**: ✅ All tests passing

2. **`test_aap001_subscription_flow.sh`**
   - Tests all 8 subscription steps
   - Verifies state transitions
   - Tests error handling
   - **Status**: ✅ 11/11 tests passing

3. **`test_aap001_error_handling.sh`**
   - Tests various error scenarios
   - Validates error responses
   - Tests idempotency protection

4. **`test_aap001_performance.sh`**
   - Sequential request performance
   - Concurrent load testing
   - End-to-end timing

### Manual Testing

```bash
# 1. Start server with AAP-001 enabled
AGENTAUTH_AAP-001_ENABLED=1 go run ./cmd/web-server

# 2. Create subscription
./scripts/test_aap001_subscription_flow.sh

# 3. Extract subscription ID
SUB_ID=$(curl -s "http://localhost:8080/api/v1/aap001/subscriptions" | jq -r '.[-1].subscription_id')

# 4. Request authorization
curl -X POST "http://localhost:8080/api/v1/aap001/authorize" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"client-app-123\",
    \"subscription_id\": \"$SUB_ID\",
    \"resource_owner_id\": \"resource-owner-99999\",
    \"poa_credential_ref\": \"mock-poa-credential-xyz\",
    \"scope\": \"read write\",
    \"context\": {\"purpose\": \"data_processing\"}
  }" | jq
```

## Development

### Code Structure

```
web/handlers/aap001/
├── authorization_handlers.go    # Authorization endpoints
└── subscription_handlers.go     # Subscription endpoints

pkg/agentauth/
├── protocol_orchestrator.go     # Authorization flow orchestration
├── compliance_validation.go     # AAP-001 compliance validation
├── extended_token_service.go    # Extended token creation
├── authorization_chain.go       # Chain validation
└── subscription_manager.go      # Subscription lifecycle

pkg/poa/
├── poa.go                       # Power of Attorney types
├── action_taxonomy.go           # Action type definitions
└── validation.go                # PoA validation logic

scripts/
├── test_aap001_end_to_end.sh       # E2E tests
├── test_aap001_subscription_flow.sh # Subscription tests
├── test_aap001_error_handling.sh    # Error tests
└── test_aap001_performance.sh       # Performance tests
```

### Adding New Features

1. **New Subscription Step**:
   - Add step handler in `subscription_handlers.go`
   - Update `SubscriptionManager` logic
   - Add route in server setup
   - Update test scripts

2. **New Compliance Check**:
   - Add validation method in `compliance_validation.go`
   - Call from `ValidateRequestCompliance()` or `ValidateGrantCompliance()`
   - Update error responses
   - Add test cases

3. **New Token Field**:
   - Update `ExtendedToken` struct in `extended_token.go`
   - Populate in `CreateExtendedToken()` method
   - Update API documentation
   - Add test verification

## Production Checklist

Before production deployment:

- [ ] Implement `ExtendedTokenStore` for token persistence
- [ ] Implement PostgreSQL storage for subscriptions
- [ ] Implement real PoA parsing and storage
- [ ] Add OAuth2 client authentication
- [ ] Implement rate limiting
- [ ] Add comprehensive audit logging
- [ ] Set up monitoring and metrics
- [ ] Configure HTTPS/TLS
- [ ] Implement token rotation
- [ ] Add backup and recovery procedures
- [ ] Security audit
- [ ] Load testing with production-like data

## Support

For issues or questions:
1. Check [AAP-001_IMPLEMENTATION_STATUS.md](../AAP-001_IMPLEMENTATION_STATUS.md)
2. Review test scripts for usage examples
3. Check server logs for detailed error messages

## License

See [LICENSE](../LICENSE) file for details.
