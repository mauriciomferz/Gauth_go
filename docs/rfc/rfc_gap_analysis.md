# RFC 111 & RFC 115 Gap Analysis (Beta Snapshot)

Date: 2025-10-26
Scope: Implementation state of delegation (RFC111) and multi-signature / serialization (RFC115) features in `main` branch.

## Legend
- IMPLEMENTED: Feature present with tests & instrumentation.
- PARTIAL: Core logic present; missing tests, docs, or edge handling.
- MISSING: Not yet implemented.
- DEFERRED: Explicitly postponed (out of beta scope).

## RFC 111 (Delegation & Validation)
| Requirement | Status | Evidence | Gap / Notes |
|-------------|--------|----------|-------------|
| Canonical POA digest deterministic across permutations | IMPLEMENTED | `canonical_permutation_test.go` | Control char escape covered; add fuzzing later (DEFERRED). |
| Replay protection (nonce/JTI) for tokens | IMPLEMENTED | `web/token_replay_nonce_test.go` | Consider distributed store integration test (PARTIAL). |
| Revocation chain integrity verification | IMPLEMENTED | Integrity checks in `ValidateDelegationRich`; revocation tests | External anchoring of revocation chain missing (PARTIAL). |
| Immediate invalidation on revocation | IMPLEMENTED | Revocation integration tests | N/A |
| Structured error taxonomy delegation/token endpoints | PARTIAL | `respondError` usage in anchor/token/BLS endpoints | Apply to all legacy endpoints (e.g. status update, audit append). |
| Scope enforcement (exact membership) | IMPLEMENTED | `ValidateDelegationRich` + semantic counters | Advanced pattern narrowing (regex/prefix/range) MISSING. |
| Restriction enforcement (max_amount, daily limits) | IMPLEMENTED | `rfc0111_daily_limit_test.go` | Multi-dimensional restrictions (currency/jurisdiction combos) MISSING. |
| Audit chain determinism / replay | IMPLEMENTED | `pkg/audit/replay_test.go` | Add external anchor observer test (PARTIAL). |
| Canonical serialization versioning | IMPLEMENTED | Canonical digest includes `version` | Formal version negotiation doc MISSING. |
| Clock skew tolerance (valid_from/valid_until grace) | IMPLEMENTED | `rfc0111_clock_skew_test.go` | Config docs need OpenAPI param (PARTIAL). |
| Multi-signature threshold semantics (structural) | IMPLEMENTED | `ValidateMultiSignature`, `verifyMultiSignatures` | Public multi-sig issuance/aggregation endpoint for POA signatures MISSING. |
| Weighted threshold verification | IMPLEMENTED | `verifyMultiSignatures` weighted path | Need explicit metrics counters for satisfied weight vs threshold (PARTIAL). |
| Semantic rejection counters (scope/restriction) | IMPLEMENTED (internal) | `semanticCounters` struct increments | Exposed endpoint `/api/v1/diagnostics/semantic` MISSING. |
| Rights & obligations serialization | MISSING | N/A | Introduce arrays in POA & enforcement hooks. |
| External anchoring of capability + revocation chain tips | PARTIAL | Combined anchor endpoint for capability; revocation not yet | Add revocation anchor emission + OpenAPI. |

## RFC 115 (Multi-Signature & Enhanced Serialization)
| Requirement | Status | Evidence | Gap / Notes |
|-------------|--------|----------|-------------|
| Domain separation & version switching (single vs multi-sig) | IMPLEMENTED | Canonical digest conditional fields | V2 transition doc MISSING. |
| Canonical JSON includes weights when multi-sig | IMPLEMENTED | `canonical_version_weights_test.go` | Add negative test for missing weight (PARTIAL). |
| Stable ordering of scope, restrictions, weights | IMPLEMENTED | Permutation test | Add ordering test for signers list (PARTIAL). |
| Digest mismatch classification (domain_conflict, tamper_suspected) | IMPLEMENTED | Metrics increments in `verifyPOASignature` | Expose mismatch reasons via diagnostics endpoint (PARTIAL). |
| Proof-of-Possession challenge issuance & verification | IMPLEMENTED | `/api/v1/crypto/bls/aggregate` `require_pop`, PoP verify endpoint | Document challenge construction formula (PARTIAL). |
| Multi-sig verification latency metrics | IMPLEMENTED | Metrics calls in aggregate/verify BLS endpoint | Provide histogram buckets (PARTIAL). |
| Weighted signature aggregation success metadata | IMPLEMENTED | `SatisfiedWeight`, `SatisfiedSignatures` | Surface in response or separate endpoint (PARTIAL). |
| Error taxonomy with RFC references | PARTIAL | `respondError` in BLS/PoP endpoints | Apply across multi-sig integrity failures (structured mapping). |
| Control character escape integrity | IMPLEMENTED | Permutation test control characters | Add explicit test for other escapes (\r, \b) (PARTIAL). |
| Grace window (clock skew) for signature validity | IMPLEMENTED | Delegation validation logic | Add config doc & metrics for skew usage (PARTIAL). |

## Cross-Cutting Observability
| Area | Status | Gap |
|------|--------|-----|
| Metrics counters (revoked, expired, scope violations) | IMPLEMENTED | Need one Prometheus doc page detailing all counters. |
| Latency measurements (multi-sig aggregate/verify) | IMPLEMENTED | Add percentile export / histogram. |
| Replay store error metrics | PARTIAL | Fail-closed toggle present; add error counter for store failures. |
| Audit chain external anchoring events | PARTIAL | Observer interface present; implement anchor callback test. |
| Semantic counters exposure | MISSING | Add endpoint + OpenAPI schema. |

## Highest Priority Gaps (Beta Readiness)
1. Complete error taxonomy coverage (delegation creation, validation, token verification, audit endpoints).
2. Scope pattern narrowing + tests (prefix, wildcard, regex, numeric range).
3. External revocation anchor emission & OpenAPI documentation.
4. Semantic counters diagnostics endpoint.
5. Multi-sig POA issuance & signer collection endpoint.
6. Rights & obligations serialization & minimal enforcement hook.
7. OpenAPI enhancements (clock skew param, mismatch reasons, multi-sig integrity failure codes).

## Deferred (Post-Beta)
- Fuzzing canonical digest stability.
- Formal version negotiation handshake for canonical digest versions.
- Extended control character / unicode normalization suite.
- Advanced jurisdiction/currency composite restrictions.

---
Generated automatically; update as features land.
