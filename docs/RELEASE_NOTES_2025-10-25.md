# Release Notes – 2025-10-25

## Overview
This release focuses on strengthening multi-signature integrity, canonical serialization, authenticity defaults, and replay protection. It introduces embedded weights, an automatic domain V2 for threshold signatures, mandatory JTI enforcement, and a structured compliance matrix.

## Key Changes
| Area | Change | Benefit |
|------|--------|---------|
| Canonical Digest | Automatic domain V2 when `Threshold > 1` (includes `thr` + sorted weights) | Prevents digest confusion between single vs multi-sig contexts |
| Multi-Signature | Embedded `Weights` map + structural validation (positive, subset, cumulative >= threshold) | Eliminates env-driven ambiguity; ensures deterministic binding |
| Canonical JSON | Added `version` and serialized `weights` (when provided) | Future evolution support; integrity across signer sets |
| Authenticity | Strict authenticity now default (missing public key -> integrity failure) | Reduces silent authenticity downgrade risk |
| Replay Protection | JTI claim mandatory unless `GAUTH_ALLOW_MISSING_JTI=1` | Closes trivial replay channel for tokens without store |
| Tests | Updated property & domain tests; added version/weights presence test | Ensures determinism & correctness under new model |
| Docs | New `rfc0111_compliance_matrix.md`, API README compliance summary, CHANGELOG entry | Transparent compliance and roadmap communication |
| Algorithm Agility | Added ECDSA P-256, BLS12-381 single, and BLS12-381 aggregated (multi-sig compression) support via registry | Enables phased adoption and scalability of multi-signer cryptography |
| Crypto Introspection | Added `/api/v1/crypto/algorithms` listing algorithms + aggregated capability flag | Operational visibility & easy integration testing |

## Security Impact
- Stronger signature context binding (domain separation tied to threshold & weights).
- Reduced reliance on mutable environment for cryptographic semantics.
- Fail-closed replay behavior improves baseline token resilience.

## Migration Guidance
1. Legacy multi-sig PoAs (pre-embedded weights) should be re-issued to gain domain V2 differentiation.
2. If existing deployments depend on soft authenticity skip, set `GAUTH_STRICT_AUTHENTICITY=0` temporarily during transition.
3. Audit storage systems should verify digest divergence for threshold PoAs after upgrade.

## Backward Compatibility
- Single-signer (`Threshold=1`) PoAs keep V1 domain; digests unchanged.
- Existing tokens with missing JTI will now fail unless override is set.

## Testing Summary
- All updated tests pass (`canonical_prop`, `canonical_domain_v2`, `rotation`, `strict_auth`, `canonical_version_weights`).
- Property assertions confirm weight order invariance and digest variation on threshold/weight changes.

## Compliance Snapshot
See `docs/rfc0111_compliance_matrix.md` for full matrix. Highlights:
- Implemented: Multi-signature threshold, canonical serialization, validity period.
- Partial: Audit logging, replay protection, cryptographic requirements (algorithm agility pending).
- Missing (before this patch): OpenAPI export, external anchoring.
	- Added in this iteration: partial revocation state (`partially_revoked`) with one-way narrowing semantics and dedicated metric `partially_revoked_delegations_total`.

### Partial Revocation (New)
Introduced lifecycle state `partially_revoked` for delegations whose effective scope has been narrowed without full termination.

Semantics:
- Allowed transitions: `active|suspended -> partially_revoked`, `partially_revoked -> terminated` (idempotent `partially_revoked` retained).
- Disallowed widening / reactivation: `partially_revoked -> active|suspended` returns HTTP 409 and increments invalid transition counters.
- Metrics:
	- `partially_revoked_delegations_total` (Prometheus / in-memory) via `IncDelegationsPartiallyRevoked()`.
	- Lifecycle breakdown keys: `delegation|active|partially_revoked|success`, `delegation|partially_revoked|terminated|success`, plus failure key `delegation|partially_revoked|active|failure`.

Operational Benefits:
- Enables progressive rights reduction before hard termination, supporting staged off-boarding & incident containment.
- Distinct observability channel for partial revocation frequency (capacity planning & security anomaly detection).

Backward Compatibility:
- Existing transitions unchanged; new state ignored by older clients (treat as unknown if not enumerated).
- Validation logic ensures no inadvertent scope widening by blocking revert to `active`.

## Roadmap (Excerpt)
1. Algorithm agility (Ed25519 + newly added ECDSA P-256 abstraction; BLS planned).
2. External audit ledger anchoring with signed entries/Merkle roots.
3. Partial revocation & suspension states; depth limits.
4. OpenAPI/Discovery contract + OTEL tracing integration.
5. Durable replay store (persistent JTI index with snapshot) – initial WAL durability, latency/error instrumentation, corruption-tolerant recovery implemented in this release.

## References
- CHANGELOG: `docs/CHANGELOG.md`
- Compliance Matrix: `docs/rfc0111_compliance_matrix.md`
- Canonical Implementation: `pkg/rfc0111/canonical.go`
- Crypto Abstraction: `pkg/crypto/signature.go`, `pkg/crypto/ecdsa_provider.go`

## Algorithm Agility Usage
Configure service with a specific algorithm using functional options:

- Ed25519 (default): existing in-memory provider or external KMS.
- ECDSA P-256: `WithInMemoryAlgorithm("ecdsa-p256")`
- BLS12-381 (single): construct provider manually (future convenience option planned) or use registry for verification.
- BLS12-381 aggregated: generate multiple BLS keys, produce individual signatures over identical canonical message, aggregate into a single signature (algorithm `bls12-381-agg`) and verify via `VerifyAggregatedAlgorithm`. Structural contract: one message, one aggregated signature, N key IDs.
	- Introspection: query `/api/v1/crypto/algorithms` for current registry (fields: `name`, `aggregated_supported`).

Signatures record the algorithm name without altering canonical digest. Digest invariance maintains structural semantics across algorithms.

Planned: algorithm introspection endpoint, heterogeneous threshold aggregation, per-algorithm metrics.

## Checksums (Informational)
The canonical digest domain prefix variants:
- V1: `GAUTH_RFC0111_POA_V1` (single-sig)
- V2: `GAUTH_RFC0111_POA_V2|thr=<T>|w=<sorted weights>` (multi-sig)

## Acknowledgments
Thanks to contributors refining threshold signatures, authenticity defaults, and compliance mapping.

---
End of release notes.

## Replay Durability & Metrics (Incremental Addition)
This release adds a metrics-instrumented, corruption-tolerant Write-Ahead Log (WAL) for replay (nonce/JTI) protection:

Features:
- WAL append on each `Record` / `RecordWithEvict` when `GAUTH_REPLAY_WAL` is set.
- Recovery on startup via line-by-line scan (`RecoverWithStats`) tolerating malformed JSON lines (skipped without aborting).
- Metrics instrumentation:
	- Latency observation for append and recovery phases (`ObserveReplayStoreLatency`).
	- Error counter increments (`IncReplayStoreErrors`) for each skipped corrupt line and initialization failures.
- Backward-compatible constructor: existing `NewReplayNonceStore(ttl)` preserved; new `NewReplayNonceStoreWithMetrics(ttl, metrics)` enables instrumentation.
- Environment variables:
	- `GAUTH_REPLAY_WAL`: path to WAL file (enables durability).
	- `GAUTH_REPLAY_CAP`: optional in-memory capacity bound with oldest eviction.

Operational Notes:
- Recovery clamps any future timestamps to `now` to prevent negative TTL windows.
- Expired entries are lazily purged during `Seen` checks; choose TTL mindful of startup recovery time.
- Corrupt lines no longer abort recovery; each is counted for observability.

Future Enhancements (Planned):
- Snapshot + compaction to control WAL growth.
- Explicit recovery latency vs append latency metric separation.
- Rolling hash / integrity chain for tamper detection.
- Size-based rotation (`GAUTH_REPLAY_WAL_MAX_MB`) and fsync toggle.

Testing Added:
- `TestReplayNonceStore_CorruptionRecovery` validates recovery of valid records and error counting for malformed line.

Security Impact:
- Increases resilience against process restarts (durable JTI storage) without sacrificing startup continuity when encountering partial corruption.

## External Anchoring (Incremental) & Combined Capability+Rotation Emission (Prototype)

This release begins expansion of external anchoring for capability registry material and introduces a prototype combined anchor emission path.

New Endpoints:
- `GET /api/v1/anchor/chain` – Returns the external anchor receipt chain (hash, timestamp, provider, version, latency_seconds, prev_hash, chain_hash) when a receipt store is configured.
- `POST /api/v1/anchor/emitCombined` – Manually emits a combined capability+rotation hash anchor receipt (concatenation `cap_hash:rotation_head`, SHA-256 hashed). Rotation head omission (empty) tolerated when no ledger is present.
- `GET /api/v1/anchor/verifyChain` – Performs incremental hash-chain verification of external receipts (`status`: ok|mismatch|empty, optional `mismatch_index`).

Receipt Persistence:
- Configure via `GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER=memory|tsa_stub` (prototype providers) and `GAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_PATH` (JSONL chain file).
- Chain semantics: Each appended receipt records `PrevHash` and cumulative `ChainHash` (rolling hash binding sequence order) enabling simple tamper-evidence.

Metrics Added:
- `combined_anchor_emitted_total` – Incremented on successful combined emission.
- `combined_anchor_failures_total` – Incremented on failure paths (no store, missing hashes, append errors).
- `external_anchor_chain_verifications_total` (planned follow-up) – Will count verification loop & manual endpoint calls.
- Existing external anchor metrics continue to record provider success/failure and forced failure conditions.

Failure Modes & HTTP Codes:
- 503: Receipt store not configured (provider or path missing).
- 400: Neither capability nor rotation hash available (no material to combine).
- 500: Append failure to receipt store.
- 401: Unauthorized combined emission when `GAUTH_COMBINED_ANCHOR_TOKEN` set and header `X-Combined-Anchor-Token` missing or mismatched.

Security / Integrity Notes:
- Duplicate combined hashes (idempotent emissions) are accepted for prototype; future versions may enforce deduplication or temporal spacing.
- Rolling `ChainHash` strengthens auditability over time; planned integrity validation endpoint will verify continuity and absence of gaps.

Planned Follow-ups:
- Automatic periodic combined emission (interval + jitter) instead of manual POST.
- Dedicated metric for chain verification invocations.
- Auth capability-based restriction (role/cap) rather than static token for emission.
- Integration of rotation ledger head once rotation subsystem exposes stable `HeadHash()` updates.
- Provider abstraction for third-party timestamping (e.g., RFC3161, blockchain notarization).
- Chain verification API + signed chain snapshots.
- Per-algorithm anchor counters (e.g., differentiating Ed25519 vs BLS aggregated anchor events).

Testing:
- `TestCombinedAnchorEmission` validates happy path emission, chain listing, metric increments, and idempotent second emission.

### Periodic Emission (New Optional Background Loop)
Set `GAUTH_COMBINED_ANCHOR_INTERVAL_SEC` (>=30) to enable an automatic background loop that emits a combined anchor receipt at the specified interval. Behavior:
- Skips emission silently when neither capability nor rotation hash is available (avoids inflating failure metrics pre-initialization).
- Increments `combined_anchor_failures_total` if the receipt store is configured incorrectly or append fails.
- Uses current rotation ledger head (when initialized) concatenated with the capability registry hash (`cap_hash:rotation_head`).
- Disabled when `GAUTH_DISABLE_BG_POLLS=1`.

Operational Guidance:
1. Export `GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER` and `GAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_PATH`.
2. Export `GAUTH_COMBINED_ANCHOR_INTERVAL_SEC=300` (example for 5m cadence).
3. Observe periodic emission logs `[combined-anchor] emitted hash=...` and metric growth.

### Rotation Ledger Integration
When `GAUTH_CAP_ANCHOR_NOTARIZE=1` and `GAUTH_ROTATION_LEDGER_PATH` are set, the combined emission now incorporates the current rotation ledger head hash. Response payload of `POST /api/v1/anchor/emitCombined` includes `rotation_head` (empty when ledger absent) enabling downstream audit differentiation between pure capability anchors and capability+rotation continuity anchors. Test coverage: `TestCombinedAnchorEmissionWithRotation` asserts hash derivation `sha256(cap_hash+":"+rotation_head)` and chain persistence.

Operational Guidance:
1. Set `GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER=memory` and `GAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_PATH=/path/to/receipts.jsonl`.
2. Issue `POST /api/v1/anchor/emitCombined` once capability registry hash computed (or after a change reload).
3. Inspect chain via `GET /api/v1/anchor/chain` (expect appended receipts with evolving `ChainHash`).
4. Monitor metrics for emission and failure counters to baseline reliability.

### Aggregated BLS Signatures (Prototype)
`POST /api/v1/crypto/bls/aggregate` adds compressed multi-signer issuance & verification.
Modes:
* issue: supply `message_b64` + `participants` (<=64) or explicit `private_keys_b64[]`.
* verify: supply `aggregated_signature_b64`, `public_keys_b64[]`, and original `message_b64`.

Issue Response: `aggregated_signature_b64`, `public_keys_b64[]`, `key_ids[]` (first 12 hex of sha256(pubkey)), `participant_count`, `latency_ms`.
Verify Response: `valid`, `participant_count`, `latency_ms`.

Metrics:
* `multi_signature_batch_size_count|total|max`
* `multi_signature_aggregate_latency_count|total_ns|max_ns`
* `multi_signature_verifications_total`
* `multi_signature_verification_failures_total`

Errors: 400 (`invalid_mode`, `missing_message`, `no_private_keys_or_participants`, `participants_too_large`, decode/deserialize failures), 500 (`bls_init_failed`).

Security Notes:
* Ephemeral issuance mode does not persist secrets; caller must retain returned public keys & aggregated signature.
* Rogue key mitigation (proof-of-possession) planned; not yet enforced.
* All signatures MUST cover identical canonical bytes; caller ensures canonicalization.

Example:
```jsonc
{"mode":"issue","message_b64":"aGVsbG8gd29ybGQ=","participants":3}
{"mode":"verify","message_b64":"aGVsbG8gd29ybGQ=","aggregated_signature_b64":"...","public_keys_b64":["...","...","..."]}
```

Planned Follow-ups: proof-of-possession, identity + weight mapping, persistent aggregated signature audit, Prometheus CounterVec for per-algorithm anchor emissions (`ed25519`,`ecdsa-p256`,`bls12-381`,`bls12-381-agg`), extended negative tests (empty key set, truncated signature), rotation ledger contextual binding.

### New Metrics (Incremental Observability Upgrade)
This release also adds a labeled Prometheus counter for per-algorithm capability anchor emissions:

- `capability_anchor_algorithm_emitted_total{algorithm="<name>"}` – increments once per capability anchor emission per cryptographic algorithm involved (multi-algorithm emissions call increment for each). Empty algorithm names normalized to `_`.

Purpose:
- Enables tracking adoption / migration across signature algorithms (e.g., measuring rollout of `bls12-381-agg`).
- Supports capacity & security analytics (detect unexpected algorithm mix changes).

Memory adapter previously exposed per-algorithm counts internally; Prometheus parity now achieved via `IncCapabilityAnchorAlgorithm`. Existing unlabeled counters remain unchanged (`capability_anchor_emitted_total`, `combined_anchor_emitted_total`).

Initial Algorithms Emitted (example taxonomy): `ed25519`, `ecdsa-p256`, `bls12-381`, `bls12-381-agg`.

Operational Guidance:
1. Scrape `/metrics` and watch the new counter distribution over time.
2. Create Grafana panels aggregating by `algorithm` to visualize adoption curves.
3. Alert on sudden appearance of unknown algorithms (regex filter not matching approved set).

Planned Extensions:
- Algorithm sunset / deprecation gauges.
- Ratio gauge (per algorithm vs total anchors) for automated deprecation readiness checks.
- Integration with rotation ledger anchoring pipeline for composite anchor algorithm reporting.

### Additional Metrics – Phase A Instrumentation
The metrics interface and Prometheus adapter were expanded to support deeper latency and failure analytics:

Attestation Proof:
- `attestation_proof_issue_latency_seconds` – issuance latency histogram (sub-ms to 100ms buckets).
- `attestation_proof_verification_failure_reason_total{reason}` – labeled failure reasons (`digest_mismatch|algo_mismatch|key_mismatch|missing_anchor|other`).

Capability Anchor Algorithms:
- `capability_anchor_algorithm_ratio{algorithm}` – per‑algorithm emission ratio (0..1) vs total; complements raw count `capability_anchor_algorithm_emitted_total`.

Lifecycle Transition Latency:
- `lifecycle_transition_latency_quantile{entity,outcome,quantile}` – computed quantile gauges (p50,p95,p99) derived from latency histograms / reservoirs.

Usage Notes:
1. Ratio gauges approximate recent distribution; sum across algorithms should be ~1.0 (floating error tolerated).
2. Quantile gauges update periodically via in-process scheduler (future PR) or test harness helper; they do not replace raw histogram but simplify SLO alerting.
3. Failure reason counters enable fine-grained alerting (e.g. spike in `digest_mismatch` may indicate canonicalization regressions).

Planned Follow-ups:
- External anchor chain verification counters (integrity scan cadence).
- Snapshot issuance counters (`anchor_snapshot_issued_total`).
- Notary provider latency/failure metrics (`notary_timestamp_latency_seconds`).



### BLS Proof-of-Possession (PoP) Flow (Prototype)
Added a PoP challenge issuance path to the aggregated BLS endpoint and a dedicated verification endpoint to prove holders possess private keys corresponding to published BLS public keys without exposing aggregated signature semantics.

Issuance:
- Request: `POST /api/v1/crypto/bls/aggregate` with `{ "mode":"issue", "message_b64":..., "participants":N, "require_pop":true }`.
- Response: `{ mode:"issue_pop", participant_count, public_keys_b64[], key_ids[], challenges_b64[], latency_ms }`.
- Challenge derivation: `challenge = SHA256("gauth-pop:" || public_key_bytes || nonce16)`.
- Metrics: increments `bls_pop_challenges_issued` per generated challenge.
- Optional (test-only) private key export when `GAUTH_ALLOW_POP_PRIV_EXPORT=1` adds `private_keys_b64[]` (DO NOT enable in production).

Verification:
- Request: `POST /api/v1/crypto/bls/pop/verify` with `{ "pairs":[{"public_key_b64","signature_b64","challenge_b64"}, ...] }` where each `signature_b64` is a BLS signature over the raw challenge bytes.
- Response: `{ success, valid, total, failures, failure_indices[], latency_ms }`.
- Metrics: `bls_pop_verifications` per successful signature; `bls_pop_verification_failures` per failed signature.

Design Notes:
- PoP flow decouples key possession attestation from aggregated signature issuance to avoid inflating multi-signature latency or revealing private material.
- Challenges embed a per-participant nonce ensuring uniqueness and replay resistance; deterministic domain prefix prevents cross-protocol confusion.
- Verification processes each tuple independently, permitting partial failure introspection (`failure_indices`).

Security Guidance:
- Never expose `GAUTH_ALLOW_POP_PRIV_EXPORT` outside controlled test environments.
- Rotate keys regularly; PoP issuance can be integrated before rotation cut-over to ensure new keys are live.
- Future hardening: signed challenge bundles, expiration timestamp, and optional aggregated PoP attestation artifact.

Test Coverage:
- `TestBLSPoPIssueAndVerify` – end-to-end issuance + verification success (3 participants).
- `TestBLSPoPFailure` – tampered signature triggers failure increments and invalid aggregate result.

Observability:
- Snapshot (`SnapshotEx`) now includes PoP counters: `bls_pop_challenges_issued`, `bls_pop_verifications`, `bls_pop_verification_failures`.
- Prometheus adapter exposes matching counters (`bls_pop_challenges_issued_total`, `bls_pop_verifications_total`, `bls_pop_verification_failures_total`).

Planned Enhancements:
- Challenge expiration & replay cache.
- Batch PoP aggregation metrics (average challenge issuance latency, size histogram).
- Optional pairing-friendly aggregated PoP verification (single operation for N proofs).


### OpenAPI Spec Expansion (0.3.2-beta)
The OpenAPI contract (`docs/openapi.yaml`) was updated and version-bumped from `0.3.1-beta` to `0.3.2-beta` to include newly implemented cryptographic and anchoring endpoints:

New Paths Added:
| Path | Method | Purpose |
|------|--------|---------|
| `/api/v1/crypto/algorithms` | GET | Enumerate registered signature algorithms and aggregated capability flag |
| `/api/v1/crypto/bls/aggregate` | POST | Issue aggregated BLS signature or PoP challenges; verify aggregated signature |
| `/api/v1/crypto/bls/pop/verify` | POST | Verify per‑participant BLS Proof‑of‑Possession signatures |
| `/api/v1/anchor/chain` | GET | Retrieve external anchor receipt chain |
| `/api/v1/anchor/emitCombined` | POST | Manually emit combined capability+rotation anchor receipt |
| `/api/v1/anchor/verifyChain` | GET | Verify integrity of external anchor receipt chain |

Schema Highlights:
- `bls/aggregate` request differentiates `mode: issue|verify` with optional `require_pop` triggering `issue_pop` response variant.
- PoP verification request uses `pairs[]` objects (public_key, signature, challenge) enabling partial failure reporting.
- Anchor chain entries expose `hash`, `prev_hash`, cumulative `chain_hash`, and `latency_seconds` for audit timing.

Testing:
- Added `TestOpenAPISpecPaths` asserting all new path substrings and version bump are present in `docs/openapi.yaml`.

Operational Guidance:
1. Serve updated spec via existing discovery endpoint (`/api/v1/openapi`).
2. Integrate client generation workflows (e.g. `oapi-codegen`, `openapi-generator`) against version `0.3.2-beta`.
3. Monitor future additions (planned: per‑algorithm anchor counters, PoP expiration metadata) for incremental spec updates.

Planned Follow-ups:
- Add schemas for future PoP batch verification aggregate artifact.
- Tag grouping refinements (`anchor`, `crypto`, `bls`) for clearer segmentation.
- Error response components (standardized `error_code`, `error_message`).
- Security scheme declarations for combined anchor emission token and future capability-based auth model.

Impact:
- Enables external tooling alignment (SDK generation, compliance scanning).
- Reduces drift between implementation and formal contract.
- Establishes foundation for automated spec diff in CI to detect undocumented endpoints.


