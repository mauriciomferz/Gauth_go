# GAP Closure Plan
Generated: 2025-10-22

This plan defines explicit closure criteria for each P0 gap (fast-track) plus structured artifacts, test additions, and KPIs. Subsequent sections outline preliminary sequencing for a two-week sprint once finalized.

## Legend
- ID: Original gap matrix identifier
- Closure Criteria: Testable end-state definition
- Artifacts: Files / docs / endpoints to produce or modify
- Tests: New tests (unit/property/fuzz/integration) required
- Metrics: Observability additions
- Dependencies: Ordering or prerequisite items

---
## P0 Gaps Detailed

### sec1.item1 – Algorithm Expansion (Ed25519 -> +ECDSA P-256, optional BLS phase 2 later)
Closure Criteria:
- Config variable `GAUTH_TOKEN_SIGNATURE_ALGS` allows comma list (e.g. `ed25519,ecdsa-p256`).
- Issuance selects default alg; verify endpoint accepts any configured alg.
- Key rotation persists key type; JWKS exposes `alg` and `crv` for ECDSA.
- Detached signature envelope supports algorithm field enumeration; multi-sig placeholder reserved.
Artifacts:
- `internal/crypto/ecdsa.go` (key gen, sign, verify)
- `internal/crypto/registry.go` update for multi-alg
- OpenAPI spec extension for `alg` listing
Tests:
- Unit: sign/verify roundtrip for Ed25519 & ECDSA.
- Negative: tampered payload, wrong alg, mismatched curve.
- Property: 1000 randomized messages produce distinct signatures; verification never false-positive.
Metrics:
- Counter `token_signature_verifications_total{alg, result}`.
Dependencies: canonical digest (done), secret storage provider (sec8.item1) optional for HSM integration.

### sec1.item2 – Claims Completeness & Semantics
Closure Criteria:
- Mandatory base claims: sub, iss, aud (array), exp, iat, nbf, jti, scope, typ.
- `typ` enforced (e.g., `gauth.token.v1`) with rejection on mismatch.
- Structured footer metadata field (JSON object) with canonical inclusion/exclusion rules.
- Aud multiple entries validated; failure if empty.
Artifacts:
- `pkg/gauth/claims.go` (refactor)
- Docs: `docs/TOKEN_CLAIMS.md` with claim semantics & examples.
Tests:
- Unit: creation/parse roundtrip.
- Property: fuzz parse maintains safety (no panic) for random valid/invalid claim sets.
- Negative: missing mandatory claims → issuance fail; expired tokens denied.
Metrics:
- Counter `token_claims_validation_fail_total{reason}`.
Dependencies: robust parsing (sec1.item3).

### sec1.item3 – Robust JSON Parser Replacement
Closure Criteria:
- Replace manual string scan with streaming decoder enforcing max depth & field whitelist.
- Reject duplicate keys; reject control chars outside allowed sets.
- Performance: ≤10% regression vs current for 10K tokens parse benchmark.
Artifacts:
- `pkg/gauth/parser.go` new implementation.
- Benchmark: `bench_test/token_parse_bench_test.go`.
Tests:
- Fuzz: 1M iterations no panic.
- Property: duplicate key rejection invariant.
- Unit: reserved claim absent → error path.
Metrics:
- Histogram `token_parse_duration_seconds`.
Dependencies: none; feeds into claims completeness.

### sec1.item5 – Multi-Alg Detached Signatures & Enforcement Flag
Closure Criteria:
- Envelope supports `signatures` array: each entry `{alg,kid,sig}` (initial single entry).
- Enforcement flag `require_multi_alg` triggers issuance with at least two algs when configured.
- Verification endpoint validates all signatures if multi present; fails on any mismatch.
Artifacts:
- `pkg/rfc0111/envelope_v2.go` modifications.
- `docs/TOKEN_SIGNATURES.md`.
Tests:
- Unit: single-alg, multi-alg issuance + verify.
- Negative: remove one signature when flag requires multi-alg → verification fail.
- Property: multi signatures over same digest identical payload cross-alg.
Metrics:
- Gauge `token_signature_multi_alg_enabled`.
Dependencies: sec1.item1 algorithm expansion.

### sec2.item1 – Richer Conflict Diagnostics (PDP)
Closure Criteria:
- Decision response includes `evaluation_path` listing policy IDs considered and denial reasons.
- Conflict taxonomy: `overrides`, `indeterminate`, `deny_overrides` surfaced.
Artifacts:
- `pkg/authz/pdp.go` enhancement.
- Docs update `AUTHORIZATION_IMPLEMENTATION.md`.
Tests:
- Unit: scenario matrix for combining algs.
- Integration: multi-policy request returns structured diagnostics.
Metrics:
- Counter `authz_conflicts_total{type}`.
Dependencies: none.

### sec2.item2 – Extensible ABAC Function Registry
Closure Criteria:
- Registry interface enabling dynamic function registration at init.
- Built-in functions documented; custom added via config file.
- Safe sandbox: limits recursion depth & execution time.
Artifacts:
- `pkg/authz/functions/registry.go`.
- Config: `config/abac_functions.json`.
Tests:
- Unit: register & invoke custom function.
- Negative: forbidden overwrite of core function.
Metrics:
- Counter `abac_function_invocations_total{name,result}`.
Dependencies: none.

### sec3.item1 – PoA Semantic Validation Expansion
Closure Criteria:
- Validator checks: issuer alignment, delegation chain integrity, scope subset, temporal overlap.
- Returns structured list of violations.
Artifacts:
- `pkg/rfc0111/poa_validator.go` expanded.
- Docs: `docs/POA_VALIDATION.md`.
Tests:
- Unit: chain with mismatch scope → violation list includes `scope_superset_violation`.
- Property: random chains preserve invariants (no false negative with intentionally broken chain).
Metrics:
- Counter `poa_validation_fail_total{type}`.
Dependencies: multi-alg signature groundwork optional for future aggregated signatures.

### sec5.item1 – Signed & Anchored Immutable Audit Ledger
Closure Criteria:
- Ledger entries hashed + signed (Ed25519/ECDSA) with `prev_hash` linking.
- External anchor: periodic (≤60s) submission to notarization (pluggable interface) producing anchor receipt.
- Verification endpoint reconstructs chain and validates signatures & anchor timestamps.
Artifacts:
- `internal/ledger/audit.go`, `internal/ledger/verify.go`.
- `web/api_v1_audit_verify.go` endpoint.
- Docs: `docs/AUDIT_LEDGER.md` (format, verification steps).
Tests:
- Unit: append sequence preserves hash chain.
- Integration: simulate anchor provider stub returning receipt.
- Property: tampering detection test (modify middle entry → verify fails).
Metrics:
- Counter `audit_ledger_append_total`.
- Gauge `audit_anchor_interval_seconds`.
Dependencies: secret storage provider for key custody (sec8.item1).

### sec8.item1 – Secure Secret Storage Provider
Closure Criteria:
- Abstraction interface with implementations: `filesystem`, `vault`.
- Secrets at rest encrypted (filesystem mode: libsodium secretbox or AES-GCM). Vault uses transit engine.
- Rotation operation updates encryption key with re-wrap.
Artifacts:
- `internal/secrets/provider.go`, `internal/secrets/filesystem.go`, `internal/secrets/vault.go`.
- Config doc: `docs/SECRETS.md`.
Tests:
- Unit: store/get/delete lifecycle.
- Integration (if Vault unavailable in CI -> use mock) verifying API contract.
Metrics:
- Counter `secret_operations_total{type,result}`.
Dependencies: none; feeds audit ledger signing & key rotation.

### sec9.item1 – Comprehensive Clause-to-Test Mapping
Closure Criteria:
- `conformance/clause_map.json` covers 100% declared RFC clauses (list enumerated in `docs/ARCHITECTURE.md` or RFC reference).
- Each clause maps to ≥1 test file path.
- CI job fails if any clause untested.
Artifacts:
- Updated `clause_map.json`.
- Validation script `scripts/verify_clause_map.go`.
Tests:
- Unit: script detects missing mapping (simulate removal).
- Conformance harness updated to print coverage %.
Metrics:
- Gauge `conformance_clause_coverage_percent`.
Dependencies: none.

### sec11.item2 – Governance Evidence Externalization (Model Limits)
Closure Criteria:
- External notarization interface invoked for every audit anchor (model limits events) producing receipt stored & stream-emitted.
- SSE stream includes `reason="anchor_notarized"` with receipt hash.
- Public receipt verification endpoint.
Artifacts:
- `internal/notary/interface.go`, `internal/notary/mock.go`.
- `web/api_v1_model_limits_receipts.go`.
- Docs update `MODEL_LIMITS.md` (receipt verification).
Tests:
- Unit: notary mock returns deterministic receipt.
- Integration: stream test for `anchor_notarized` reason.
Metrics:
- Counter `model_limits_notarization_total{result}`.
Dependencies: audit ledger anchoring (sec5.item1).

---
## Cross-Cutting KPIs
- P0 gaps closed count / total (target 100%).
- Mean token parse latency pre/post parser switch (<10% increase).
- Audit ledger verification time for N=10K entries (<2s local).
- Fuzz iteration count across parser & signatures with zero crashes.
- Conformance clause coverage percent (100%).

## Non-P0 Early Consideration (Risk Mitigation)
Items with strong dependency enabling: begin scaffolding for sec1.item3 property tests early; stub notary provider while audit ledger built.

## Preliminary Sequencing (Draft)
Week 1 (Foundations): sec8.item1, sec1.item3, sec1.item2, sec1.item1, sec1.item5.
Parallelizable: sec2.item1 + sec2.item2; sec9.item1 mapping.
Week 1 End Milestone: Multi-alg signatures operational; parser & claims refactor merged; secret provider integrated; initial conformance coverage script pass.
Week 2 (Ledger & Governance): sec5.item1, sec3.item1, sec11.item2 + stream reason, finalize diagnostics & ABAC registry docs.
Week 2 Stretch: Begin P1 embedding PoA (sec3.item2) & durable replay store scaffolding (sec6.item1).

## Review & Sign-Off Process
- Daily status: metrics snapshot, test coverage diff.
- Acceptance checklists stored in `docs/GAP_ACCEPTANCE_CHECKLIST.md` (to be generated).

## Next Steps
1. Implement sequencing file (detailed daily breakdown).
2. Map ownership for each artifact.
3. Start scaffolding secret storage provider & parser replacement.

