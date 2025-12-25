---
title: Beta Readiness Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Beta Readiness Plan (RFC 0111 & RFC 0115 Compliance)

## Objective
Deliver a demonstrably compliant, transparent, and resilient beta implementation of core RFC0111 (delegation, revocation, cryptographic integrity, replay protection) and RFC0115 (Power of Attorney structure, multi-signature semantics, canonical serialization, validity period) for live demo usage.

## Scope In Beta
Included:
- Canonical PoA digest (version + weights, domain-separated)
- Threshold + weighted multi-signature enforcement
- Mandatory JTI (fail-closed unless override)
- Replay protection (in-memory + metrics, WAL scheduled for remediation)
- Revocation chain inclusion proof + trivial consistency endpoint (Phase 1)
- Detached PoA signature (Ed25519)
- Rotation summary chain with metrics
- Auditor CLI modes (revocation, multisig, PoA, consistency sizes)

Excluded (Documented Roadmap):
- Algorithm agility beyond Ed25519
- Partial / suspension revocation semantics
- Delegation depth limits
- Advanced obligations/advice engine
- OTEL tracing and full distributed telemetry

## Clause Compliance Summary (Beta-Critical)
| RFC | Clause | Status | Evidence | Beta Action |
|-----|--------|--------|----------|-------------|
| 0111 | Multi-Signature Threshold | Implemented | `pkg/rfc0111/rfc0111.go` verifyMultiSignatures | None |
| 0111 | Replay Protection | Partial | `web/server_clean.go` JTI checks; `web/replay_store.go` | Add WAL & persistence tests |
| 0111 | Cryptographic Requirements | Partial | Canonical digest Ed25519 only | Add agility interface (stub ECDSA) |
| 0111 | Audit Logging | Partial | Hash chain ledger | Add per-entry signature |
| 0111 | Delegation & Revocation | Partial | Chain + basic revoke | Placeholder for partial/suspend |
| 0115 | PoA Structure | Partial | Embedded version & weights | Add verifier helper + size guard |
| 0115 | Validity Period | Implemented | Canonical digest excludes mutable fields | None |
| 0115 | Canonical Serialization | Implemented | `pkg/rfc0111/canonical.go` + tests | None |
| 0115 | Joint Signatures | Implemented | Weighted + threshold logic | None |

## Remediation Tasks (P0/P1)
| ID | Task | Priority | Deliverable | Success Criteria |
|----|------|----------|-------------|-----------------|
| T1 | Durable Replay Store (WAL+Snapshot) | P0 | `internal/replay/wal_store.go` | Recovery retains JTI; duplicate rejected post-restart |
| T2 | Signature Algorithm Interface | P0 | `internal/crypto/signalgo/*.go` | Ed25519 + stub ECDSA selectable; digest stable |
| T3 | Policy Bundle Signed Manifest | P1 | `internal/policy/manifest_sign.go` | Tampering detected; signature test passes |
| T4 | Ledger Entry Signatures | P1 | Extend entry struct + verification | Modified entry fails verification |
| T5 | Embedded PoA Verifier Helper | P1 | `pkg/rfc0111/embedded_verify.go` | Oversize PoA rejected; digest equivalence confirmed |
| T6 | Discovery Endpoint | P1 | `/well-known/gauth/config` handler | Returns algorithms, required claims consistent with runtime config |

## Sequence for Implementation
1. T2 (interface) before T1 (replay WAL) to freeze digest format.
2. T1 durable replay to secure token lifecycle early.
3. T4 ledger signatures to enhance audit integrity before demo data capture.
4. T3 bundle manifest to align policy governance narrative.
5. T5 embedded verifier to simplify external validation.
6. T6 discovery endpoint for interoperability showcase.

## Demo Narrative Alignment
- Multi-signature issuance (weights + threshold) -> canonical digest shown.
- Token creation with embedded PoA and JTI -> verification success.
- Replay attempt (same token) -> fail closed replay taxonomy.
- Revocation event -> inclusion proof + auditor verification.
- Consistency sizes endpoint -> trivial proof demonstration + roadmap mention.
- Discovery endpoint -> external agent retrieves config (algorithms, strict JTI enforcement).
- Ledger inspection -> entry signatures + chain head hash.

## Testing Additions
| Area | Test Type | File (Planned) |
|------|-----------|----------------|
| Replay WAL | Crash recovery integration | `web/replay_wal_recovery_test.go` |
| Algo Agility | Digest invariance cross-algo | `pkg/rfc0111/algorithm_agility_test.go` |
| Ledger Entry Sig | Negative mutation test | `pkg/ledger/entry_signature_test.go` |
| Manifest Signing | Tamper detection test | `internal/policy/manifest_sign_test.go` |
| Embedded PoA | Size cap + digest equivalence | `pkg/rfc0111/embedded_verify_test.go` |
| Discovery | Schema + value correctness | `web/discovery_endpoint_test.go` |

## Metrics & Observability Enhancements (Post-Remediation)
- Add counters: `gauth_replay_wal_recoveries_total`, `gauth_replay_wal_errors_total`
- Histogram: `gauth_token_validation_latency_seconds` (ensure p95 tracked)
- Gauge: `gauth_ledger_head_age_seconds` after signature integration
- OTEL spans (future): create_poA, verify_multi_sig, validate_token, generate_revocation_proof

## Acceptance Criteria
- All T1–T6 tasks merged & tests green.
- p95 token validation latency < 25ms (local benchmark).
- Replay WAL recovery test passes with retained JTI set.
- Digest invariance tests confirm no regression when ECDSA stub disabled/enabled.
- Discovery endpoint reflects runtime flags (authenticity, algorithms, JTI requirement).
- Compliance matrix updated (replay protection moved to Implemented; cryptographic requirements partial w/ agility). 

## Rollback / Safety
- Each remediation feature gated by env flags (`GAUTH_ENABLE_WAL`, `GAUTH_ENABLE_ECDSA_STUB`, etc.).
- Canonical digest version remains unchanged unless multi-sig semantics altered.
- Clear migration note appended to `CHANGELOG.md` for new mandatory manifest signature.

## Documentation Updates
- `docs/ARCHITECTURE.md`: Add replay WAL diagram.
- `docs/DEMO_FLOW.md`: Stepwise script referencing endpoints & expected outputs.
- `docs/CHANGELOG.md`: Beta readiness section.

---
Generated: 2025-10-26
