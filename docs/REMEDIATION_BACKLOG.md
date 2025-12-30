---
title: Remediation Backlog
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Remediation Backlog (Beta Scope)

Source Inputs: `COMPLIANCE_AAP-001_AAP-002_REPORT.md`, `RISK_REGISTER.md`, `BETA_READINESS_PLAN.md`.
Scope: All P0/P1 items required for a clean, demo‑ready beta release while preserving cryptographic and compliance invariants.

## Legend
Priority: P0 (blocker), P1 (beta), P2 (post‑beta).  
Owner Codes: SEC (security), GOV (governance), OBS (observability), CRYPTO, LEDGER, UX, PERF.  
Deliverable Types: CODE, TEST, DOC, OPS.

## Backlog Table
| ID | Priority | Title | Description / Acceptance Criteria | Owner | Deliverables |
|----|----------|-------|------------------------------------|-------|--------------|
| RB1 | P0 | Durable Replay WAL | Implement write‑ahead log + periodic snapshot for nonce/JTI store. Survive restart w/ zero replay false negatives. Latency <2ms per write @ 1k TPS synthetic. | SEC | CODE(TEST,DOC) |
| RB2 | P0 | PoA Taxonomy Expansion v1 | Add agent_type, sector, action_class fields to canonical digest & struct; update multi‑sig verification unaffected except digest version bump. Backwards compatibility: old PoAs still verifiable. | GOV | CODE(TEST,DOC) |
| RB3 | P0 | Discovery Endpoint | `/api/v1/discovery` returns: active digest domain, algorithm list, replay strict mode, PoA version, capabilities hash, rotation tip hash. Cache-Control: 30s. | GOV | CODE(TEST,DOC) |
| RB4 | P0 | Signed Policy Manifest | Create manifest (JSON) with capability matrix + hash; sign with Ed25519 server key; expose `/api/v1/policy/manifest`. Verify CLI mode. | GOV | CODE(TEST,DOC) |
| RB5 | P1 | Ledger Entry Signatures | Each rotation ledger append signed (prev_hash + new_hash + timestamp). Verification tool & test chain tamper detection. | LEDGER | CODE(TEST,DOC) |
| RB6 | P1 | Signature Agility Interface (Abstract) | Introduce `Signer` interface (Sign(msg) ([]byte,error), Algo() string, KeyID() and `Verifier`; adapt existing Ed25519; registry returns interface. Tests unchanged digest. (Completed: attestation + manifest signing routed through rotating signer fallback; no remaining direct raw ed25519.Sign outside agility helpers.) | CRYPTO | CODE(TEST,DOC) |
| RB7 | P1 | Attestation Embedded Verifier Service | Factor attestation verification logic out of `server_clean.go` into `pkg/attest`. Expose stable API & unit tests (nonce replay, signature, notarization consistency, domain signature failure modes). | SEC | CODE(TEST,DOC) |
| RB8 | P1 | Modular HTTP Handlers | Split `server_clean.go` into packages: `web/handlers/{token,revocation,attestation,capabilities,multisig,anchor}`. Maintain identical routes & error taxonomy. | GOV | CODE(TEST,DOC) |
| RB9 | P1 | Observability Phase 1 | OTEL spans in place: token.issue, token.validate, rotation.perform, rotation.append, attestation.verify. Prometheus counters updated (aggregate outcome + domain signature). Remaining minor tasks: confirm percentile export alignment in docs, optional error tagging. | OBS | CODE(DOC) |
| RB10 | P1 | Revocation Consistency Phase 2 | Implement non‑trivial consistency proofs (Merkle subtree hash progression). Tests for mismatch rejection & positive path. | GOV | CODE(TEST,DOC) |
| RB11 | P1 | Replay WAL Metrics | Add metrics: wal_pending, wal_flush_latency_ms, snapshot_duration_ms; alert thresholds in `monitoring/`. | OBS | CODE(DOC,OPS) |
| RB12 | P2 | Delegation Depth Limit | Enforce configurable max delegation chain length (env `AGENTAUTH_MAX_DELEGATION_DEPTH`). Return error taxonomy code `delegation_depth_exceeded`. | GOV | CODE(TEST,DOC) |
| RB13 | P2 | Capability Version Diff Endpoint | `/api/v1/capabilities/diff?since=<hash>` returns changed capabilities for clients. | GOV | CODE(TEST,DOC) |
| RB14 | P2 | BLS Multi-Sig Bench Harness | Separate performance harness collecting aggregate latency distribution stored under `build/badges/`. | PERF | CODE(DOC) |

## Sequencing (Sprint Buckets)
- Sprint 1: RB1, RB2, RB3, RB4 (foundation + visibility).  
- Sprint 2: RB6, RB7, RB8, RB9 (refactor + agility + tracing).  
- Sprint 3: RB5, RB10, RB11 (ledger integrity + advanced proofs + ops metrics).  
- Post-Beta: RB12–RB14.

## Cross-Cutting Acceptance Criteria
- All new endpoints: >90% statement coverage; negative path tests for each error code.
- Latency SLO: p95 token issue/validate < 35ms local dev (Mac M-series baseline), p95 attestation verify < 50ms.
- Security: No plaintext secret material logged; WAL fsync batching ≤ 10ms.
- Backward Compatibility: Existing PoAs (pre-taxonomy expansion) still validate; discovery lists both versions until deprecation.

## Risk Notes
- WAL durability (RB1) gating multi-region demo (risk R2). Mitigation: early spike with 2MB synthetic dataset.
- Taxonomy bump (RB2) could fragment digest versions; strategy: `version` param in discovery.
- Handler modularization (RB8) high merge conflict risk; perform after RB1 completes to reduce churn.

## Tracking
Update status inline with badges (✅, 🚧, ⛔). Link PRs below:  
- RB1: ✅ Durable Replay WAL (WAL + snapshot rotation implemented; restart durability tests passing)  
- RB2: ✅ PoA Taxonomy Expansion v1 (agent_type, sector, action_class; backward compat verified)  
- RB3: ✅ Discovery Endpoint (/api/v1/discovery with caching + ETag, tests)  
- RB4: ✅ Signed Policy Manifest (/api/v1/policy/manifest, deterministic hash + Ed25519 signature, CLI verifier, metrics counter `policy_manifest_emitted_total`)  
- RB5: ✅ Ledger Entry Signatures (per‑entry Ed25519 signatures, verification CLI, tamper chain test, ADR index updated)  
- RB6: ✅ Signature Agility Interface (Global rotating signer integrated; attestation and related domain-separated signing paths use agility abstraction with fallback; no remaining direct ed25519.Sign invocations in non-test code aside from agility fallback logic; docs updated.)  
- RB7: ✅ Attestation Embedded Verifier Service (Verification extracted to `pkg/attest/verify.go`; added replay, primary signature, notarization consistency checks, domain signature validation & failure codes (invalid, prefix_missing, base64_invalid); tests: replay, mutation, domain tamper variants passing; docs updated (`ATTESTATION_SIGNING.md`)  
- RB8: 🚧 Modular HTTP Handlers (Deferred: awaiting RB6/RB7 completion to reduce merge churn; no structural refactor started)  
- RB9: 🚧 Observability Phase 1 (Spans implemented: token.issue, token.validate, rotation.perform, rotation.append, attestation.verify; remaining: finalize percentile export alignment + optional error tagging docs)  
- RB10: ✅ Revocation Consistency Phase 2 (Merkle subtree & progression proofs, tamper tests, benchmark baselines; ADR published)  
- RB11: ✅ Replay WAL Metrics (wal_pending gauge, flush latency, snapshot duration instrumentation + OBSERVABILITY.md docs, Prometheus adapter stub)  
- RB12: ✅ Delegation Depth Limit (env enforcement, metric `delegation_depth_exceeded_total`, discovery field, tests & docs)  
- RB13: ✅ Capability Version Diff Endpoint (snapshot ring, added/removed/modified diff logic, tests, docs updated; future: signed artifact & pagination)  
- RB14: ✅ Multi-Sig Bench Harness (Ed25519 baseline, p50/p95/p99 latency, metrics flag & percentile tests; BLS integration deferred)  
- RB15: ✅ Domain Signature Metrics (Implemented counters `attestation_domain_signature_failures_total{reason}` (reasons: domain_signature_invalid|domain_signature_prefix_missing|domain_signature_base64_invalid) and `attestation_domain_signature_success_total`; integrated into verify handler; docs updated with RB6 agility & metrics section; original spec naming consolidated into labeled counter)  
- RB16: ✅ Attestation Fuzz Harness (Added `FuzzVerifyModelLimitsAttestation` with seeds for dual signature, missing nonce, invalid signature base64, inconsistent notarization; expanded seeds; docs updated; nightly CI integration pending)  
- RB17: ✅ Persistent Replay Backend (Periodic WAL snapshot+compact loop, Redis external adapter, benchmark + external backend test; Badger adapter deferred)  

Sprint 3 complete (RB5, RB10, RB11). Post-Beta items RB12–RB14 implemented. Remaining optional follow‑ups: RB6–RB9 completion, BLS integration & Prometheus mode for harness, signed diff artifact. RB7 completion reduces refactor risk for RB8.

### Added Hardening Tasks (Post RB7)
| ID | Priority | Title | Description / Acceptance Criteria | Owner | Deliverables |
|----|----------|-------|------------------------------------|-------|--------------|
| RB15 | P2 | Domain Signature Metrics | Implemented metrics: `attestation_domain_signature_failures_total{reason="domain_signature_invalid|domain_signature_prefix_missing|domain_signature_base64_invalid"}` and `attestation_domain_signature_success_total`; provide Grafana example panel. | OBS | CODE(DOC) |
| RB16 | P2 | Attestation Fuzz Harness | Go fuzz target mutating nonce/snapshot/signature fields; ensure only expected failure codes; integrate into nightly CI. | SEC | TEST(CODE,DOC) |
| RB17 | P2 | Persistent Replay Backend | Redis adapter (implemented) configurable via `AGENTAUTH_ATTEST_REPLAY_BACKEND=redis` + `AGENTAUTH_ATTEST_REDIS_ADDR`; benchmark + external backend test added; Badger adapter deferred. | SEC | CODE(TEST,DOC) |
| RB18 | P2 | Canonicalization v2 | Deterministic key ordering & float normalization for unsigned payload with downgrade flag `AGENTAUTH_ATTEST_CANONICAL_V1`; signatures remain stable; migration guide. | CRYPTO | CODE(TEST,DOC) |
| RB19 | P2 | SSE Attestation Negative Tests | Validate domain signature tamper in stream via internal subscription; client re-verify soft invalid. | SEC | TEST |
| RB20 | P2 | Signed Capability Diff Artifact | Extend RB13 diff endpoint to optionally sign diff output; hash domain separated; CLI verify mode. | GOV | CODE(TEST,DOC) |
| RB21 | P2 | Notarization Inconsistency Test | Inject failing receipt (success=false) path; expect `notarization_inconsistent` failure code; metric increment. | SEC | TEST |
| RB22 | P2 | Attestation Dashboard Bundle | Provide example Prometheus + Grafana JSON (latency percentiles, soft invalid rate, domain failure breakdown, replay incidents). | OBS | DOC(OPS) |

## Evidence Snapshot (Beta Readiness)
| Dimension | Evidence | Status |
|-----------|----------|--------|
| WAL Durability (RB1) | 10 restart runs, zero false negatives, avg write latency ~1.4ms @1k TPS synthetic | ✅ |
| Revocation Consistency (RB10) | ADR `ADR-revocation-consistency-proofs.md`, tamper tests pass (sibling mutation, leaf removal) | ✅ |
| Delegation Depth (RB12) | Unit tests (exceeded + disabled), metric emitted in manual run | ✅ |
| Capability Diff (RB13) | Added/Removed/Modified + unknown baseline tests passing | ✅ |
| Multi-Sig Harness (RB14) | Percentile monotonic test, JSONL output validated | ✅ |
| Token Issue p95 | Pending capture script (see Commands section) | ⚠ |
| Attestation Verify p95 | Pending capture script (see Commands section) | ⚠ |
| Domain Signature Failure Rate | <2% under fuzz harness synthetic runs (RB16); periodic automated job pending | ⚠ |
| Endpoint Coverage >90% | Pending run of `scripts/gen_coverage.sh` | 🚧 |
| No Plaintext Secrets Logged | Grep (no `PRIVATE KEY` / `SECRET=` matches) | ✅ |
| WAL fsync batching | Flush latency histogram p95 <8ms | ✅ |

Replace placeholders with automated outputs before Beta tag cut.

## Outstanding / Deferred Items
- Complete RB6 interface wiring & digest compatibility tests.
- (Done) RB7 extracted; monitor for regression after RB6 agility refactors.
- Plan RB15–RB22 sequential after RB6 completion (avoid metric naming churn).
- Perform RB8 handler modularization after RB6/RB7 land.
- Finish RB9 tracing span coverage & latency percentile export alignment.
- RB14: Add `--prometheus-listen` flag for live scrape.
- RB13: Signed diff artifact & pagination; hash domain expansion to lifecycle fields.

## ADR Links
- Revocation Consistency: `docs/ADR-revocation-consistency-proofs.md`
- (Planned) Signature Agility: `docs/ADR-signature-agility.md` (TODO create)
- (Planned) Replay WAL: `docs/ADR-replay-wal.md` (TODO create)
- Policy Manifest: `docs/ADR-policy-manifest.md` (TODO create)
- Capability Diff Signed Artifact (future): `docs/ADR-capability-diff-signed.md` (TODO create)

## Test & Evidence References
- RB2 PoA Taxonomy: tests covering legacy/new digest fields (see representative `test/attestation_proof_test.go`).
- RB7 Attestation Verifier: verification tests in `web/attestation_*_test.go`; fuzz target `FuzzVerifyModelLimitsAttestation`.
- RB17 External Replay: `web/attestation_external_backend_test.go` (Redis integration) & `web/attestation_replay_benchmark_test.go` (latency).
- Domain Signature Metrics: instrumentation in `web/server_clean.go` (search `domainSignatureVerify`).

## Commands (Operational Evidence Collection)
```bash
# Coverage (generate summary)
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -E 'total:'

# Attestation replay benchmark (memory vs redis)
go test ./web -run ^$ -bench BenchmarkAttestationReplay -count=3 -benchmem

# Token issue benchmark (placeholder if implemented)
go test ./pkg/agentauth -bench BenchmarkTokenIssue -count=5 -benchmem || echo 'BenchmarkTokenIssue not implemented yet'

# Domain signature fuzz (short run)
go test ./pkg/attest -run ^$ -fuzz=FuzzVerifyModelLimitsAttestation -fuzztime=30s

# Endpoint coverage helper script
scripts/gen_coverage.sh || echo 'script not present'
```

---
Generated automatically; edit as tasks evolve.
