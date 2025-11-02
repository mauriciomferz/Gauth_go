# ADR: Revocation Consistency Proofs (Merkle Subtree Progression)

Status: Proposed (Implemented Core Logic; Documentation Finalization Pending)  
Date: 2025-10-27  
Authors: Security & Governance Track  
Related Backlog Items: RB10 (Revocation Consistency Phase 2)  
Related Code: `pkg/delegation/*revocation*`, `web/revocation_merkle_tamper_test.go`, `web/revocation_consistency_v2_benchmark_test.go`.

## 1. Context
Revocation events must be verifiable for integrity (no removal, no reordering, no silent mutation). Phase 1 exposed basic revocation recording with linear chain hashing. Phase 2 (RB10) introduces non-trivial *consistency proofs* allowing a client or auditor to incrementally advance from a previously trusted tree root to a newer root **without re-downloading the entire revocation set**.

Primary drivers:  
- Efficiency: Avoid O(N) re-hash of entire event set for every sync; enable logarithmic progression.  
- Tamper Detection: Ensure any removed/replaced leaf or sibling alteration invalidates the proof chain.  
- Extensibility: Provide foundation for future batch proof aggregation (e.g., Merkle partials, multi-tree stitching, forward-secure snapshot notarization).

## 2. Problem Statement
Auditors holding an older revocation tree root require assurance that a new advertised root legitimately extends the prior state. Naïve approaches (download all events + re-hash) are costly and scale poorly. We need a standardized progression proof to:
1. Minimize bandwidth.  
2. Provide cryptographic soundness against leaf deletion/reordering/sibling tampering.  
3. Enable benchmarkable performance with predictable asymptotics.

## 3. Decision Drivers
- Performance: Logarithmic (or near) verification vs linear replay.  
- Simplicity: Reuse established Merkle inclusion/consistency semantics (RFC 6962 inspiration) without full transparency log overhead.  
- Testability: Straightforward deterministic test vectors; negative tamper tests (sibling nibble flip) already implemented.  
- Extensibility: Allow future upgrade to Merkle Mountain Range (MMR) or append-only transparency log if cadence or scale demands.  
- Operational Observability: Benchmarks underpin latency expectations across tree sizes (64, 256, 1024 measured).  

## 4. Options Considered
| Option | Summary | Pros | Cons | Decision |
|--------|---------|------|------|----------|
| Full Re-hash (Linear Sync) | Client re-downloads all revocations and recomputes root. | Simple | O(N) bandwidth & compute each sync | Rejected |
| Append-Only Hash Chain Only | Each revocation hashes previous tip (like rotation ledger). | Minimal complexity | No efficient subtree proofs; removal still detectable but progression inefficient | Rejected |
| Merkle Subtree Consistency (Chosen) | Provide sibling path + intermediate progression nodes bridging old -> new root. | Logarithmic size; well-studied | Requires careful assembly rules | Accepted |
| Vector Clocks / Version Map | Track per-issuer revocation monotonic counters; combine at client. | Captures causal independence | Not cryptographically aggregating leaves; harder tamper detection | Rejected |
| RSA Accumulator | Single short proof for inclusion/exclusion. | Constant-size proofs | Complex updates, witness management overhead | Deferred |
| Merkle Mountain Range (MMR) | Efficient append proofs + historical indexing. | Optimized for append transparency | Added structural complexity; not needed at current scale | Deferred |
| Sparse Merkle Tree | Large fixed address space supporting absence proofs. | Absence proofs capability | Overkill for sequential revocation IDs | Deferred |

## 5. Decision Outcome
Adopt Merkle subtree progression consistency proofs resembling RFC 6962 semantics: given OldRoot and NewRoot with event count N_old < N_new, server constructs a minimal path of nodes enabling the client to recompute NewRoot starting from OldRoot without full replay. Roots are domain-separated using a revocation-specific prefix (future enhancement) to prevent cross-structure collision.

### Key Properties
- Proof Size: O(log N) sibling digests.  
- Verification: Deterministic re-hashing from OldRoot -> NewRoot.  
- Tamper Resistance: Any mutation of siblings or leaf digest yields mismatch.  
- Replay Safety: OldRoot must match a previously trusted state; otherwise fallback to full inclusion proof or linear sync.

## 6. Data Structures
```
struct RevocationEvent {
  id: uint64                    // sequential
  subject: string
  reason_code: string
  timestamp_unix: int64
  leaf_digest: []byte           // H(domain || serialized_event)
}

struct ConsistencyProofV2 {
  oldSize: uint64
  newSize: uint64
  subtree: []HashNode           // ordered nodes bridging old -> new
}

struct HashNode {
  left: []byte
  right: []byte
}
```
Future: add `domain_prefix` constant: `revocation:v1:` for leaf hashing.

## 7. Verification Algorithm (Simplified)
1. Validate `oldSize < newSize` and non-zero boundaries.  
2. Initialize working root = OldRoot.  
3. Replay subtree nodes: for each node combine (left,right) according to position parity (mirroring RFC6962 progression) advancing partial root.  
4. After processing all nodes, expect working root == NewRoot.  
5. If mismatch -> integrity failure.

Negative test implemented in `revocation_merkle_tamper_test.go` mutates a sibling digest nibble; verification fails as expected.

## 8. Benchmark Results (Current Baseline)
From `revocation_consistency_v2_benchmark_test.go` (Mac M-series dev environment):
- 64 leaves: ~9.6e4 ns/op  
- 256 leaves: ~3.7e5 ns/op  
- 1024 leaves: ~1.56e6 ns/op  
Scaling shows near-linear in number of leaves for generation but logarithmic proof size; acceptable at current cardinalities.

## 9. Security & Integrity Considerations
- Hash Domain Separation: Planned hardening; current digest stable but lacks explicit domain prefix.  
- Collision Resistance: SHA-256 assumed; upgrade path to SHA-512 or BLAKE3 possible with version tagging.  
- Replay Protection: Auditor must pin OldRoot via a signed checkpoint (future ledger anchoring integration).  
- Tamper Resistance: Any mutation in leaf set invalidates progression chain; detection through mismatch root comparison.

## 10. Operations & Observability
- Metrics: Integrity failure increments `revocation_integrity_failures_total`.  
- Benchmark speeds recorded to guide alert thresholds if revocation throughput spikes.  
- Planned: Add histogram for proof generation latency if operational frequency increases.

## 11. Migration Plan
Phase 2 already deployed. Future migration steps:  
1. Introduce domain-prefixed leaf hashing with version bump `revocation_digest_v2`.  
2. Emit signed revocation root checkpoints to rotation ledger for external notarization.  
3. Optional upgrade to MMR if revocation entries > 10M or require efficient historical queries.

## 12. Alternatives Rejected – Rationale Summary
See Options table. Primary rejections due to complexity vs current scale (accumulators, sparse trees) or insufficient efficiency (full re-hash, basic chaining).

## 13. Future Work
- Domain-separated hashing & explicit version metadata.  
- External notarization of periodic revocation root (TSA or transparency log).  
- Batched diff endpoint returning minimal set of newly revoked subjects with proof anchor.  
- Aggregated multi-root bridging (weekly snapshots) enabling skip validation for mid-interval states.  
- Consider witness compression or inclusion of leaf index mapping for constant-time lookups.

## 14. Acceptance Criteria
- Positive verification from OldRoot -> NewRoot passes with valid proof.  
- Tamper test (sibling mutation) fails verification.  
- Benchmark provides stable latency within documented ranges.  
- Integrity metric increments only on deliberate tamper tests, not false positives.  

## 15. References
- RFC 6962: Certificate Transparency (consistency proofs).  
- BLAKE3 design notes (future hashing agility).  
- Merkle Mountain Range (Grin / blockchain append-only structures).

## 16. Status & Lifecycle
Current Status: Proposed (core implementation merged).  
Transition to Accepted once domain separation + signed checkpoints added or upon beta freeze.  
Supersession Criteria: Adoption of transparency log or accumulator-based scheme providing stronger compact proofs.

---
