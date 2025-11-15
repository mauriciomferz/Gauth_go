# React UI - Backend API Integration Guide

**Date**: November 15, 2025  
**Status**: In Progress 🔄  
**Backend**: http://localhost:8080  
**Frontend**: http://localhost:3000

---

## Executive Summary

The React UI (100% complete, 2,531 lines) needs API endpoint mappings updated to match the GAuth Go backend structure. The backend uses `/api/v1/beta/` prefix with different payload structures than initially designed in the UI.

---

## Server Status ✅

- **Backend**: Running on port 8080 (`go run ./cmd/web-server`)
- **Frontend**: Running on port 3000 (`npm run dev`)
- **Proxy**: Vite dev server proxies `/api/*` → `http://localhost:8080`
- **Health Check**: ✅ `GET /api/v1/beta/health` returns `{"status":"healthy"}`

---

## API Endpoint Mappings

### Current UI Implementation vs Backend Reality

| UI Page | UI Endpoint (Expected) | Backend Endpoint (Actual) | Status |
|---------|----------------------|--------------------------|---------|
| **Tokens** | `POST /api/v1/token/create` | `POST /api/v1/beta/delegation/create` | ❌ Mismatch |
| **Tokens** | `POST /api/v1/token/validate` | ❓ TBD | ❌ Not found |
| **PVP** | `POST /api/v1/pvp/verify` | ❓ TBD | ❌ Not found |
| **Registry** | `POST /api/v1/registry/verify-entity` | ❓ TBD | ❌ Not found |
| **Registry** | `POST /api/v1/registry/verify-signatory` | ❓ TBD | ❌ Not found |
| **PIP** | `POST /api/v1/pip/validate` | `POST /api/v1/beta/authz/evaluate` | ❌ Mismatch |
| **PoA** | `POST /api/v1/poa/create` | ❓ TBD | ❌ Not found |
| **PoA** | `POST /api/v1/poa/validate` | ❓ TBD | ❌ Not found |
| **Metrics** | `GET /api/v1/metrics` | `GET /api/v1/beta/metrics/prometheus` | ❌ Mismatch |
| **Health** | `GET /api/v1/health` | `GET /api/v1/beta/health` | ✅ Works |

---

## Backend Available Endpoints (Verified)

### System & Health
```
GET  /api/v1/beta/health             ✅ Working - Returns {"status":"healthy"}
GET  /api/v1/beta/info               ✅ Available
GET  /api/v1/beta/ping               ✅ Available
```

### Policy Management
```
GET  /api/v1/beta/policy/provenance
GET  /api/v1/beta/policy/chain
GET  /api/v1/beta/policy/head/policies
GET  /api/v1/beta/policy/metrics
GET  /api/v1/beta/policy/metrics/prometheus
POST /api/v1/beta/policy/bundles
GET  /api/v1/beta/policy/bundles/:hash
POST /api/v1/beta/policy/rollback
POST /api/v1/beta/policy/evaluate
GET  /api/v1/beta/policy/diff
GET  /api/v1/beta/policy/timeline
```

### Delegation (Token) Management
```
POST /api/v1/beta/delegation/create        ⚠️ Different payload than UI expects
POST /api/v1/beta/delegation/revoke
POST /api/v1/beta/delegation/status/update
```

**Delegation Create Payload** (Backend expects):
```json
{
  "delegation_id": "string",
  "subject": "string", 
  "delegate": "string",
  "claims": {
    "cap": ["cap.delegation.create"]
  }
}
```

**Token Create Payload** (UI currently sends):
```json
{
  "clientId": "string",
  "ownersAuthorizer": "string",
  "clientOwner": "string",
  "scope": "string",
  "expirationHours": number
}
```

### Authorization & Evaluation
```
POST /api/v1/beta/authz/evaluate
GET  /api/v1/beta/authz/metrics
GET  /api/v1/beta/authz/metrics/prometheus
GET  /api/v1/beta/metrics/decisions
```

### Capabilities
```
GET  /api/v1/beta/capabilities
POST /api/v1/beta/capabilities/reload
POST /api/v1/beta/capabilities/negotiate
GET  /api/v1/beta/capabilities/anchor/metrics/prometheus
```

### Audit & Compliance
```
GET  /api/v1/beta/audit
GET  /api/v1/beta/policy/audit-consistency
```

### Metrics & Monitoring
```
GET  /api/v1/beta/metrics/violations
GET  /api/v1/beta/metrics/violations/prometheus
GET  /api/v1/beta/metrics/prometheus            ✅ Prometheus format
GET  /metrics                                    ✅ Root-level Prometheus
```

### Rotation & Keys
```
GET  /api/v1/beta/rotations/summary
GET  /api/v1/beta/rotations/summary/v2
GET  /api/v1/beta/rotations/verification
GET  /api/v1/beta/keys/eddsa
```

### Examples & Testing
```
GET  /api/v1/beta/examples/catalog
POST /api/v1/beta/examples/run
GET  /api/v1/beta/examples/run/:id/status
GET  /api/v1/beta/examples/run/:id/logs
GET  /api/v1/beta/examples/run/jobs
POST /api/v1/beta/examples/run/jobs/:id/cancel
```

---

## Integration Strategy

### Phase 1: Core Token/Delegation Mapping ⏳
**Goal**: Make Tokens page functional

**Tasks**:
1. ✅ Identify backend delegation endpoints
2. ❌ Map UI token creation to delegation creation
3. ❌ Find or mock token validation endpoint
4. ❌ Update `createToken()` in api.ts
5. ❌ Update `validateToken()` in api.ts
6. ❌ Test Tokens page end-to-end

### Phase 2: Authorization/PIP Integration ⏳
**Goal**: Make PIP page functional

**Tasks**:
1. ❌ Map to `/beta/authz/evaluate`
2. ❌ Update payload structure
3. ❌ Update `validateAuthorization()` in api.ts
4. ❌ Test PIP page

### Phase 3: Metrics Integration ⏳
**Goal**: Make Metrics page functional

**Tasks**:
1. ❌ Parse Prometheus metrics format
2. ❌ Transform to frontend chart data
3. ❌ Update `getMetrics()` in api.ts
4. ❌ Test Metrics page with real data

### Phase 4: Registry & PVP (Mock or Find) ⏳
**Goal**: Make Registry and PVP pages functional

**Options**:
- Option A: Find existing backend endpoints (if they exist)
- Option B: Create mock adapter layer in UI
- Option C: Implement backend endpoints (out of UI integration scope)

**Current Status**: Need to search backend for PVP/Registry endpoints or implement mocking

### Phase 5: PoA Integration ⏳
**Goal**: Make PoA page functional

**Tasks**:
1. ❌ Find PoA endpoints in backend (may not exist yet)
2. ❌ Mock or implement integration
3. ❌ Update PoA API methods

---

## Payload Transformation Examples

### Token Creation Mapping

**UI Form Data**:
```typescript
{
  clientId: "test-client-123",
  ownersAuthorizer: "owner-auth-456",
  clientOwner: "client-owner-789",
  scope: "read",
  expirationHours: 24
}
```

**Backend Delegation Payload** (proposed mapping):
```json
{
  "delegation_id": "test-client-123",
  "subject": "owner-auth-456",
  "delegate": "client-owner-789",
  "claims": {
    "cap": ["cap.delegation.create"],
    "scope": "read",
    "exp": "<calculated_timestamp>"
  }
}
```

---

## Mock vs Real Integration Decision Matrix

| Feature | Backend Exists? | Complexity | Strategy |
|---------|----------------|------------|----------|
| Token Create | ✅ Yes (delegation) | Medium | **Real** - Adapt payload |
| Token Validate | ❓ Unknown | Low | **Mock** - Client-side validation |
| Authorization | ✅ Yes (authz/evaluate) | Medium | **Real** - Adapt payload |
| PVP Verify | ❓ Unknown | Low | **Mock** - Simulated verification |
| Registry Entity | ❓ Unknown | Low | **Mock** - Sample data |
| Registry Signatory | ❓ Unknown | Low | **Mock** - Sample data |
| PoA Create | ❓ Unknown | Low | **Mock** - In-memory PoA |
| PoA Validate | ❓ Unknown | Low | **Mock** - Validation logic |
| Metrics | ✅ Yes (Prometheus) | High | **Real** - Parse Prometheus |

---

## Next Steps (Immediate)

1. **Search backend for missing endpoints**:
   - PVP verification
   - Registry entity/signatory
   - PoA management
   - Token validation

2. **Update API client** (`src/lib/api.ts`):
   - Add `/beta/` prefix to routes
   - Transform payloads for delegation
   - Add Prometheus parser for metrics

3. **Test each page**:
   - Tokens: delegation create + mock validate
   - PIP: authz/evaluate
   - Metrics: Prometheus data
   - Others: Mock or find endpoints

4. **Document findings**:
   - Which endpoints exist
   - Which need mocking
   - Which need backend implementation

---

## Questions for Backend Team

1. **Token Validation**: Is there a JWT validation endpoint? Or should UI decode client-side?
2. **PVP Verification**: Does `/api/v1/beta/pvp/*` exist? Or should we use RFC-0111 service?
3. **Registry**: Are entity/signatory verification endpoints implemented?
4. **PoA**: Are Power of Attorney endpoints available? Or use delegation endpoints?
5. **JWT Structure**: What claims/structure do delegation JWTs have?

---

## Current Blockers

1. ❌ **Payload mismatch**: UI token creation !== backend delegation creation
2. ❌ **Missing endpoints**: PVP, Registry, PoA not found in backend routes
3. ❌ **Metrics format**: Prometheus text format needs parsing to chart data
4. ❌ **No JWT validation endpoint**: Need to decode client-side or find endpoint

---

## Success Criteria

- [ ] Tokens page: Create delegation (token) successfully
- [ ] Tokens page: Validate token (real or mock)
- [ ] PIP page: Evaluate authorization successfully
- [ ] Metrics page: Display real Prometheus metrics as charts
- [ ] Registry page: Verify entities (real or mock)
- [ ] PVP page: Verify identities (real or mock)
- [ ] PoA page: Create/validate PoA (real or mock)
- [ ] E2E Testing page: Run tests against real backend
- [ ] All pages: No console errors, smooth UX

---

**Report Status**: Diagnostic phase complete, integration phase starting  
**Next Action**: Search backend for PVP/Registry/PoA endpoints, then update API client
