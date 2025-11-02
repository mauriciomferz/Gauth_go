# Revocation Consistency Proofs (consistency_v2)

## Objective
Provide cryptographically verifiable append-only assurances for the revocation (delegation) event log by enabling third parties (auditors) to:
1. Fetch consecutive signed tree heads (STHs) of the revocation chain.
2. Verify a compact RFC6962-style consistency proof that the later tree is an append-only extension of the earlier tree.
3. Detect any fork / rewrite attempts (split view) or removal of historical revocation events.

## Scope (Phase 1 / Stub)
Phase 1 introduces the API surface and auditor mode returning a placeholder proof for identical head comparisons (trivial consistency). Implementation focuses on wiring and contracts; Merkle proof generation between arbitrary sizes lands in Phase 2.

## Data Structures
```go
// TreeHead represents a signed snapshot of the revocation chain state.
type TreeHead struct {
    Version          int           // protocol version (start with 1)
    ChainLength      int           // number of revocation events included
    MerkleRoot       string        // hex encoded root hash
    AggregateHash    string        // optional aggregate chaining hash
    Timestamp        time.Time     // RFC3339 when head sealed
    Signatures       []Signature   // multi-signature support (threshold already implemented)
    Threshold        int           // required total weight
    WeightsTotal     int           // total available signing weight
    SatisfiedWeight  int           // achieved signing weight for this head
}

// ConsistencyProof (Phase 2 target)
type ConsistencyProof struct {
    OlderSize  int            // N
    NewerSize  int            // M >= N
    Path       []string       // ordered sibling hashes required to recompute older root within newer tree
    RootOlder  string         // Merkle root at size N (redundant but explicit)
    RootNewer  string         // Merkle root at size M
}
```

## API Endpoints
| Endpoint | Method | Phase | Description |
|----------|--------|-------|-------------|
| `/api/v1/revocation/head` | GET | existing | Returns latest tree head (already provided via rotation / revocation endpoints). |
| `/api/v1/revocation/consistency?older=<n>&newer=<m>` | GET | Phase 1 (stub) | Returns JSON with requested sizes, available heads, and placeholder proof if `n==m`. Error if unknown sizes. |

### Phase 1 Response Shape (Stub)
```json
{
  "success": true,
  "older_size": 42,
  "newer_size": 42,
  "head_older": { "chain_length": 42, "merkle_root": "sha256:..." },
  "head_newer": { "chain_length": 42, "merkle_root": "sha256:..." },
  "proof": { "older_size": 42, "newer_size": 42, "path": [], "root_older": "sha256:...", "root_newer": "sha256:...", "trivial": true }
}
```

### Error Taxonomy (extend RFC111 style)
Code | Reason | HTTP | RFC Ref | Notes
-----|--------|------|---------|------
`consistency_invalid_params` | invalid_params | 400 | `rfc111:revocation_consistency` | Non-positive or older>newer.
`consistency_head_unknown` | head_unknown | 404 | `rfc111:revocation_consistency` | Requested size not sealed into a head yet.
`consistency_proof_unavailable` | proof_unavailable | 501 | `rfc111:revocation_consistency` | Phase 1 placeholder for differing sizes.

## Auditor CLI Mode
New mode: `revocation-consistency`.
Usage examples:
```
auditor -mode revocation-consistency -base https://gauth.example -older 100 -newer 150
```
Phase 1 behavior:
- Fetch `/api/v1/revocation/consistency?older=100&newer=150`.
- If 501 proof_unavailable: emit informative message, exit 0 (non-fatal while feature incubates).
- If success: verify trivial proof logic (sizes equal & path empty) or, in Phase 2, recompute root from path.

## Verification Algorithm (Phase 2)
Given `OlderSize=N`, `NewerSize=M`, and `Path` of sibling hashes:
1. Reconstruct root_N by iteratively hashing along path using RFC6962 semantics: `Hash(node) = SHA256(0x00 || leaf)` / `SHA256(0x01 || left || right)`.
2. Starting from the set of leaves up to N (not all stored: reconstruct using path), ascend to root_N.
3. Using remaining path components and knowledge of M, embed root_N into larger tree and confirm final recomputed root equals `RootNewer`.
4. Compare supplied `MerkleRoot` of current head with recomputed `RootNewer`; mismatch => append-only violation.

Edge Cases:
- N == M (trivial): path empty; proof valid immediately.
- N == 0: treat empty tree root as predefined constant (e.g., `sha256:EMPTY`).
- Path length upper bound: `O(log M)`.
- Malformed hash values (non-hex) => reject.

## Storage Considerations
To support arbitrary historical head queries:
- Persist a slice of TreeHeads (`revocationHeads[]`), each sealed on event append or scheduled rotation.
- Optional compaction: keep every head, or only periodic checkpoints (hourly) + on rotation; consistency proofs across gaps still valid if requested sizes match existing checkpoints.

## Metrics (Future)
Metric | Type | Description
-------|------|------------
`gauth_revocation_consistency_requests_total` | counter | Total consistency queries.
`gauth_revocation_consistency_proofs_total` | counter | Proofs successfully generated.
`gauth_revocation_consistency_failures_total` | counter | Failed proof verifications.
`gauth_revocation_consistency_latency_seconds` | histogram | Proof build time.

## Phase 1 Deliverables
- `docs/CONSISTENCY_PROOFS.md` (this document)
- Server route stub returning trivial proof when `older==newer==latest_length`.
- Auditor CLI mode consuming stub.
- Test ensuring JSON contract and trivial verification pass.

## Phase 2 Deliverables
- Actual consistency proof generation between arbitrary sealed heads.
- Auditor verification logic implementing RFC6962 reconstruction.
- Negative tests: tampered path, wrong root, size mismatch.

## Security Considerations
- Ensures append-only log transparency; prevents silent revocation removal.
- Multi-signature STHs reduce single key compromise risk.
- Auditor independence: all verification local; only requires heads + proof.

## Open Questions
- Persist every head vs. checkpoint strategy? (Initial: every head for simplicity.)
- Compression of path (store bytes vs hex)? (Initial: hex for readability.)
- Canonical domain separation for Merkle node hashing (prefix 0x00/0x01 or textual tags)? (Adopt RFC6962 bytes for interoperability.)

## Next Steps
1. Implement stub endpoint & CLI mode.
2. Add trivial proof test.
3. Introduce head persistence slice with bounded capacity (configurable; e.g. env `GAUTH_REVOCATION_HEAD_CAP`).
4. Implement real proof construction.
