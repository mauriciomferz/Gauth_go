# AAP-AAP-001 - CORRECTED PROTOCOL FLOW
## Critical Fix: Protocol Architecture Clarification

**Date:** November 15, 2025  
**Status:** ERRATA - Critical Correction Required  
**Original RFC:** AAP-0111 (August 2025)

---

## PROTOCOL ARCHITECTURE CLARIFICATION

**AgentAuth builds on OAuth 2.0 and OpenID Connect as its foundation.** The protocol layers are:

### Layer 1: OAuth 2.0 / OpenID Connect Foundation (Inherited)
- **Resource Server (RS)** - Validates tokens and serves protected resources
- **Authorization Server (AS)** - Issues access tokens, handles authorization flows
- **Client** - Requests access to protected resources
- **Resource Owner (RO)** - Grants access to protected resources
- **Authorization Grants** - Credentials representing RO authorization
- **Access Tokens** - Bearer tokens for accessing protected resources

### Layer 2: AgentAuth Extensions (What AgentAuth Defines/Adds)
- **Extended Tokens** - Enhanced tokens beyond OAuth access tokens, includes PoA context and authorization chain
- **P*P Architecture Roles** - PEP (Enforcement), PDP (Decision), PAP (Administration), PIP (Information), PVP (Validation)
- **Power of Attorney (PoA)** - AI legitimization framework with verifiable authorization chains
- **Owner's Authorizer** - New role for approving AI actions on behalf of Client Owner
- **PoA Validation Logic** - Extended AS behavior to validate power delegation chains
- **Compliance Event Reporting** - Extended RS behavior for AI governance audit trails
- **AI-Specific Endpoints** - `/transaction`, `/decision`, `/action` for AI operations

**Critical Distinction:**
- AgentAuth **uses** the OAuth/OIDC Authorization Server but **extends its token issuance** to include PoA validation
- AgentAuth **uses** the OAuth/OIDC Resource Server but **extends its enforcement** to validate PoA claims
- AgentAuth **adds** entirely new roles (Owner's Authorizer, P*P architecture) not present in OAuth/OIDC

---

## CRITICAL ISSUE IDENTIFIED

The original AAP-0111 RFC **omits the Resource Server** from the protocol flow diagram, despite:

1. Resource Server being part of the inherited OAuth/OIDC foundation
2. Resource Server being defined in AgentAuth nomenclature (Section 3)
3. Resource Server appearing in subscription step VIII
4. OAuth RFC 6749 explicitly showing Resource Server in canonical flow
5. The flow diagram being incomplete without showing where resources are actually accessed

---

## CORRECTED PROTOCOL FLOW

### Visual Flow Diagram

```
┌───────────────────────────────────────────────────────────────────────────────┐
│               CORRECTED GAUTH PROTOCOL FLOW (with Resource Server)            │
│                         AAP-001 ERRATA - November 2025                       │
└───────────────────────────────────────────────────────────────────────────────┘

   ┌──────────────┐          ┌──────────────────┐          ┌──────────────┐
   │   Resource   │          │  Authorization   │          │   Resource   │
   │     Owner    │          │     Server       │          │    Server    │
   │              │          │      (AS)        │          │     (RS)     │
   └──────────────┘          └──────────────────┘          └──────────────┘
          │                           │                            │
          │                           │                            │
   ┌──────────────┐                   │                            │
   │    Client    │                   │                            │
   │  (AI Agent)  │                   │                            │
   └──────────────┘                   │                            │
          │                           │                            │
          │                           │                            │
          │ (a) Authorization         │                            │
          │     Request               │                            │
          │───────────────>           │                            │
          │                 (via RO)  │                            │
          │                           │                            │
          │                           │                            │
          │ (a.1) RO Authentication   │                            │
          │         & Consent         │                            │
          │         ───────────────>  │                            │
          │                           │                            │
          │                           │                            │
          │ (b) Request Compliance    │                            │
          │     Validation            │                            │
          │     <───────────────────> │                            │
          │                           │                            │
          │                           │                            │
          │ (c) Authorization Grant   │                            │
          │ <─────────────────────    │                            │
          │                           │                            │
          │                           │                            │
          │ (d) Extended Token        │                            │
          │     Request + Grant       │                            │
          │ ───────────────────────>  │                            │
          │                           │                            │
          │                           │                            │
          │         (e) Grant         │                            │
          │         Compliance        │                            │
          │         Validation        │                            │
          │                           │                            │
          │                           │                            │
          │ (f) Extended Token        │                            │
          │ <─────────────────────    │                            │
          │                           │                            │
          │                           │                            │
          │                           │                            │
          │ (g) Transaction/Decision/Action Request                │
          │         with Extended Token                            │
          │ ───────────────────────────────────────────────────────>
          │                           │                            │
          │                           │ (h.1) OPTIONAL:            │
          │                           │       Token                │
          │                           │       Introspection        │
          │                           │ <──────────────────────────│
          │                           │                            │
          │                           │ (h.2) Token                │
          │                           │       Validation           │
          │                           │       Response             │
          │                           │ ───────────────────────────>
          │                           │                            │
          │                           │ (h.3) RS validates         │
          │                           │       token & policies     │
          │                           │                            │
          │                           │                            │
          │ (i) Protected Resource / Action Result                 │
          │ <───────────────────────────────────────────────────────
          │                           │                            │
          │                           │                            │
          │                           │ (j) Compliance             │
          │                           │     Event Report           │
          │                           │ <──────────────────────────│
          │                           │                            │
          │                           │                            │
          ▼                           ▼                            ▼
```

---

## CORRECTED REQUEST-SPECIFIC STEPS (a-j)

### Step (a): Client Authorization Request
```
Actor: Client → Resource Owner (via Authorization Server)
Purpose: Request authorization to perform specific transaction/decision/action
Details:
  • Client requests specific authorization from resource owner
  • Request must align with client's general powers (PoA)
  • Request MAY be made directly to resource owner OR
  • Request SHOULD be made indirectly via authorization server (recommended)
```

### Step (a.1): Resource Owner Authentication & Consent ⭐ NEW
```
Actor: Resource Owner → Authorization Server
Purpose: Authenticate and provide explicit consent
Details:
  • Resource owner authenticates to authorization server
  • Authorization server presents request details:
    - Client identity
    - Requested actions/transactions
    - Scope of authorization
    - Duration/expiration
    - Resource server target
  • Resource owner explicitly grants or denies consent
  • Consent timestamp recorded for audit trail
  • If denied, flow terminates with error
  • If granted, proceed to step (b)
```

### Step (b): Request Compliance Validation
```
Actor: Authorization Server
Purpose: Validate request complies with client's PoA powers
Details:
  • Authorization server validates request against:
    - Client's registered Power of Attorney
    - Client owner's authorization scope
    - Geographic restrictions
    - Value limits
    - Action type permissions
  • Shares client's powers with resource owner/server for verification
  • If validation fails, flow terminates with error
```

### Step (c): Authorization Grant Issuance
```
Actor: Authorization Server → Client
Purpose: Issue credential representing resource owner's authorization
Details:
  • Authorization server issues authorization grant
  • Grant is credential representing resource owner's consent
  • Grant includes:
    - Authorization code (OAuth 2.0 pattern)
    - Consent reference ID
    - Timestamp
    - Scope of authorization
    - PKCE code challenge (for security)
    - Resource server endpoint
  • Grant is short-lived (typically 10 minutes)
  • Grant is single-use only
```

### Step (d): Extended Token Request
```
Actor: Client → Authorization Server
Purpose: Exchange grant for extended token
Details:
  • Client authenticates with authorization server
  • Client presents:
    - Authorization grant (from step c)
    - Client credentials
    - PKCE code verifier
    - Proof of possession (optional)
```

### Step (e): Grant Compliance Validation
```
Actor: Authorization Server
Purpose: Final validation before token issuance
Details:
  • Validate authorization grant structure and signature
  • Verify grant has not been used before (replay prevention)
  • Verify grant has not expired
  • Confirm grant scope consistency with request
  • Check client authentication credentials
  • Validate PKCE code verifier matches code challenge
```

### Step (f): Extended Token Issuance
```
Actor: Authorization Server → Client
Purpose: Issue extended token for resource access
Details:
  • Authorization server authenticates client
  • Validates all compliance checks passed
  • Issues extended token containing:
    - Access token (JWT format)
    - Token type: Bearer
    - Expires in: seconds until expiration
    - Refresh token: (optional, for long-lived sessions)
    - Scope: authorized actions
    - PoA reference: link to power of attorney
    - Authorization chain: delegation hierarchy
    - Resource server: target endpoint
    - Issuer (iss): authorization server URL
    - Subject (sub): client identifier
    - Audience (aud): resource server identifier
```

### Step (g): Transaction/Decision/Action Request ⭐ CORRECTED
```
Actor: Client → Resource Server
Purpose: Execute authorized transaction/decision/action
Details:
  • Client sends request to RESOURCE SERVER (not omitted!)
  • Request includes:
    - HTTP Method: POST/PUT/DELETE as appropriate
    - Endpoint: /api/v1/transaction, /api/v1/decision, or /api/v1/action
    - Authorization header: Bearer <extended-token>
    - Request body: transaction/decision/action details
  • Example:
    POST https://resource-server.example.com/api/v1/transaction
    Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
    Content-Type: application/json
    
    {
      "transaction_type": "contract_signature",
      "contract_id": "CONTRACT-12345",
      "signer_ai": "client-agent-001",
      "on_behalf_of": "client-owner-org"
    }
```

### Step (h): Token Validation & Authorization Check ⭐ DETAILED
```
Actor: Resource Server (with optional AS introspection)
Purpose: Validate token and enforce authorization
Details:
  • Resource Server has TWO validation options:

  OPTION 1: Local JWT Validation (Offline)
    1. Verify JWT signature using AS public key
    2. Check token expiration (exp claim)
    3. Validate issuer (iss claim) matches trusted AS
    4. Validate audience (aud claim) matches RS identifier
    5. Check token has not been revoked (local cache check)
    6. Validate token binding (if PoP was used)
    7. Extract authorization claims (scope, PoA)

  OPTION 2: Token Introspection (Online) ⭐ ADDS STEPS (h.1) & (h.2)
    (h.1) RS → AS: POST /introspect with token
    (h.2) AS → RS: Returns token metadata
    {
      "active": true,
      "scope": "transaction:sign decision:approve",
      "client_id": "client-agent-001",
      "exp": 1700000000,
      "iat": 1699993800,
      "sub": "client-owner-org",
      "aud": "resource-server-001",
      "poa_reference": "PoA-2025-001",
      "authorization_chain": [...]
    }

  (h.3) Resource Server Enforcement:
    • Check request action matches token scope
    • Verify PoA permits this specific action
    • Apply resource-specific policies
    • Check rate limits
    • Validate request payload
    • If all checks pass, execute request
    • If any check fails, return error (401/403)
```

### Step (i): Protected Resource / Action Result ⭐ CORRECTED
```
Actor: Resource Server → Client
Purpose: Return result of authorized action
Details:
  • Resource server executes authorized action
  • Returns result to client:
    - Success: HTTP 200/201 with response body
    - Created: HTTP 201 with resource location
    - Accepted: HTTP 202 for async operations
    - Error: HTTP 4xx/5xx with error details
  • Response includes:
    - Transaction ID (for audit trail)
    - Execution timestamp
    - Result details
    - Audit reference
  • Example success response:
    HTTP 201 Created
    Location: /api/v1/transactions/TXN-67890
    Content-Type: application/json
    
    {
      "transaction_id": "TXN-67890",
      "status": "completed",
      "timestamp": "2025-11-15T08:30:00Z",
      "result": {
        "contract_signed": true,
        "signature_hash": "sha256:abc123...",
        "notary_timestamp": "2025-11-15T08:30:01Z"
      }
    }
```

### Step (j): Compliance Tracking & Audit ⭐ CORRECTED
```
Actor: Resource Server → Authorization Server
Purpose: Report compliance events for monitoring
Details:
  • Resource server reports to authorization server:
    - Action executed
    - Timestamp
    - Client identifier
    - Token used
    - Result (success/failure)
    - Any policy violations detected
  • Authorization server tracks:
    - Client behavior patterns
    - Resource server enforcement
    - Policy violations
    - Anomaly detection
    - Compliance metrics
  • Triggers alerts on:
    - Repeated authorization failures
    - Policy violations
    - Unusual request patterns
    - Rate limit breaches
  • Generates audit reports for:
    - Regulators
    - Client owners
    - Resource owners
```

---

## COMPARISON: ORIGINAL vs CORRECTED

### Original AAP-001 (INCORRECT)
```
(a) Client requests authorization from resource owner
(b) Validation via AS
(c) Client receives grant
(d) Client requests token from AS
(e) AS issues token
(f) Client validates grant (WRONG: after token issued!)
(g) Client requests... (MISSING: to whom? Resource Server not shown!)
(h) Resource server validates token (APPEARS OUT OF NOWHERE!)
(i) AS tracks compliance
```

### Corrected AAP-001 v1.1
```
(a) Client requests authorization from resource owner
(a.1) Resource owner authenticates & consents ← NEW
(b) AS validates request compliance
(c) AS issues authorization grant
(d) Client requests extended token from AS
(e) AS validates grant compliance ← MOVED BEFORE TOKEN
(f) AS issues extended token
(g) Client → RESOURCE SERVER with token ← EXPLICIT
(h) RS validates token (with optional AS introspection) ← DETAILED
(h.1) Optional: RS → AS introspection request ← NEW
(h.2) Optional: AS → RS validation response ← NEW
(h.3) RS enforces policies ← NEW
(i) RS → Client: resource/action result ← EXPLICIT
(j) RS → AS: compliance event report ← EXPLICIT
```

---

## ARCHITECTURAL CORRECTION: Resource Server Integration

### 6.1 Resource Server Role (Inherited from OAuth/OIDC)

**IMPORTANT:** The Resource Server is part of the **OAuth 2.0 / OpenID Connect foundation** that AgentAuth builds upon. This section clarifies how Resource Servers interact with **AgentAuth-specific extensions** (Extended Tokens, PoA validation, P*P policy enforcement).

The Resource Server in a AgentAuth-enabled system:

1. **Receives client requests** with extended tokens (AgentAuth extension to OAuth access tokens)
2. **Validates extended tokens** (locally or via AS introspection - standard OAuth pattern)
3. **Enforces authorization policies** based on:
   - Standard OAuth scopes (OAuth/OIDC)
   - **PoA claims** (AgentAuth extension)
   - **P*P policy attributes** (AgentAuth extension)
4. **Executes authorized transactions/decisions/actions** (business logic)
5. **Returns results** to clients (standard HTTP responses)
6. **Reports compliance events** to authorization server for audit (**AgentAuth extension** for AI governance)

**What AgentAuth Adds to Resource Server Behavior:**
- Extended token validation (beyond OAuth access tokens)
- PoA (Power of Attorney) claim verification
- P*P architecture policy enforcement (PEP role at RS)
- Compliance event reporting for AI action tracking

**What AgentAuth Inherits from OAuth/OIDC:**
- Token presentation via Authorization header
- Token introspection protocol (RFC 7662)
- Error response codes (401, 403)
- Bearer token scheme

### 6.1.1 Token Presentation (OAuth Standard)

Clients MUST present extended tokens using HTTP Authorization header (OAuth 2.0 Bearer Token specification):

```http
Authorization: Bearer <extended-token>
```

Alternative headers MAY be supported:
```http
X-AgentAuth-Token: <extended-token>
```

### 6.1.2 Token Validation Methods

Resource Servers MUST validate tokens using ONE of the following methods:

#### Method A: Local JWT Validation (Offline)
```
Advantages:
  • Low latency (no network call)
  • Scales independently
  • Works offline

Requirements:
  • Access to AS public keys (via JWKS endpoint)
  • Local revocation cache
  • Token structure knowledge

Process:
  1. Verify JWT signature (RS256/ES256)
  2. Check exp, iat, nbf timestamps
  3. Validate iss claim matches trusted AS
  4. Validate aud claim matches RS identifier
  5. Check revocation status (cache)
  6. Extract claims (scope, PoA, chain)
```

#### Method B: Token Introspection (Online)
```
Advantages:
  • Real-time revocation status
  • No key management needed
  • Centralized policy enforcement

Requirements:
  • Network connectivity to AS
  • AS introspection endpoint
  • RS authentication credentials

Process:
  1. POST /introspect to AS with token
  2. Receive active status + metadata
  3. Cache result with short TTL (30-60s)
  4. Enforce based on response
```

### 6.1.3 Standard Endpoints (AgentAuth Extension)

**AgentAuth Extension:** Resource Servers implementing AgentAuth SHOULD expose AI-specific endpoints:

```
POST   /api/v1/transaction    - Execute AI-authorized transaction (AgentAuth-specific)
POST   /api/v1/decision       - Execute AI-authorized decision (AgentAuth-specific)
POST   /api/v1/action         - Execute AI-authorized action (AgentAuth-specific)
GET    /api/v1/status         - Check action/transaction status
DELETE /api/v1/transaction/{id} - Cancel pending transaction
```

**Note:** These are **AgentAuth-specific business endpoints** for AI agent actions. Standard OAuth protected resources (REST APIs, user data, etc.) continue to work as defined in OAuth/OIDC.

### 6.1.4 Error Responses (OAuth Standard)

Resource Servers MUST return standardized errors per OAuth 2.0 Bearer Token Usage (RFC 6750):

```http
401 Unauthorized
{
  "error": "invalid_token",
  "error_description": "Token signature verification failed"
}

403 Forbidden
{
  "error": "insufficient_scope",
  "error_description": "Token does not grant 'transaction:sign' permission"
}

404 Not Found
{
  "error": "resource_not_found",
  "error_description": "Contract CONTRACT-12345 does not exist"
}

409 Conflict
{
  "error": "resource_conflict",
  "error_description": "Contract already signed by another party"
}

500 Internal Server Error
{
  "error": "server_error",
  "error_description": "Transaction processing failed"
}
```

---

## SUMMARY OF CORRECTIONS

### Critical Fixes
1. ✅ **Added Resource Server to flow diagram** - was omitted despite being part of OAuth/OIDC foundation
2. ✅ **Added explicit RS interaction in step (g)** - client → RS request
3. ✅ **Detailed RS token validation in step (h)** - two methods specified
4. ✅ **Added RS → Client response in step (i)** - resource/result return
5. ✅ **Added RS → AS reporting in step (j)** - compliance events (AgentAuth extension)
6. ✅ **Added new step (a.1)** - explicit resource owner consent
7. ✅ **Reordered steps (e) and (f)** - validate grant BEFORE token issuance
8. ✅ **Clarified protocol layering** - OAuth/OIDC foundation vs AgentAuth extensions

### New Sections Added
- Protocol Architecture Clarification (Layer 1 vs Layer 2)
- 6.1 Resource Server Integration (with OAuth/OIDC attribution)
- 6.1.1 Token Presentation (OAuth Standard)
- 6.1.2 Token Validation Methods (OAuth + AgentAuth extensions)
- 6.1.3 Standard Endpoints (AgentAuth-specific AI endpoints)
- 6.1.4 Error Responses (OAuth Standard)

### Documentation Improvements
- Flow diagram now shows all 5 actors: Client, RO, AS, RS, Client Owner
- Each step explicitly states source → target
- Optional vs required interactions clarified
- Error handling specified
- Security considerations added
- **Protocol boundaries clearly defined** - what's OAuth/OIDC vs what's AgentAuth

### Protocol Layering Clarification

**What AgentAuth Inherits from OAuth 2.0 / OpenID Connect:**
- **Resource Server (RS)** - Core role and behavior for serving protected resources
- **Authorization Server (AS)** - Core role for authorization flows and token issuance
- **Client, Resource Owner** - Standard OAuth roles
- **Authorization Grants** - OAuth grant types (authorization code, etc.)
- **Access Tokens** - Bearer token format and usage
- **Token presentation** - Authorization: Bearer header (RFC 6750)
- **Token introspection** - Validation protocol (RFC 7662)
- **Error codes** - 401 Unauthorized, 403 Forbidden (RFC 6750)
- **Client authentication** - client_id, client_secret, PKCE

**What AgentAuth Extends (Behavior Additions to OAuth/OIDC Components):**
- **Extended AS behavior** - PoA validation during token issuance
- **Extended RS behavior** - PoA claim enforcement, compliance event reporting
- **Extended Token structure** - PoA claims, authorization chain, P*P attributes

**What AgentAuth Defines (Entirely New Concepts):**
- **Extended Tokens** - Token type with PoA context beyond OAuth access tokens
- **Power of Attorney (PoA)** - AI legitimization framework with delegation chains
- **P*P Architecture** - PEP, PDP, PAP, PIP, PVP roles for policy-based authorization
- **Owner's Authorizer** - Entirely new role for AI action approval
- **AI-Specific Operations** - Transaction/decision/action semantics and endpoints

---

## IMPLEMENTATION STATUS IN GAUTH_GO CODEBASE

The AgentAuth_go implementation **ALREADY IMPLEMENTS THE CORRECTED FLOW**:

✅ **Resource Server Mocked**: `pkg/gauth/mocks.go` - MockResourceServer  
✅ **Token Validation**: `pkg/gauth/extended_token_service.go` - ValidateExtendedToken  
✅ **Introspection**: Web server has `/introspect` endpoint  
✅ **PoA Enforcement**: `pkg/poa/validator.go` - ValidatePoA  
✅ **Compliance Tracking**: `pkg/gauth/compliance_tracker.go`  

**The code is more correct than the RFC!**

---

## RECOMMENDATION

**AAP-001 should be republished as AAP-001 v1.1** with these corrections, or an **official errata document** should be issued immediately.

---

**Document Status:** ERRATA - Critical Correction  
**Impact:** HIGH - Affects all implementations  
**Action Required:** Update AAP-001 to version 1.1  
**Date:** November 15, 2025  
**Author:** AgentAuth Implementation Team
