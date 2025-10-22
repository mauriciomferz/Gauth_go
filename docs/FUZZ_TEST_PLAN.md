# Fuzz & Property Test Plan

Date: 2025-10-19
Status: Draft
Scope: Strengthen robustness & integrity assurances for capability governance, PoA validation, expression engine, and canonical hashing.

## 1. Objectives
- Detect panics, invariant violations, and silent logic drifts under malformed / adversarial inputs.
- Prove canonical hash stability across ordering permutations while detecting semantic differences.
- Validate PoA definition semantic rules against wide randomized domain inputs.
- Ensure expression engine evaluation terminates within safety budgets and preserves correctness vs. reference implementation.

## 2. Targets
1. Capability Loader
   - Input: Random JSON structure (capabilities array, action_mappings object, schema_version field).
   - Properties:
     - Loader either fully accepts and atomically replaces registry OR rejects without mutation.
     - Hash stability: permutations of ordering yield identical hash.
     - Duplicate IDs, dangling references → rejection (no state change & previous hash preserved).
2. Canonical Hash Function
   - Input: Generated capability sets + mappings.
   - Properties:
     - Idempotent on repeated computation.
     - Sensitive to semantic changes (added capability, removed capability, altered mapping list membership).
     - Insensitive to ordering permutations.
3. PoA Validator
   - Input: Randomized parties, jurisdictions, validity windows, numeric limits.
   - Properties:
     - Validity window must satisfy: Start < End; duration ≤ 30 days (current rule set).
     - Grantor != Grantee when not wildcard.
     - Numeric limits produce violation counters when exceeded (requires harness combining validator + simulation).
4. Expression Engine (Policy PDP)
   - Input: Random ASTs within grammar subset, bounded depth.
   - Properties:
     - Evaluation must terminate within time budget (no runaway recursion).
     - Deterministic results given same variable bindings.
     - Safe handling of invalid regex patterns / divide-by-zero scenarios.
5. Audit Pagination
   - Input: Random sequences of audit entries with timestamps & actions.
   - Properties:
     - Pagination stable ordering preserved (monotonic insertion index).
     - Cursor semantics: successive pages concatenate to full set with no gaps/duplication.

## 3. Tooling Approach
- Use Go's built-in fuzzing (Go 1.18+) with `testing.F` for loader, hash, and expression evaluation.
- Property tests via quickcheck-style generation (e.g., `pgregory.net/rapid`) for nuanced invariants (ordering vs semantics).
- Custom generators for capability JSON ensuring controllable permutation & mutation operations.
- Limit resource usage: cap iterations, memory bounds, and execution time per fuzz function.

## 4. Fuzz Harness Sketch
```go
func FuzzCapabilityLoader(f *testing.F) {
    seed := `{"schema_version":1,"capabilities":[{"id":"cap.a"}],"action_mappings":{"token:issue":["cap.a"]}}`
    f.Add(seed)
    f.Fuzz(func(t *testing.T, input string) {
        beforeHash := currentHash()
        ok := attemptLoad(input)
        afterHash := currentHash()
        if !ok && afterHash != beforeHash { t.Fatalf("hash mutated on failed load") }
    })
}
```

## 5. Metrics & Observability
- Track rejected vs. accepted fuzz cases counts.
- Emit histogram of loader execution time.
- Count invariant violation occurrences (should remain zero after stabilization).

## 6. Exit Criteria
- Zero panics after ≥100k fuzz iterations across loader & hash functions.
- Canonical hash invariants hold for ≥10k permutation trials.
- PoA validator retains rule consistency over randomized inputs (no silent acceptance of invalid definitions).
- Expression engine evaluation remains bounded (no timeouts > configured threshold).

## 7. Risk & Mitigation
- Risk: False positives due to overly strict invariants during early evolution → Mitigate by tagging unstable rules and gating fuzz assertions behind feature flags.
- Risk: Resource exhaustion → enforce iteration & time budgets.
- Risk: Hash collisions (extremely unlikely with SHA256) → monitor for identical hash on clearly different semantic inputs; add secondary semantic diff assertion.

## 8. Roadmap
Phase 1 (Short Term): Loader + hash fuzzing & property tests.
Phase 2: PoA validator fuzz harness & numeric limit simulation.
Phase 3: Expression engine structural fuzz (AST generation) & evaluation sandbox.
Phase 4: Audit pagination correctness under randomized batch insertion & paging.

## 9. References
- `ADR-capability-governance.md`
- `GAP_MATRIX.md` Section 11 (AI Capability & Governance)
- Go Fuzzing Proposal: https://go.dev/doc/fuzz/

---
