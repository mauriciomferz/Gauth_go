---
title: Roadmap Next Sprint
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Next Sprint Roadmap (Target Start: 2025-10-27)

## Objective
Elevate cryptographic agility, external verifiability, and interoperability while reducing partial gaps in replay and lifecycle controls.

## Prioritized Work Items (P0 / P1)
| ID | Priority | Item | Description | Success Criteria |
|----|----------|------|-------------|------------------|
| CRYPTO-ALG-01 | P0 | Signature Algorithm Abstraction | Introduce interface + registry for Ed25519, ECDSA P-256, BLS aggregate | PoA issuance configurable; tests for each algo; canonical unchanged |
| LEDGER-ANCHOR-01 | P1 | External Anchoring | Sign audit entries + periodic Merkle root publish | `anchor_publish_test.go` passing; CLI anchor verification |
| REVOCATION-EXT-01 | P1 | Partial Revocation & Suspension | Add `suspended`, `partially_revoked` statuses + validation logic | State transitions tested; canonical digest unaffected |
| OPENAPI-EXPORT-01 | P1 | OpenAPI/Discovery | Generate OpenAPI spec + `/well-known/agentauth/config` endpoint | Spec file + endpoint integration tests |
| REPLAY-DURABLE-01 | P1 | Durable Replay Store | Persistent JTI index with TTL + compaction snapshot | Recovery test after restart retains anti-replay guarantees |

## Secondary (P2)
- Tracing: OTEL spans on delegation create/validate.
- Property tests for enhanced PoA semantic validator.
- Depth limit enforcement (max chaining) configurable.

## Risks & Mitigations
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Algorithm plugin complexity | Delays release | Start with internal interface + simple factory |
| External anchoring latency | Slower issuance | Async anchoring + fallback logging |
| Replay store size growth | Memory/disk pressure | Periodic compaction & TTL purge |

## Metrics Additions
- `delegation_suspensions_total`
- `signature_algorithm_usage{algo}` gauge
- `replay_jti_persist_latency_seconds` histogram

## Definition of Done
- All P0/P1 items merged with passing tests & docs updated.
- CHANGELOG updated with new features.
- Conformance run shows reduction in Missing items by ≥3.

## References
- Current compliance matrix: `docs/aap001_compliance_matrix.md`
- Latest release notes: `docs/RELEASE_NOTES_2025-10-25.md`
