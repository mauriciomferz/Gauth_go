---
title: Snapshot Pruning Design
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Snapshot & Pruning Design (Scaffold)

Status: Draft / Scaffold
Date: 2025-10-20

## Purpose
Long-lived receipt chains will grow without bound. To manage storage and verification cost we introduce snapshots and (future) pruning.

Snapshots capture an authenticated summary of the chain state at a point in time, enabling:
- Faster integrity verification from a known trusted root.
- Archival of older segments while preserving auditability.
- Batching external transparency log / TSA anchoring operations.

Pruning (future) will allow safe removal or cold-storage of older receipt segments once a snapshot + external anchor provides forward integrity.

## Snapshot Object (Proposed)
```jsonc
{
  "version": 1,
  "generated_at": "<RFC3339Nano UTC>",
  "receipt_count": <int>,
  "chain_head": "<chain_hash_of_last_receipt>",
  "merkle_root": "<optional_merkle_root_if_enabled>",
  "rotation_head": "<optional_last_rotation_receipt_hash>",
  "previous_snapshot_hash": "<hash_of_previous_snapshot_object_or_empty>",
  "external_anchors": [
     {
       "type": "tsa|tlog",
       "provider": "<provider_name>",
       "reference": "<opaque_external_id_or_serial>",
       "timestamp": "<RFC3339Nano UTC>"
     }
  ],
  "signature": "<optional_signature_over_canonical_snapshot_bytes>",
  "hash": "<sha256_of_canonical_snapshot_bytes>"
}
```

### Canonicalization
Canonical snapshot bytes = deterministic JSON (sorted keys) or CBOR. For scaffold we plan JSON with stable field ordering produced by dedicated encoder (future).

### Integrity Semantics
- `hash` allows quick verification of snapshot file integrity.
- `signature` (future): signed by current active key; may embed previous key signature for rotation continuity.
- `previous_snapshot_hash` forms a hash chain for snapshots (separate from receipt chain) enabling linear history proof.

### External Anchors
List of external timestamp or transparency proofs associated with the snapshot creation window. Enables auditors to map snapshot to externally verifiable inclusion points (e.g., TSA serial, transparency log leaf index).

## Verification Workflow (Future)
1. Load latest snapshot.
2. Verify signature(s) and chain continuity (`previous_snapshot_hash`).
3. Optionally check external anchors (RFC3161 token validation, transparency log inclusion proof).
4. Recompute Merkle root / head chain hash for receipts since snapshot to tip using incremental verification.
5. Alert if any mismatch or stale snapshot age threshold exceeded.

## Pruning Strategy (Future)
- Only prune receipts strictly before the earliest snapshot still required by retention policy.
- Maintain at least one overlapping window (e.g., last N receipts before snapshot) until snapshot externally anchored and verified.
- Provide tooling to export pruned segment to cold archive with its Merkle subtree and chain head for offline audit.

## Open Questions
- Snapshot interval heuristic (fixed time vs receipt growth factor vs rotation events).
- Signed tree head format: adopt existing transparency log STH or custom minimal structure.
- Multi-key rotation bridging: embed both old and new key signatures in first snapshot after rotation.
- Compression approach (zstd vs gzip) for archived segments.

## Next Steps
- Implement Snapshot struct & serialization (without signature).
- Add snapshot creation trigger (manual + periodic) behind feature flag.
- Provide CLI subcommand to generate and verify snapshot.
- Extend threat model to cover snapshot tampering & pruning risks.

## Merkle Incremental Optimization (Plan)

Current Approach: Full Merkle recompute each append (O(n) per entry). Acceptable for small n but scales poorly.

Target Optimization:
1. Maintain leaf hash slice plus a parallel slice of upper level nodes.
2. On append, compute new leaf hash and update only affected path up the tree:
   - If previous leaf count was odd, duplicate last leaf logic replaced by pairing new leaf with last; recompute one chain of hashes (O(log n)).
   - Maintain a map[level][]nodes; each level constructed lazily.
3. Store partial tree in memory only; persist full Merkle root per receipt (no extra on-disk structure).
4. Provide function `IncrementalMerkleUpdate(prevState, newLeafHash) -> (newRoot, newState)`.
5. Fall back to full recompute if any integrity mismatch detected or state lost.

Data Structure Sketch:
```go
type MerkleState struct {
  Leaves [][]byte // raw 32-byte hashes
  Levels [][][32]byte // Levels[0] = Leaves (as [32]byte), Levels[i] = nodes at depth i
}
```

Update Algorithm:
1. Append leaf; if Levels empty, initialize.
2. Propagate combining up until single root remains adjusting for odd node duplication rule.
3. Complexity: O(log n) per append.

Verification Shortcut:
Use stored chain hash + merkle root together: chain hash ensures ordering & tamper evidence; merkle root enables subtree proof extraction (future).

## Dual-Signature Rotation Design (Placeholder)

Enhanced Rotation Descriptor Fields (planned):
```jsonc
"rotation": {
  "old_key_id": "ed25519:old",
  "new_key_id": "ed25519:new",
  "effective_time": "2025-10-20T12:00:00Z",
  "reason": "scheduled",
  "prev_rotation_hash": "...",
  "old_key_sig": "base64url( signature_over_descriptor )",
  "new_key_sig": "base64url( signature_over_descriptor )"
}
```

Verification Workflow (future):
1. Canonicalize descriptor (excluding signatures) and domain-separate context (e.g., `gauth:rotation:v1`).
2. Verify both signatures using known public keys at time of rotation.
3. Enforce monotonic `effective_time` and continuity via `prev_rotation_hash`.
4. Fail issuance/verification if rotation descriptor signature checks fail when descriptor present.

Open Questions:
- Grace period semantics (allow both keys for limited overlap?).
- Multi-signer / threshold rotation (support more than two keys?).
- Snapshot-captured rotation attestation (snapshot includes both signatures to seal rotation event).

