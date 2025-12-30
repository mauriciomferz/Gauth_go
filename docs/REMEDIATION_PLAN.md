---
title: Remediation Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Remediation Plan (Toward Clean Beta)

> Updated: 2025-10-26
> Scope: Targeted tasks to reach "clean demo-ready" state; supersedes earlier long-range framework roadmap while retaining strategic elements below.

## Guiding Principles (Demo Horizon)
1. Integrity artifacts must be verifiable live (rotation, attestation, revocation proof, PoA digest).
2. Replay & signature failures demonstrably blocked (negative paths ready).
3. Minimal spec & auditor tooling enable external trust narrative.
4. Additive changes only; avoid refactors that risk stability days before demo.

## Phase Overview
| Phase | Narrow Demo-Focused Goal | Exit Criteria |
|-------|-------------------------|---------------|
| A Demo Hardening | Durable replay + PoA canonical digest | Restart replay test passes; digest stable tests green |
| B Transparency | Auditor CLI + PoA revocation anchoring | CLI verifies all artifacts; PoA revocation appears in Merkle proof |
| C Interop | OpenAPI + discovery config | `/api/openapi.json` + `/.well-known/agentauth/config` served |
| D Polish | Conditional interpreter + diagrams + risk register | Conditions enforced in issuance; diagrams finalized; risks documented |

## Detailed Epics
### Epic A: Durable Replay & Canonical PoA (Phase A)
- Implement full JWT (or PASETO) claim set: `iss, aud, sub, exp, nbf, iat, jti, scope, kid`.
- Introduce `kid` header; maintain key ring with active + previous + scheduled next.
- Key rotation daemon: time-based schedule + manual trigger; events recorded in audit ledger.
- Acceptance: Attestation test issues token, rotates key, validates old token with previous key; fails with removed key; jti uniqueness stored.

### Epic B: Minimal Multi-Sig PoA (Phase A)
- Embed canonical PoA JSON inside token envelope or sidecar structure; include digest & signature set.
- Support M-of-N signatures (Ed25519). Aggregation record lists signers, threshold, verification status.
- Provide deterministic canonicalization doc & property/fuzz tests.
- Acceptance: Issuance fails if threshold not met; tampering with embedded PoA invalidates verification test.

### Epic C: PoA Digest Tests (Phase A)
- Fuzz tests: random field order permutations, benign whitespace changes, ensure digest stability.
- Negative tests: altering semantic fields changes digest.

### (Deferred) PDP Governance & Versioning
- Add policy version field, timestamp, author; immutable append log.
- Rollback API: mark head pointer to previous version; provenance ties decision to version hash.
- Acceptance: Policy chain verification test detects tampering; rollback updates evaluation head without data loss.

### (Deferred) Obligations & Advice Execution
- Define `ObligationService` interface; execution pipeline after decision combining.
- Obligation results recorded in audit log (hash-chained).

### Epic F: Durable Replay Store (Merged into Epic A)
- Persistent store (Redis + WAL fallback) storing used JTIs until expiry.
- Config flag `AGENTAUTH_REPLAY_FAIL_CLOSED` to deny decisions if store unreachable.

### (Deferred) Jurisdiction & Legal Validator
- `LegalFrameworkValidator` with rule set (JSON DSL) mapping jurisdiction codes to constraints (e.g. max amounts, restricted actions).
- Acceptance: Decision evaluation fails fast with jurisdiction rule violation; test suite covers each rule variant.

### Epic H: OpenAPI Spec & Discovery (Phase C)
- Generate OpenAPI from handlers; publish at `/openapi.json`.
- Add `/.well-known/agentauth-configuration` returning algorithms, endpoints, supported PoA features.

### (Deferred) Rights & Obligations Engine
- Parse & enforce rights/obligations inside evaluation (pre/post decision). Distinguish must vs advisory obligations.

### Epic J: Metrics Additions (Selective)
- Export labeled counters: `agentauth_decisions_total{action,resource,allow}`.
- OTel spans: issuance → delegation issuance → decision evaluation chain.

### Epic K: PoA & Rotation Anchor Scheduler (Phase B/C)
- Periodic anchoring: hash tips submitted to external timestamp (RFC3161 or transparency log API).
- Verification command replays anchor proofs.

### (Deferred) SBOM + Signing
- CI step: build -> syft SBOM -> cosign sign -> attest SLSA provenance.

### (Deferred) Persistent Policy Store & Decision Cache
- Durable store (Postgres) with migrations, indices (subject, resource, action).
- Cache with invalidation on append/rollback events.

### (Deferred) Distributed Revocation & Ledger Sync
- Gossip or pub/sub (e.g., NATS) broadcasting revocation events + ledger increments.
- Convergence test ensures all nodes have same chain head within SLA.

## Cross-Cutting (Demo Scope)
| Task | Description |
|------|-------------|
| Spec Clause Mapping | Create `docs/RFC_MAP.md` linking each RFC clause → code symbol → test name.
| Security Posture Upgrade | Integrate `govulncheck`, `gosec`, container scan; fail CI on critical issues.
| Threat/Risk Register | `docs/RISK_REGISTER.md` enumerating mitigations & residual risk with owners.
| Feature Flags | `internal/flags` package controlling beta vs hardened modes.
| ADRs | Each epic yields an ADR documenting rationale & alternatives.

## Acceptance Metrics
| Metric | Target |
|--------|--------|
| Attestation verify latency | p95 < 40ms local |
| Rotation signature verify latency | p95 < 25ms local |
| Replay attempt after restart | 0 passes (all blocked) |
| Auditor CLI end-to-end runtime | < 2s |
| PoA digest invariance tests | 100% pass |

## Risk Prioritization
High: Missing replay protection fail-closed, multi-signature enforcement, public verifiable tokens.  
Medium: Jurisdiction validator, obligations execution, ledger anchoring.  
Low: Well-known discovery, metrics labels, advisory rights engine.

## Compressed Timeline (5-Day Sprint)
Day 1: Durable replay store + PoA canonical digest + negative tests
Day 2: Multi-sig PoA issuance (count-based) + PoA revocation anchoring
Day 3: Auditor CLI + OpenAPI + discovery config
Day 4: Conditional interpreter MVP + metrics additions + diagrams finalize
Day 5: Risk register + polish + demo dry-run script

## Test Strategy (Immediate)
- Add property tests for PoA field permutations.  
- Add concurrency tests (simulated multi-writer) for persistent policy store.  
- Benchmark harness for delegation issuance & revocation verification under load.

## Tooling (Immediate vs Deferred)
- Adopt `golang-jwt/jwt/v5` or `paseto` library (remove manual parser).  
- Use `OpenTelemetry` SDK for spans & metrics pipeline.  
- Add `go-fuzz` / `github.com/dvyukov/go-fuzz` integration (or `oss-fuzz` harness) for canonicalization & expression parsing.

## Deferred (Post-Demo)
- Formal verification / model checking.  
- Homomorphic or zero-knowledge proofs of delegation.  
- Multi-tenant isolation & namespace boundaries (Phase 5+).  

---
Maintain this plan: update status per epic in PR descriptions; regenerate summary table monthly.
