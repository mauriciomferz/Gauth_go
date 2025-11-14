---
title: RFC-0111 Implementation Architecture
category: architecture
status: active
lastUpdated: 2025-11-12
owners: architecture-team
---
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

(Existing content continues – unchanged from original.)
