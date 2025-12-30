---
title: Project Changelog
category: release-notes
status: active
lastUpdated: 2025-11-12
owners: release-engineering
---
# Changelog

The format is inspired by Keep a Changelog and uses date-based sections.

## 2025-12-11
### Added - GNAP Integration (RFC 9635)
- **Grant Negotiation**: Full RFC 9635 grant request/response implementation with continuation support
- **Interaction Modes**: Redirect and user_code interaction patterns with hash verification (§4.2.3)
- **Token Management**: Token rotation and revocation endpoints
- **HTTP Message Signatures**: RFC 9421 support for Ed25519, ECDSA, RSA key types
- **Discovery Endpoint**: `/.well-known/gnap-as-rs` for client configuration
- **Auth Middleware**: Signature verification and token validation middleware
- **PoA Bridge**: Link GNAP grants to Power of Attorney credentials
- **Prometheus Metrics**: 6 metrics for grants, tokens, and signature verification
- **Audit Logging**: 6 new event types for GNAP operations

### Files Created
- `pkg/gnap/` - Core types, stores, interaction handling
- `pkg/gnap/httpsig/` - RFC 9421 HTTP signatures
- `web/handlers/gnap/` - Endpoints, middleware, metrics, PoA bridge
- `examples/gnap_client/` - Working client example
- `docs/GNAP_DEPLOYMENT.md` - Production deployment guide

### Documentation
- `docs/RFC_MAP.md` updated with RFC 9635 clause mappings
- `README.md` updated with GNAP features

## 2025-11-05
### Added
- **Interactive Web Features**: Complete dynamic pattern simulation system with 25+ unique authorization patterns
- **Pattern Explorer**: Fully dynamic pattern loading and simulation without hardcoded scenarios
- **GAP Matrix Updates**: Comprehensive status updates with 10 implemented features (up from 8)
  - Model limit checks (sec11.item2): Missing → Implemented
  - Delegation depth limits (sec12.item2): Missing → Implemented  
  - Threat model synchronization (sec14.item1): Partial → Implemented
  - Residual risk register (sec14.item2): Missing → Implemented
  - Audit ledger (sec5.item1): Partial → Implemented
  - OpenAPI specification (sec10.item1): Partial → Implemented
- **Implementation Progress Summary**: Added comprehensive roadmap and milestone tracking in GAP_MATRIX.auto.md
- **Test Coverage**: 100% conformance (8/8 clauses, 24/24 symbols)

### Fixed
- **Web Interface**: Fixed non-working "Check Authorization" and "Generate Power of Attorney" buttons
- **Jurisdiction Validation**: Corrected invalid jurisdiction from 'US-CA' to 'US' (valid jurisdictions: US, EU, UK, DE)
- **Pattern Simulation**: Removed all hardcoded pattern IDs and data-action attributes
- **Authorization Demo**: Updated defaults to show ALLOW decisions with alice@example.com and report:finance
- **JavaScript Errors**: Fixed incomplete scenario object causing page breakage

### Changed
- **Status Summary**: Updated to 10 Implemented, 24 Partial, 9 Missing (from 8/23/12)
- **Priority Categorization**: Reorganized gaps into P0-P3 tiers with clear roadmap
- **Documentation**: Enhanced GAP matrix with recent achievements section and next milestones (Q4 2025, Q1 2026)

### Deployed
- **GitHub Repositories**: Successfully pushed updates to both repositories
  - mauriciomferz/AgentAuth (main branch)
  - AgentAuth-Foundation/AgentAuth_Platform-AgentAuth_Server_Prototype (web-interactive-forms-fix branch)

- `DISCLAIMER.md` centralizing NOT production ready rationale and enumerating intentionally missing controls.
- Badge in `README.md` highlighting NOT production ready status.
- Documentation table row linking to `DISCLAIMER.md`.
- Cross-file disclaimer references added to: `docs/ARCHITECTURE.md`, `docs/CRYPTOGRAPHY_IMPLEMENTATION.md`, `docs/GETTING_STARTED.md`, `docs/TESTING.md`, `docs/PATTERNS_GUIDE.md`, `docs/API_REFERENCE.md`.
- `scripts/lint-disclaimers.sh` to detect unqualified production-readiness phrases.

### Changed
- Consolidated and standardized disclaimers across all markdown files (removed duplicate or divergent phrasing).
- Softened authoritative or compliance claims in architecture, cryptography, and combined RFC demo docs to educational wording.
- README disclaimers unified and linked to central disclaimer.

- Explicitly documented missing production features (key management, policy engine, observability, multi-tenancy, supply chain, etc.) to reduce misinterpretation.
- Added roadmap cues (see `docs/ARCHITECTURE.md` hardening section) for future production maturation efforts.

---
## 2025-10-13
### Changed
### Removed
### Deprecated
- Sliding window and distributed rate limiter benchmarks (skipped placeholder remains).
- Pending sync to `docs/API_REFERENCE.md` to reflect the above API shifts (will follow in same commit series).
### Added
- Policy engine expression enhancements: logical OR (`||`) and numeric comparisons (`> >= < <=`) with short‑circuit evaluation.
- New policy chain pagination endpoint: `GET /api/v1/beta/policy/chain?offset=&limit=` returning slice, total, verification metadata.
- Audit-policy consistency endpoint: `GET /api/v1/beta/policy/audit-consistency` detecting evaluation head drift / tamper.
- Store abstraction (`pkg/policy/store.go`) introducing `Store` interface and `InMemoryStore` implementation for future persistence backends.
- Benchmark cases for OR and numeric expressions plus audit anchoring coverage.
- Audit logger anchor callback (`MemoryLogger.SetAnchor`) for external root hash anchoring.
- Tamper detection tests for both audit event chain and policy bundle chain.

### Changed
- README updated with new endpoints, expression grammar, store abstraction section, limitations matrix extended (consistency & expressions).
- Policy evaluation provenance integrated into audit log entries (action `evaluate`).

### Internal / Tooling
- Expanded benchmarks measuring performance impact of additional expression operators.
- Added integration tests for pagination & consistency endpoints and stale head detection scenario.

### Limitations (Documentation)
- README now explicitly lists remaining expression gaps (NOT, parentheses, regex) and future persistence roadmap.

### Notes
- All features remain experimental & in-memory; production hardening (persistence, authenticated multi-tenant admin model, external anchoring) deferred to future milestones.

## 2025-10-16
### Added
- Initial RFC compliance planning matrix (`docs/RFC_COMPLIANCE_MATRIX.md`).
- PR template (`.github/pull_request_template.md`) standardizing merge metadata.
- PR summary draft (`docs/BETA_REFACTOR_PR_SUMMARY.md`) for `beta-refactor` merge.
- Bundle substitution test scaffolds (now structured skipped tests preparing for detection logic).
 - Policy bundle substitution tamper detection test (mid-chain mutation invalidates chain).
 - Delegation / POA chain with scope narrowing and expiry validation tests.
 - Minimal practical delegation section added to `docs/COMPLIANCE_IMPLEMENTATION.md`.
 - Revocation placeholder struct and skipped test scaffold.
 - Web asset production substitution helper `BetaServer.applyBundleSubstitution` with SRI and strict mode failure markers plus passing tests (SRI substitution & strict placeholder retention).
 - Delegation chain integration into `/api/v1/poa/authorize` with metadata response (`delegation.chain_verified`) and tests for success, scope widening denial, and expired delegation rejection.
 - Basic delegation scope enforcement (requested scope must contain delegated action token) plus scope violation and expired tests.
 - Delegation revocation enforcement (ID-based) in `/api/v1/poa/authorize` with tests for revoked head, middle, and success path (`web/delegation_revocation_test.go`).

### Changed
- `.gitignore` updated to exclude generated web artifacts (`web/static/js/app-*.js`, `asset-manifest.json`).
- Documentation clarified around partial compliance (matrix marks missing areas: delegation/POA, revocation, expiry, substitution detection, crypto specification).

### Tagged
- Release tag `v0.3.0-beta-refactor` capturing compliance scaffolds and asset hygiene improvements.

### Notes
 - Revocation logic, signature-based authenticity, and full cryptographic specification remain pending.
 - Next milestone targets: integrate delegation evaluation into authorization engine, implement revocation chain & signature scheme, expand RFC citation coverage.

## 2025-10-17
### Added
- Regex compilation cache lifecycle: LRU capacity limit (default 256) and optional TTL with eviction metrics.
- Pre-validation of regex patterns on policy reload (compile success/error counted prior to first decision use).
- Per-pattern successful match counters (aggregate total exported; internal per-pattern map retained for future top-N export).
- Decision latency histogram with fixed nanosecond buckets (50µs–100ms) exposed in Prometheus and metrics snapshot.
- Prometheus metrics: `authz_regex_matches_total`, `authz_regex_evictions_total`, latency bucket counters `authz_latency_bucket{le="..."}`.
- Tests: capacity eviction, TTL expiry eviction, regex pre-validation invalid pattern, match frequency, latency histogram.

### Changed
- Refactored regex evaluation to eliminate nil dereference panic and improper map writes under read lock.
- Documentation (`docs/AUTHORIZATION_IMPLEMENTATION.md`) expanded with sections for regex cache tuning, lifecycle, metrics, latency histogram interpretation, and observability reference.

### Fixed
- Concurrency race in decision caching metadata mutation (cache hit path now copies metadata map before annotation).
- Regex evaluation panic on invalid pattern due to dereference of nil compiled object.

### Metrics Summary
- New counters: compiles, compile errors, evictions, matches; histogram for latency distribution enabling SLO definition.

### Notes
- Per-pattern match export beyond aggregate total is deferred; future enhancement will include top-N patterns and adaptive bucket sizing.
- Histogram currently uses static buckets; consider HDR or dynamic buckets for production accuracy.
- Eviction strategy is simple LRU scan; acceptable at current cardinalities (< few thousand). Monitor if scaling up.

## 2025-10-18
### Added
- Full OpenAPI path coverage (44/44) with automated enforcement (specgen + Makefile `openapi_coverage`).
- Canonical parameter normalization (`:param` → `{param}`) in coverage tool to eliminate false negatives.
- CI workflow `openapi-coverage.yml` failing builds when coverage drops below 100%.
- ETag + optional HMAC signature headers for discovery, JWKS, and OpenAPI spec endpoints (integrity & caching).
- Deprecation timeline metadata (`deprecated_after`, `sunset_after`) surfaced in discovery document.
- Health/info/ping endpoints and composite export endpoints documented and implemented.
- README documentation section "OpenAPI Route Coverage Enforcement" with usage guidance and failure modes.

## 2025-10-20
### Added
- Revocation auto-sign instrumentation: internal counters (emitted, skipped_empty, skipped_duplicate) with Prometheus exposition endpoint `GET /api/v1/beta/metrics/revocation/auto-sign/prometheus` and OpenTelemetry observable gauges (`agentauth_revocation_auto_sign_emitted`, `..._skipped_empty`, `..._skipped_duplicate`).
- Prometheus test coverage ensuring revocation counters appear with expected values after simulated rotations (`web/revocation_prometheus_metrics_test.go`).
### Changed
- OpenTelemetry metrics initialization now guarded via `sync.Once` to prevent duplicate exporter setup when multiple `BetaServer` instances are constructed in test processes (eliminates noisy duplicate metric registrations and ensures idempotent instrumentation).
### Internal / Testing
- Production accessor for revocation auto-sign counters removed; replaced by test-only accessor (`revocation_metrics_access_test.go`) to avoid expanding public API surface.
### Notes

## 2025-10-21
### Added
- Multi-signature verification granular metrics: counters for structural, digest, public key missing, invalid signature, threshold, and weight failures plus dedicated latency histogram (`multi_signature_verification_latency_seconds`).
- Satisfied signature/weight fields surfaced in `PowerOfAttorney` struct after successful multi-signature validation.
- Domain separation v2 (`AGENTAUTH_MULTI_SIG_DOMAIN_V2=1`) embedding threshold and sorted weight map into digest prefix to prevent cross-mode collisions.
### Changed
- Hardened weight map parsing (duplicate keys, unknown signer, non-positive / overflow weights, total possible weight < threshold invalidates weighted mode fallback to count-based semantics).
- Verification path instrumentation now records granular counters before generic failure counter increments.
### Documentation
- `docs/OBSERVABILITY.md` updated with new multi-signature metrics taxonomy, PromQL alert examples, migration guidance for domain v2.
### Notes
- Domain v2 disabled by default for backward compatibility; enabling requires re-issuance of multi-signature delegations. Future roadmap includes dual-domain acceptance window.

### Added (Migration & Sunset Instrumentation)
- Envelope versioning rollout monitoring: gauge `agentauth_rfc0111_envelope_v2_adoption_ratio` computing V2 issuance ratio vs total (updated each issuance).
- Digest integrity counter `agentauth_rfc0111_envelope_digest_mismatch_total` detecting canonical digest mismatches during verification.
- `OBSERVABILITY.md` extended with rollout monitoring & mismatch troubleshooting sections (alert examples, ratio thresholds, security considerations).
- ADR `docs/ADR-envelope-v1-sunset.md` defining phased deprecation lifecycle: Pilot → Broad → Stabilization → Soft Deprecation → Sunset → Post-Verification.
- GAP matrix updated to reflect implemented adoption ratio gauge & mismatch counter, and to adjust remediation targets toward sunset execution tasks.
- Prometheus recording & alert rules scaffold (`deployments/observability/recording-rules-envelopes.yaml`) providing adoption smoothing, mismatch ratio, readiness composite, regression & spike alerts, and sunset readiness alert.

### Changed (Governance Docs)
- `AAP-001_0115_remediation_plan.md` updated: new epic for Envelope V1 Sunset Execution; success metrics extended with adoption & mismatch criteria.

### Added (Cadence & Phase Instrumentation)
- Issuance cadence histogram (`agentauth_rfc0111_envelope_issuance_cadence_seconds`) recording inter-issuance intervals.
- Sunset phase gauge (`agentauth_rfc0111_envelope_v1_sunset_phase`) exposing lifecycle phase (Pilot→Post-Verification) for V1 deprecation governance.

### Added (Policy Governance)
- Policy bundle versioning: each appended bundle assigned monotonically increasing `version` incorporated into provenance hash.
- Rollback API endpoint: `POST /api/v1/beta/policy/rollback?version=NN` allowing temporary reversion to historical bundle without mutating chain history.
- Policy evaluation response now includes `policy_version` tagging decisions with effective bundle version for audit correlation.
- Chain pagination endpoint extended with `versions` array aligned to returned `hashes` plus `active_version` field.
- Prometheus metrics: `agentauth_policy_revisions_total` (counter) and `agentauth_policy_active_version` (gauge) for governance observability and rollback detection.
- Test coverage: version auto-increment, rollback activation, rollback clearing on new append, evaluation version tagging (`web/policy_version_rollback_test.go`).

### Changed (Policy Governance)
- `EvalDecision` extended with `policy_version`.
- `/api/v1/beta/policy/bundles` response now returns `policy_version` for the appended head.
- `/api/v1/beta/policy/evaluate` response schema extended with `policy_version`.
- `/api/v1/beta/policy/chain` response extended with `versions` and `active_version` fields.

### Notes (Policy Governance)
- Rollback sets a head override pointer; subsequent append clears override restoring forward progression.
- Version included in hash calculation ensures immutable provenance (hash changes on version mismatch tamper).
- Future roadmap: RBAC on rollback, audit event emission for rollback operations, persistence backend, Merkle accumulator for bundle contents.
- Mismatch reason labeling (canonicalization_error, tamper_suspected, domain_conflict).
- Automated phase controller referencing adoption ratio & mismatch thresholds.
- Dual-domain acceptance window for multi-signature digest migration (optional).

### Notes (Migration Risk)
- Rapid drops in adoption ratio (<0.8 for 10m) should trigger regression investigation before continuing rollout.
- Sustained mismatch ratio >0.005 for any 1h window blocks sunset execution per ADR criteria.

- Future roadmap: optionally expose revocation counters in aggregated JSON metrics snapshot; consider histogram for time between auto-sign events and labeled reasons for skips beyond empty/duplicate.
- Coverage badge in README referencing workflow status.
 - Path parameter description coverage enforcement (`make openapi_param_coverage`, combined `make spec-contract`).
 - Query parameter description coverage enforcement (`make openapi_query_param_coverage`).
 - Schema property description coverage enforcement (`make openapi_schema_prop_coverage`).
- Operation example coverage enforcement (`make openapi_example_coverage`) ensuring each operation includes ≥1 illustrative example.

### Changed
- Rebuilt and normalized `docs/openapi.yaml` (removed malformed fragments, added security scheme, applied array `maxItems`, cleaned quoting) to achieve lint cleanliness.
- Updated discovery and JWKS handlers to compute & return ETags consistently.
- Spec paths now strictly aligned with registered Gin routes; legacy placeholder discrepancies resolved.

### Tooling
- Introduced `internal/specgen` package and `cmd/specgen` CLI for repeatable coverage reporting (`docs/openapi.coverage.json`).
- Makefile enforcement target integrates into local developer workflow to catch drift pre-commit.
 - New parameter description coverage computation fields in coverage report (path_params_total, parameter_description_coverage, missing_param_descriptions).
 - Added query parameter coverage metrics (query_params_total, query_parameter_description_coverage, missing_query_param_descriptions).
 - Added schema property coverage metrics (schema_props_total, schema_property_description_coverage, missing_schema_prop_descriptions).
- Added operation example coverage metrics (operations_total, operation_example_coverage, missing_operation_examples) and integrated into `spec-contract`.
 - Added error response example coverage enforcement (`make openapi_error_example_coverage`) requiring all 4xx/5xx responses to include examples.
 - Extended coverage report with `error_responses_total`, `error_responses_with_example`, `error_response_example_coverage`, and `missing_error_response_examples`.
 - Updated `spec-contract` Makefile target to include error response example coverage gate.

### Integrity & Caching
- Standardized weak ETag computation (SHA256-derived) across spec-related endpoints.

### Documentation
- CHANGELOG updated to reflect coverage & normalization milestone.

### Notes
- Future enhancements: configurable coverage threshold, multi-file spec modularization, signed discovery responses, automated client generation pipelines.
- Lint style warnings in workflow resolved by restructuring branch/paths lists (avoiding redundant quoting).

### Cryptography Upgrade (Ed25519 JWKS)
#### Added
- Dual token signing modes (`AGENTAUTH_TOKEN_SIG_MODE=hmac|eddsa`) with Ed25519 (EdDSA) public-key issuance.
- Unified JWKS endpoint now publishes OKP Ed25519 keys (`kty=OKP`, `crv=Ed25519`, `alg=EdDSA`, `use=sig`, `kid`, `expires_at`).
- Discovery document fields: `eddsa_enabled`, `eddsa_keys`, `eddsa_rotation_hours` for client introspection.
- Manual rotation utility (`cmd/rotate-key`) and Makefile targets: `crypto-rotate`, `crypto-test`.
- HMAC compatibility tests ensure legacy issuance/validation unaffected.
#### Changed
- JWKS handler refactored to aggregate RSA/HMAC metadata and Ed25519 keys under single endpoint.
- README extended with comprehensive cryptography section (modes table, rotation workflow, environment variables).
#### Notes
- Key manager is in-memory only; persistence, automatic rotation scheduler, and revocation list exposure remain on roadmap.
- OKP key publication demo integrity only—do not rely on for production secrecy.

### Revocation Chain Signatures (Phase 2)
#### Added
- Ed25519 signatures for revocation events (`sig_kid`, `signature` fields on `RevocationEvent`).
- Signing performed at append-time using active rotating Ed25519 key (Phase 1 key manager).
- Verification extends hash chain validation to include signature integrity (unknown kid / bad signature produce failure).
- New endpoint `/api/v1/token/revocation/verify` returns per-event signature presence/validity and aggregate chain hash.
- Discovery metadata enhanced with `revocation_support.signatures_enabled` and `revocation_support.signing_kids`.
#### Changed
- `RevocationEvent` struct expanded; legacy unsigned events remain valid for hash linkage (backward compatible).
- README updated with "Revocation Chain Signatures" section including data model, usage, tamper rationale, and roadmap.
#### Notes
- In-memory key manager only—restart invalidates signing lineage; persistence & external anchoring deferred.
- No Merkle accumulator yet; aggregate hash remains whole-chain SHA256 composite.
- Unsigned events are treated as acceptable (signature optional); future milestone may enforce signature requirement post-stabilization.

## 2025-10-19
### Changed
- Conformance CLI consolidated: legacy deprecated entry point replaced with minimal stub printing migration notice. Canonical runner is `cmd/conformance`.
### Deprecated
- `conformance/cmd/agentauth-conformance` stub scheduled for removal after 2025-10-25; external scripts should migrate now.
### Notes
- Full removal will occur in subsequent refactor once CI/scripts confirm no lingering dependence on legacy path.

## 2025-10-20
### Added
- Rotation ledger signed summary Prometheus metrics:
	- `agentauth_rotation_summary_latency_seconds` (histogram)
	- `agentauth_rotation_summary_total{outcome}` (success/error)
	- `agentauth_rotation_summary_anchor_total{result}` (anchored/skipped/error)
- README documentation section: "Rotation Summary Metrics" with operational guidance, failure modes, and troubleshooting steps.
- Client verification helper (`pkg/verification/rotation_summary.go`) for fetching & verifying signed rotation summary (unit tests for valid + tampered signature).
 - Rotation summary gauges: `agentauth_rotation_summary_chain_length` and `agentauth_rotation_summary_head_age_seconds` providing instantaneous ledger size and freshness metrics; README updated.
 - Rotation summary gauge: `agentauth_rotation_summary_last_anchor_age_seconds` tracking seconds since last successful anchor to detect missed anchoring cadence; README updated.
 - Canonical rotation summary signing payload helper ensuring deterministic JSON ordering across signer and verifier (eliminates prior non-deterministic signature failures under map serialization). Added stability test and README rationale section.
 - External capability anchor receipt persistence (append-only hash chain):
	 - New env vars: `AGENTAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_PATH`, `AGENTAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_VERIFY_INTERVAL`.
	 - New endpoints: `/api/v1/beta/capabilities/anchor/external/receipts`, `/latest`, `/verify`.
	 - Metrics: `capability_external_anchor_receipts_integrity`, `capability_external_anchor_receipts_last_verify_age_seconds`, `capability_external_anchor_receipts_total`.
	 - Background verification loop with integrity + age gauges.
	 - Test coverage for persistence, chain integrity, and tamper detection (see `web/external_anchor_receipt_persistence_test.go`).
	 - Architecture & Getting Started docs updated with feature overview and alerting guidance.
	 - Deterministic test seeding env var `AGENTAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED` for `tsa_stub` provider (eliminates probabilistic flakiness in external anchor retry metrics tests; documented in `OBSERVABILITY.md`).
	 - Forced initial failures env var `AGENTAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS` enabling guaranteed failure-before-success sequences for retry metrics validation (takes precedence over probability; documented in `OBSERVABILITY.md`).
	 - New Prometheus counters distinguishing deterministic forced failures from probabilistic failures: `agentauth_rfc0111_external_anchor_forced_failures_total` and provider-labeled `agentauth_rfc0111_external_anchor_forced_failures_provider_total` (subset of `external_anchor_failures_total`). Provides clean separation for organic failure alerting; tests added (`web/capability_anchor_external_metrics_forced_failure_test.go`).
### Changed
- `/api/v1/beta/rotations/summary` endpoint instrumented to record latency, outcome, and anchoring result in a single code path.
### Notes
- Metrics extend existing rotation verification observability allowing separation between descriptor verification failures and summary generation/anchoring issues.
- Future enhancements planned: chain length gauge, last anchored head age gauge, outcome-labeled latency buckets, external client verification counters.
 - External anchor persistence roadmap: multi-provider quorum anchoring, Merkle accumulation for high-volume chains, signed chain head snapshots, TSA / transparency log real provider integration.

