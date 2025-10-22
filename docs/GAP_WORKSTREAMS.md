# GAP Closure Workstreams

Generated: 2025-10-22

This document groups RFC gap matrix items into actionable workstreams with priority focus (P0/P1 fast-track). Each gap lists its original ID for traceability.

## Summary of Fast-Track Scope
P0 objective is to establish cryptographic completeness, core PoA validation, governance integrity, durable auditability, secret management, and conformance coverage. P1 extends to distributed/rollback features, embedding semantics, capability & replay hardening, interoperability enrichment.

## Workstream 1: Cryptographic Foundations (P0)
Focus: Algorithm flexibility, claims completeness, parser robustness, token integrity extensions.
- sec1.item1 (alg expansion): Add ECDSA (P-256), optional BLS aggregate; config selectable algs.
- sec1.item2 (claims completeness): Enforce typ, aud set semantics, structured footer metadata, optional custom claims registry.
- sec1.item3 (robust parsing): Replace manual scan with streaming JSON decoder + strict validation; maintain fuzz/property tests.
- sec1.item5 (public token integrity extensions): Multi-alg detached signatures + enforcement flag + batch signature test vectors.
Acceptance dependency: canonical digest stability (sec1.item6 already implemented) provides foundation.

## Workstream 2: Authorization Engine Enhancements (P0/P1)
- P0: sec2.item1 (richer conflict diagnostics), sec2.item2 (extensible ABAC function registry).
- P1: sec2.item4 (policy versioning & rollback metadata).
- P2: sec2.item3 (obligations/advice persistent channel), sec2.item5 (distributed PDP + cache invalidation).

## Workstream 3: PoA & Delegation Core (P0/P1)
- P0: sec3.item1 (semantic validation expansion beyond BasicPoAValidator).
- P1: sec3.item2 (embed full PoA in token), sec3.item3 (joint/collective signature enforcement), sec12.item2 (delegation chaining depth limits).
- P2: sec3.item4 (conditional runtime interpreter).

## Workstream 4: Persistence & Ledger Integrity (P0/P1)
- P0: sec5.item1 (immutable audit ledger: add signature chaining + external anchor integration).
- P1: sec6.item1 (durable replay store with fail-closed mode), sec8.item2 (rotation audit trail external sink), sec11.item2 (external chain entry notarization for model limits evidence).
- P2: sec5.item2 (delegation storage indexing/pruning), sec5.item3 (revocation anchoring external notarization), sec6.item3 (replay persistence recovery WAL snapshot).

## Workstream 5: Secret & Key Management (P0/P1)
- P0: sec8.item1 (secure secret storage provider: Vault/HSM abstraction layer).
- P1: Extend multi-tenant segregation for keys, rotation segregation (sec8.item2 remaining aspects).

## Workstream 6: Model Governance & Capability Enforcement (P1)
- P1: sec11.item1 (capability policy binding + multi-tenant segregation), advanced rate algorithms (token bucket / leaky bucket) extension of sec11.item2 gap list.
- P2: Streaming head events public receipt publication (remaining governance enhancements).

## Workstream 7: Testing & Conformance (P0/P1/P2)
- P0: sec9.item1 (expand clause-to-test mapping to remaining RFC sections).
- P1: sec9.item2 (property tests for parsing & semantic validators).
- P2: sec9.item3 (load/stress benchmark harness).

## Workstream 8: Interoperability & API (P1/P2)
- P1: sec10.item1 (OpenAPI completion: error schemas, provenance/audit endpoints).
- P2: sec10.item2 (JWKS integrity signature + deprecation metadata fields).

## Workstream 9: Observability & Metrics (P2/P3)
- P2: Enrich decision metrics taxonomy (sec7.item1 remaining), semantic snapshot archival & surge alert hooks (sec7.item3 remaining gaps).
- P3: sec7.item2 (collector registration framework), sec7.item4 (distributed tracing integration).

## Workstream 10: Compliance & Jurisdiction (P1/P2/P3)
- P1: sec4.item1 (jurisdiction-specific branching).
- P2: sec4.item2 (compliance attestation ingestion pipeline).
- P3: sec4.item3 (arbitration / dispute hooks).

## Workstream 11: Replay & Token Security (P1/P2)
- P1: sec6.item1 (fail-closed with durable store) overlaps Workstream 4.
- P2: sec6.item3 (recovery via WAL snapshot), sec6.item2 remaining skew checks.

## Workstream 12: Data Hygiene & Risk (P2/P3)
- P2: sec13.item2 (structured numeric limits multi-period + currency conversion + audit persistence), sec14.item1 (threat model synchronization with mitigations matrix).
- P3: sec13.item1 (UTF-8 control char filtering metrics), sec14.item2 (residual risk register tracking).

## Fast-Track (P0) Aggregate List
1. sec1.item1 algorithm expansion.
2. sec1.item2 claims completeness.
3. sec1.item3 robust parser replacement + fuzz/property tests backup.
4. sec1.item5 multi-alg detached signatures + enforcement flag.
5. sec2.item1 conflict diagnostics.
6. sec2.item2 ABAC function registry.
7. sec3.item1 PoA semantic validation expansion.
8. sec5.item1 signed & anchored immutable audit ledger.
9. sec8.item1 secret storage provider.
10. sec9.item1 comprehensive clause-to-test mapping.
11. sec11.item2 external chain entry notarization + streaming head events (governance evidence).

## Dependencies & Ordering Notes
- Multi-alg signatures (sec1.item1 & sec1.item5) precede joint signature enforcement (sec3.item3).
- Robust parser (sec1.item3) precedes property tests expansion (sec9.item2).
- Audit ledger signing (sec5.item1) precedes external publication & notarization in governance (sec11.item2) and revocation anchoring (sec5.item3).
- Secret storage provider (sec8.item1) precedes HSM integration & key segregation (sec8.item2).
- Claims completeness (sec1.item2) feeds replay store durable design (sec6.item1) for JTI semantics.

## Success Metrics (Preliminary)
- 100% P0 gaps closed with passing tests & documented artifacts.
- Multi-alg signature suite test vectors (Ed25519/ECDSA) passing + property fuzz no panics for 1M iterations.
- Audit ledger external anchor interval <= 60s with verifiable hash chain signature.
- Secret provider supports at least filesystem (dev) + Vault (prod) backends.
- Clause map coverage: 100% of enumerated RFC sections.

## Next Step
Define closure criteria per gap (acceptance tests, artifacts, metrics) to be captured in GAP_CLOSURE_PLAN.md.
