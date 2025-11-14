---
title: ADR Capability Governance (Schema Versioning & Hash Anchoring)
category: adr
status: proposed
lastUpdated: 2025-11-12
owners: architecture-team
source: internal
refreshCadence: on-change
---
# ADR: Capability Governance (Schema Versioning, Hash Anchoring, Transactional Loader, Audit Pagination)

Date: 2025-10-19
Status: Proposed (Implemented in beta-refactor branch)
Context Version: v0.3.1-beta-refactor

## 1. Context & Problem Statement
Capability-based governance was added to enforce which actions (issuance, delegation create/revoke) are permitted. Early implementation lacked:
- Explicit schema versioning for capability file evolution.
- Deterministic integrity artifact to detect tampering or unintended drift.
- Transactional safety during reload (partial failures mutated global state).
- Scalable retrieval (audit export could grow unbounded without pagination/filtering).

We need a robust, versioned, integrity-verifiable capability registry with safe reload semantics and production-ready audit observability.

## 2. Decision Drivers
- Integrity: Prevent silent mutation or partial application of new capability sets.
- Governance & Compliance: Provide auditable provenance (source path, load timestamp, schema version) and anchorable hash.
- Safety: Fail closed on invalid capability files without corrupting in-memory registry.
- Scalability & UX: Efficient audit retrieval for operational triage & compliance review.
- Determinism: Stable ordering ensures reproducible hashes, discovery ETag, and test predictability.

## 3. Considered Options
1. Naïve reload (mutate on parse) + no hash + no version.
2. Version field only (no hash, non-transactional).
3. Hash only (no explicit schema version negotiation).
4. Transactional loader + schema_version + canonical hash + paginated audit (Chosen).

Rejected options lacked at least one of integrity verification, graceful evolution path, or operational scalability.

## 4. Decision Outcome
Implement Option 4: transactional validator & atomic registry reset with enforced `schema_version` and deterministic SHA256 `capability_registry_hash`; add paginated audit endpoint.

### 4.1 Key Elements
- Capability File Requirements: JSON with `schema_version` (string), `capabilities` (list, unique `id`), `action_mappings` (map: action -> list of capability ids).
- Validation Pipeline (all-or-nothing): parse -> validate schema_version -> uniqueness check -> dangling reference check -> canonical sort -> hash compute -> atomic `Reset`.
- Canonical Hash: SHA256 over JSON serialization of sorted capabilities (by id) and sorted actions (lexicographically) with each capability list sorted; includes schema_version to prevent replay across versions.
- Discovery Additions: `capability_registry_schema_version`, `capability_registry_hash`, provenance (`capability_registry_source`, `capability_registry_last_loaded_at`).
- Enforcement Flag: `GAUTH_CAPABILITY_ENFORCE=1` gates action capability checks at token issuance & delegation endpoints.
- Audit Enhancements: `capability_enforce` events logged; `/api/v1/audit/capabilities` supports `limit`, `cursor`, optional filters `action`, `outcome`; returns `entries`, `count`, `next_cursor`, `has_more`, `total_filtered`.

### 4.2 Atomicity & Failure Behavior
- On validation failure, existing registry remains unchanged and previous hash preserved.
- Hash changes only occur on successful complete replacement.

### 4.3 Deterministic Ordering Benefits
- Stable hash comparisons across environments.
- Predictable pagination sequence (insertion index ordering).
- Facilitates external anchoring (future Merkle root or timestamping service).

## 5. Data Structures
- In-Memory Registry: slice of `Capability{ID, Description, ...}` behind `Reset([]Capability)`.
- Action Mapping: `map[string][]string` validated for referenced capability IDs.
- Audit Log Entry: `{ts, action, outcome, provided_capabilities, required_capabilities, missing_capabilities, reason}`.
- Cursor: Opaque numeric offset (string encoded) referencing next start index.

## 6. API Surface Changes
- Discovery: new fields enumerated above.
- Reload Endpoint: `/api/v1/beta/capabilities/reload` unchanged path, now transactional logic & error semantics.
- Audit Export: `/api/v1/audit/capabilities?limit=...&cursor=...&action=...&outcome=...`.

## 7. Security & Integrity Considerations
- Hash prevents unnoticed drift; plan to externally anchor (future ADR) by periodic submission of hash to timestamping or ledger.
- Failure paths log errors but do not mutate state, reducing risk of partial poisoning.
- Potential future extension: include signed capability file digest using server key; integrate with JWKS.

## 8. Operational & Monitoring
- Metrics: `capability_denied` violation counter feeds anomaly detector.
- Recommended Alerts: spike in `capability_denied` or frequent failed reload attempts.
- Observability Gap: need to expose last hash change timestamp + previous hash for differential monitoring.

## 9. Alternatives & Trade-offs
- Merkle tree per capability vs flat canonical hash: deferred (current scale small; simplicity favored). Merkle tree would improve partial proof but adds complexity.
- Database-backed registry vs file-backed: deferred until multi-tenant or dynamic editing required.

## 10. Migration & Rollout
- Introduce `schema_version` starting at `v1`. Older files without version now rejected (document upgrade path).
- Provide sample `capabilities.json` with version & commentary.
- Rollout Steps: deploy code -> add versioned file -> enable `GAUTH_CAPABILITY_ENFORCE` after readiness checks.

## 11. Future Work (Follow-ups)
- External anchoring of `capability_registry_hash` (timestamp/Merkle service).
- Multi-version negotiation & deprecation schedule metadata (e.g., `supported_versions`, `deprecated_after`).
- Fuzz/property tests for loader & hash stability.
- Signed capability file (Ed25519) + signature verification in loader.
- Streaming or incremental audit export (server-sent events) if growth demands.

### 11.1 External Anchoring Placeholder
Initial candidate approaches under evaluation:
1. RFC3161 Timestamping Authority: submit hash periodically, retain timestamp receipts.
2. Public Transparency Log (Trillian / Sigstore): append hash entries, retrieve inclusion proof.
3. Internal Merkle Ledger: accumulate capability hash + other integrity artifacts (ledger tip, semantic snapshot) into a daily Merkle root anchored externally once per interval.

Selection criteria: latency, cost, independent verifiability, audit API maturity. Decision deferred to dedicated anchoring ADR.

## 12. Acceptance Criteria
- Successful reload updates hash; failed reload leaves previous hash intact (tests passing).
- Discovery returns consistent canonical hash across identical files (test verified).
- Pagination test traverses >1 page returning deterministic ordering & correct `has_more`/`next_cursor` semantics.
- Negative tests ensure registry unchanged after invalid file attempts.

## 13. References
- GAP Matrix Section 11 (AI Capability & Governance) updated 2025-10-19.
- Source: `web/server_clean.go` (loader, discovery, audit pagination), `internal/capability/registry.go` (Reset), tests in `web/capability_persistence_test.go`, `web/discovery_capabilities_test.go`.

## 14. Obligations & Advice Execution (Prototype Integration)

An initial obligations execution pipeline was added to the PDP engine (Phase 2 prototype scope):

* Policies may declare an `Obligations` slice; if any rule in the policy matches, those obligations are attached to the final decision.
* Each obligation can set `mandatory=true`. When the engine is configured via `WithObligationFailureDenies(true)`, any failure of a mandatory obligation converts an allow decision into a deny with reason `Denied due to mandatory obligation failure` and metadata key `mandatory_obligation_failures` listing failed IDs.
* Execution occurs prior to recording allow/deny counters so final outcome reflects mandatory semantics.
* Metrics: successful executions increment `obligations_executed_total`; failures increment `obligations_failed_total` via `metrics.Metrics` surface.
* Optional JSONL audit file records one line per obligation: `{ts, subject, action, resource, allow, obligation, index, success, duration_ms, error?}`.
	* `duration_ms` provides per-obligation execution latency (sequential approximation in current executor implementations) enabling SLO tracking and future alerting.
* Context cancellation propagates into executor; cancelled obligations are counted as failures.
* Future evolution will add richer advisory vs mandatory policies, retry/circuit isolation, and parameterized attributes.

This prototype advances compliance observability without coupling authorization correctness to side-effect channels.

---
