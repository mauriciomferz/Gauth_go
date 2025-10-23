# GAuth RFC Gap Matrix (Generated)

> Generated: 2025-10-23T18:02:09Z

**Status Summary:** Implemented=8 | Partial=23 | Missing=12 | Conceptual=0 | Total=43

## Capability Snapshot

| Capability ID | Version | Stable |
|---------------|---------|--------|
| cap.transfer | 1.0 | true |
| cap.issue | 1.0 | true |
| cap.delegation.create | 1.0 | true |
| cap.delegation.revoke | 1.0 | true |

Schema Version: 1

## Cryptographic & Authenticity

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec1.item1 | Mandatory POA signature at issuance | Implemented | P0 | Need configurable algorithms (Ed25519 only) | docs/GAP_MATRIX.md:12\|pkg/rfc0111/signature_negative_test.go |
| sec1.item2 | Full JWT/PASETO claims | Partial | P0 | sub,scope,exp,iat,iss,aud,jti,nbf implemented; missing advanced (claims set metadata, structured nested PASETO footer, typ semantic enforcement) | pkg/gauth/gauth.go\|pkg/gauth/gauth_claims_test.go |
| sec1.item3 | Robust JSON parsing | Partial | P0 | Manual string scanning; property + fuzz tests cover legacy parser for safety | pkg/gauth/gauth.go\|pkg/gauth/gauth_prop_test.go\|pkg/gauth/gauth_fuzz_test.go |
| sec1.item4 | Key rotation & lifecycle | Partial | P1 | Scheduler + disk persistence implemented (env driven); missing multi-tenant segregation & external HSM integration | internal/crypto/keys.go\|internal/crypto/keys_persist_test.go |
| sec1.item5 | Public verifiable token integrity | Implemented | P0 | Detached signature (EnvelopeV2: detached_sig+alg+kid) implemented; remaining: alt algs (ecdsa/bls/batch) & property/fuzz tests & mandatory enforcement flag | docs/GAP_MATRIX.md:14\|pkg/rfc0111/rfc0111_detached_signature_test.go |
| sec1.item6 | Canonical digest stability fuzzing | Implemented | P2 | Property + fuzz tests validate determinism & mutable field exclusion | docs/GAP_MATRIX.md:15\|pkg/rfc0111/canonical.go\|pkg/rfc0111/canonical_prop_test.go\|pkg/rfc0111/canonical_fuzz_test.go |

## Authorization Engine

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec2.item1 | PDP combining algorithms | Implemented | P0 | Need richer conflict diagnostics | docs/GAP_MATRIX.md:20 |
| sec2.item2 | ABAC expression evaluation | Implemented | P0 | No extensible function registry | docs/GAP_MATRIX.md:21 |
| sec2.item3 | Obligations & advice processing | Partial | P2 | Executor skeleton present; lacks advice emission semantics & persistent audit channel | docs/GAP_MATRIX.md:25 |
| sec2.item4 | Policy versioning & rollback | Partial | P1 | In-memory version snapshots + rollback API; missing persistent store + audit trail | docs/GAP_MATRIX.md:23\|pkg/authz/policy_version_test.go |
| sec2.item5 | Distributed PDP & caching | Missing | P2 | No clustering or cache invalidation | docs/GAP_MATRIX.md:24 |

## PoA Definition (RFC0115)

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec3.item1 | Full semantic validation | Partial | P0 | BasicPoAValidator only | docs/GAP_MATRIX.md:32\|pkg/rfc0111/validator.go |
| sec3.item2 | Embed full PoA in token | Missing | P1 | Envelope lacks full definition | docs/GAP_MATRIX.md:33 |
| sec3.item3 | Joint/collective signature enforcement | Partial | P1 | No aggregated digest signature (batch/compact) & multi-algorithm sets | docs/GAP_MATRIX.md:33 |
| sec3.item4 | Conditional/special conditions evaluation | Missing | P2 | No runtime interpreter | docs/GAP_MATRIX.md:34 |

## Legal / Jurisdiction / Compliance

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec4.item1 | Jurisdiction-specific enforcement | Missing | P1 | No runtime branching | docs/GAP_MATRIX.md:40 |
| sec4.item2 | Compliance attestation proof | Missing | P2 | No evidence ingestion | docs/GAP_MATRIX.md:41 |
| sec4.item3 | Arbitration / dispute hooks | Missing | P3 | No code path | docs/GAP_MATRIX.md:42 |

## Persistence & Durability

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec5.item1 | Immutable audit ledger | Partial | P0 | BoltDB lacks signatures & external anchor | docs/GAP_MATRIX.md:48 |
| sec5.item2 | Delegation storage durability | Partial | P2 | No indexing or pruning | docs/GAP_MATRIX.md:49 |
| sec5.item3 | Revocation anchoring | Partial | P2 | No external notarization | docs/GAP_MATRIX.md:50 |

## Replay & Token Security

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec6.item1 | Fail-closed replay mode | Partial | P1 | In-memory JTI map + optional ReplayStore reject duplicates/errors; missing durable persistence & eviction controls | pkg/gauth/gauth.go\|pkg/gauth/gauth_claims_test.go |
| sec6.item2 | JTI format validation | Implemented | P2 | Need skew checks | docs/GAP_MATRIX.md:56 |
| sec6.item3 | Replay persistence recovery | Missing | P2 | No WAL snapshot | docs/GAP_MATRIX.md:57 |

## Observability & Metrics

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec7.item1 | Decision metrics (allow/deny + action/resource labels) | Implemented | P2 | Reason taxonomy limited; no JSON labeled export yet | docs/GAP_MATRIX.md:62\|internal/metrics/prometheus_adapter.go\|docs/OBSERVABILITY.md |
| sec7.item2 | Metrics export adapter | Partial | P3 | No collector registration | docs/GAP_MATRIX.md:63 |
| sec7.item3 | Violation & semantic counters (adaptive anomaly) | Implemented | P2 | Counters + per-category 60s/300s rates + adaptive anomaly detector (EWMA + Welford variance) with z-score export via JSON/Prometheus/OTEL; anomaly EWMA state persisted & restored with hash chain verification. Remaining gaps: external anchoring & archival rotation of semantic snapshots, historical rate archive beyond EWMA, surge alert hooks. | internal/observability/violations.go\|pkg/gauth/gauth.go\|pkg/rfc0111/rfc0111.go\|web/server_clean.go\|docs/OBSERVABILITY.md\|web/persistence_verify_test.go\|web/server_anomaly_test.go\|web/server_semantic_persistence_test.go |
| sec7.item4 | Distributed tracing | Missing | P3 | No span linking | docs/GAP_MATRIX.md:65 |

## Key & Secret Management

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec8.item1 | Secure secret storage | Partial | P0 | Secret provider abstraction + memory + vault stub; missing real backend + encryption at rest | docs/GAP_MATRIX.md:70\|pkg/secret/provider.go |
| sec8.item2 | Rotation audit trail | Partial | P1 | JSON rotation log + hash chain (prev_hash -> hash) implemented; still missing external append-only sink & multi-tenant segregation | internal/crypto/keys.go\|internal/crypto/keys_rotation_log_test.go\|internal/crypto/keys_rotation_hash_chain_test.go |

## Testing & Conformance

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec9.item1 | Clause-to-test mapping | Partial | P0 | Harness maps 8 mapped clause entries (100% of declared set); broader RFC sections still unmapped | docs/GAP_MATRIX.md:76\|conformance/clause_map.json\|report.md |
| sec9.item2 | Fuzzing / property tests | Partial | P1 | Canonical digest covered; parsing & semantic validators still lack property tests | docs/GAP_MATRIX.md:77\|pkg/rfc0111/canonical_prop_test.go\|pkg/rfc0111/canonical_fuzz_test.go |
| sec9.item3 | Load/stress benchmarks | Missing | P2 | No high-load harness | docs/GAP_MATRIX.md:78 |

## Interoperability / External Interfaces

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec10.item1 | OpenAPI for PoA & delegation | Partial | P1 | Spec published (issue/validate/status/delegation/metrics); missing error schemas & provenance/audit endpoints | docs/GAP_MATRIX.md:83\|docs/openapi.yaml\|web/server_clean.go |
| sec10.item2 | Well-known discovery endpoints | Partial | P2 | Missing JWKS integrity signature & structured deprecation metadata (deprecated_after/sunset_after) | web/server_clean.go\|web/jwks_integrity_test.go |

## AI Capability & Governance

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec11.item1 | Capability matrix enforcement | Partial | P1 | Runtime enforcement & anchoring present (flag-gated); missing dedicated fuzz tests | external timestamp integration |
| sec11.item2 | Model limit checks | Partial | P2 | Multi-dimension enforcement (input/output tokens + per-minute rate) + per-user scoped quotas and exceed audit hash chain with verification endpoint; metrics counters (model_limit_exceeded_total | model_output_limit_exceeded_total |

## Advanced Delegation Lifecycle

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec12.item1 | Suspension / partial revocation | Missing | P2 | Only revoked/expired statuses | docs/GAP_MATRIX.md:96 |
| sec12.item2 | Delegation chaining depth limits | Missing | P2 | No depth enforcement | docs/GAP_MATRIX.md:97 |

## Data Hygiene & Validation

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec13.item1 | UTF-8 & control char filtering | Partial | P3 | No metrics instrumentation | docs/GAP_MATRIX.md:102 |
| sec13.item2 | Structured numeric limit parsing | Partial | P2 | No multi-period limits; lacks currency conversion & audit persistence | docs/GAP_MATRIX.md:103 |

## Risk & Threat Modeling

| ID | Requirement | Status | Priority | Gap | Evidence |
|----|-------------|--------|----------|-----|----------|
| sec14.item1 | Threat model synchronization | Partial | P2 | No mitigations matrix | docs/GAP_MATRIX.md:108 |
| sec14.item2 | Residual risk register | Missing | P3 | No tracking of remaining exposures | docs/GAP_MATRIX.md:109 |

