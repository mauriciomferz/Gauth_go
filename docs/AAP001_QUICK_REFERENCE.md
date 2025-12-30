---
title: Rfc0111 Quick Reference
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# RFC-0111 Quick Reference Guide

## Quick Commands

```bash
# Start server with RFC-0111 enabled
GAUTH_RFC0111_ENABLED=1 go run ./cmd/web-server

# Run all tests
./scripts/test_rfc0111_end_to_end.sh

# Create subscription
curl -X POST http://localhost:8080/api/v1/rfc0111/subscriptions \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-app-123","client_owner_identity":{"subject_id":"owner-123"},"owners_authorizer_identity":{"subject_id":"auth-123"},"pip_token":"pip-token","pvp_token":"pvp-token"}'

# Get authorization token
curl -X POST http://localhost:8080/api/v1/rfc0111/authorize \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-app-123","subscription_id":"sub_XXX","resource_owner_id":"ro-123","scope":"read write"}'
```

## Authorization Flow States

```
Subscription:
  initial → pip_verified → user_confirmed → chain_built 
  → poa_submitted → poa_validated → finalized → completed

Authorization:
  (a) Request → (b) Validate → (c) Grant → (d) Token Request 
  → (e) Token Issue → (f) Grant Check → (g-i) Use/Track
```

## Common OAuth Scopes → RFC-0111 Actions

| Scope    | Action                         |
|----------|--------------------------------|
| `read`   | `ActionNonPhysicalAnalyzing`  |
| `write`  | `ActionNonPhysicalDocumenting`|
| `delete` | `ActionNonPhysicalApproving`  |
| `admin`  | `ActionNonPhysicalApproving`  |

## Extended Token Fields

```
access_token          - OAuth access token
power_of_attorney     - Full PoA structure
authorization_chain   - 3-level chain (authorizer→owner→client)
legal_framework       - Governing law, jurisdiction
compliance_status     - Compliance validation result
verification_proof    - Multi-party verification
audit_trail          - Complete audit log
```

## Response Status Codes

| Code | Meaning                                      |
|------|----------------------------------------------|
| 200  | Success                                      |
| 201  | Subscription created                         |
| 400  | Bad request (validation error)               |
| 401  | Unauthorized (invalid token)                 |
| 404  | Subscription not found                       |
| 500  | Internal error (check logs)                  |

## Error Codes

| Error Code                  | Meaning                                |
|-----------------------------|----------------------------------------|
| `invalid_request`           | Malformed request or missing fields    |
| `authorization_failed`      | Authorization flow step failed         |
| `subscription_not_found`    | Subscription doesn't exist             |
| `step_already_completed`    | Step executed twice (idempotency)      |
| `missing_scope`             | Request scope required                 |
| `missing_poa`               | Power of Attorney required             |
| `missing_legal_framework`   | Legal framework required               |
| `request_compliance_failed` | Compliance validation failed           |

## Troubleshooting

### Token not validating
- **Issue**: Extended tokens not persisted
- **Workaround**: Validate using embedded metadata
- **Fix**: Implement ExtendedTokenStore (future)

### Subscription not found
- **Issue**: In-memory store cleared on restart
- **Fix**: Recreate subscription

### Authorization fails at step (b)
- **Check**: Subscription must be "completed" status
- **Check**: PoA must have active operational status
- **Check**: Legal framework present in PoA

### Step II fails with "already completed"
- **Reason**: Idempotency protection
- **Solution**: This is expected - step already executed

## Testing Checklist

- [ ] Server running with `GAUTH_RFC0111_ENABLED=1`
- [ ] Can create subscription (8 steps)
- [ ] Subscription reaches "completed" status
- [ ] Can request authorization token
- [ ] Token contains all required metadata
- [ ] Compliance status shows "Compliant: true"
- [ ] Authorization chain has 3 levels

## Performance Tips

- Sequential subscriptions: ~10-15s each
- Token issuance: ~200-500ms
- Concurrent requests: Supported (tested with 5 parallel)

## Production Readiness

**Current Status**: ✅ Fully operational for pilot/demo  
**Missing for Production**:
- Token persistence (ExtendedTokenStore)
- PostgreSQL storage
- Real PoA parsing
- OAuth2 client auth
- Rate limiting
- Production monitoring

## File Locations

- **Handlers**: `web/handlers/rfc0111/`
- **Core Logic**: `pkg/gauth/`
- **PoA Types**: `pkg/poa/`
- **Tests**: `scripts/test_rfc0111_*.sh`
- **Docs**: `docs/RFC0111_README.md`
- **Status**: `RFC0111_IMPLEMENTATION_STATUS.md`
