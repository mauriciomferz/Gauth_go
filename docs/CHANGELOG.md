---
title: Documentation Changelog (Supplemental)
category: release-notes
status: active
lastUpdated: 2025-11-12
owners: release-manager
source: manual-curation
refreshCadence: monthly
---
# CHANGELOG

## 2025-10-29 (PoA Beta MVP Integration)
### Delegation, Persistence & Extended Token Integrity
- Added legal & evidentiary fields to `PowerOfAttorney` (jurisdiction, witnesses, attestations, revocation metadata) excluded from canonical digest to preserve signature stability.
- Implemented minimal PoA issuance endpoint (`POST /demo/poa/issue`) and revocation endpoint (`POST /demo/poa/:id/revoke`).
- Linked enforcement decisions to PoA via `poa_id` and stored canonical digest (`poa_digest`) in decisions table (schema columns added).
- Added extended token issuance (`POST /demo/poa/:id/token`) producing HS256 JWT with embedded PoA integrity claims: `poa_id`, `poa_digest`, `poa_version` and capped lifetime (≤2h).
- Auth middleware now validates embedded PoA claims with integrity binding (revoked / expired / not found / digest mismatch / version mismatch) returning structured reasons (`poa_not_found`, `poa_revoked`, `poa_expired`, `poa_digest_mismatch`, `poa_version_mismatch`).
- Added metrics: `ai_demo_poa_validations_total{result}`, `ai_demo_poa_revocations_total`, and `ai_demo_poa_integrity_failures_total{reason}` (digest_mismatch|version_mismatch).
- Added optional BoltDB persistence for PoA records via `GAUTH_AI_DEMO_POA_DB_PATH` (fallback to in-memory map when unset). Includes secondary index bucket for principal queries (future expansion).
- Updated README to document new PoA endpoints, metrics, revised decision schema, troubleshooting entries, and JWT error reason extensions.

### Testing
- Added unit tests (`poa_validation_test.go`) covering success, revoked, expired, and scope mismatch PoA validation scenarios.

### Compatibility & Migration Notes
- Existing SQLite `decisions` tables automatically migrated: code issues `ALTER TABLE` for missing `poa_id` / `poa_digest` columns during startup (legacy manual migration guidance retained for reference). Recommended manual verification:
	```sql
	ALTER TABLE decisions ADD COLUMN poa_id TEXT;
	ALTER TABLE decisions ADD COLUMN poa_digest TEXT;
	```
- Extended token issuance depends on `GAUTH_AI_DEMO_JWT_SECRET`. Without it PoA issuance still functional but token endpoint returns error (`jwt_secret_not_configured`). Integrity claims (`poa_digest`, `poa_version`) only present when token issued.

### Follow-Up Roadmap (Updated)
- Persistence-backed PoA repository (BoltDB/Postgres) & audit ledger anchoring.
- Formal revocation reason taxonomy & structured codes.
- Sub-delegation depth controls and dual-control revocation workflow.
- Weighted multi-signature activation linking to issuance flow (integration with existing multisig manager).
- Potential next extended token format upgrade (separate key, explicit header version field beyond embedded `poa_version`).
- Compliance matrix doc update incorporating PoA lifecycle coverage.

## 2025-10-29 (AI Capability Demo Enhancements)
### Demo Feature Expansion
 - Added latency histograms `ai_demo_enforcement_duration_seconds{action}` and `ai_demo_conflict_batch_duration_seconds` for performance visibility.
 - Added optional decision log retention pruning via `GAUTH_AI_DEMO_DB_MAX_ROWS` (keeps most recent rows, deletes oldest beyond limit after each insert).
 - Added age-based decision pruning (`GAUTH_AI_DEMO_DB_MAX_AGE_DAYS`) and statistics endpoint `/demo/decisions/stats` (total, oldest/newest timestamps, top actions). Introduced prune operations counter `ai_demo_prune_operations_total`.
 - Added decision store row gauge `ai_demo_decisions_store_rows` (tracks current persisted decision count) updating after retention/age pruning.
- RS256 JWT verification via JWKS (env: `GAUTH_AI_DEMO_JWKS_URL`, `GAUTH_AI_DEMO_JWKS_CACHE_SECONDS`, optional `GAUTH_AI_DEMO_JWT_EXPECT_ALG`) with new error codes (`unsupported_alg`, `jwks_fetch_error`, `kid_not_found`, `rsa_verification_failed`). Added JWKS metrics: `ai_demo_jwks_fetch_total{result}` and `ai_demo_jwks_keys_loaded` gauge for key cache size.

- JWKS resilience improvements:
	- Background proactive refresh (env: `GAUTH_AI_DEMO_JWKS_BG_REFRESH_FACTOR`, default 0.5) to refetch keys before cache expiry, smoothing latency and reducing cold path fetch races.
	- Negative KID caching (env: `GAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS`) to store missing `kid` lookups for a short TTL, suppressing redundant JWKS fetch attempts for invalid or noisy tokens.
	- Added negative cache metrics: `ai_demo_jwks_negative_hits_total` (count of served negative cache hits preventing network fetch) and `ai_demo_jwks_negative_entries` (current size of negative cache) for operational visibility into invalid kid traffic patterns.
	- Added background refresh metrics: `ai_demo_jwks_bg_refresh_total` (number of proactive refreshes) and `ai_demo_jwks_cache_ttl_remaining_seconds` (live TTL gauge) for monitoring refresh efficacy and detecting timing anomalies.
	- Introduced `GAUTH_AI_DEMO_JWKS_NEGATIVE_MAX_ENTRIES` to cap negative cache size with oldest-entry eviction to prevent unbounded memory growth under malicious random kid sprays.
		- Added eviction metric `ai_demo_jwks_negative_evictions_total` to quantify pressure on negative cache capacity; supports detection of random kid spray attempts.
		- Added optional jitter control `GAUTH_AI_DEMO_JWKS_BG_REFRESH_JITTER_SECONDS` to de-synchronize background JWKS refresh across horizontally scaled replicas, reducing coordinated thundering herd after deploy or TTL boundary.
		- Added targeted unit test `TestAuthMiddleware_RS256_NegativeKidEviction` ensuring eviction increments metric and maintains capped cache size.

- Added constant-time signature comparison helper to reduce timing disclosure risk.

### Documentation Updates
- Extended `examples/ai_capability_demo/README.md` with new environment variables, persistence schema, decisions pagination, conflict simulation usage, tracing spans, JWT verification details, and metrics note.

### Backward Compatibility
- Existing demo usage without new env vars continues to function (all features gated by env presence).
- No breaking changes to capability matrix or existing enforcement APIs.

### Follow-Up / Roadmap
- Potential additions: Prometheus metrics exporter, richer JWT claim validation (`aud`, `iss`, `nbf`), filtering / search on decisions endpoint, OTLP exporter integration, retention policies for decision log.
- Consider promoting conflict simulation into core API set if adoption increases.

---

## 2025-10-25 (Post Multi-Sig Remediation)
### Security & Integrity Enhancements
- Canonical digest upgraded: automatic domain V2 when `Threshold > 1`, binding `thr` and sorted embedded weights to prevent replay/confusion attacks between single vs aggregated signature contexts.
- Embedded `Version` and deterministic `weights` object into canonical JSON (future evolution + integrity across weighted signer sets).
- Multi-signature weights now stored inside `PowerOfAttorney` (no env injection), eliminating configuration race and increasing auditability.
- Structural validation for weighted threshold signatures: positive weights, subset of signers, cumulative weight ≥ threshold.
- Strict authenticity enabled by default (missing signature public key -> integrity failure). Override only via `GAUTH_STRICT_AUTHENTICITY=0` for transitional migration.
- Mandatory `jti` claim enforced unless `GAUTH_ALLOW_MISSING_JTI=1` set, closing trivial replay gaps without a replay store.

### Test Suite Updates
- Refactored canonical property tests (`canonical_prop_test.go`) to use embedded weights instead of env variables.
- Added domain transition test (`canonical_domain_v2_test.go`) verifying digest changes without canonical JSON mutation.
- Introduced version/weights presence test (`canonical_version_weights_test.go`).
- Adjusted rotation & strict authenticity tests to reflect default strict mode (`rfc0111_rotation_test.go`, `rfc0111_strict_auth_test.go`).

### Documentation
- New compliance matrix: `docs/rfc0111_compliance_matrix.md` summarizing Implemented / Partial / Missing clauses & gap roadmap.
- API README updated with compliance summary and remediation highlights.

### Backward Compatibility Notes
- Single-sig PoAs (Threshold=1) retain V1 domain (digests unchanged).
- Multi-sig PoAs previously relying on `GAUTH_MULTI_SIG_WEIGHTS` will produce different digests when re-issued with embedded weights; re-issue recommended.
- Strict authenticity may surface new integrity failures for legacy delegations missing public key—use env override during phased rollout.

### Roadmap (Next Targets)
- Algorithm agility (add ECDSA/BLS PoA signature provider abstraction).
- External audit ledger anchoring + signed entries.
- Partial revocation & suspension states; delegation depth limits.
- OpenAPI/Discovery contract & OTEL tracing integration.
- Durable replay store with snapshot/compaction.

---
