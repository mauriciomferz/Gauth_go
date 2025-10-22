# GAP Closure Two-Week Sprint Schedule
Generated: 2025-10-22

Goal: Close 100% of P0 gaps, reach readiness for initial P1 follow-ons.

## Week 1 – Foundations & Core Crypto
Day 1 (Parser & Secrets)
- Implement secret provider interfaces (filesystem mock encryption path).
- Begin parser replacement scaffolding (stream decoder skeleton + tests harness).
- Add baseline benchmarks (existing parser vs stub).
Milestone: Secret provider compiled; parser baseline test passes.

Day 2 (Parser Completion & Claims Start)
- Complete robust parser (duplicate key rejection, depth limits).
- Fuzz harness initial integration (target 100K iterations local).
- Claims refactor structure sketch.
Milestone: Parser unit tests + initial fuzz green; parse latency benchmark recorded.

Day 3 (Claims Semantics & Alg Registry Init)
- Enforce mandatory claims (typ/aud semantics).
- Structured footer metadata support.
- ECDSA keygen/sign/verify implementation.
Milestone: Claims tests pass; ECDSA verification successful; JWKS includes alg/crv.

Day 4 (Multi-Alg Detached Signatures)
- Envelope signatures array support.
- Config enforcement flag for multi-alg.
- Property tests (multi-alg consistency).
Milestone: Multi-alg issuance & verification passing; metrics counters exposed.

Day 5 (ABAC & PDP Diagnostics + Clause Map)
- Conflict diagnostics structure & response fields.
- ABAC function registry (load custom functions from config).
- Expand clause_map.json to full coverage + verification script.
Milestone: Authorization enhancements merged; clause coverage 100%; KPIs updated.

## Week 1 End Review
- Parser perf delta <10%.
- Fuzz corpus seeds stored.
- Secrets provider integrated with key rotation path (dev mode).

## Week 2 – Ledger, PoA, Governance Evidence
Day 6 (Audit Ledger Signing)
- Implement signed hash-chain ledger (append & verify modules).
- Metrics instrumentation (append counter, anchor interval gauge stub).
Milestone: Ledger verification test passes; tamper test fails as expected.

Day 7 (External Anchoring & Notarization Interface)
- Notary interface + mock provider.
- Periodic anchor worker (≤60s) writing receipt artifacts.
- Stream emission reason `anchor_notarized`.
Milestone: Notarized receipt visible in SSE stream; persistence test passes.

Day 8 (PoA Semantic Validator Expansion)
- Chain integrity & scope subset validation.
- Validator returns structured violations.
- Metrics for PoA failures.
Milestone: PoA tests pass; negative case shows violation taxonomy.

Day 9 (Integration & Hardening)
- End-to-end token issuance with multi-alg + PoA + ledger append + anchor receipt.
- Add property tests for PoA chain variations.
- Documentation updates (TOKEN_SIGNATURES.md, AUDIT_LEDGER.md, POA_VALIDATION.md).
Milestone: Comprehensive scenario test passes; docs reviewed.

Day 10 (Buffer / Risk Mitigation / Stretch)
- Address any lagging tasks; optimize parser if >10% perf regression.
- Begin scaffolding durable replay store (P1) & embed PoA (sec3.item2) if time.
- Final KPI snapshot and acceptance checklist generation.
Milestone: All P0 gaps closed; acceptance checklist complete.

## Daily Reporting Template (Short)
- Completed Tasks
- Tests Added/Run (count, pass/fail)
- Metrics Changes
- Blockers/Risks
- Next Day Focus

## Contingency
- If cryptographic tasks slip: allocate Day 9 morning to finalize multi-alg, push property tests to Day 10.
- If ledger anchoring complexity: stub anchor with deterministic hash before real external integration.

## Post-Sprint Follow-On (Initial P1)
- Policy versioning & rollback metadata.
- Durable replay store with fail-closed eviction.
- Embed full PoA structure in envelope.
- Capability policy binding & tenant segregation.

