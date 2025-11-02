# Sprint 2 Consolidated Changelog (RB6–RB9)

Date Range: 2025-10-?? to 2025-10-27
Status: Completed (All core goals implemented)

## RB6 – Signature Agility
- Added signer / verifier abstraction (`Signer`, `Verifier`) preparing multi-algorithm support.
- Implemented Ed25519 signer wrapper over existing key manager; preserved legacy registry.
- Non-breaking: existing issuance & verification paths continue to function; interface adoption deferred for some downstream sites.
- Tests validate parity of signatures (old vs new path).

## RB7 – Embedded Attestation Verifier
- Extracted verification logic into `pkg/attest` (`VerifyModelLimitsAttestation`).
- Added replay strategies (memory + durable) via adapter interfaces.
- Comprehensive test coverage: valid, soft invalid, nonce replay, notarization mismatch, unknown key id, invalid JSON/base64, tampered snapshot hash, combined hash correctness.
- Endpoint now delegates cleanly; error taxonomy unchanged.

## RB8 – Modular HTTP Handlers
- Migrated capability, anchor, audit, external receipt endpoints into discrete handler packages.
- Added accessor methods to `BetaServer` to supply dependencies cleanly (avoids field leakage).
- Preserved JSON schemas and error codes for all migrated routes.
- Implemented tests for negotiation, anchoring, audit verify/anchor, external receipt listing/latest/verify scenarios.
- Stabilized rotation summary metrics test via `/metrics` route and implicit memory anchor client initialization.

## RB9 – Observability Phase 1
- Introduced tracing spans: `token.issue`, `token.validate`, `attestation.verify`, `rotation.perform`.
- Environment gating: `GAUTH_TRACING_ENABLED` (primary) + legacy `GAUTH_OTEL_ENABLE`.
- Sampling ratio semantics: `GAUTH_TRACING_SAMPLE_RATIO` with current implementation (ratio<=0 => always sample). Documented in ADR.
- Added latency percentile endpoint `/api/v1/beta/metrics/latency` exposing p50/p95/p99 for key histograms.
- Fixed early error span closure path in attestation verify (body read failure).
- Added rotation span tagging (prev_kid, new_kid, ttl_hours, history_size, error).
- Tests: tracing enabled/disabled, ratio=0 deterministic sampling, latency endpoint shape.
- Documentation updated (`OBSERVABILITY.md`, Sprint plan and ADR).

## Cross-Cutting Improvements
- Enhanced error taxonomy consistency across extracted handlers.
- Added `Spans()` accessor on tracer provider for white-box testing.
- Percentile calculation logic implemented via Prometheus bucket scan (no external dependency).

## Backward Compatibility
- All endpoint paths and response shapes maintained.
- Legacy tracing flag retained; signature flows untouched.
- No migrations required for existing deployment configs beyond optional adoption of new tracing flag.

## Known Deferred / Follow-Ups (Non-blocking)
- Potential inversion of sampling semantics (make 0 => disable) via transitional mode flag.
- Mid-value sampling distribution test (e.g., ratio=0.5 multi-iteration).
- OTLP exporter integration & W3C propagation improvements.
- Golden snapshot for selected anchor/audit error payload (hardening).
- Additional latency histograms for token issue/validate if instrumentation expanded.

## Risk Notes
- Sampling semantics may confuse operators—explicit documentation mitigates.
- Handler dependency surface growth monitored; consider future consolidation interface.

## Verification Summary
- All newly added tests pass (`go test` subsets for web, crypto, attest packages).
- Rotation system integration test green post instrumentation.
- No observed regression in metrics emission after tracing introduction.

## References
- `SPRINT2_PLAN.md` (updated with final RB9 implementation details)
- `ADR-tracing-sampling-semantics.md`
- `web/tracing_basic_test.go`
- `OBSERVABILITY.md`

---
