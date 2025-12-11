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
| RFC111-C2 | Bundle Substitution Detection | `VerifyChain` error returns | pkg/policy/registry_tamper_test.go | Implemented | Verified error reporting for substitutions |
| RFC111-C3 | Provenance Query Endpoint | `BetaServer` `apiPolicyProvenance` | web/policy_provenance_test.go | Implemented | Strict hash syntax validation added |
| RFC111-C4 | Decision Traceability | `audit.Append` with policy metadata | pkg/audit/trace_test.go, web/audit_policy_trace_test.go | Implemented | Verified multi-bundle replay traceability via integration test |
| RFC111-C5 | Delegation Chain Revocation Hash Linking | `RevocationChain` `RevocationChain.Append` `RevocationChain.Verify` `IsDelegationRevoked` | pkg/delegation/revocation_chain_test.go, pkg/delegation/revocation_anchor_test.go | Implemented | Added External Anchor Observer hook and verified via test |
| RFC111-C6 | Replay Protection | `ReplayNonceStore` `ReplayNonceStore.RecordWithEvict` `apiTokenCreate` `apiTokenValidate` | web/token_replay_test.go | Implemented | Verified JTI extraction and fail-closed mode |
| RFC111-C7 | Token Integrity (Public) | `Service.RequestToken` `Service.ValidateToken` `New` | web/token_public_integrity_test.go | Implemented | Verified RS256 JWT public key validation |
| RFC111-C8 | Policy Evaluation Combining | `policy.ChainEngine` strategies | pkg/policy/eval_combining_test.go | Implemented | Need conflict diagnostic metadata |

## RFC 0115 (Power-of-Attorney Definition)
| Clause ID | Title | Implementation Symbols | Tests | Status | Notes |
|-----------|-------|------------------------|-------|--------|-------|
| RFC115-C1 | Parties Structure | `poa.Parties`, `Principal`, `Organization`, `ClientOwnerInfo` | pkg/poa/parties_validation_test.go | Implemented | Verified strict structure validation |
| RFC115-C2 | Authorization Scope Encoding | `PowerOfAttorney` `Scope` `ValidateScopeNarrowing` | pkg/poa/scope_narrowing_test.go | Implemented | Verified regex/wildcard scope validation |
| RFC115-C3 | Validity Period | `PowerOfAttorney.ValidFrom` `PowerOfAttorney.ValidUntil` `ValidateDelegationCtx` | pkg/poa/validity_skew_test.go | Implemented | Verified validity period with clock skew tolerance |
| RFC115-C4 | Formal Requirements | `PowerOfAttorney.Requirements.FormalRequirements` | pkg/poa/formal_requirements_test.go | Implemented | Verified structure and serialization |
| RFC115-C5 | Power Limits | `Requirements.PowerLimits`, `EnforcePowerLimits` | pkg/poa/power_limits_test.go | Implemented | Verified numeric limits and logical consistency |
| RFC115-C6 | Rights & Obligations | `Requirements.RightsObligations`, `EnforceReportingCompliance` | pkg/poa/obligations_execution_test.go | Implemented | Verified reporting duty logic and rule structures |
| RFC115-C7 | Special Conditions | `SpecialConditions` | pkg/poa/special_conditions_test.go | Implemented | Verified persistence and mock interpreter extensibility |
| RFC115-C8 | Joint / Collective Signatures | `VerifyMultiSig` | pkg/poa/multi_signature_test.go | Implemented | Verified threshold logic and signature counting |
| RFC115-C9 | Canonical Serialization & Digest | `CanonicalPOADigest` | pkg/gauth_rfc_001/canonical_fuzz_test.go | Implemented | Verified via fuzzing (stability, invariance) |
| RFC115-C10 | Revocation Semantics | `RevocationChain`, `IsDelegationRevoked` | pkg/delegation/revocation_semantics_test.go | Implemented | Verified chain filtering and revocation checks |

## Cross-Cutting
| Area | Implementation | Tests | Gap |
|------|---------------|-------|-----|
| Error Taxonomy | Generic HTTP errors, `pkg/rfc/errs` | `pkg/poa/error_taxonomy_test.go` | Implemented | Standardized via leaf package |
| Metrics Export | `authz.PrometheusMetricsProvider` | `pkg/authz/metrics_labels_test.go` | Implemented | Verified Prometheus labels support |
| Discovery Metadata | `/.well-known/gauth-configuration` | `web/discovery_test.go` | Implemented | Added jwks_uri & revocation endpoints |

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
| RFC111-R1 | Revocation events SHALL be hash-linked to permit tamper detection | `RevocationChain.Append`, `RevocationChain.Verify` (hash + prev linkage) | `pkg/delegation/revocation_chain_test.go` | Implemented | Hash mismatch + prev link break tests present |
| RFC111-R2 | Revocation events MAY be cryptographically signed for authenticity | `RevocationEvent.signature`, `RevocationEvent.sig_kid`, signing in `Append` using `crypto.GlobalEdDSARegistry` | `pkg/crypto/keys_test.go` | Implemented | Added archival for expired keys |
| RFC111-R3 | Verification MUST fail if any signature is invalid or kid unknown | Signature branch in `RevocationChain.Verify` invoking `ValidateSignature` | pkg/delegation/revocation_chain_sig_test.go | Implemented | Verified tamper & unknown KID failure |
| RFC111-R4 | System SHOULD expose chain head & aggregate integrity digest | `/api/v1/token/revocation/head` & `RevocationChain.AggregateHash()` | web/revocation_endpoints_test.go | Implemented | Verified head & aggregate availability |
| RFC111-R5 | Verifier/RP MUST check per-event signature presence (if advertised) and validity | `Verify` integration | (web/revocation_endpoints_test.go) | Implemented |
| RFC111-R6 | Discovery MUST advertise `revocation_signing_alg_values_supported` | `discovery` endpoint | (web/revocation_endpoints_test.go) | Implemented |
| RFC111-R7 | Audit log MUST record each successful revocation append | `OnRevocationAppended` hook emitting AuditEntry | web/audit_revocation_test.go | Implemented |

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

---
