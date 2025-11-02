# Revocation Merkle Inclusion Proof

This document explains the structure and verification process of a GAuth revocation inclusion proof artifact.

## Artifact Structure
Example: `examples/revocation_inclusion_proof.json`

```jsonc
{
  "revocation_event": { "id": "rev-42", "hash": "sha256:<leaf>" },
  "merkle_proof": {
    "leaf_hash": "sha256:<leaf>",
    "siblings": ["sha256:<sibling1>", ...],
    "index": 42,
    "tree_size": 64,
    "computed_root": "sha256:<root>"
  },
  "signed_tree_head": {
    "chain_length": 64,
    "merkle_root": "sha256:<root>",
    "aggregate_hash": "sha256:<aggregate>",
    "threshold": 1,
    "satisfied_weight": 1,
    "signatures": [{"kid": "ed25519:<id>", "mode": "EdDSA", "signature": "<b64url>" }]
  }
}
```

## Verification Steps
1. Canonical serialization of `revocation_event` → hash = `leaf_hash` (SHA-256 domain separation recommended, e.g. prefix `GAUTH_REVOCATION_EVENT:`).
2. Iteratively combine `current = H( left || right )` using sibling ordering determined by bits of `index` (LSB first) until `computed_root` derived.
3. Assert `computed_root == signed_tree_head.merkle_root`.
4. Canonical tree head payload (excluding signatures) signed with Ed25519 under prefix `GAUTH_REVOCATION_STH:`.
5. Verify all signatures; ensure `satisfied_weight >= threshold`.
6. Accept inclusion if all checks pass.

## Pseudocode
```go
type Proof struct { Leaf string; Siblings []string; Index int }
func ComputeRoot(p Proof) string {
    cur := p.Leaf
    idx := p.Index
    for _, sib := range p.Siblings {
        if idx&1 == 0 { // leaf on left
            cur = Hash(cur + sib)
        } else {
            cur = Hash(sib + cur)
        }
        idx >>= 1
    }
    return cur
}
```

## Security Considerations
- Domain separation prevents cross-protocol hash collisions.
- Threshold signatures (future) mitigate single key compromise risk.
- Duplicate Signed Tree Head suppression prevents replay inflation.
- Persistence of latest tree head enables detection of rollback attempts.

## Future Enhancements
| Feature | Benefit |
|---------|---------|
| Complete sibling hashes in example | Enables full end-to-end test verification. |
| Batch verification API | Efficient multi-proof validation. |
| Signature aggregation (e.g. BLS) | Reduces artifact size with multi-signer setups. |
| Audit log proof integration | Extends transparency beyond revocations. |

---
See also: `docs/diagrams.md` (revocation flow), `docs/RELEASE_NOTES_beta.md`.
