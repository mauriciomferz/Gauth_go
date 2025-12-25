# RFC 0111 / 0115 Clause → Implementation → Test Map

Generated: 2025-10-17
Status: Draft (placeholder clause numbering until official spec ingestion)

Legend:
- Clause IDs temporary (e.g., RFC111-C1) – replace with actual section/subsection numbers.
- Status: Implemented | Partial | Missing
- Each row should eventually include line references (file:line-range) auto-updated by CI script.

## RFC 0111 (GAuth Protocol Core)
| Clause ID | Title | Implementation Symbols | Tests | Status | Notes |
|-----------|-------|------------------------|-------|--------|-------|
| RFC111-C1 | Policy Bundle Hash Chaining | `policy.Registry`, `AddBundle`, `VerifyChain` | pkg/policy/registry_tamper_test.go | Implemented | Verified multi-hop tamper detection |
| RFC111-C2 | Bundle Substitution Detection | `VerifyChain` | pkg/policy/registry_tamper_test.go | Implemented | Verified error reporting for substitutions |
| RFC111-C3 | Provenance Query Endpoint | `API`, `Provenance` | web/policy_provenance_test.go | Implemented | Strict hash syntax validation added |
| RFC111-C4 | Decision Traceability | `audit.Append` | pkg/audit/trace_test.go, web/audit_policy_trace_test.go | Implemented | Verified multi-bundle replay traceability via integration test |
| RFC111-C5 | Delegation Chain Revocation Hash Linking | `RevocationChain`, `Append`, `Verify`, `IsDelegationRevoked` | pkg/delegation/revocation_chain_test.go, pkg/delegation/revocation_anchor_test.go | Implemented | Added External Anchor Observer hook and verified via test |
| RFC111-C6 | Replay Protection | `ReplayNonceStore`, `RecordWithEvict`, `apiTokenCreate`, `ValidateToken` | web/token_replay_test.go | Implemented | Verified JTI extraction and fail-closed mode |
| RFC111-C7 | Token Integrity (Public) | `Service.RequestToken`, `Service.ValidateToken`, `New` | web/token_public_integrity_test.go | Implemented | Verified RS256 JWT public key validation |
| RFC111-C8 | Policy Evaluation Combining | `policy.ChainEngine` | pkg/policy/eval_combining_test.go | Implemented | Need conflict diagnostic metadata |

## RFC 0115 (Power-of-Attorney Definition)
| Clause ID | Title | Implementation Symbols | Tests | Status | Notes |
|-----------|-------|------------------------|-------|--------|-------|
| RFC115-C1 | Parties Structure | `poa.Parties`, `Principal`, `Organization`, `ClientOwnerInfo` | pkg/poa/parties_validation_test.go | Implemented | Verified strict structure validation |
| RFC115-C2 | Authorization Scope Encoding | `PowerOfAttorney`, `Scope`, `ValidateScopeNarrowing` | pkg/poa/scope_narrowing_test.go | Implemented | Verified regex/wildcard scope validation |
| RFC115-C3 | Validity Period | `ValidityPeriod`, `StartTime`, `EndTime`, `ValidateDelegationCtx` | pkg/poa/validity_skew_test.go | Implemented | Verified validity period with clock skew tolerance |
| RFC115-C4 | Formal Requirements | `FormalRequirements` | pkg/poa/formal_requirements_test.go | Implemented | Verified structure and serialization |
| RFC115-C5 | Power Limits | `Requirements`, `PowerLimits`, `EnforcePowerLimits` | pkg/poa/power_limits_test.go | Implemented | Verified numeric limits and logical consistency |
| RFC115-C6 | Rights & Obligations | `Requirements`, `RightsObligations`, `EnforceReportingCompliance` | pkg/poa/obligations_execution_test.go | Implemented | Verified reporting duty logic and rule structures |
| RFC115-C7 | Special Conditions | `SpecialConditions` | pkg/poa/special_conditions_test.go | Implemented | Verified persistence and mock interpreter extensibility |
| RFC115-C8 | Joint / Collective Signatures | `VerifyMultiSig` | pkg/poa/multi_signature_test.go | Implemented | Verified threshold logic and signature counting |
| RFC115-C9 | Canonical Serialization & Digest | `CanonicalPOADigest` | pkg/gauth_rfc_001/canonical_fuzz_test.go | Implemented | Verified via fuzzing (stability, invariance) |
| RFC115-C10 | Revocation Semantics | `RevocationChain`, `IsDelegationRevoked` | pkg/delegation/revocation_semantics_test.go | Implemented | Verified chain filtering and revocation checks |

## Cross-Cutting
| Clause ID | Feature | Implementation Symbols | Tests | Status | Notes |
|-----------|---------|------------------------|-------|--------|-------|
| Cross-1 | Error Taxonomy | `pkg/rfc/errs` | pkg/poa/error_taxonomy_test.go | Implemented | Standardized via leaf package |
| Cross-2 | Metrics Export | `authz.PrometheusMetricsProvider` | pkg/authz/metrics_labels_test.go | Implemented | Verified Prometheus labels support |
| Cross-3 | Discovery Metadata | `/.well-known/gauth-configuration` | web/discovery_test.go | Implemented | Added jwks_uri & revocation endpoints |

## CI Conformance Script (Planned)
Script will:
1. Parse this map.
2. Verify each Implementation Symbol exists (via `go list -json` + grep).
3. Check test file presence for each clause with Status != Missing.
4. Fail build if any Implemented/Partial clause lacks at least one test reference.

## Next Steps
- Replace placeholder clause IDs with actual RFC section numbers once spec text imported.
- Add line-level evidence references using generator script.
- Implement missing tests and update statuses accordingly.

---
Maintainers: Update after each feature merge affecting RFC-related structures.

## Revocation Provenance Mapping (Phase 2 Addendum)
| Clause (Placeholder) | Normative Intent (Summary) | Implementation Artifacts | Tests | Status | Notes |
|----------------------|----------------------------|--------------------------|-------|--------|-------|
| RFC111-R1 | Revocation events SHALL be hash-linked to permit tamper detection | `RevocationChain`, `Append`, `Verify` | `pkg/delegation/revocation_chain_test.go` | Implemented | Hash mismatch + prev link break tests present |
| RFC111-R2 | Revocation events MAY be cryptographically signed for authenticity | `RevocationEvent`, `Signature`, `SigKid`, `crypto.GlobalEdDSARegistry` | `pkg/crypto/keys_test.go` | Implemented | Added archival for expired keys |
| RFC111-R3 | Verification MUST fail if any signature is invalid or kid unknown | `RevocationChain`, `Verify`, `ValidateSignature` | pkg/delegation/revocation_chain_sig_test.go | Implemented | Verified tamper & unknown KID failure |
| RFC111-R4 | System SHOULD expose chain head & aggregate integrity digest | `RevocationChain`, `AggregateHash` | web/revocation_endpoints_test.go | Implemented | Verified head & aggregate availability |
| RFC111-R5 | Verifier/RP MUST check per-event signature presence (if advertised) and validity | `RevocationChain`, `Verify` | web/revocation_endpoints_test.go | Implemented |
| RFC111-R6 | Discovery MUST advertise `revocation_signing_alg_values_supported` | `registerRB3Discovery` | web/revocation_endpoints_test.go | Implemented |
| RFC111-R7 | Audit log MUST record each successful revocation append | `RevocationChain`, `OnRevocationAppended` | web/audit_revocation_test.go | Implemented |

### Evidence Trace
Implementation lines (approximate):
- Hash linking & verification: `pkg/delegation/revocation_chain.go` lines ~30-140.
- Signature fields & signing: `revocation_chain.go` lines ~18-40, ~60-90.
- Signature verification logic: `revocation_chain.go` lines ~96-120.
- Aggregate hash: `revocation_chain.go` lines ~180-210.
- JWKS / key exposure: `web/server_clean.go` JWKS handler lines (search `/.well-known/jwks.json`).
- Discovery metadata injection: `web/server_clean.go` lines ~500-530.
- Verification API: `web/server_clean.go` (search `revocation/verify`).
- Audit hook registration: `web/server_clean.go` lines ~200-230.


### Gaps & Roadmap
1. Persistent key storage & historical signature retention (rotation continuity beyond process lifetime).
2. External anchoring of head or aggregate hash (transparency / notarization layer) - **Implemented (File-based prototype)**.
3. Merkle or incremental accumulator for efficient partial proofs - **Implemented (Verified)**.
4. Clause ID replacement with official RFC section numbering once authoritative text ingested.
5. End-to-end provenance query combining delegation chain + revocation chain signed proofs - **Implemented (Verified)**.
6. GNAP Integration (RFC 9635) - **Implemented (Verified)**.

---

## RFC 9635 (GNAP - Grant Negotiation and Authorization Protocol)
| Clause ID | Title | Implementation Symbols | Tests | Status | Notes |
|-----------|-------|------------------------|-------|--------|-------|
| RFC9635-§3 | Grant Request | `pkg/gnap.GrantRequest`, `GrantRequest` | pkg/gnap/gnap_test.go | Implemented | Full grant request handling |
| RFC9635-§4 | Grant Response | `pkg/gnap.GrantResponse`, `GrantState`, `AccessToken` | pkg/gnap/gnap_test.go | Implemented | Response construction with continuation |
| RFC9635-§5 | Grant Continuation | `Continue`, `ContinueUpdate`, `ContinueCancel` | pkg/gnap/gnap_test.go | Implemented | Full continuation lifecycle |
| RFC9635-§6 | Token Management | `pkg/gnap.TokenStore`, `TokenRotate`, `TokenRevoke` | pkg/gnap/gnap_test.go | Implemented | Rotation and revocation |
| RFC9635-§7.3 | HTTP Signature Binding | `pkg/gnap/httpsig.Signer`, `Verifier` | pkg/gnap/httpsig/signer_test.go | Implemented | Ed25519, ECDSA, RSA support |
| RFC9635-§9 | AS Discovery | `DiscoveryResponse`, `discovery` | web/discovery_manual_test.go | Implemented | Metadata endpoint |

### GNAP Implementation Artifacts
- Core Types: `pkg/gnap/types.go`
- Grant Store: `pkg/gnap/store.go` (in-memory)
- Token Store: `pkg/gnap/token_store.go` (in-memory)
- Interaction: `pkg/gnap/interaction.go`
- HTTP Signatures: `pkg/gnap/httpsig/signer.go`
- HTTP Handlers: `web/handlers/gnap/handler.go`
- Client Example: `examples/gnap_client/`

### GAuth Extensions to GNAP
- `PowerOfAttorneyRef` - Links GNAP grants to PoA credentials
- `AuthorizationChain` - Recursive delegation attestation per RFC 0111
- `ComplianceLevel` - Policy compliance attestation

---

## RFC 8628: OAuth 2.0 Device Authorization Grant

| Section | Feature | Conformance | Implementation Notes |
| :--- | :--- | :--- | :--- |
| **3.1** | Device Authorization Request | ✅ Full | `POST /device/authorize` |
| **3.2** | Device Authorization Response | ✅ Full | Returns `device_code`, `user_code`, `verification_uri` |
| **3.3** | User Interaction | ✅ Full | User enters code at `verification_uri` |
| **3.4** | Device Access Token Request | ✅ Full | `POST /device/token` with polling |
| **3.5** | Device Access Token Response | ✅ Full | Issues access token on approval |

**Implementation Status:**
- `pkg/device` implements core types and store.
- `web/handlers/device` implements protocol endpoints.
- `web/server_factory.go` integrates handlers.
- `web/templates/device_verify.html` provides user interface.

---

## A2A Authorization Profile (Draft)

| Section | Feature | Conformance | Implementation Notes |
| :--- | :--- | :--- | :--- |
| **Draft** | Agent Identity | ✅ Full | `pkg/a2a.AgentIdentity` |
| **Draft** | Call Chain | ✅ Full | `pkg/a2a.A2ACallContext`, `CallHop` |
| **Draft** | Chain Integrity | ✅ Full | Hash-linked hops with `ComputeHash` |
| **Draft** | Transaction Token | ✅ Full | `POST /a2a/token` issues linked tokens |

**Implementation Status:**
- `pkg/a2a` implements chain builder and validation logic.
- `web/handlers/a2a` implements token issuance and verification endpoints.
- Implements "Agent-to-Agent Authorization" patterns for AI workflows.

---

## RFC 7523: JWT Profile for OAuth 2.0 Client Auth & Grants

| Section | Feature | Conformance | Implementation Notes |
| :--- | :--- | :--- | :--- |
| **2.2** | Private Key JWT Authentication | ✅ Full | Uses `client_assertion` signed by private key. Implemented in `pkg/auth/client_auth.go` (`PrivateKeyJWTValidator`). |
| **2.1** | JWT Bearer Token Grant | ✅ Full | `urn:ietf:params:oauth:grant-type:jwt-bearer` supported at `/oauth/token`. |
| **3** | JWT Format & Processing | ✅ Full | Validates `iss`, `sub`, `aud`, `exp`, `jti`. |

**Implementation Status:**
- `pkg/auth` implements validator logic.
- `web/handlers/grant_jwt` implements the grant type handler.
- `web/handlers/device` and `a2a` support the client authentication method.



## RFC 9396: Rich Authorization Requests (RAR)
| Clause ID | Title | Implementation Symbols | Tests | Status | Notes |
|-----------|-------|------------------------|-------|--------|-------|
| RFC9767-3 | Authorization Details Data Model | `gauth.AuthorizationDetail` | pkg/gauth/rar_validator_test.go | Implemented | Types and JSON serialization |
| RFC9767-5 | Authorization Request | `RFCCompliantAuthorizationRequest` | web/handlers/token/rfc_integration_test.go | Implemented | 'authorization_details' parameter support |
| RFC9767-6 | Token Response | `ExtendedToken`, `ExtendedTokenService` | web/handlers/token/rfc_integration_test.go | Implemented | Details persisted in token and JWT |
| RFC9767-7 | Resource Server Validation | `RARValidator`, `ValidateExtendedTokenWithRAR` | pkg/gauth/resource_server_test.go | Implemented | Enforces PoA scope narrowing |
```
