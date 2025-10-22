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
| RFC111-C1 | Policy Bundle Hash Chaining | `policy.Registry`, `AddBundle`, `VerifyChain` | test/policy_bundle_substitution_test.go | Partial | Need tamper multi-hop, collision simulation |
| RFC111-C2 | Bundle Substitution Detection | `VerifyChain` error returns | test/policy_bundle_substitution_test.go | Partial | Add explicit error code mapping |
| RFC111-C3 | Provenance Query Endpoint | `BetaServer` `apiPolicyProvenance` | web/policy_provenance_test.go | Partial | Add negative cases (unseeded registry) |
| RFC111-C4 | Decision Traceability | `audit.Append` with policy metadata | pkg/audit/trace_test.go | Partial | Need multi-bundle replay test |
| RFC111-C5 | Delegation Chain Revocation Hash Linking | `RevocationChain` `RevocationChain.Append` `RevocationChain.Verify` `IsDelegationRevoked` | pkg/delegation/revocation_chain_test.go | Partial | External anchor missing |
| RFC111-C6 | Replay Protection | `ReplayNonceStore` `ReplayNonceStore.RecordWithEvict` `apiTokenCreate` `apiTokenValidate` | web/token_replay_test.go | Partial | Needs real JTI extraction & fail-closed mode |
| RFC111-C7 | Token Integrity (Public) | `Service.RequestToken` `Service.ValidateToken` `New` | web/token_public_integrity_test.go | Partial | Switch to JWT/PASETO public mode |
| RFC111-C8 | Policy Evaluation Combining | `policy.ChainEngine` strategies | pkg/policy/eval_combining_test.go | Implemented | Need conflict diagnostic metadata |

## RFC 0115 (Power-of-Attorney Definition)
| Clause ID | Title | Implementation Symbols | Tests | Status | Notes |
|-----------|-------|------------------------|-------|--------|-------|
| RFC115-C1 | Parties Structure | `poa.Parties`, `Principal`, `Organization`, `ClientOwnerInfo` | pkg/rfc0111/rfc0111_strict_auth_test.go | Partial | Missing full validation coverage |
| RFC115-C2 | Authorization Scope Encoding | `PowerOfAttorney` `Scope` `CanonicalPOADigest` | pkg/rfc0111/canonical_test.go | Partial | Need advanced narrowing (range, regex) |
| RFC115-C3 | Validity Period | `PowerOfAttorney.ValidFrom` `PowerOfAttorney.ValidUntil` `ValidateDelegationCtx` | pkg/rfc0111/rfc0111_verify_token_test.go | Partial | Clock skew tests missing |
| RFC115-C4 | Formal Requirements | `PowerOfAttorney.Requirements.FormalRequirements` | (TODO formal_requirements_test.go) | Missing | Add requirement presence & serialization tests |
| RFC115-C5 | Power Limits | `Requirements.PowerLimits` | (TODO power_limits_test.go) | Missing | Numeric parsing & enforcement |
| RFC115-C6 | Rights & Obligations | `Requirements.RightsObligations` | (TODO obligations_execution_test.go) | Missing | Execution engine absent |
| RFC115-C7 | Special Conditions | `Requirements.SpecialConditions` | (TODO special_conditions_test.go) | Missing | Interpreter not implemented |
| RFC115-C8 | Joint / Collective Signatures | `RepresentationType`, placeholder count check | (TODO multi_signature_test.go) | Missing | Threshold & aggregated digest verification |
| RFC115-C9 | Canonical Serialization & Digest | `CanonicalPOADigest` | pkg/rfc0111/canonical_test.go, pkg/rfc0111/canonical_fuzz_test.go | Partial | Fuzz invariants: scope permutation, control char escaping, stable digest |
| RFC115-C10 | Revocation Semantics | (planned) `Revocation` struct | (TODO revocation_semantics_test.go) | Missing | Implement chain filter on verification |

## Cross-Cutting
| Area | Implementation | Tests | Gap |
|------|---------------|-------|-----|
| Error Taxonomy | Generic HTTP errors | (TODO error_taxonomy_test.go) | Missing structured codes |
| Metrics Export | `policyMetrics`, `authz.PrometheusHandler` | (TODO metrics_dimension_test.go) | Partial: missing action/resource labels |
| Discovery Metadata | `/.well-known/gauth-configuration` | `web/discovery_test.go` | Partial: lacks jwks_uri, revocation_endpoint |

## Planned Test Artifacts (TODO Files)
Create the following test files to achieve clause coverage:
- `web/policy_provenance_test.go`
- `policy/eval_combining_test.go`
- `delegation/revocation_chain_test.go`
- `pkg/rfc0111/formal_requirements_test.go`
- `pkg/rfc0111/power_limits_test.go`
- `pkg/rfc0111/obligations_execution_test.go`
- `pkg/rfc0111/special_conditions_test.go`
- `pkg/rfc0111/multi_signature_test.go`
- `pkg/rfc0111/revocation_semantics_test.go`
- `pkg/rfc0111/error_taxonomy_test.go`
- `pkg/rfc0111/metrics_dimension_test.go`

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
| RFC111-R2 | Revocation events MAY be cryptographically signed for authenticity | `RevocationEvent.signature`, `RevocationEvent.sig_kid`, signing in `Append` using `crypto.GlobalEdDSARegistry` | `pkg/delegation/revocation_chain_sig_test.go` | Partial | Key persistence & rotation continuity TBD |
| RFC111-R3 | Verification MUST fail if any signature is invalid or kid unknown | Signature branch in `RevocationChain.Verify` invoking `ValidateSignature` | `revocation_chain_sig_test.go` tamper case | Implemented | Unknown kid scenario pending dedicated test |
| RFC111-R4 | System SHOULD expose chain head & aggregate integrity digest | `/api/v1/token/revocation/head` & `RevocationChain.AggregateHash()` | `web` integration tests (TODO) | Partial | Add head tamper detection endpoint test |
| RFC111-R5 | System SHOULD expose per-event signature presence & validity | `/api/v1/token/revocation/verify` endpoint | `revocation_chain_sig_test.go` (indirect) | Partial | Need HTTP-level response validation test |
| RFC111-R6 | Discovery SHOULD advertise revocation signature capability | `revocation_support.signatures_enabled`, `signing_kids` in discovery handler | (TODO discovery_revocation_signatures_test.go) | Partial | Add discovery metadata assertion test |
| RFC111-R7 | Audit log MUST record each successful revocation append | `delegation.OnRevocationAppended` hook emitting `AuditEntry` | (TODO audit_revocation_append_test.go) | Partial | Validate meta fields & ordering |

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

### Planned Tests To Elevate Status
| Test File | Purpose |
|-----------|---------|
| `web/revocation_verify_endpoint_test.go` | Assert JSON fields (signature_present, signature_valid) and aggregate hash consistency |
| `web/discovery_revocation_signatures_test.go` | Ensure discovery lists signing_kids when signatures enabled |
| `audit/audit_revocation_append_test.go` | Validate audit entry meta population & sequence ordering |
| `delegation/revocation_unknown_kid_test.go` | Force signature with removed key to ensure verification failure |
| `delegation/revocation_chain_head_tamper_test.go` | Tamper last event to confirm head endpoint reports unverified |

### Gaps & Roadmap
1. Persistent key storage & historical signature retention (rotation continuity beyond process lifetime).
2. External anchoring of head or aggregate hash (transparency / notarization layer).
3. Merkle or incremental accumulator for efficient partial proofs.
4. Clause ID replacement with official RFC section numbering once authoritative text ingested.
5. End-to-end provenance query combining delegation chain + revocation chain signed proofs.

---
