# Feature Expansion Roadmap – 2025-10-26

This document sequences implementation for the 12 requested features. Each item lists: Contract, Metrics, Security, Dependencies, and Acceptance Criteria.

## 1. BLS Aggregated Proof-of-Possession (PoP)
Goal: Mitigate rogue public key injection in aggregated signature issuance.
Contract:
- Issue mode (optional): `require_pop=true` -> returns `challenges[]` (hashes) for each public key.
- Verify PoP submission endpoint: POST `/api/v1/crypto/bls/pop` with `public_key_b64`, `challenge_b64`, `pop_signature_b64`.
Metrics:
- `bls_pop_challenges_issued_total`
- `bls_pop_verifications_total`
- `bls_pop_verification_failures_total`
Security:
- Challenge = SHA256(domain || public_key || nonce).
Dependencies: Existing BLS issuance; attestation metrics.
Acceptance: All keys in aggregated issuance have verified PoP before aggregate signature accepted when `require_pop` set.

## 2. External Anchor Chain Verification Metrics & Endpoint
Goal: Continuous integrity monitoring of external anchor receipt chain.
Endpoint: `GET /api/v1/anchor/verifyExternalChain` -> `{status: ok|mismatch|empty, mismatch_index?, receipt_count}`.
Metrics:
- `external_anchor_chain_verifications_total`
- `external_anchor_chain_verification_failures_total`
Security: Detect broken hash continuity.
Dependencies: Existing receipt persistence.

## 3. OpenAPI / Discovery Contract
Goal: Machine-consumable API spec.
Artifacts:
- `openapi.yaml` generation via Go structs + minimal annotations.
Endpoint: `GET /api/v1/openapi` returns JSON.
Acceptance: Validated by `swagger-cli validate`; versioned in docs.

## 4. Replay WAL Snapshot & Compaction
Goal: Bound WAL growth & faster recovery.
Mechanism:
- Trigger on size > `GAUTH_REPLAY_WAL_MAX_MB` or line count.
- Compact algorithm: load live set -> write new file -> atomic rename -> rotate old to `.bak`.
Metrics:
- `replay_wal_compactions_total`
- `replay_wal_compaction_failures_total`
- `replay_wal_compaction_latency_seconds`
- `replay_wal_compaction_bytes_reclaimed_total`

## 5. Per-Algorithm Adoption Ratio Gauge
Goal: Track migration progress.
Implementation:
- Compute ratio = algo_anchor_count / sum(all_algos) periodically.
Metric: `capability_anchor_algorithm_ratio{algorithm="..."}` (added).
Acceptance: Ratios sum ~1.0 (floating tolerance <1e-6).

## 6. Attestation Proof Metrics Expansion
Goal: Granular latency + failure analytics.
Added:
- `attestation_proof_issue_latency_seconds` histogram
- `attestation_proof_verification_failure_reason_total{reason}`
Reasons: `digest_mismatch,algo_mismatch,key_mismatch,missing_anchor,other`.
Acceptance: Counters increment under simulated failure injection tests.

## 7. RFC3161 Timestamp Notary Provider
Goal: Third-party time attestation.
Interface: `NotaryProvider{Notarize(hash []byte) (receipt, error)}`.
Metrics: `notary_timestamp_latency_seconds`, `notary_timestamp_failures_total`.
Security: Validate TSA signature; future: certificate pinning.

## 8. Signed Chain Snapshot Export
Goal: Portable tamper-evident state.
Data: JSON snapshot of receipts + Merkle root + signature.
Endpoint: `GET /api/v1/anchor/snapshot/latest`.
Metrics: `anchor_snapshot_issued_total`, `anchor_snapshot_failures_total`.
Signature: Use primary signing key (Ed25519) over canonical JSON of `{root, count, generated_at}`.

## 9. BLS Aggregated Extended Negative Tests + Fuzz
Goal: Robustness.
Fuzz target: `FuzzBLSAggregateEndpoint` hitting malformed JSON, truncated base64, random binary in signature fields.
Metrics: None new (reuse failure counters).
Acceptance: Runs for N iterations locally without panic.

## 10. Grafana Dashboard Artifacts
File: `dashboards/gauth_observability.json`.
Panels:
- Multi-signature batch size histogram heat.
- Per-algorithm anchor ratio pie/time series.
- Attestation issuance & verification latency percentiles.
- External anchor chain status single-stat.

## 11. Capability Anchor Algorithm Sunset Workflow
Goal: Decommission legacy algorithm safely.
Config: `GAUTH_ALGO_SUNSET_TARGET=ed25519:2026-03-01`.
Metrics:
- `capability_anchor_algorithm_sunset_target_timestamp{algorithm}` (gauge unix seconds)
- `capability_anchor_algorithm_sunset_progress{algorithm}` (ratio). Progress = anchors_new_algo / (anchors_all_after_announce).
Alert: ratio < threshold within T days before target.

## 12. Lifecycle Transition Latency Percentiles Export
Goal: SLA visibility.
Approach: Derive quantiles from histogram OR reservoir (memory) every interval.
Metrics: `lifecycle_transition_latency_quantile{entity,outcome,quantile="p50|p95|p99"}`.
Acceptance: Quantiles update on test transitions.

## Sequencing Plan (Phases)
- Phase A (Completed / In Progress): 5,6,12 instrumentation primitives.
- Phase B: 2 endpoint + metrics; 9 fuzz harness.
- Phase C: 1 PoP; 7 RFC3161.
- Phase D: 8 snapshot export.
- Phase E: 4 replay compaction.
- Phase F: 3 OpenAPI + 10 dashboards + 11 sunset workflow augmentation.

## Testing Strategy
- Unit tests for counters increment conditions.
- Integration tests for new endpoints (200, 400 paths, mismatch scenarios).
- Fuzz tests gated behind `go test -fuzz` for aggregated endpoint.
- Manual Grafana JSON lint (ensure UID fields optional).

## Security Considerations
- PoP prevents malicious aggregator injecting chosen public keys.
- Chain snapshot signing ensures audit portability.
- Timestamp notary adds external temporal binding.
- Replay compaction must fsync before rename to avoid partial durability.

## Follow-up / Deferred Items
- Merkle tree incremental update optimization (O(log n)).
- Multi-algorithm aggregated threshold (heterogeneous curves) validation.
- TSA certificate rotation strategy.

---
Generated: 2025-10-26
