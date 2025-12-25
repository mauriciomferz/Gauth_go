---
title: Risk Register
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth Risk Register (Initial Beta)

Generated: 2025-10-28
Scope: Architectural & operational risks for RFC-0111 / RFC-0115 compliance features. Updated per remediation milestone.

| ID | Risk | Description | Impact | Likelihood | Current Mitigation | Residual Risk | Owner | Status |
|----|------|-------------|--------|------------|--------------------|---------------|-------|--------|
| R1 | Replay WAL Corruption | WAL file truncation or partial write causes loss of nonce history enabling replay window after restart | Medium (limited to restart window) | Low | WAL snapshot + rotation (`web/replay_store.go`), recovery skips malformed lines, external Redis backend option | Small replay gap if corruption on last entries | Platform Eng | Open |
| R2 | Key Compromise (Signing) | Private key exfiltration enables forging attestation / policy manifest / POA signatures | High | Low | Rotating signer abstraction + optional KMS/Vault providers; domain prefix reduces cross-protocol misuse | Attack window until rotation detected; limited algorithm diversity | Security Eng | Open |
| R3 | Taxonomy Drift / Uncontrolled Expansion | Unvetted additions dilute semantic clarity of agent_type/sector/action_class | Medium | Medium | Hard-coded enumerations + validation (`taxonomy.go`); planned ADR for extension process | Potential inconsistent downstream mapping until ADR enforced | Governance | Planned |
| R4 | Revocation Chain Tamper | Insertion/deletion of events breaks append-only properties | High | Low | Merkle inclusion + consistency proofs (`revocation_chain.go`); integrity failure metrics | If auditor tooling not used regularly tamper may persist briefly | Platform Eng | Open |
| R5 | External Anchoring Outage | Capability registry / notarization providers unavailable cause transparency gaps | Medium | Medium | Latency/failure metrics; receipt chain integrity gauge; local snapshot fallback | Delayed external evidence of state transitions | Ops | Open |
| R6 | Auditor Tool Spoofing | Unsigned or modified auditor outputs mislead stakeholders | Medium | Medium | Source-available CLI; plan to sign releases; reproducible build scripts | Users may trust unsanctioned binaries before signing implemented | DevRel | Planned |
| R7 | Replay Backend TTL Misconfiguration | Excessive TTL increases memory/Redis usage and widens replay window | Low | Medium | Sensible defaults (1h), metrics for size/latency, capacity env var | Elevated resource use until config corrected | Platform Eng | Open |
| R8 | Multi-Sig Weight Ambiguity | Without weights governance decisions misinterpreted | Medium | High | Current threshold signature listing; design doc forthcoming | Perception risk in audits; limited policy expressiveness | Governance | Planned |
| R9 | Algorithm Monoculture | Ed25519 only for core artifacts increases systemic risk if vuln emerges | High | Low | Experimental BLS path; agility abstraction (`agility.go`) | Slow emergency migration until additional curves integrated | Security Eng | Planned |
| R10 | Error Taxonomy Fragmentation | Distributed error codes hinder observability & client handling | Medium | High | Structured respondError usage; plan centralized catalog | Harder automated remediation until catalog generated | Platform Eng | Planned |
| R11 | WAL Growth / Performance | Large WAL slows recovery increasing cold start risk | Low | Medium | Snapshot + rotation + pending gauge; planned Bloom filter | Longer restart latency until optimization | Platform Eng | Open |
| R12 | Manifest Replay / Stale ETag Abuse | Clients accept stale policy manifest due to caching weaknesses | Low | Low | ETag based on canonical hash; signature check; deterministic build | Minor risk if clients ignore ETag mismatch logic | Platform Eng | Open |
| R13 | Merkle Feature Optionality | Disabled Merkle root reduces receipt chain confidence | Medium | Low | Env gate; verification logic still recomputes when present | Reduced assurance until feature always-on policy | Governance | Planned |
| R14 | Consistency Proof Size Growth | Linear proof variant increases bandwidth for large chains | Low | Medium | Logarithmic V2 variant implemented; optimize sizes endpoint later | Slight inefficiency in stub sizes path | Platform Eng | Open |
| R15 | Dual Domain Signature Rollout | Partial adoption may cause inconsistent verification expectations | Low | Medium | Soft-invalid classification; metrics separated for domain signature | Transition confusion until fully mandatory | Security Eng | Open |

## Risk Treatment Pipeline
- Planned: R3 (taxonomy ADR), R8 (weighted multi-sig), R9 (algorithm agility), R10 (error catalog)
- Open: Monitoring R1, R2, R4, R5, R7, R11, R12, R14, R15

## Review Cadence
Monthly during beta; on major feature merges.

## Next Additions
- Supply chain: dependency CVE tracking integration
- Privacy: logging minimization audit
- Chaos scenarios: simulated Redis outage impact measurement
# Risk Register (Beta Scope)

| ID | Risk | Description | Residual Level | Mitigation / Planned Action | Owner | Target Sprint | Evidence Artifact |
|----|------|-------------|----------------|-----------------------------|-------|--------------|------------------|
| R1 | Crypto Agility Absence | Only Ed25519 supported for PoA + detached; no algorithm negotiation | High | Introduce pluggable signature interface (Ed25519 + stub ECDSA) | Crypto Lead | Current | Planned: `internal/crypto/signalgo` |
| R2 | Replay Persistence Loss | In-memory JTI/nonce store; crash loses history | Medium | Implement WAL + snapshot w/ compaction & TTL eviction | Security Eng | Current | Existing: `web/replay_store.go` (baseline) |
| R3 | Policy Bundle Tampering | Bundle integrity partial; manifest unsigned | Medium | Signed manifest digest (Ed25519) + verification step | Policy Eng | Current | Existing: `pkg/policy/engine.go` |
| R4 | Ledger Entry Alteration | Hash chain only; per-entry signature missing | Medium | Add entry-level signature fields + verification | Platform Eng | Current | `pkg/ledger/bolt.go` (base) |
| R5 | Partial Revocation Missing | Cannot suspend / partially revoke; binary state only | Low | Extend status enums + placeholder endpoints (501 until implemented) | Delegation Eng | Next | `delegation/revocation_chain.go` |
| R6 | No Depth Limits | Delegation chain depth not enforced | Low | Add max depth constant + validation + tests | Delegation Eng | Next | `pkg/rfc0111/rfc0111.go` |
| R7 | Missing Tracing | No OTEL spans around critical flows | Low | Introduce tracer wrapper (create/validate/revoke/multisig) | Observability Eng | Next | `internal/observability/violations.go` |
| R8 | Lack Load Bench | No sustained throughput metrics; risk of regression | Medium | Add benchmark harness (token validate + multisig verify) | QA | Next | `pkg/rfc0111/bench_test.go` (seed) |
| R9 | Replay Store Error Handling | Fail-closed mode present; persistence errors not surfaced clearly | Low | Expand error taxonomy to include replay persistence failures | Security Eng | Current | `pkg/rfc0111/rfc0111_replay_failclosed_test.go` |
| R10 | Discovery Incompleteness | No public config introspection endpoint | Medium | Implement `/well-known/gauth/config` with algorithms & requirements | Platform Eng | Current | `web/server_clean.go` (add handler) |
| R11 | Canonical Evolution Risk | Adding fields could break digest inadvertently | Low | Maintain versioned canonical schema doc; property tests for invariance | Crypto Lead | Ongoing | `pkg/rfc0111/canonical_version_weights_test.go` |
| R12 | External Notarization Coverage | Revocation anchoring partial; limited external receipts | Low | Integrate with TSA / blockchain provider adapters | Platform Eng | Future | `pkg/ledger/external_anchor.go` |

## Status Codes
- High: Must remediate for beta before demo.
- Medium: Preferred in beta; document if deferred.
- Low: Acceptable to defer with documented roadmap.

## Monitoring & Update Process
- Reviewed weekly until beta freeze.
- Changes reflected by updating `Evidence Artifact` with new file or test additions.
- Migration risks (R1, R11) require sign-off after implementation property/fuzz test results.

---
Generated: 2025-10-26
