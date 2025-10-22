# GAP Risk & Dependency Analysis
Generated: 2025-10-22

## High-Level Dependency Graph
- Secret Provider (sec8.item1) → Audit Ledger Signing (sec5.item1) → Governance Notarization (sec11.item2)
- Algorithm Expansion (sec1.item1) → Multi-Alg Signatures (sec1.item5) → Joint Signatures (sec3.item3, P1)
- Parser Replacement (sec1.item3) → Claims Completeness (sec1.item2) → Durable Replay Fail-Closed (sec6.item1, P1)
- Audit Ledger Signing (sec5.item1) → Revocation Anchoring (sec5.item3, P2) & Conformance Evidence Integrity

## Key Risks & Mitigations
| Risk | Impact | Probability | Mitigation | Fallback |
|------|--------|-------------|-----------|----------|
| Parser performance regression >10% | Latency SLA breach | Medium | Early benchmark & optimize decoding strategy (reuse buffers) | Retain legacy parser behind flag |
| ECDSA implementation subtle bug | Token verification failures | Low-Med | Use vetted library, cross-check test vectors | Disable alg via config until fixed |
| Ledger anchoring external provider unstable | Missed anchor intervals | Medium | Implement retry + local receipt queue | Fallback deterministic hash stub |
| Secret provider Vault connectivity | Block signing operations at startup | Medium | Graceful degrade to filesystem provider with warning | Auto-switch + metrics alert |
| Multi-alg signature envelope complexity delays | Week 1 slip | Medium | Implement single array structure with forward-compatible reserved fields | Ship minimal version (single entry array) |
| Clause map expansion underestimated | Incomplete P0 closure | Low | Parallelize mapping; daily progress metric | Reduce scope temporarily but mark gap (avoid) |

## Parallelization Opportunities
- ABAC registry & PDP diagnostics can proceed independently of cryptographic tasks.
- Clause map coverage expansion runs parallel with parser & secret provider work.
- Notarization interface stub can start before ledger signing complete (compile-time interface only).

## Critical Path Summary
1. Secret Provider
2. Parser Replacement
3. Claims Semantics
4. Algorithm Expansion
5. Multi-Alg Signatures
6. Ledger Signing
7. Notarization Integration
8. PoA Semantic Validator

Delays in 1-5 cascade; ensure daily check-ins.

## Monitoring & Early Warning Metrics
- Parser perf benchmark delta gauge.
- Anchor interval gauge trending above threshold.
- Multi-alg issuance success rate.
- Secret provider backend selection (labelled metric) to detect fallback usage.

## Go/No-Go Checkpoints
- End Day 3: All crypto primitives operational; if not, reallocate Day 5 tasks.
- End Day 6: Ledger signing functional; if not, expand Day 7 with pair programming focus.

