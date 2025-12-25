---
title: Post Beta Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Post-Beta Plan (RB12–RB14)

Date: 2025-10-27  
Scope: Transition tasks after Beta P0/P1 completion. Focus on strengthening boundaries (delegation depth), client diff efficiency, and performance transparency for multi-signatures.

## Objectives
| ID | Goal | Outcome Metric | Risk Level |
|----|------|----------------|-----------|
| RB12 | Enforce delegation chain max depth | 100% of chains rejected > configured limit, no false denies at boundary | Medium (incorrect off-by-one) |
| RB13 | Efficient capability diff retrieval | Clients pull only changed capabilities since hash; reduces payload size >70% vs full manifest | Medium (hash invalidation semantics) |
| RB14 | Multi-sig performance harness | Baseline aggregate latency distributions (p50/p95/p99) under synthetic signer counts | Low |

## RB12: Delegation Depth Limit
### Acceptance Criteria
- Env var `GAUTH_MAX_DELEGATION_DEPTH` honored at validation time.
- Error taxonomy code: `delegation_depth_exceeded` with HTTP 400 or 422 (consistent with existing validation errors).
- Negative tests: depth=limit, depth=limit+1, empty chain, single delegation.
- Instrumentation: Counter `delegation_depth_exceeded_total` (add if not present) + optional gauge for current max observed depth.
- Docs: `OBSERVABILITY.md` updated (depth env + metric + alert example) & discovery docs (`DISCOVERY_ENDPOINT.md`) expose `max_delegation_depth` field.

### Implementation Notes
- Compute depth via parent traversal; cache depth in delegation struct after first validation to avoid repeated traversal.
- Guard recursion; prefer iterative loop to avoid stack growth.
- Future: Soft warning mode (log + metric) before hard enforcement (migration flag `GAUTH_DELEGATION_DEPTH_ENFORCE_STRICT`).

### Risks & Mitigations
- Off-by-one interpretation (root depth = 0 or 1). Mitigation: define depth contract explicitly in code comment & tests.
- Existing delegations exceeding limit post-deployment. Mitigation: warm audit report listing deepest chains pre-switch.

## RB13: Capability Version Diff Endpoint
### Endpoint
`GET /api/v1/capabilities/diff?since=<hash>` returns JSON: `{ base_hash: string, current_hash: string, added: [...], removed: [...], modified: [...] }`.

### Acceptance Criteria
- Hash references existing anchored capability set or returns 404 if unknown.
- Modified item includes before/after selective fields (exclude unchanged metadata).
- Latency: p95 < 25ms for 5k capabilities diff.
- Tests: unknown hash, no changes (empty arrays), adds only, removes only, modified mixed set.
- Docs: Extend capability governance ADR with diff semantics & invariants.
- Metrics: Counter `cap_diff_requests_total`, histogram `cap_diff_latency_seconds`.

### Implementation Notes
- Maintain in-memory map hash->capability snapshot (bounded by last N versions; evict oldest beyond threshold N=20).
- Diff computation: map compare (ID keyed). Modified detection via deep comparison of fields affecting hash.
- Security: Only allow anchored hashes (signed manifest emitted) to prevent diff against uncommitted transient state.

### Extended Design Details
* Alternative query style (future): `?from=<version>&to=<version>` when explicit semantic versions are introduced; maps to historical anchored snapshots.
* Caching: Provide ETag header using a stable hash derived from (base_hash || current_hash); clients can revalidate cheaply.
* Pagination (deferred): For extremely large diffs (>1000 changed items) support `?limit` + opaque `?cursor` token.
* Error Cases:
	- Unknown `since` hash -> 404 `capability_version_not_found`.
	- `since` equals current hash -> 200 with empty arrays (no error) OR optional 400 `capability_diff_empty_request` (decide after client feedback).
* Metrics Granularity: Consider label `diff_size_bucket` derived from count of changed items (coarse buckets: small <10, medium <100, large >=100).
* Performance Guardrails: Abort diff if expected memory usage > configurable cap (e.g. 32MB) and return 413 with retry-after suggestion.
* Roadmap: Signed diff artifact (JSON + signature) for offline reconciliation.

### Risks & Mitigations
- Memory growth storing historical snapshots. Mitigation: configurable limit + persistent on-disk manifest for cold load.
- Race condition if diff requested during concurrent capability update. Mitigation: RW lock; capture pre/post snapshots atomically.

## RB14: BLS Multi-Sig Bench Harness
### Acceptance Criteria
- Harness executable `build/bin/gauth-multisig-bench` or Go benchmark under `test/` producing JSON summary `build/badges/multisig_latency.json`.
- Measures aggregate signature creation & verification across signer counts (e.g., 4,8,16,32).  
- Outputs p50/p95/p99 & max, plus error rate (should be 0 in controlled run).
- CI badge generation: convert summary into small table for README.
- Deterministic: Two consecutive runs deviate <10% on p95 aggregate latency.
- Emits optional metrics when run with `--metrics`: `multisig_aggregate_latency_seconds`, `multisig_verify_latency_seconds`, `multisig_bench_runs_total`.

### Implementation Notes
- Use synthetic key generation (in-memory) with deterministic seed for repeatability.
- Provide flag `--threshold` to vary N-of-M semantics.
- Avoid external network calls; pure crypto + memory to keep stable.
- Optionally integrate with memory Metrics adapter; not required for baseline.
 - Output format: newline-delimited JSON records to stdout plus aggregate summary file.
 - Include metadata block: `{ git_commit, go_version, cpu_arch, cpu_features, start_time }`.

### Future Enhancements
- Compare Ed25519 vs BLS vs aggregated BLS for signing throughput.
- Introduce batch verification path performance (vectorization or parallelization).
 - SVG curve generation (signers vs latency) artifact.
 - Integration with perf dashboard importer.

## Sequencing & Timeline
1. RB12 depth limit (foundation & low external dependency).  
2. RB13 diff endpoint (builds on stable capability registry).  
3. RB14 bench harness (can proceed in parallel but depends on finalized multi-sig interface maturity).  

## Shared Quality Gates
- >90% test statement coverage for RB12 & RB13 new paths.  
- Bench harness deterministic output validated in CI (non-flaky).  
- All new metrics documented in `OBSERVABILITY.md`.

## Metrics Additions Summary
| Metric | Type | Description |
|--------|------|-------------|
| `delegation_depth_exceeded_total` | Counter | Number of delegations rejected for exceeding depth limit. |
| `delegation_max_depth_observed` | Gauge | Highest delegation depth observed since start (optional). |
| `cap_diff_requests_total` | Counter | Total capability diff endpoint requests. |
| `cap_diff_latency_seconds` | Histogram | Latency distribution for diff computations. |
| `multisig_bench_run_total` | Counter | Number of bench harness runs (optional). |

## Risks Overview
- Depth enforcement rollout may surprise existing clients: stage with warning-only period.  
- Capability diff complexity grows with large manifests: precompute hash-indexed snapshots and bound history.  
- Performance harness producing inconsistent numbers if other system load fluctuates: isolate run or pin CPU affinity (optional).

## Acceptance Verification Plan
- RB12: Inject synthetic chain depths via test fixtures; verify rejection boundary & metric increments.  
- RB13: Golden diff test vectors against static capability snapshots.  
- RB14: Compare sequential runs; deviation >10% on p95 flagged.

## Open Questions
- Should capability diff include semantic versioning or just structural diff? (Deferred)  
- Introduce streaming diff for very large deltas? (Not needed yet)  
- Persist delegation depth stats for audit? (Consider small JSON snapshot at shutdown like other metrics persistence.)

---
