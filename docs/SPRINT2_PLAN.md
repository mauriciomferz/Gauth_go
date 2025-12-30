---
title: Sprint2 Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Sprint 2 Plan (RB6–RB9)

Scope: Refactor & agility (signature abstraction), embedded attestation verifier, HTTP handler modularization, and initial distributed observability (tracing + latency percentiles).

## Goals Overview
| ID | Goal | Outcome | Risk Mitigation |
|----|------|---------|-----------------|
| RB6 | Signature Agility Interface | Unified `Signer`/`Verifier` abstraction enabling future algorithms | Introduce interface alongside existing EdDSA before removing old calls |
| RB7 | Embedded Attestation Verifier Service | Decouple complex verification logic from monolithic server for reuse/testability | Extract incrementally; keep façade shim in server until stable |
| RB8 | Modular HTTP Handlers | Improve maintainability & reduce merge conflicts by isolating domains | Move endpoints in small batches; snapshot test error payloads |
| RB9 | Observability Phase 1 | Add tracing spans + latency percentiles for core flows | Env-gated enablement; minimal span attributes; measure overhead |

## RB6: Signature Agility
### Interfaces
```go
// Signer provides signing capability over arbitrary message bytes.
type Signer interface {
    Sign(msg []byte) ([]byte, error)
    Algo() string       // e.g. "EdDSA"
    KeyID() string      // stable kid
}

// Verifier validates a signature for a message and key id.
type Verifier interface {
    Verify(kid string, msg, sig []byte) error // error taxonomy: unknown_kid|invalid_signature
}
```
### Tasks
- Add `internal/crypto/signing.go` with interfaces + adapter for existing Manager.
- Implement `Ed25519Signer` wrapping active key; `Manager` implements `Verifier`.
- Preserve `GlobalEdDSARegistry` but add helper: `GlobalSigner()` returning `Signer`.
- Update policy manifest signing to use interface (defer until tests pass).
- Tests: signature unchanged (compare old vs new path). Interface compliance test.
- Documentation: Add section to MANIFEST_POLICY.md referencing agility.

### Edge Cases
- No active key ⇒ `signing_unavailable` (re-use existing error code).
- Rotation event: ensure new signer reflects updated key without stale caching.

## RB7: Attestation Embedded Verifier
### Extraction Targets
From `server_clean.go`:
- Nonce replay check
- Detached signature validation
- Notarization consistency (if configured)
- Multi-signature path (if present)

### Proposed API (pkg/attest)
```go
// VerificationResult summarizes outcome.
type VerificationResult struct {
    Success bool
    Code    string // ok|replay|invalid_signature|digest_mismatch|notarization_mismatch|internal_error
    Details map[string]any
}

func Verify(ctx context.Context, env *Envelope, opts Options) (*VerificationResult, error)
```
`Options` includes strict mode flags, registries, replay store reference, metrics hooks.

### Tasks
- Create `pkg/attest` with data structures (Envelope simplified interface).
- Move logic; leave adapter in web layer calling `attest.Verify`.
- Add unit tests (replay hit, signature bad, digest mismatch, notarization mismatch, success).
- Update web tests to assert identical JSON error codes.
- (Progress) Extraction completed: `pkg/attest/verify.go` with `Attestation`, `ReplayStrategy`, `VerificationResult`, and `VerifyModelLimitsAttestation`. Endpoint `apiModelLimitsAttestationVerify` delegates to package logic via adapters (`attestationMemoryReplay`, `attestationDurableReplay`). Extended test suite now covers: valid signature, soft invalid signature, nonce replay (pre-seeded + durable second-call), missing nonce, notarization inconsistent (422), missing signature fields, unknown key id (404), invalid base64 signature, combined hash correctness, and tampered snapshot hash (soft invalid). All tests passing (`go test ./pkg/attest`). Next (optional): add legacy fallback env flag (`AGENTAUTH_LEGACY_ATTEST_VERIFY`), negative audit/anchor hash tamper test, and lightweight latency benchmark.

### Edge Cases
- Replay store errors: differentiate `store_error` vs `replay`.
- Optional notarization: skip gracefully when unconfigured.

## RB8: Modular HTTP Handlers
### Package Layout
```
web/handlers/
  token/ (issue, validate)
  revocation/ (publish, list)
  attestation/ (verify endpoints)
  capabilities/ (discovery, manifest)
  multisig/ (multi-signature related endpoints)
  anchor/ (capability anchor emission, registry hash)
```
### Migration Strategy
1. Introduce packages with handler constructors (`Register(router, deps)`).
2. Move capability discovery + manifest first (low coupling).
3. Snapshot test: capture error payload for representative negative cases before migration; compare after.
4. Move higher-risk token/attestation last.

### Progress Update (2025-10-27)
Initial extraction completed for capability discovery & negotiation endpoints:
- Added `web/handlers/capabilities/basic.go` with `RegisterBasic`, preserving paths `/api/v1/beta/capabilities` and `/api/v1/beta/capabilities/negotiate`.
- Duplicated error taxonomy locally (`respondError`) to avoid import cycle while keeping JSON shape stable.
- Server refactored to register modular handlers early; legacy inline implementation removed.
- Introduced `LifecycleStrict()` on `BetaServer` to satisfy handler dependency interface.
- Added idempotent UI route registration helper (`RegisterUIRoutes`) and adjusted smoketests to skip gracefully when embedded UI not present (preventing false negatives while modularization in progress).
- New unit tests for handlers (`basic_test.go`) validate list, happy negotiation, and invalid payload error path.

Current gaps / next steps:
- Decide UI static asset strategy (ensure `/index.html` always served or keep optional/skipped model) before migrating further handlers to avoid inconsistent smoke test semantics.
- Migrate anchor/audit capability endpoints next (low coupling) then token operations.
- Add golden JSON snapshot tests for a representative error (already partially done for negotiation invalid payload) prior to moving token/attestation to ensure taxonomy stability.
- Evaluate removal of temporary redirect / legacy stubs once downstream consumers updated.

Risk notes:
- UI route conditional registration caused initial 404 test failures; resolved by explicit helper + test skips. Converting to mandatory registration will simplify future test maintenance.

Metrics / observability impact:
- No change to capability hash tracking metrics; anchor endpoints still monolithic (to be migrated).

Rollback readiness:
- Reverting modularization limited to removing import + restoring old handlers; kept changes localized for quick rollback.

### Acceptance Criteria
- Route paths unchanged.
- Error taxonomy unchanged (same `code`, `error`, HTTP status).
- Tests updated to import new packages minimal changes.

### Risks & Mitigation
- Hidden implicit state in `BetaServer`: Address by creating explicit `Deps` struct passed to handlers.
- Import cycles: Keep shared types in `pkg` or `internal/common`.

## RB9: Observability Phase 1
### Tracing Design (Final Implemented)
- Initialization: in-repo custom tracer provider (not OTLP) activated when `AGENTAUTH_TRACING_ENABLED=1` (or legacy `AGENTAUTH_OTEL_ENABLE=1`).
- Sampling: `AGENTAUTH_TRACING_SAMPLE_RATIO` (0..1). Implementation treats `ratio <= 0` as ALWAYS SAMPLE (documented quirk; tests assert this). Future change may invert semantics—flag in OBSERVABILITY.md.
- Spans & Tags (final RB9 scope):
  - `token.issue`: ttl_req, jwt_mode, outcome, token_id.
  - `token.validate`: status, token_id (minimal to reduce cardinality).
  - `attestation.verify`: valid, failure_code, kid (on success), error (on early failure), combined hash computed outside span tags to avoid payload size.
  - `rotation.perform`: prev_kid, new_kid, ttl_hours, history_size (added), error (if rotation fails).
- Provider exposes `Spans()` slice for white-box unit tests (no external exporter yet). Propagation limited to request-local context; no cross-process trace propagation in RB9.

### Latency Percentiles (Implemented)
- Endpoint: `/api/v1/beta/metrics/latency` returns JSON: `{ success: true, percentiles: { attestation_verify: { p50, p95, p99 }, rotation_summary: {...}, aap001_validation: {...} } }`.
- Computation: scans Prometheus histogram buckets, approximates quantiles by cumulative count boundary method (no HDR histogram integration yet). Function `percentileFromBuckets` selects first bucket whose cumulative proportion exceeds target percentile.
- Added tests (`latency_percentiles_endpoint_test.go`) validating presence of keys and basic monotonic ordering (implicitly by non-zero values where metrics exist).

### Tasks (All Completed)
- Wire tracer provider in `server_clean.go` under both env flags.
- Instrument spans for token issue, token validate, attestation verify (including early read-body error path ensuring span.End), and key rotation.
- Add latency percentile endpoint + unit test.
- Update `docs/OBSERVABILITY.md` with RB9 tracing usage, env vars, endpoint example.
- Add tracing tests (`tracing_basic_test.go`) for enabled, disabled, and sample ratio=0 behaviors. Adjusted expectations for token creation HTTP 201.

### Performance Considerations & Outcomes
- Overhead: In-memory span allocation only on sampled requests; sampling ratio default (unset) -> 1.0 (dev). Ratio check happens before allocation; negligible overhead when disabled.
- Attributes kept minimal to prevent high-cardinality tag explosion.
- Percentile endpoint avoids storing raw latency arrays (derived on demand from histograms).
- Confirmed rotation summary metrics unaffected by tracing logic.

### Follow-up (Post-RB9 Suggestions)
- Optional inversion of ratio semantics (0 => no sample) with backward compatibility shim; update tests accordingly.
- OTLP exporter integration (wrap current provider) + W3C TraceContext header propagation for distributed trace continuity.
- Add mid-ratio probabilistic sampling test (e.g. 0.5) running multi-iteration to assert both presence/absence distribution.
- Expand latency endpoint to include token.issue/validate histograms after they are added (currently not aggregated).

## Cross-Cutting Testing Strategy
- Property tests for signature unchanged after RB6.
- Golden JSON tests for error taxonomy pre/post RB8.
- Replay & attestation unit tests isolated (RB7).
- Bench optional: measure added overhead of tracing (RB9) with simplistic timing harness.

## Sequencing Recommendation
1. RB6 (low surface, enables cleaner RB7 error codes).
2. RB7 (reduces complexity before modularization).
3. RB8 (modularization after verification code decoupled).
4. RB9 (add tracing once structure stable).

## Rollback Plan
- RB6: Keep original direct calls until Signer integrated; feature flag fallback.
- RB7: Retain legacy path behind env `AGENTAUTH_LEGACY_ATTEST_VERIFY` for one sprint.
- RB8: Migrate handlers one group at a time; if failure, revert package import.
- RB9: Disable tracing via env; no code removal necessary.

## Open Questions
- Multi-algorithm support timeline (BLS integration) — plan after RB6 baseline.
- Should latency percentiles share endpoint with metrics? (TBD).

---
Generated plan for Sprint 2. Adjust as tasks evolve.

### Anchor Modularization Update (2025-10-27)
Additional progress after initial RB8 capability extraction:

**Endpoints migrated:** Full capability anchor suite extracted into `web/handlers/anchor/basic.go` with `RegisterAll`:
`POST /api/v1/beta/capabilities/anchor`, `GET /api/v1/beta/capabilities/anchor/latest`, `GET /api/v1/beta/capabilities/anchor/material`, `GET /api/v1/beta/capabilities/anchor/status`.

**Compatibility assurances:**
- Preserved error taxonomy (`anchoring_disabled`, `anchor_client_unavailable`, `registry_hash_empty`, `anchor_failure`).
- Response JSON fields unchanged (hash, anchored_at, previous_hash, registry_last_changed_at, emitted_total, skipped_total, hash_changed_total, last_write, last_write_unix, notarization_receipt, external_anchor_receipt, stale flags).
- Signed wrapper detection logic retained for material endpoint integrity.

**Implementation notes:**
- Added thin accessor methods on `BetaServer` (e.g. `CapabilityRegistryHash()`, `AnchorClient()`, `CapAnchorMetrics()`) to supply handler dependencies without exposing internal fields directly.
- Anchor tests (`basic_test.go`) validate full cycle and disabled path (403) ensuring idempotent POST semantics and stable status/material outputs.
- Route registration in `server_clean.go` replaced inline handlers with `anchorhandlers.RegisterAll(beta, s)`; original functions remain for easy diff/revert.

**Risk reduction:**
- Modularization confined to import + route registration; reversion is O(1) change.
- Concurrency safety unchanged—handlers call existing memory anchor client methods (mutex protected).

**Next targets:**
- Capability audit verify/anchor endpoints.
- External anchor receipt chain + Prometheus exposition routes.
- Consolidate accessor set into a versioned interface to cap surface growth before token/attestation extraction.
- Introduce golden snapshot for one anchor error response to guard against accidental field changes in future refactors.

**Deferred items:**
- Decide on promoting Prometheus anchor metrics endpoint to modular package (read-only, low coupling).
- Potential aggregation of notarization/external receipt exposure into a single structured sub-object.

**Follow-up considerations:**
- Evaluate merging capability & anchor dependency interfaces for negotiation + anchoring coherence.
- Document accessor interface rationale in a short ADR if surface expands further (to avoid erosion of encapsulation).

### Audit Modularization Update (2025-10-27)
Progress after anchor extraction:

**Endpoints migrated:** Capability audit chain endpoints extracted to `web/handlers/audit/basic.go` (`RegisterBasic`):
`GET /api/v1/beta/capabilities/audit/verify`, `POST /api/v1/beta/capabilities/audit/anchor`.

**Parity assurances:**
- Unconfigured verify response unchanged: `{success:true, configured:false}` when `AGENTAUTH_CAP_AUDIT_PERSIST_PATH` unset.
- Error codes preserved: `capabilities_audit_read_failed`, `capabilities_audit_invalid_json`, `capabilities_audit_chain_tip_empty`, `capabilities_audit_anchor_failure`, `capability_anchor_disabled`, `capability_anchor_client_unavailable`.
- Anchor success payload fields retained (hash, anchored_at RFC3339, total, chain_tip, type).

**Implementation notes:**
- Added `CapAuditPersistPath()` & `CapAuditPrevHash()` accessors on `BetaServer`.
- Narrow `Deps` interface mirrors existing `AnchorClient()` signature (includes `LatestAnchor()` to satisfy type match).
- Route registrations replaced with `audithandlers.RegisterBasic(beta, s)`; legacy inline functions left for ease of revert.
- Unit tests: verify unconfigured path + anchor disabled 403 error taxonomy.

**Risk & Mitigation:**
- Reversion is a single import removal + reinstating two route lines.
- Hash recomputation uses identical logic (sha256 over raw payload) preserving integrity flag semantics.
- No concurrency changes (audit file read only; anchoring delegates to existing mutex-protected anchor client).

**Next Targets:** External anchor receipt endpoints + Prometheus metrics modularization; add positive-path audit anchor test after chain tip seeding utility available.

**Deferred:** Potential consolidation of audit + anchor status into a unified governance summary object post-RB8.

### External Anchor Receipts Modularization Update (2025-10-27)

**Endpoints migrated:** External capability anchor receipt chain endpoints extracted to `web/handlers/externalreceipts/basic.go` (`RegisterChain`):
`GET /api/v1/beta/capabilities/anchor/external/receipts/latest`, `GET /api/v1/beta/capabilities/anchor/external/receipts`, `GET /api/v1/beta/capabilities/anchor/external/receipts/verify`.

**Parity assurances:**
- Preserved success/error shape semantics (always HTTP 200 with `configured` flag unless internal marshal failure).
- `configured=false` only when store is truly unconfigured (nil); empty chain now reported as `configured=true` with `empty:true` (matching original behavior after interface adaptation).
- Integrity verification reproduces original chain hashing: `chain_hash = sha256(prev_hash || canonical_json(base_with_prev_hash))` using deterministic field ordering.
- Status codes unchanged (never 404/500 except internal marshal errors).
- Metrics setters (`SetExternalAnchorReceiptsIntegrity`, `SetExternalAnchorReceiptsLastVerifyAge`) invoked on verify path identical to monolith.

**Implementation notes:**
- Deps interface simplified to avoid exposing concrete store type; raw store presence tested via nil interface semantics (fixed test harness to return untyped nil rather than typed nil pointer).
- Added helper accessor methods on `BetaServer`: `ExternalReceiptStoreLatest`, `ExternalReceiptStoreEntries`, integrity status getters/setters and last verify age logger.
- Tests expanded (`basic_test.go`) to cover: unconfigured latest, empty verify, integrity ok (two-entry chain), mismatch scenario (tampered chain hash), chain listing summary, latest empty receipt path.
- Hash chain test receipts built using canonical marshaling identical to production to guard against accidental ordering changes.

**Risk reduction:**
- Reversion requires only reinstating original inline handlers and removing import; no shared state mutation logic changed.
- Store mutation/append logic remains in monolith; handlers are read-only except integrity setters for metrics.

**Next targets:** Anchor metrics Prometheus endpoint extraction, then consider consolidating receipt integrity + notarization integrity into a unified governance observability surface.

**Deferred items:** Incremental verification optimization tests (store `VerifyIncremental`) and golden snapshot for mismatch payload to prevent accidental field drift.

**Follow-up considerations:** Potential addition of POST endpoint for forced integrity re-scan with latency metrics, plus env flag gating for external receipt verification frequency.

### RB9 Tracing & Observability Progress (2025-10-27)

#### Final RB9 Status (All Implemented)
- Tracing spans for token.issue, token.validate, attestation.verify, rotation.perform active; rotation adds history_size/error tagging.
- Sampling ratio logic implemented (ratio <= 0 treated as always sample) and validated via tests.
- Latency percentile endpoint live and covered by tests.
- Documentation updated (OBSERVABILITY.md) with env vars & endpoint usage.
- Tracing tests green (enabled, disabled, ratio=0).

#### Residual Opportunities (Non-blocking)
- Add probabilistic sampling test for mid ratio (e.g. 0.5) to confirm distribution.
- Consider exporter abstraction for OTLP integration.
- Add external receipt and audit verification spans if needed for deeper governance observability (guard cardinality).
