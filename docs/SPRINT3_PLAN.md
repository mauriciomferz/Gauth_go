---
title: Sprint3 Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Sprint 3 Plan (RB5, RB10, RB11)

Scope: Ledger integrity hardening (signed rotation entries), revocation consistency proofs, and operational WAL observability.
Sprint Start: 2025-10-27
Target End: 2025-11-07 (approx. 10 working days)

## Goals Overview
| ID  | Goal                              | Outcome                                                        | Risk Mitigation                                      |
|-----|-----------------------------------|----------------------------------------------------------------|------------------------------------------------------|
| RB5 | Ledger Entry Signatures           | Each ledger append signed; tamper detection tool & tests       | Start with append-only interface wrapper; golden test |
| RB10| Revocation Consistency Proofs V2  | Efficient Merkle/subtree progression proofs + mismatch rejection| Incremental Merkle builder; cache subtree roots       |
| RB11| Replay WAL Metrics & Alerts       | WAL operational health metrics + alert thresholds              | Keep flush path isolated; use histogram+gauge combo   |

## RB5: Ledger Entry Signatures
### Objectives
- Sign every rotation ledger append: payload = `prev_hash || new_hash || timestamp || entry_index`.
- Provide verification CLI: `gauth-ledger-verify <path>` returns chain validity + first tamper index.
- Integrate signing with existing Ed25519 key manager (reuse active signing key).

### Tasks
- Add `internal/ledger/signing.go` (SignLedgerEntry, VerifyLedgerChain).
- Extend ledger struct to store `signature_b64` and `kid` (backward compat: treat missing signature as legacy unsigned entry).
- CLI tool under `cmd/ledger-verify/main.go`.
- Tests:
  - Happy chain verify.
  - Tamper single entry (`new_hash` mutated) -> detection.
  - Missing signature (legacy entries) accepted when `--allow-unsigned` flag set.
- Add golden JSON output for CLI verification summary.

### Edge Cases
- Key rotation mid-ledger: signature `kid` differs; verification must pick correct public key.
- Empty ledger path ⇒ exit code 2 with message `ledger_empty`.
- Unrecognized signature algorithm ⇒ `unsupported_alg` error.

### Acceptance Criteria
- `go test ./internal/ledger` >90% coverage of signing functions.
- CLI verifies tamper on first modified entry.
- Documentation section added to `LEDGER.md`.

## RB10: Revocation Consistency Proofs Phase 2
### Objectives
- Provide proof API for revocation state: `/api/v1/revocation/proof?ids=<comma>` returns inclusion + cumulative subtree hash for those IDs.
- Implement Merkle progression: maintain rolling root after each revocation event.
- Detect mismatches (client recomputed root != server root) with explicit error taxonomy (`revocation_root_mismatch`).

### Tasks
- Add `internal/revocation/merkle.go` (structure: nodes, Append(id), Root(), Proof(id)).
- Persist minimal subtree checkpoints every N revocations (tunable via env `GAUTH_REVOCATION_PROOF_CHECKPOINT_INTERVAL`).
- HTTP handler module `web/handlers/revocation/proofs.go`.
- Tests:
  - Single revocation inclusion proof.
  - Multiple revocations ordering stability.
  - Mismatch scenario (client intentionally recompute with altered leaf).
- Benchmark: `BenchmarkRevocationProofAppend` (target throughput >50k ops/s local).

### Edge Cases
- Duplicate revocation IDs (idempotent) should not alter root.
- Large number of revocations (simulate 10k) memory growth bounded; subtree caching leveraged.

### Acceptance Criteria
- Proof API returns stable JSON: `{success:true, root:"sha256:..", proofs:[{id:"...", hash:"sha256:..", path:["sha256:.."]}]}`.
- Mismatch test triggers HTTP 409 with code `revocation_root_mismatch`.
- Benchmark throughput logged to `build/badges/revocation_proof_throughput.txt`.

## RB11: Replay WAL Metrics & Alerts
### Objectives
- Expose metrics: `gauth_wal_pending_entries`, `gauth_wal_flush_latency_seconds` (histogram), `gauth_wal_snapshot_duration_seconds` (histogram), `gauth_wal_last_flush_age_seconds` (gauge).
- Alert rules: flush latency p95 > 0.250s, snapshot duration p95 > 1s, pending entries > 500.
- Document operational runbook.

### Tasks
- Instrument WAL write/flush code (in existing replay store implementation) adding metrics via Prometheus.
- Add file `monitoring/alerts/wal_rules.yml` with defined thresholds.
- Update `OBSERVABILITY.md` WAL metrics section.
- Tests: unit test metrics emission (trigger writes, flush, snapshot) asserting non-zero observations.
- Add optional debug endpoint `/api/v1/wal/status` returning current counters.

### Edge Cases
- WAL flush failure: increment `gauth_wal_flush_failures_total`; alert if >0 over 5m window.
- Snapshot interruption: record partial state metric `gauth_wal_snapshot_incomplete_total`.

### Acceptance Criteria
- Metrics visible on `/metrics` after simulated workload (test harness).
- Alert file passes basic Prometheus rule lint (manual step or script placeholder).
- Documentation includes troubleshooting (high latency, backlog growth).

## Cross-Cutting Concerns
- Backward compatibility: unsigned ledger entries accepted unless strict mode env `GAUTH_LEDGER_STRICT_SIGN`=1.
- Performance: revocation Merkle operations O(log n) for proofs; append amortized O(1) with rolling hash strategy.
- Security: Signatures verify before trusting ledger chain; proof API rejects unknown revocation IDs gracefully.

## Risks & Mitigation
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Large revocation sets degrade memory | Proof latency spikes | Checkpoint subtrees & limit per-node children |
| Signature overhead on high rotation frequency | Increased latency | Batch append signing if rotations > X/sec (future) |
| WAL instrumentation adds flush overhead | Reduced throughput | Use histograms with minimal label cardinality |

## Sequencing & Weekly Milestones
Week 1:
- Implement RB5 core signing + tests.
- Scaffold revocation Merkle structure & basic append.
Week 2:
- Finish RB10 proofs + API + mismatch tests.
- Add WAL metrics & alert rules (RB11).
- Bench & finalize docs.

## Metrics & SLO Targets
- Ledger verification CLI runtime: <150ms for 1k entries local.
- Revocation proof generation: p95 < 5ms for single ID, < 15ms for batch of 25.
- WAL average flush latency < 50ms; snapshot < 500ms.

## Deliverables Checklist
- CODE: new packages (`internal/ledger`, `internal/revocation`), handlers, metrics instrumentation.
- TEST: unit & golden tests, benchmarks.
- DOC: `LEDGER.md` update, `OBSERVABILITY.md` WAL section, new ADRs.
- OPS: alert rule file + runbook excerpt.

## Exit Criteria
- All acceptance criteria satisfied for RB5, RB10, RB11.
- Changelog updated with Sprint 3 summary.
- New ADRs merged & indexed.
- No regression in existing tracing/latency endpoints.

## Future (Post-Sprint Follow-Ups)
- Batch ledger signing optimization.
- Sparse Merkle integration for revocation set (memory + proof size reduction).
- Structured WAL compaction metrics.

---
Generated plan; adjust as scope evolves.
