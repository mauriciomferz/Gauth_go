---
title: Compliance AAP-001 AAP-002
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth AAP-001 & AAP-002 Compliance Matrix (Beta)

This document inventories implemented features and gaps for AAP-001 (core protocol, governance, anchoring, replay & multi-signature) and AAP-002 (semantic diagnostics, anomaly detection, integrity chain) strictly from repository evidence.

## Summary

| Area | AAP-001 Status | AAP-002 Status | Notes |
|------|---------------|---------------|-------|
| Token Validation / Replay | Implemented (nonce + JTI replay) | N/A (no semantic tie-in) | Added structured respondError codes (token_replay_detected, token_invalid_signature). |
| Model Limits & Rate | Implemented (per-user, global, audit hooks) | Potential future semantic feed | Comprehensive error taxonomy tagged AAP-001:model_limits. |
| Multi-Signature (BLS) | Implemented & tagged | N/A | Issue/verify modes, participant ceiling, base64 validation. |
| Attestation Verify | Basic payload validation & tagging | N/A | Minimal coverage; could expand cryptographic assertions. |
| External Anchoring | Implemented (capability + rotation coalesce) | N/A | Idempotent append, authorization & store checks. |
| Capability Anchoring & Audit | Implemented & tagged | N/A | Registry hash & audit chain tip anchoring. |
| Rotations | Minimal tagging (type mismatch) | N/A | Additional invariants could be tagged. |
| Revocation Anchor Emission | Implemented & tagged | N/A | Merkle root emission + idempotent anchoring. |
| Semantic Diagnostics | Not tagged previously | Implemented (now tagged) | Counters, rates (60s/300s), EWMA z-scores, history, integrity chain. |
| Integrity Chain (Semantic) | Internal only previously | Implemented (exposed) | prev_hash/current_hash plus integrity_status surfaced. |
| Error Taxonomy Uniformity | Mostly standardized | N/A | jwtError legacy path replaced; all use respondError for failures. |

## AAP-001 Evidence (Selected)

- `web/server_clean.go`: multiple `respondError(..., "AAP-001:<clause>")` for: pop, model_limits, attestation_verify, external_anchor, multi_sig, capabilities_* , capability_anchor, rotations, revocation_anchor.
- Replay protection: strict JTI mode and duplicate detection mapped to `AAP-001:replay_protection` via token_replay_detected.

## AAP-002 Evidence (Added)

- Semantic diagnostics handler annotated with comments: `AAP-002:semantic_diagnostics`, `AAP-002:anomaly_detection`, `AAP-002:integrity_chain`.
- OpenAPI `docs/openapi.yaml` diagnostics path: example error references `AAP-002:semantic_diagnostics`; schema extended with `prev_hash`, `current_hash`.

## New/Updated Error Codes

| Code | Error | HTTP | RFC Ref | Scenario |
|------|-------|------|---------|----------|
| token_invalid_signature | invalid_signature | 401 | AAP-001:token_validation | Signature mismatch. |
| token_replay_detected | replay_detected | 401 | AAP-001:replay_protection | JTI duplicate in strict mode. |
| token_invalid_algorithm | invalid_algorithm | 400 | AAP-001:token_validation | Unsupported / mismatched alg. |
| token_expired | token_expired | 401 | AAP-001:token_validation | Exp beyond skew tolerance. |
| token_malformed | malformed_token | 400 | AAP-001:token_validation | Structural or missing JTI (strict). |

## Remaining Gaps / Future Work

1. Attestation verification cryptographic depth (currently only JSON shape validation tagged).
2. Rotation ledger invariants (only one error path tagged). Add checks for signature set completeness, sequence monotonicity.
3. Semantic metrics unavailability path not implemented (OpenAPI documents 500 example). Provide conditional error when internal maps disabled.
4. Cross-link semantic anomaly scores to proactive revocation or throttling decisions (AAP-002 extension).
5. Additional persistence integrity verification (detect mismatch vs expected prev_hash).

## Testing Status

- Unwired semantic diagnostics test: validates structure when service absent.
- (To Add) Wired semantic diagnostics test: ensures history accumulation, anomaly score non-zero after induced rate changes, integrity chain hashes stable across snapshots.

## Integrity Chain Logic

- Hash computation: sorted key iteration of counters; SHA256 over `key=value;` sequence.
- Chain exposure: `prev_hash`, `current_hash`, `integrity_status` in diagnostics payload.
- Status states: `ok` | `mismatch` | `legacy` | `unconfigured` (currently only `ok` / `unconfigured` used).

## Demo Readiness Improvements

1. Add wired diagnostics test and small load generator to produce non-zero scores.
2. Extend diagrams with semantic integrity & anomaly feedback loop.
3. Provide one sample signed capability registry anchor artifact in `examples/` for transparency demonstration.
4. Add revocation anchor Merkle path example (inclusion proof) for a revoked token.
5. Include short README section mapping live endpoints to RFC clauses for audience clarity.

## Conclusion

AAP-001 coverage is broad and well-tagged post standardization. AAP-002 foundational features (semantic diagnostics, anomaly detection, integrity hashing) now surfaced and tagged; remaining work focuses on deeper integration (reactive controls) and test expansion.
