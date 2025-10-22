# GAuth RFC0111 & RFC0115 Remediation Plan (Beta → Framework Conformant)

> Date: 2025-10-17
> Scope: Elevate current beta demo to a standards-aligned implementation of GAuth Framework per RFC0111 (Authorization) & RFC0115 (PoA Credential Definition).

## Guiding Principles
1. **Integrity First**: Strengthen cryptographic provenance (tokens, PoA, ledger, revocations) before adding surface area.
2. **Incremental Hardening**: Add features behind explicit opt-in flags; preserve educational clarity while enabling production pathways.
3. **Traceable Conformance**: Every RFC clause mapped to code + test IDs.
4. **Non-Destructive Evolution**: Introduce interfaces (`SignerService`, `PoAStore`, `PolicyVersionStore`) to avoid breaking existing demos.

## Phase Overview
| Phase | Goal | Key Epics | Exit Criteria |
|-------|------|-----------|---------------|
| 0 Foundation | Secure token & PoA core | Token Lifecycle, Multi-Signature PoA, Canonical Digest Hardening | Tokens carry full claims + `kid`; PoA embedded & verified; digest fuzz tests pass |
| 1 Governance | Policy/versioning & obligations | PDP Governance, Obligations Execution, Replay Protection | Policy version rollback works; obligations logged; unicity (JTI) enforced fail-closed option |
| 2 Compliance & Interop | Jurisdiction & discovery | Jurisdiction Validator, OpenAPI & Well-Known, Rights & Obligations Engine | Discovery endpoint live; jurisdiction rules enforced in decisions; rights/obligations executed |
| 3 Observability & Supply Chain | Metrics/tracing & provenance | Metrics Expansion, OTel Spans, Ledger Anchoring, SBOM+Signing | Allow/deny metrics labeled; spans show token→delegation→decision; chain tips anchored externally |
| 4 Scalability & Persistence | Durable stores & distribution | Persistent Policy Store, Decision Cache, Distributed Revocation, Horizontal Ledger Sync | Multi-node PDP coherent with cache; revocation events propagate; persistent audit with external anchors |

## Detailed Epics
### Epic A: Robust Token Lifecycle (Phase 0)
- Implement full JWT (or PASETO) claim set: `iss, aud, sub, exp, nbf, iat, jti, scope, kid`.
- Introduce `kid` header; maintain key ring with active + previous + scheduled next.
- Key rotation daemon: time-based schedule + manual trigger; events recorded in audit ledger.
- Acceptance: Attestation test issues token, rotates key, validates old token with previous key; fails with removed key; jti uniqueness stored.

### Epic B: Multi-Signature PoA & Embedding (Phase 0)
- Embed canonical PoA JSON inside token envelope or sidecar structure; include digest & signature set.
- Support M-of-N signatures (Ed25519). Aggregation record lists signers, threshold, verification status.
- Provide deterministic canonicalization doc & property/fuzz tests.
- Acceptance: Issuance fails if threshold not met; tampering with embedded PoA invalidates verification test.

### Epic C: Canonical Digest Hardening (Phase 0)
- Fuzz tests: random field order permutations, benign whitespace changes, ensure digest stability.
- Negative tests: altering semantic fields changes digest.

### Epic D: PDP Governance & Versioning (Phase 1)
- Add policy version field, timestamp, author; immutable append log.
- Rollback API: mark head pointer to previous version; provenance ties decision to version hash.
- Acceptance: Policy chain verification test detects tampering; rollback updates evaluation head without data loss.

### Epic E: Obligations & Advice Execution (Phase 1)
- Define `ObligationService` interface; execution pipeline after decision combining.
- Obligation results recorded in audit log (hash-chained).

### Epic F: Replay Protection & JTI Store (Phase 1)
- Persistent store (Redis + WAL fallback) storing used JTIs until expiry.
- Config flag `GAUTH_REPLAY_FAIL_CLOSED` to deny decisions if store unreachable.

### Epic G: Jurisdiction & Legal Validator (Phase 2)
- `LegalFrameworkValidator` with rule set (JSON DSL) mapping jurisdiction codes to constraints (e.g. max amounts, restricted actions).
- Acceptance: Decision evaluation fails fast with jurisdiction rule violation; test suite covers each rule variant.

### Epic H: OpenAPI Spec & Well-Known (Phase 2)
- Generate OpenAPI from handlers; publish at `/openapi.json`.
- Add `/.well-known/gauth-configuration` returning algorithms, endpoints, supported PoA features.

### Epic I: Rights & Obligations Engine (Phase 2)
- Parse & enforce rights/obligations inside evaluation (pre/post decision). Distinguish must vs advisory obligations.

### Epic J: Metrics Expansion & OTel Spans (Phase 3)
- Export labeled counters: `gauth_decisions_total{action,resource,allow}`.
- OTel spans: issuance → delegation issuance → decision evaluation chain.

### Epic K: Ledger & Revocation Anchoring (Phase 3)
- Periodic anchoring: hash tips submitted to external timestamp (RFC3161 or transparency log API).
- Verification command replays anchor proofs.

### Epic L: SBOM + Signing (Phase 3)
- CI step: build -> syft SBOM -> cosign sign -> attest SLSA provenance.

### Epic M: Persistent Policy Store & Decision Cache (Phase 4)
- Durable store (Postgres) with migrations, indices (subject, resource, action).
- Cache with invalidation on append/rollback events.

### Epic N: Distributed Revocation & Ledger Sync (Phase 4)
- Gossip or pub/sub (e.g., NATS) broadcasting revocation events + ledger increments.
- Convergence test ensures all nodes have same chain head within SLA.

## Cross-Cutting Tasks
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
| Token validation latency (ms) | p95 < 5ms (local) |
| Decision latency (ms) | p95 < 10ms (local) |
| Policy rollback time | < 250ms |
| Replay store failure detection | < 2s |
| Multi-signature validation | All signers verified; negative tamper test fails |
| Fuzz test coverage (digest) | 1M exec / run, zero crashes |

## Risk Prioritization
High: Missing replay protection fail-closed, multi-signature enforcement, public verifiable tokens.  
Medium: Jurisdiction validator, obligations execution, ledger anchoring.  
Low: Well-known discovery, metrics labels, advisory rights engine.

## Initial Implementation Sequence (Weeks Estimate)
1. Weeks 1–2: Epics A, C (token + digest hardening).  
2. Weeks 3–4: Epic B (multi-signature PoA), start Epic D (versioning).  
3. Weeks 5–6: Epics E, F (obligations + replay protection).  
4. Weeks 7–8: Epics G, H, I (compliance & interop).  
5. Weeks 9–10: Epics J, K, L (observability + supply chain).  
6. Weeks 11–12: Epics M, N (scalability & distribution).

## Test Strategy Enhancements
- Add property tests for PoA field permutations.  
- Add concurrency tests (simulated multi-writer) for persistent policy store.  
- Benchmark harness for delegation issuance & revocation verification under load.

## Tooling Additions
- Adopt `golang-jwt/jwt/v5` or `paseto` library (remove manual parser).  
- Use `OpenTelemetry` SDK for spans & metrics pipeline.  
- Add `go-fuzz` / `github.com/dvyukov/go-fuzz` integration (or `oss-fuzz` harness) for canonicalization & expression parsing.

## Out-of-Scope (For Later)
- Formal verification / model checking.  
- Homomorphic or zero-knowledge proofs of delegation.  
- Multi-tenant isolation & namespace boundaries (Phase 5+).  

---
Maintain this plan: update status per epic in PR descriptions; regenerate summary table monthly.
