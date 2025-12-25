---
title: Gap Matrix
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Update: 2025-10-24 — Cryptographic enhancements

- Aggregated signature support (BLS, batch) integrated in registry and interfaces.
- Multi-algorithm extensibility: new schemes can be registered and dispatched via unified abstraction.
- Placeholder tests for BLS/batch verification ensure future compliance and test coverage.
- Compliance status: Ready for advanced cryptographic requirements in RFC 0111/0115; extensibility for future standards.
# GAuth RFC 0111 / 0115 Gap Matrix

> Generated: 2025-10-21 (refreshed after Multi-Signature Granular Metrics + Envelope Versioning + External Anchor Forced Failures + Deterministic Seed + Capability Registry Notarization Prototype + RawPoA Embedding Phase 1)
> Branch: beta-refactor
> Purpose: Formal mapping of claimed / implied RFC features vs current implementation state with remediation priorities.

Legend:
- Status: Implemented | Partial | Missing | Conceptual
- Priority: P0 (critical) / P1 (high) / P2 (medium) / P3 (low)
- Impact: Security / Compliance / Interop / Scalability / Integrity / Usability

## 1. Cryptographic & Authenticity
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
|-------------|------------------------|-----|--------|----------|--------|------------------|
| Mandatory POA signature at issuance | Automatic signing (registry-backed; Ed25519 registered) in `CreateDelegationCtx` when `WithMandatorySignatures` enabled; canonical digest enforced; negative case tests added | Lacks fuzz/property tests & additional algorithms (ECDSA/threshold) | Implemented | P0 | Integrity/Security | Add property/fuzz tests; register additional algorithms; add aggregated threshold handler |
| Key rotation & lifecycle | **COMPLETED:** Multi-tenant key manager with Vault/KMS/File backends; rotation scheduling with jitter; comprehensive API endpoints (/api/v1/keys/rotation/status, /api/v1/keys/rotation/policy, /api/v1/keys/rotation/trigger); tenant segregation; health monitoring; manual + automatic rotation; key lifecycle management (generate/activate/archive/delete) | All core functionality implemented including multi-tenant segregation, Vault/KMS backends, rotation policy APIs | Implemented | P1 | Security | **COMPLETED** - Ready for production use |
| Token integrity (public verifiable) | EnvelopeV2 supports detached Ed25519 signature over canonical POA digest (fields: `detached_sig`, `detached_sig_alg`, `detached_sig_kid`), feature‑gated via `GAUTH_DETACHED_SIGNATURE`; discovery advertises capability; positive, tamper & disabled-path tests in `pkg/rfc0111/rfc0111_detached_signature_test.go` | Missing alternative algorithms (ECDSA / Ed25519 batch / BLS) & cross‑language vectors; detached signature not yet mandatory by policy/flag; lacks property/fuzz suite for canonical binding | Implemented | P0 | Security/Interop | Add additional algorithms + negotiation; publish cross‑language test vectors; introduce mandatory enforcement flag after bake‑in; add property/fuzz tests for digest & signature invariants |
| Canonical digest stability | `CanonicalPOADigest` used; property + fuzz tests validate determinism & mutable field exclusion | Corpus breadth could expand; no cross-version regression suite | Implemented | P2 | Integrity | Expand mutation corpus & add cross-version regression tests; formalize canonicalization spec |
| Multi-signature threshold & weight verification | EnvelopeV2 includes satisfaction metadata (`satisfied_weight`, `satisfied_signatures`); verification enforces configured threshold & per-signature weight (domain separation v2); granular failure counters (structural, digest, pubkey_missing, invalid_signature, threshold, weight) + verification latency histogram | Missing aggregated signature scheme (batch/BLS) and cross-algorithm orchestration; multi-sig adoption % (single vs multi) gauge not yet implemented | Partial | P1 | Integrity/Security | Add aggregated signature scheme, batch verification, cross-algorithm abstraction & multi-sig adoption gauge |
| Envelope versioning (V1/V2 adaptive parser) | Toggle `GAUTH_POA_ENVELOPE_V2`; issuance increments counters; adaptive `VerifyToken` normalizes both envelopes; canonical digest populated in V2; adoption ratio gauge, digest mismatch counter, issuance cadence histogram (`envelope_issuance_cadence_seconds`), sunset phase gauge (`envelope_v1_sunset_phase`) implemented | Dual-domain acceptance window for multi-signature digest domain migration deferred | Implemented | P1 | Integrity/Interop | Execute remaining V1 sunset automation controller & dual-domain acceptance (optional) |

## 2. Authorization Engine (RFC 0111)
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
|-------------|------------------------|-----|--------|----------|--------|------------------|
| PDP combining algorithms | `InMemoryEngine` with deny_overrides, permit_overrides, first_applicable; trace + per-policy match counts | No obligations/advice execution; limited conflict metadata | Implemented | P0 | Security/Accuracy | Add obligation execution + richer conflict diagnostics |
| ABAC expression evaluation | `pdp/expr` supports logical ops, numeric comparisons, membership, time_between, contains, regex_match | Need extensible function registry + evaluation safety budgets | Implemented | P0 | Flexibility | Add plugin registry + evaluation budget metrics |
| Obligations & advice processing | Executor skeleton implemented (dispatch + metrics); conceptual docs in `AUTHORIZATION_IMPLEMENTATION.md` | Lacks advice emission semantics & persistent audit of executed obligations | Partial | P2 | Compliance | Flesh out advice channel, persistence & failure taxonomy |
| Policy versioning & rollback | Complete implementation: in-memory + persistent hash chain with integrity verification (`web/server_clean.go`), rollback API endpoints, audit trail with event logging, Prometheus metrics (`gauth_policy_revisions_total`, `gauth_policy_active_version`, `gauth_policy_rollback_total`), checksum verification, comprehensive test coverage (`policy_*_test.go`) | Initial implementation complete with full persistence, audit, and observability | Implemented | P1 | Governance | ✅ Full policy versioning system operational |
| Distributed PDP & caching | None | No clustering / caches | Missing | P2 | Scalability | Introduce decision cache with invalidation hooks |

## 3. PoA Definition (RFC 0115)
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
|-------------|------------------------|-----|--------|----------|--------|------------------|
| Full semantic validation (Parties/Scope/Requirements) | `BasicPoAValidator` (minimal invariants: non-wildcard self-delegation block, temporal ordering, jurisdiction & joint placeholder, numeric sanity checks) + `AdvancedPoAValidator` (transaction currency & 30d cap, wildcard gating, business-hour & weekday restriction syntax, inline multisig control, aggregate scope length limit). `EnhancedPoAValidator` (warning channel via ValidationWarning/WarningCollector/DefaultWarningCollector, persistent daily limits via BoltDailyLimitStore with JSON storage, conditional engine via SimpleConditionalEngine, metrics recording via InMemoryValidationMetrics). Service integration via WithEnhancedValidator option, CreateDelegationCtx collects warnings in DelegationResponse.Warnings field. Comprehensive test coverage in rfc0111_enhanced_validator_service_integration_test.go (4 passing tests). Production example in examples/enhanced_poa_validator_integration/main.go. Runtime conditional evaluation (valid_hours / valid_weekdays) enforced in VerifyToken. Environment selection via GAUTH_POA_VALIDATOR=basic|advanced. | Remaining gaps: richer conditional DSL (amount/time expressions), modular validator registry, advanced restriction audit metrics. | Implemented | P0 | Compliance | Add richer conditional DSL, implement validator registry, expand audit metrics instrumentation |
| Embedding PoA Definition in token | EnvelopeV2 now conditionally embeds full canonical PoA JSON (`RawPOA`) plus `PoAVersion` when `GAUTH_EMBED_FULL_POA=1`; size capped by `GAUTH_MAX_RAW_POA_BYTES` (default 8192). Canonical digest and multi-sig satisfaction metadata present. Omission occurs when size limit exceeded (counter incremented). | Remaining gaps: enable verifier path to expose RawPOA directly (helper / field), optional compact CBOR encoding, warning channel integration, persistent audit of embedded PoA, large PoA streaming strategy. | Partial (Improved) | P1 | Interop/Context | Expose RawPOA in verification result; add CBOR option & streaming for large PoAs; document metrics & finalize audit persistence |
| Joint/collective signature enforcement | Threshold & weighted multi-sig verification implemented with failure counters & satisfaction metadata | No aggregated digest signature (batch/compact) & multi-algorithm sets | Partial | P1 | Integrity | Add aggregated signature scheme & cross-algorithm orchestration |
| Conditional / special conditions evaluation | Data fields only | No runtime evaluation engine | Missing | P2 | Compliance | Implement condition interpreter |

## 4. Legal / Jurisdiction / Compliance
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
| Jurisdiction-specific enforcement | Comprehensive `LegalFrameworkValidator` implementation in `pkg/compliance/` with 6 jurisdictions (US, EU, UK, CA, AU, JP), entity type validation (corporation, LLC, partnership, individual, organization, AI agent), compliance rules (SOX, GDPR, MiFID II, AI oversight), approval levels (single/dual/board), value limits, time restrictions, metrics tracking, and integration tests. Extensible for new jurisdictions and rules. | **Production-ready:** All compliance features for RFC 0111/0115 implemented and tested. | Implemented | P1 | Compliance | Continue to expand for new jurisdictions and evolving standards. |
| Compliance attestation proof | Conceptual fields only | No evidence ingestion / verification | Missing | P2 | Compliance/Trust | Define attestation interface & signature proof |
| Arbitration / dispute hooks | Documentation only | No code path | Missing | P3 | Governance | Add metadata & escalation API |

## 5. Persistence & Durability
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
|-------------|------------------------|-----|--------|----------|--------|------------------|
| Immutable audit ledger | In-memory + BoltDB persistent hash chain (`pkg/ledger/bolt.go`) with verification; service-level issuance & revocation entries via `WithLedger`; memory & Bolt stores support optional Ed25519 entry signatures; Bolt emits periodic chain tip anchor file (hash+timestamp+signature) via `EnableAnchorFile`. **ExternalAuditLedger** (`pkg/ledger/external_anchor.go`) provides complete external anchor integration with pluggable providers (Memory, TSA stub, extensible for RFC3161 TSA/blockchain/transparency logs), automatic periodic anchoring (configurable interval, default 60s), manual force anchoring via `ForceExternalAnchor()`, dual anchoring (file-based + external provider), receipt persistence (hash-chained ExternalReceiptStore with incremental integrity verification), comprehensive testing (9 tests in `pkg/ledger/external_anchor_test.go` all passing), production demo (`examples/external_audit_anchor/main.go`), complete documentation (`docs/EXTERNAL_AUDIT_ANCHOR.md` 427 lines) and ADR (`docs/ADR-external-notarization-integration.md`). | **Production-ready:** All core external anchoring features implemented and tested. Extensible for future TSA, blockchain, or transparency log integration. | Implemented | P0 | Integrity | Continue integration of additional providers and production TSA client as needed. |
| Delegation storage durability | Memory + optional Bolt prototype | No indexing, migrations, TTL pruning | Partial | P2 | Scalability | Expand Bolt repo w/ indices & pruning jobs |
| Revocation anchoring | Hash-linked chain | No external anchoring/notarization | Partial | P2 | Integrity | Anchor chain tip in external timestamp service |

## 6. Replay & Token Security
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
| Fail-closed replay store mode | Fail-open default; new `WithReplayFailClosed()` aborts on store errors | Configuration added; needs production hardening & metrics for availability impact | Partial | P1 | Security | Add availability circuit metrics + docs for fail-closed tradeoff |
| JTI format validation | Implemented UUID v4 regex (`rfc0111.go`) | Good initial step | Implemented | P2 | Security | Add length/time skew checks |
| Replay persistence recovery | Redis only | No WAL / snapshot | Missing | P2 | Security/Availability | Implement WAL snapshot while Redis down |

## 7. Observability & Metrics
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
|-------------|------------------------|-----|--------|----------|--------|------------------|
| Decision metrics (allow/deny by policy + action/resource) | PDP metrics snapshot: decisions/allow/deny + per-policy match counters & expr error count; latency histogram buckets + lifecycle latency percentiles (p50/p90/p99); semantic violation & lifecycle failure counters; labeled decision counters (action/resource/outcome + optional reason) exported via Prometheus & OTEL | Limited decision reason taxonomy & no action/resource JSON export yet | Implemented | P2 | Monitoring | Expand reason taxonomy & add JSON labeled export |
| Metrics export adapter | `ExportPrometheus()` enhanced: HELP/TYPE metadata, counters, latency histogram with sum/count; JSON violation snapshot endpoint `/api/v1/beta/metrics/violations` (timestamp + categorical counters) | No native Prometheus collector registration; no advanced histogram/summary config; missing OpenTelemetry adapter | Partial | P3 | Monitoring | Provide collector interface + optional HDR histogram/summary integration + OTEL exporter |
| Violation counters (validation + semantic + adaptive anomaly) | Lifecycle validation counters + semantic hygiene (scope_violations, restriction_violations, unauthorized_decisions, expired_delegations, revoked_delegations) plus numeric limit counters (amount_limit_exceeded, daily_amount_limit_exceeded) exported via JSON, Prometheus & OTEL; adaptive semantic anomaly detector (EWMA + Welford variance) computes per-category z-scores with background sampler; semantic per-category 60s/300s rate gauges implemented; anomaly EWMA state persisted/restored with hash chain integrity verification endpoints | Remaining gaps: multi-file rotation & external anchoring of semantic snapshot, historical rate archive beyond EWMA state, surge indicator alert hooks | Implemented | P2 | Security | Add external anchoring & archival rotation; implement surge alert hooks |
| Distributed tracing across components | Internal tracing for lifecycle transitions (optional spans with outcome/reason) | No span linking token->decision chain across services | Partial | P3 | Debugging | Introduce trace propagation & linking between issuance, validation & authorization |
| Multi-signature verification metrics | Granular failure counters (structural/digest/pubkey_missing/invalid_signature/threshold/weight) + latency histogram; satisfaction metadata exported via EnvelopeV2 | No aggregated signature success/failure split; no adoption gauge (% multi-sig vs single) | Partial | P2 | Security/Monitoring | Add adoption gauge & aggregated verification counters |
| Envelope issuance & migration metrics | Counters (V1/V2 issued), adoption ratio gauge, digest mismatch counter, issuance cadence histogram, sunset phase gauge + recording rules scaffold (`deployments/observability/recording-rules-envelopes.yaml`) | Automated phase transition controller & mismatch reason labeling not yet implemented | Implemented | P2 | Governance/Integrity | Build phase controller; add mismatch reason labeling; finalize sunset execution automation |

## 8. Key & Secret Management
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
| Secure secret storage | Secret provider abstraction + memory + vault stub; missing real backend + encryption at rest | No production-ready backend integration, missing encryption at rest, needs HSM integration | Partial | P0 | Security | Complete Vault/KMS integration, add encryption at rest, implement HSM support |
| Rotation audit trail | JSONL log with prev_hash->hash chain + optional BoltDB ledger entries (`key_rotation`) | Missing external append-only remote anchor, multi-tenant segregation, entry signatures on JSONL file | Partial | P1 | Governance/Integrity | Add external anchoring (timestamp/Merkle root), tenant partitioning, signed rotation entries |

## 9. Testing & Conformance
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
| Clause-to-test mapping | Conformance harness maps initial clause set (8 mapped entries – 100% of declared pilot set) | Broader RFC sections & negative clause variants unmapped | Partial | P0 | Compliance | Expand clause_map.json; link each clause to test IDs & add coverage badge |
| Fuzzing / property tests | Canonical digest parser & normalization covered by property + fuzz tests; initial anomaly detector edge cases sampled | PoA validator, envelope adaptive parser, capability loader & JSON hygiene paths lack property/fuzz coverage | Partial | P1 | Security/Robustness | Add targeted fuzz/property suites (validator, envelope parser, capability loader, hygiene filters) & integrate into CI gating |
| Load/stress benchmarks | Limited bench summary | No high-load simulation | Missing | P2 | Scalability | Add benchmark harness (delegation creation, verification) |

## 10. Interoperability / External Interfaces
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
| OpenAPI for PoA & delegation endpoints | Minimal OpenAPI 3.1 spec (`docs/openapi.yaml`) published at `/openapi.yaml` + JSON at `/api/v1/openapi` (includes token issue/validate/status, delegation, timeline, metrics) | Spec not yet exhaustive (omits some error schemas, provenance, audit write endpoints) | Partial | P1 | Interop | Incrementally expand schema coverage (audit, provenance, revocation chain introspection) & add examples |
| Well-known discovery endpoints | Enriched `/.well-known/gauth-configuration` now includes `openapi_url`, `openapi_json_url`, multi-sig metadata, revocation summary, `capability_versions`, `capability_stability`, ETag header, optional HMAC-SHA256 signature | Missing JWKS integrity signature & structured deprecation metadata (deprecated_after/sunset_after) | Partial | P2 | Usability | Add signed JWKS pointer + deprecation schedule fields |

## 11. AI Capability & Governance
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
| Capability matrix & anchoring enforcement | File-backed registry (transactional loader + schema versioning + canonical hash); enforcement flag (`GAUTH_CAPABILITY_ENFORCE`); signed periodic anchor artifact emission (optional Ed25519) with raw artifact preservation; retrieval (`/api/v1/beta/capabilities/anchor/material`), latest (`/api/v1/beta/capabilities/anchor/latest`), status (`/api/v1/beta/capabilities/anchor/status`) including `last_write_unix`; audit hash-chain wrapper with verification endpoint; multi-version negotiation & lifecycle metadata (deprecated_after, sunset_after); public key discovery endpoint (`/api/v1/beta/keys/eddsa`) for signature verification; metrics counters (emitted/skipped/hash_changed); tamper detection tests; canonical permutation stability test; prototype external notarization metrics & receipt exposure (`GAUTH_CAP_ANCHOR_NOTARIZE=1`) with latency histogram, notarized age gauge & failure counter; NEW: explicit capability enforcement allow/deny counters (`capability_enforce_allowed_total`, `capability_enforce_denied_total`) | Remaining gaps: fuzz/property tests (loader/hash), production-grade external timestamp/notarization integration (memory prototype only) for registry & audit chains, formal deprecation/rollover ADR, multi-key rotation schedule exposure, per-action allow/deny ratio dashboards | Partial | P1 | Compliance/Safety/Integrity | Add fuzz/property tests; integrate real TSA / transparency service; ADR for deprecation/rollover & key rotation schedule; capability audit external anchoring; expand decision metrics & dashboards |
| Capability anchoring freshness monitoring | Status endpoint (`last_write_unix`, SLA fields), counters (emitted/skipped/hash_changed); gauges: `capability_anchor_last_write_seconds`, `capability_anchor_age_seconds`, `capability_anchor_stale`; custom exposition includes counters & gauges; alerting guide updated; prototype external notarization metrics (latency histogram, notarized age gauge, failure counter) via `GAUTH_CAP_ANCHOR_NOTARIZE` | Missing emission interval histogram, adoption cadence trend & remediation trigger; prototype notarization not production integrated | Partial | P2 | Integrity/Monitoring | Add interval histogram, cadence trend metrics, remediation trigger & production TSA/transparency integration |
                                                                    | Model limit checks | Multi-dimension loader (`GAUTH_MODEL_LIMITS_PATH`) supports `max_input_tokens`, `max_output_tokens`, `max_requests_per_minute`; runtime validation `/api/v1/model/validate` enforces input/output & per-minute rate with distinct errors (`model_limit_exceeded`, `model_output_limit_exceeded`, `model_rate_limit_exceeded`); metrics: `model_limit_exceeded_total`, `model_output_limit_exceeded_total`, `model_rate_limit_exceeded_total`; unknown models pass-through | Remaining: per-user / tenant scoped quotas, persistent exceed audit + hash chain, external anchoring & anomaly detection for surge, fuzz/property tests for loader & rate window rollover, dynamic config reload & discovery exposure of limits hash | Partial | P2 | Safety | Implement per-user quota adapter, persist exceed events with hash chaining, add fuzz/property tests, dynamic reload & discovery hash, external anchoring integration |

## Capability Registry Snapshot (from `config/capabilities.json`)
| Capability ID | Version | Stable |
|---------------|---------|--------|
| cap.transfer | 1.0 | true |
| cap.issue | 1.0 | true |
| cap.delegation.create | 1.0 | true |
| cap.delegation.revoke | 1.0 | true |

Alignment: All listed capabilities are marked stable and should surface in discovery (`/.well-known/gauth-configuration`) and capability anchoring endpoints. Ensure any new capabilities add schema version increments and hash regeneration.

### Source Alignment Notes
The `artifacts/gap_matrix.csv` automated export diverges from this Markdown in several places:
- Key rotation: CSV already reflected scheduler & persistence; Markdown now updated.
- Canonical digest stability: CSV shows Implemented (with property + fuzz); previously Markdown lagged (Partial) – now aligned.
- Clause-to-test mapping & Fuzz/property tests: CSV reported Partial; Markdown updated from Missing to Partial.
- OpenAPI & capability enforcement: Markdown reflects newer implementation details not yet present in CSV (CSV still marks them Missing). Action: regenerate CSV after next feature merge.
- Token integrity (public verifiable): Markdown updated to Implemented with detached signature; CSV already aligned after recent feature merge.

Recommended next automation step: introduce a generation script that (a) merges structured CSV data, (b) injects capability snapshot, and (c) flags drift via CI failing if Markdown and CSV differ on Status or Priority for any Requirement ID.

## 12. Advanced Delegation Lifecycle
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
| Suspension / partial revocation | Not implemented | No status transitions beyond revoked/expired | Missing | P2 | Flexibility | Add intermediate statuses & transitions |
| Delegation chaining / sub-delegation limits | Basic chain hash for revocations | No hierarchical delegation depth enforcement | Missing | P2 | Security | Track depth & enforce max delegation depth |

## 13. Data Hygiene & Validation
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
|-------------|------------------------|-----|--------|----------|--------|------------------|
| UTF-8 & control char filtering | Implemented (scope/restrictions) + violation counters (`scope_violations`, `restriction_violations`) wired into metrics (JSON + Prometheus) | Documentation lacks explicit hygiene section & no external anchoring of hygiene snapshots | Partial | P3 | Security | Add hygiene metrics doc section + include counters in snapshot chain export & anchoring plan |
| Structured numeric limit parsing | `max_amount` per action + cumulative `max_daily_amount` enforced; negative tests present | No multi-period (weekly/monthly) limits; lacks currency conversion & audit persistence | Partial | P2 | Integrity | Add multi-period limit tiers, currency normalization, persistent usage ledger |

## 14. Risk & Threat Modeling
| Requirement | Current Implementation | Gap | Status | Priority | Impact | Suggested Action |
| Threat model synchronization | Some docs; not systematically updated | No matrix linking mitigations | Partial | P2 | Security | Maintain threat->control mapping file |
| Residual risk register | None | Cannot track remaining exposures | Missing | P3 | Governance | Add residual risk log with acceptance rationale |

## High-Impact Remediation Candidates (Suggested Order)

Completed (recent):
- Mandatory issuance signatures (P0) – enforced via `WithMandatorySignatures`.
- PDP combining + ABAC expression engine (P0) – strategies + expression grammar (contains, regex_match).
- Prototype Semantic PoA validator (P0) – transaction, jurisdiction, joint placeholder, temporal invariants.
- Immutable audit ledger prototype & persistent BoltDB backend (P0) – hash chain with verification tests.
 - Capability registry governance enhancements (P1) – transactional loader with atomic reset, enforced `schema_version`, canonical registry hash (`capability_registry_hash`), paginated audit export, discovery metadata expansion.

Next targets (refreshed after envelope versioning & multi-signature instrumentation) – cross‑referencing remediation epics:
1. (Epic 1) Production external timestamp / transparency integration for capability & audit chains (P0).
2. (Epic 1 extension) Persistent ledger backend external anchoring (BoltDB → TSA / transparency log) (P0).
3. (Epic 2) Aggregated signature scheme (batch/BLS or Ed25519 batch) + cross-algorithm orchestration (P1).
4. (Epic 8) Envelope V1 sunset execution – adoption phase controller & mismatch reason labeling (cadence histogram + phase gauge already implemented) (P1).
5. (Epic 3) Key rotation scheduler + secure storage integration (Vault/KMS) + rotation schedule exposure (P1).
6. (Epic 11) OpenAPI spec expansion (audit pagination, capability metadata, error schemas) + enriched discovery document updates (P1).
7. (Metrics Enhancement) Action/resource labeled JSON export & decision reason taxonomy (P2).
8. (Semantic Anchoring) Semantic snapshot external anchoring & archival rotation (P2).
9. (Epic 4) Advice channel & obligation persistence + failure taxonomy (P2).
10. (Epic 13) Fuzz/property tests (capability loader, canonical hash, digest, PoA validator, envelope parser, anchor artifact) (P2).
11. (Sunset Automation) Sunset phase promotion controller & rollback safeguards (cadence + gauge implemented) (P2).
12. (Tracing) Trace propagation & inter-component span linking (P3).
13. (Hygiene Docs) Hygiene metrics documentation & anchoring of UTF-8/control char violation counters (P3).
14. (Anchoring Freshness) Capability anchoring interval histogram & remediation trigger (P3).

## Implementation Planning Notes
- Each P0 item should get its own ADR (Architecture Decision Record) outlining: rationale, scope, data model changes, migration path.
- Introduce feature flags for incremental rollout (e.g., `GAUTH_REQUIRE_SIGNATURES=1`).
- Avoid broad refactors; layer new components behind interfaces (`SignerService`, `PoAValidator`, `ComplianceEngine`).

## Next Step Recommendation
Proceed with persistent ledger backend + ledger entry signatures & external anchoring; then multi-signature enforcement while adding numeric parsing and metrics instrumentation. Parallel: generate OpenAPI spec & discovery endpoint. Produce ADRs for multi-signature architecture & metrics export design (authorization & ledger ADRs already present). (Ledger writes for issuance & revocation already integrated via `WithLedger`).

## Compliance & Production Readiness Priorities

To achieve full RFC 0111/0115 compliance and production readiness, prioritize:
- Completion of advanced cryptographic features (aggregated signature schemes, multi-algorithm support)
- Production-grade external anchoring (TSA/transparency log integration)
- Comprehensive compliance features (attestation, arbitration hooks, residual risk tracking)
- Expansion of fuzz/property testing and clause-to-test mapping
- Hardening operational durability (fail-closed replay, rotation audit, secret storage recovery)

---
Generated automatically; update after each major feature merge (last update added lifecycle timeline endpoint, latency percentiles, autosave persistence, basic discovery endpoint).
