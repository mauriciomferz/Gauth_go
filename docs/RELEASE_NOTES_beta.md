---
title: Release Notes Beta
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth Beta Release Notes

> Version: Beta (2025-10-26)
> Scope: RFC-0111 / RFC-0115 Compliance Enhancements

## Summary

This beta release focuses on deepening integrity, rotation invariants, revocation transparency, and reactive semantic safeguards aligned with RFC-0111 (Attestation & Transparency) and RFC-0115 (Proof-of-Authorization Definition).

## Key Additions

### Attestation Integrity (RFC-0111)
- Canonical Ed25519 attestation signature verification with explicit error taxonomy.
- Structured error codes: `attestation_signature_invalid`, `attestation_kid_unknown`, `attestation_registry_unavailable`.
- Tests: success path and invalid signature rejection.

### Rotation Summary & Invariants
- Continuity enforcement (`prev_hash` chain validation) and signature integrity.
- Signed rotation summary artifact with canonical serialization prefix: `GAUTH_ROTATION_SUMMARY:`.
- Error codes: `rotation_continuity_gap`, `rotation_signature_missing`, `rotation_signature_invalid`.
- Anchoring optional via `GAUTH_ANCHOR_ROTATIONS=1` (single anchoring per head hash).

### Semantic Integrity Chain
- Persistent hash continuity file enabling detection of tampering or divergence.
- Integrity status exposed via diagnostics endpoint with mismatch code path.
- Test simulates persisted hash tamper → mismatch flag.

### Reactive Semantic Throttle (RFC-0115 Resilience Extension)
- Z-score anomaly detection (EWMA + Welford variance) triggers throttle state.
- Throttle metadata fields returned in diagnostics: `semantic_throttle_active`, threshold, current score.
- Demo endpoint `/api/v1/beta/throttle/demoAction` returns RFC-coded 429 denial when active.

### Revocation Transparency
- Inclusion proof example (`revocation_inclusion_proof.json`) documenting Merkle verification workflow.
- Revocation anchor emission endpoint guarded by chain state; duplicate suppression for identical Signed Tree Heads.
- Error codes: `revocation_chain_empty`, `revocation_anchor_client_unavailable`.

### Cryptographic Artifacts & Documentation
- Mermaid diagrams for: attestation verification, rotation invariants, diagnostics feedback loop, revocation inclusion flow, anchoring.
- Multi-signature rotation summary conceptual artifact (`rotation_summary_multisig.json`) for forward-looking threshold design.

## New Environment Flags
| Flag | Purpose |
|------|---------|
| `GAUTH_ROTATIONS_SIGN=1` | Enables signing rotation descriptors & summary. |
| `GAUTH_ANCHOR_ROTATIONS=1` | Anchors rotation summary once per unique head hash. |
| `GAUTH_SEMANTIC_INTEGRITY_PERSIST_PATH` | File path for persisted hash chain continuity. |
| `GAUTH_SEMANTIC_ANOMALY_Z_THRESHOLD` | Z-score threshold that activates semantic throttle. |
| `GAUTH_ROTATION_LEDGER_PATH` | Path for rotation ledger persistence. |

## Testing Overview
| Test | Focus |
|------|-------|
| `attestation_integrity_test.go` | Signature validation + failure mode. |
| `rotation_invariants_test.go` | Continuity gap, missing signature, valid signature. |
| `rotation_summary_endpoint_test.go` | Anchoring single-attempt semantics. |
| `semantic_integrity_mismatch_test.go` | Persistence tamper detection. |
| `semantic_throttle_test.go` | Reactive throttle activation & RFC error path. |
| `revocation_autosign_test.go` | Duplicate suppression for signed tree heads. |
| `anchor_revocation_emit_test.go` | Anchor emission success & failure cases. |
| `revocation_inclusion_artifact_test.go` | Structural validation of inclusion proof artifact. |

## Error Taxonomy Expansion
All new errors return structured JSON `{ code, message, rfc_ref, ... }` improving automated correlation:
- Attestation: `attestation_signature_invalid`, `attestation_kid_unknown`.
- Rotation: `rotation_continuity_gap`, `rotation_signature_missing`, `rotation_signature_invalid`.
- Revocation: `revocation_chain_empty`, `revocation_anchor_client_unavailable`.
- Reactive Controls: `semantic_throttle_active` (429 path).

## Observability
- Prometheus exposition for revocation auto-sign counters (emitted, skipped_empty, skipped_duplicate).
- OTEL gauges for rotation summary generation & anchoring metrics.
- Diagnostics loop surfaces semantic integrity & throttle state for external monitoring.

### Multi-Signature Rotation Summary (RFC-0115 Extension)
Runtime now supports multi-signature emission for rotation summaries when enabled:
| Flag | Behavior |
|------|----------|
| `GAUTH_ROTATIONS_MULTISIG=1` | Aggregates signatures from all active Ed25519 keys into `Signatures[]`. |
| `GAUTH_ROTATIONS_THRESHOLD` | Optional integer >0 declaring required signature quorum; if unmet returns 400 `rotation_threshold_unsatisfied`. |

`RotationSummary` fields added:
| Field | Purpose |
|-------|---------|
| `Signatures[]` | Array of `{kid, signature}` entries (canonical ordering). |
| `Threshold` | Configured required signature weight/quorum. |
| `SatisfiedWeight` | Count of valid signatures collected at emission time. |

Backward compatibility: If multisig disabled the legacy single `Signature` field is still populated for existing clients.

Tests:
| Test | Focus |
|------|-------|
| `rotation_multisig_test.go` | Happy path: multiple signatures & threshold satisfied. |
| `rotation_multisig_threshold_unsatisfied_test.go` | Negative path: threshold > available signatures returns structured error. |

Error Taxonomy Addition:
| Code | Condition |
|------|-----------|
| `rotation_threshold_unsatisfied` | Requested quorum exceeds available valid signatures. |

### Forward-Looking Design
- Adaptive anomaly thresholds, Merkle proof cryptographic verification enhancements.
- Multi-signer governance policy evolution (weight classes, key retirement scheduling).

## Upgrade & Integration Notes
1. Enable signing/anchoring via environment flags before production-like benchmarking.
2. Persist semantic integrity file on durable storage to benefit from mismatch detection across restarts.
3. Load JWKS early for rotation summary signature verification client-side.
4. Monitor throttle activations to tune `GAUTH_SEMANTIC_ANOMALY_Z_THRESHOLD`.

## Known Limitations
- Inclusion proof siblings truncated (documentation artifact only).
- Multi-signature summary not emitted by current endpoint (conceptual JSON only).
- Throttle scoring uses simplified EWMA; no adaptive decay yet.

## Security Considerations
- Canonical JSON payload signing reduces ambiguity exploits.
- Persistence mismatch detection guards against silent state rewrites.
- Duplicate suppression prevents replay amplification of identical signed roots.
- Structured RFC codes facilitate automated policy reaction (e.g., escalated audit triggers).

## Next Recommended Enhancements
| Item | Rationale |
|------|-----------|
| Full Merkle proof verification test | Strengthen revocation trust chain validation. |
| Threshold signature verification logic | Prepare for multi-signer governance rollouts. |
| Adaptive anomaly windowing | Reduce false positives under bursty legitimate load. |
| Audit trail hashing | Extend integrity chain to audit log entries. |

---
For architecture references see: `docs/diagrams.md`.
