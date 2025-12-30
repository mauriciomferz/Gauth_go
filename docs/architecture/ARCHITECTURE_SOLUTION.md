---
title: AAP-001 Implementation Architecture
category: architecture
status: active
lastUpdated: 2025-11-12
owners: architecture-team
---
# AAP-001 Implementation Architecture
## Visual Guide to the Solution

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AAP-001 COMPLIANT AGENTAUTH                            │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    ONE-OFF SUBSCRIPTION FLOW                        │    │
│  │                         (Steps I-VIII)                              │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│    NEW: SubscriptionFlowManager                                             │
│    ┌──────────────────────────────────────────────────────────────────┐     │
│    │ Step I:   Owner's Authorizer Identity Proof                      │     │
│    │           → PVP.VerifyIdentityProof()                            │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ Step II:  Owner's Authorizer Authorization Proof                 │     │
│    │           → CommercialRegisterClient.Verify()                    │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ Step III: Client Owner Identity Proof                            │     │
│    │           → PVP.VerifyIdentityProof()                            │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ Step IV:  Client Owner Authorization Proof                       │     │
│    │           → AuthChainValidator.Validate()  ✓ CONNECTS EXISTING   │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ Step V:   Client Authorization                                   │     │
│    │           → FormalReqValidator.Validate()  ✓ CONNECTS EXISTING   │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ Step VI:  Resource Owner Identity Proof                          │     │
│    │           → PVP.VerifyIdentityProof()                            │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ Step VII: Resource Owner Authorization Proof                     │     │
│    │           → AuthChainValidator.Validate()  ✓ CONNECTS EXISTING   │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ Step VIII: Resource Server Authorization                         │     │
│    │           → Store server registration                            │     │
│    └──────────────────────────────────────────────────────────────────┘     │
│                               ↓                                             │
│                    [Subscription Status: COMPLETED]                         │
│                               ↓                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    REQUEST-SPECIFIC FLOW                            │    │
│  │                         (Steps a-i)                                 │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│    NEW: ProtocolOrchestrator                                                │
│    ┌──────────────────────────────────────────────────────────────────┐     │
│    │ (a) Client Authorization Request                                 │     │
│    │     → RFCCompliantAuthorizationRequest received                  │     │
│    │     → Verify subscription completed ✓                            │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ (b) Request Compliance Validation                                │     │
│    │     → ComplianceValidator.ValidateRequestCompliance() ✓ CALLED   │     │
│    │     → Checks request vs client's authorized powers               │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ (c) Authorization Grant Issuance                                 │     │
│    │     → IssueAuthorizationGrant()                                  │     │
│    │     → Embed PoA credential, auth chain, compliance result        │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ (d) Extended Token Request                                       │     │
│    │     → Grant serves as token request (implicit)                   │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ (e) Extended Token Issuance                                      │     │
│    │     → ExtendedTokenService.CreateExtendedToken() ✓ CALLED        │     │
│    │     → NOT jwt.NewWithClaims() ❌                                 │     │
│    │     → Returns AAP-001 ExtendedToken ✓                           │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ (f) Grant Compliance Validation                                  │     │
│    │     → ComplianceValidator.ValidateGrantCompliance() ✓ CALLED     │     │
│    │     → Checks grant vs resource owner/server powers               │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ (g) Transaction/Decision/Action Request                          │     │
│    │     → Prepare token metadata for downstream                      │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ (h) Token Validation & Request Fulfillment                       │     │
│    │     → Embed all validation results in token                      │     │
│    ├──────────────────────────────────────────────────────────────────┤     │
│    │ (i) Compliance Tracking                                          │     │
│    │     → ComplianceTracker.StartTracking() ✓ CALLED                 │     │
│    │     → Monitor ongoing behavior vs authorized scope               │     │
│    └──────────────────────────────────────────────────────────────────┘     │
│                               ↓                                             │
│                    Return AAP-001 ExtendedToken                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Current vs. Fixed Architecture

### CURRENT ARCHITECTURE (BROKEN)

```
┌──────────────┐
│   Client     │
└──────┬───────┘
       │
       │ POST /v1/token
       │ { grant_id, scope }
       │
       ↓
┌─────────────────────────┐
│   RequestToken()        │  ← PROBLEM: Direct JWT generation
│                         │
│   jwt.NewWithClaims()   │  ← No validation
│   token.SignedString()  │  ← No extended token
│                         │  ← No compliance checking
│   return TokenResponse  │  ← Wrong type
└─────────────────────────┘
       │
       │
       ↓
┌──────────────────────────────────────┐
│  UNUSED VALIDATION FUNCTIONS         │
│  - ValidateRequestCompliance() ❌    │  ← 200 lines, never called
│  - ValidateGrantCompliance() ❌      │  ← 150 lines, never called
│  - ValidateAuthorizationChain() ❌   │  ← 720 lines, never called
│  - ValidateFormalRequirements() ❌   │  ← 814 lines, never called
│  - CreateExtendedToken() ❌          │  ← 400 lines, never called
└──────────────────────────────────────┘
```

**Result**: OAuth server, NOT AgentAuth. No RFC compliance.

---

### FIXED ARCHITECTURE (RFC-COMPLIANT)

```
┌──────────────┐
│   Client     │
└──────┬───────┘
       │
       │ Step 1: Complete Subscription (ONCE)
       │ ↓
       │ POST /v1/subscriptions
       │ → Execute Steps I-VIII
       │ → Get subscription_id
       │
       │ Step 2: Request Authorization (PER REQUEST)
       │ ↓
       │ POST /v1/token/rfc
       │ {
       │   subscription_id,
       │   requested_scope,
       │   poa_credential_ref
       │ }
       │
       ↓
┌─────────────────────────────────────────────────────────────┐
│  RequestTokenRFC()                                          │
│  → NEW: Uses ProtocolOrchestrator                           │
└─────────────────────────────────────────────────────────────┘
       │
       ↓
┌─────────────────────────────────────────────────────────────┐
│  ProtocolOrchestrator.ExecuteRFCCompliantFlow()             │
│  → NEW: Orchestrates steps (a)-(i)                          │
└─────────────────────────────────────────────────────────────┘
       │
       │ (a) Verify subscription completed
       │ ↓
       │ SubscriptionStore.GetSubscription() ✓
       │
       │ (b) Validate request compliance
       │ ↓
       │ ComplianceValidator.ValidateRequestCompliance() ✓
       │
       │ (c) Issue grant
       │ ↓
       │ IssueAuthorizationGrant() ✓
       │
       │ (d)-(e) Create extended token
       │ ↓
       │ ExtendedTokenService.CreateExtendedToken() ✓
       │   → AuthChainValidator.Validate() ✓
       │   → FormalReqValidator.Validate() ✓
       │
       │ (f) Validate grant compliance
       │ ↓
       │ ComplianceValidator.ValidateGrantCompliance() ✓
       │
       │ (i) Start compliance tracking
       │ ↓
       │ ComplianceTracker.StartTracking() ✓
       │
       ↓
┌─────────────────────────────────────┐
│  Return RFCCompliantTokenResponse   │
│  {                                  │
│    extended_token: {                │
│      power_of_attorney: {...},      │
│      authorization_chain: {...},    │
│      verification_proof: {...},     │
│      compliance_level: "rfc-0111"   │
│    },                               │
│    grant_validation: {...},         │
│    compliance_status: {...}         │
│  }                                  │
└─────────────────────────────────────┘
```

**Result**: True AAP-001 AgentAuth implementation. ✓

---

## Component Interaction Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         EXISTING COMPONENTS                             │
│                         (Well-implemented)                              │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
         ┌──────────────────────────┼──────────────────────────┐
         │                          │                          │
         ↓                          ↓                          ↓
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│ Authorization   │      │   Compliance    │      │  Formal Req     │
│ Chain Validator │      │    Validator    │      │   Validator     │
│                 │      │                 │      │                 │
│ 720 lines ✓     │      │ 500+ lines ✓    │      │ 814 lines ✓     │
└─────────────────┘      └─────────────────┘      └─────────────────┘
         │                          │                          │
         └──────────────────────────┼──────────────────────────┘
                                    │
                                    │ Currently: DISCONNECTED ❌
                                    │ After fix: CONNECTED ✓
                                    │
         ┌──────────────────────────┼──────────────────────────┐
         │                          │                          │
         ↓                          ↓                          ↓
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│  Extended Token │      │   Protocol      │      │  Subscription   │
│    Service      │      │  Orchestrator   │      │  Flow Manager   │
│                 │      │                 │      │                 │
│ 400 lines ✓     │      │ NEW ~500 lines  │      │ NEW ~600 lines  │
└─────────────────┘      └─────────────────┘      └─────────────────┘
         │                          │                          │
         │                          │                          │
         └──────────────────────────┼──────────────────────────┘
                                    │
                                    │ Coordinates all components
                                    │
                                    ↓
                          ┌─────────────────┐
                          │  RequestToken   │
                          │      RFC()      │
                          │                 │
                          │  NEW entry      │
                          │     point       │
                          └─────────────────┘
```

---

## Data Flow: Token Request

### Step-by-Step Data Transformation

```
1. CLIENT REQUEST
   ┌───────────────────────────────────┐
   │ RFCCompliantAuthorizationRequest  │
   │ {                                 │
   │   client_id: "llm_gpt4",          │
   │   subscription_id: "sub_12345",   │
   │   requested_scope: {              │
   │     sectors: ["K"],               │
   │     transactions: ["Purchase"],   │
   │     value_limits: {max: 10000}    │
   │   }                               │
   │ }                                 │
   └───────────────────────────────────┘
                 ↓

2. VERIFY SUBSCRIPTION (NEW)
   ┌────────────────────────────────────┐
   │ SubscriptionStore.GetSubscription  │
   │ → Returns Subscription with:       │
   │   - Steps I-VIII completed ✓       │
   │   - PoA credential                 │
   │   - Authorization chain            │
   │   - All identity proofs            │
   └────────────────────────────────────┘
                 ↓

3. VALIDATE REQUEST (STEP b) - NOW CALLED ✓
   ┌────────────────────────────────────┐
   │ ValidateRequestCompliance()        │
   │ → Checks:                          │
   │   - Request within client powers?  │
   │   - Sector authorized?             │
   │   - Transaction type allowed?      │
   │   - Value within limits?           │
   │ → Returns: RequestComplianceResult │
   └────────────────────────────────────┘
                 ↓

4. ISSUE GRANT (STEP c)
   ┌────────────────────────────────────┐
   │ IssueAuthorizationGrant()          │
   │ → Creates grant with:              │
   │   - PoA credential embedded        │
   │   - Authorization chain            │
   │   - Compliance validation result   │
   │   - Expiry (short-lived)           │
   └────────────────────────────────────┘
                 ↓

5. CREATE EXTENDED TOKEN (STEP e) - NOW CALLED ✓
   ┌────────────────────────────────────┐
   │ CreateExtendedToken()              │
   │ → NOT jwt.NewWithClaims() ❌       │
   │ → Creates ExtendedToken with:      │
   │   - Access token                   │
   │   - Power of attorney              │
   │   - Authorization chain            │
   │   - Client owner info              │
   │   - Owner's authorizer info        │
   │   - Legal framework                │
   │   - Verification proof             │
   │   - Restrictions                   │
   │   - Audit trail                    │
   └────────────────────────────────────┘
                 ↓

6. VALIDATE GRANT (STEP f) - NOW CALLED ✓
   ┌────────────────────────────────────┐
   │ ValidateGrantCompliance()          │
   │ → Checks:                          │
   │   - Grant vs resource owner powers │
   │   - Grant vs resource server rules │
   │   - All constraints satisfied      │
   │ → Returns: GrantComplianceResult   │
   └────────────────────────────────────┘
                 ↓

7. START TRACKING (STEP i) - NEW ✓
   ┌────────────────────────────────────┐
   │ ComplianceTracker.StartTracking()  │
   │ → Initiates monitoring:            │
   │   - Token usage tracking           │
   │   - Constraint enforcement         │
   │   - Violation detection            │
   │   - Audit logging                  │
   └────────────────────────────────────┘
                 ↓

8. RETURN RESPONSE
   ┌────────────────────────────────────┐
   │ RFCCompliantTokenResponse          │
   │ {                                  │
   │   extended_token: {                │
   │     access_token: "ext_...",       │
   │     token_type: "AgentAuth-Extended",  │
   │     expires_in: 3600,              │
   │     power_of_attorney: {...},      │
   │     authorization_chain: {...},    │
   │     verification_proof: {...},     │
   │     compliance_level: "rfc-0111"   │
   │   },                               │
   │   grant_validation: {              │
   │     valid: true                    │
   │   },                               │
   │   compliance_status: {             │
   │     compliant: true,               │
   │     violations: []                 │
   │   }                                │
   │ }                                  │
   └────────────────────────────────────┘
```

---

## File Structure

```
pkg/agentauth/
├── agentauth.go                          (EXISTING - UPDATE)
│   ├── RequestToken()                 ← Update to use orchestrator
│   ├── RequestTokenLegacy()           ← NEW: Keep old JWT mode
│   └── RequestTokenRFC()              ← NEW: RFC-compliant entry point
│
├── subscription_flow.go              (NEW - 600 lines)
│   ├── SubscriptionFlowManager       ← Manages Steps I-VIII
│   ├── ExecuteStepI()                 ← Owner's authorizer identity
│   ├── ExecuteStepII()                ← Commercial register verification
│   ├── ExecuteStepIII()               ← Client owner identity
│   ├── ExecuteStepIV()                ← Client owner authorization
│   ├── ExecuteStepV()                 ← Client authorization
│   ├── ExecuteStepVI()                ← Resource owner identity
│   ├── ExecuteStepVII()               ← Resource owner authorization
│   └── ExecuteStepVIII()              ← Resource server authorization
│
├── protocol_orchestrator.go          (NEW - 500 lines)
│   ├── ProtocolOrchestrator          ← Orchestrates Steps (a)-(i)
│   ├── ExecuteRFCCompliantFlow()     ← Main orchestration method
│   └── RFCCompliantAuthorizationRequest ← New request type
│
├── compliance_tracker.go             (NEW - 300 lines)
│   ├── ComplianceTracker             ← Implements Step (i)
│   ├── StartTracking()                ← Begin monitoring
│   └── CheckCompliance()              ← Periodic checks
│
├── subscription_store.go             (NEW - 200 lines)
│   └── SubscriptionStore interface    ← Persist subscription state
│
├── subscription_store_memory.go      (NEW - 150 lines)
│   └── MemorySubscriptionStore        ← In-memory implementation
│
├── extended_token_store_postgres.go  (COMPLETE - 479 lines) ✓
│   └── PostgresExtendedTokenStore     ← PostgreSQL persistence
│       ├── SaveToken()                ← Store tokens with JSONB
│       ├── GetToken()                 ← Retrieve by access token
│       ├── GetTokenByRefreshToken()   ← Retrieve by refresh token
│       ├── ListTokensByClient()       ← Query by client_id (JSONB path)
│       ├── RevokeToken()              ← Mark as revoked
│       ├── DeleteExpiredTokens()      ← Cleanup task
│       └── Close()                    ← Connection cleanup
│
├── authorization_chain_validation.go (EXISTING - 720 lines) ✓
│   └── NOW CALLED by Steps IV, VII
│
├── compliance_validation.go          (EXISTING - 500+ lines) ✓
│   └── NOW CALLED by Steps (b), (f)
│
├── formal_requirements_validation.go (EXISTING - 814 lines) ✓
│   └── NOW CALLED by Step V
│
└── extended_token_service.go         (EXISTING - 400 lines) ✓
    └── NOW CALLED by Step (e)
```

---

## Summary: Before → After

### Before (Current)
```
❌ No subscription flow
❌ No protocol orchestration
❌ Direct JWT generation
❌ Validation functions never called
❌ OAuth server, not AgentAuth
❌ RFC compliance: 58/100
```

### After (Fixed)
```
✅ Subscription flow (Steps I-VIII)
✅ Protocol orchestrator (Steps a-i)
✅ Extended tokens with PoA metadata
✅ All validation functions integrated
✅ True AAP-001 AgentAuth implementation
✅ RFC compliance: 95+/100
```

---

## Key Takeaway

> **We have all the pieces. We just need to connect them.**

The code quality is excellent. The validation functions are comprehensive. 
The data structures are perfect. We're not rewriting from scratch.

**We're adding 3 new files (~1,400 lines) to orchestrate the existing ~5,000 
lines of validation code into an AAP-001 compliant flow.**

---

## Persistence Layer (COMPLETED ✅)

### PostgreSQL Integration for Extended Tokens

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       PRODUCTION STORAGE LAYER                          │
└─────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────┐      ┌──────────────────────────┐
│   ExtendedTokenStore     │      │   SubscriptionStore      │
│      (Interface)         │      │      (Interface)         │
└────────────┬─────────────┘      └────────────┬─────────────┘
             │                                 │
       ┌─────┴─────┐                      ┌────┴────┐
       │           │                      │         │
       ↓           ↓                      ↓         ↓
┌───────────┐ ┌──────────────┐    ┌──────────┐ ┌──────────────┐
│  Memory   │ │  PostgreSQL  │    │  Memory  │ │ PostgreSQL   │
│   Store   │ │    Store     │    │   Store  │ │ Store (TODO) │
│  ✓ Done   │ │   ✅ DONE    │    │ ✓ Done   │ │              │
└───────────┘ └──────────────┘    └──────────┘ └──────────────┘
                     │
                     │
        ┌────────────┴────────────┐
        │                         │
        ↓                         ↓
┌─────────────────┐    ┌─────────────────────┐
│ extended_tokens │    │  Schema Features    │
│     (Table)     │    │                     │
│                 │    │ • 28 columns        │
│ • access_token  │    │ • 5 indexes         │
│ • token_type    │    │ • JSONB storage     │
│ • expires_in    │    │ • Connection pool   │
│ • refresh_token │    │ • pq.Array() for    │
│ • scope []text  │    │   PostgreSQL arrays │
│ • issued_at     │    │ • Nullable issued_by│
│ • expires_at    │    │ • AAP-001 metadata │
│ • revoked_at    │    │                     │
│ • client_id     │    │ Performance:        │
│   (JSONB path)  │    │ • ~10ms/token       │
│ • grant_id      │    │ • <1ms storage      │
│ • authorization │    │ • 25 max conns      │
│   _chain JSONB  │    │ • 5 idle conns      │
│ • power_of      │    │ • 5min conn lifetime│
│   _attorney     │    │                     │
│   JSONB         │    │ Testing:            │
│ • compliance    │    │ • ✅ 5/5 tokens     │
│   _level        │    │ • ✅ Full JSONB     │
│ • ...           │    │ • ✅ All indexes    │
└─────────────────┘    └─────────────────────┘
```

### Environment Configuration

```bash
# Token Store Selection
export AGENTAUTH_TOKEN_STORE=postgres        # Options: memory, postgres (default: memory)

# PostgreSQL Connection
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=agentauth
export DB_USER=agentauth_user
export DB_PASSWORD=secure_password
export DB_SSLMODE=require               # For production
```

### Key Implementation Details

**PostgreSQL Array Handling** (Critical Fix):
```go
// lib/pq doesn't auto-convert Go slices to PostgreSQL arrays
// MUST use pq.Array() wrapper

// INSERT
_, err = db.ExecContext(ctx, query,
    token.AccessToken,
    pq.Array(token.Scope),  // ← Required for []string → text[]
    // ...
)

// SCAN
err := db.QueryRowContext(ctx, query, accessToken).Scan(
    &token.AccessToken,
    pq.Array(&scope),  // ← Required for text[] → []string
    // ...
)
```

**JSONB Storage for Complex Structures**:
```sql
-- Full AAP-001 authorization chains stored as JSONB
authorization_chain JSONB,      -- Complete auth chain with all entities
power_of_attorney JSONB,        -- PoA credential details
client_owner JSONB,             -- Client owner information
owners_authorizer JSONB,        -- Owner's authorizer details
verification_proof JSONB,       -- Cryptographic verification
restrictions JSONB,             -- Access restrictions
audit_trail JSONB,              -- Compliance audit trail

-- Efficient querying with JSONB path operators
CREATE INDEX idx_extended_tokens_client_id ON extended_tokens(((authorization_chain->'client'->>'entity_id')));
```

**Database Schema** (`schema/migrations/001_create_extended_tokens.sql`):
- 28 columns for complete AAP-001 token metadata
- 5 indexes for performance (client_id JSONB path, grant_id, issued_at, revoked_at partial, created_at)
- Nullable `issued_by` column (matches optional ExtendedToken.IssuedBy field)
- Auto-applied migrations on first connection

**Integration Pattern**:
```go
// Handler integration (non-blocking storage)
if h.tokenStore != nil && response.ExtendedToken != nil {
    if err := h.tokenStore.SaveToken(ctx, response.ExtendedToken); err != nil {
        c.Header("X-Token-Storage-Warning", "Token storage failed: "+err.Error())
        // Continue - return token even if storage fails
    }
}
```

### Testing Results

```sql
-- Verification query
SELECT COUNT(*) as total_tokens,
       COUNT(DISTINCT ((authorization_chain->'client'->>'entity_id')) as unique_clients
FROM extended_tokens;

-- Result: ✅
total_tokens: 5
unique_clients: 1

-- Sample token data
SELECT LEFT(access_token, 30) as token_prefix,
       token_type,
       expires_in,
       compliance_level
FROM extended_tokens LIMIT 3;

-- Result: ✅
token_prefix              | token_type | expires_in | compliance_level
--------------------------+------------+------------+------------------
agentauth_at_a6cca05dc8eb2... | Bearer     | 3600       | rfc-0111-compliant
agentauth_at_cba6bea9fea47... | Bearer     | 3600       | rfc-0111-compliant
agentauth_at_8d66456cdf96c... | Bearer     | 3600       | rfc-0111-compliant
```

**Performance Metrics**:
- Token creation: ~10ms per token
- Storage overhead: <1ms additional
- Concurrent handling: 5/5 requests successful
- Database size: ~2KB per token with full authorization chain

### Deployment

**Docker Compose** (Development):
```bash
docker-compose up -d
# Starts: PostgreSQL 16 + Redis 7 + Auto-migrations
```

**Production Setup**:
```bash
# 1. Set environment variables
export AGENTAUTH_TOKEN_STORE=postgres
export DB_HOST=prod-postgres.example.com
export DB_SSLMODE=require

# 2. Run migrations (auto-applied on first start)
# 3. Start web server
./bin/web-server
```

### Documentation

- **Setup Guide**: `docs/POSTGRESQL_SETUP.md` (850+ lines)
- **Schema**: `schema/migrations/001_create_extended_tokens.sql`
- **Implementation**: `pkg/agentauth/extended_token_store_postgres.go` (479 lines)
- **Tests**: Verified with 5 tokens, full JSONB data persistence

---

**Document**: Architecture Diagram
**Purpose**: Visual guide to implementation solution
**Status**: PostgreSQL persistence layer complete ✅
**Next**: Start with subscription_flow.go (Week 1-6)
