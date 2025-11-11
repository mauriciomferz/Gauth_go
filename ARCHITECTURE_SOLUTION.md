# RFC-0111 Implementation Architecture
## Visual Guide to the Solution

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         RFC-0111 COMPLIANT GAUTH                            │
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
│    │     → Returns RFC-0111 ExtendedToken ✓                           │     │
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
│                    Return RFC-0111 ExtendedToken                            │
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
│  - ValidateRequestCompliance() ❌     │  ← 200 lines, never called
│  - ValidateGrantCompliance() ❌       │  ← 150 lines, never called
│  - ValidateAuthorizationChain() ❌    │  ← 720 lines, never called
│  - ValidateFormalRequirements() ❌    │  ← 814 lines, never called
│  - CreateExtendedToken() ❌           │  ← 400 lines, never called
└──────────────────────────────────────┘
```

**Result**: OAuth server, NOT GAuth. No RFC compliance.

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

**Result**: True RFC-0111 GAuth implementation. ✓

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
   │     token_type: "GAuth-Extended",  │
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
pkg/gauth/
├── gauth.go                          (EXISTING - UPDATE)
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
❌ OAuth server, not GAuth
❌ RFC compliance: 58/100
```

### After (Fixed)
```
✅ Subscription flow (Steps I-VIII)
✅ Protocol orchestrator (Steps a-i)
✅ Extended tokens with PoA metadata
✅ All validation functions integrated
✅ True RFC-0111 GAuth implementation
✅ RFC compliance: 95+/100
```

---

## Key Takeaway

> **We have all the pieces. We just need to connect them.**

The code quality is excellent. The validation functions are comprehensive. 
The data structures are perfect. We're not rewriting from scratch.

**We're adding 3 new files (~1,400 lines) to orchestrate the existing ~5,000 
lines of validation code into an RFC-0111 compliant flow.**

---

**Document**: Architecture Diagram  
**Purpose**: Visual guide to implementation solution  
**Next**: Start with subscription_flow.go (Week 1-6)
