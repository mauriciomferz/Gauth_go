# Architecture & Interaction Diagram Spec

Purpose: Provide inputs for generating visually consistent diagrams (PlantUML / Mermaid) covering system topology and protocol flows.

## Diagram 1: System Topology
Nodes:
- Client (Agent A)
- Client (Agent B)
- GAuth Web Server
- Replay WAL Store (durable) *(RB1)*
- Capability Manifest Store
- Rotation Ledger (append-only + signatures) *(RB5)*
- Revocation Tree Store (Merkle)
- External Anchor Provider
- Metrics / Prometheus
- OTEL Collector *(RB9)*
- Auditor CLI (external)
Edges:
- Agents -> Server: PoA issuance requests / token issuance
- Server -> WAL: JTI/nonce writes
- Server -> Ledger: rotation append (signed)
- Server -> Revocation Store: revoke PoA
- Server -> Anchor: combined rotation+capability hash anchoring
- Auditor CLI -> Server: read-only verification endpoints
- Server -> Metrics/OTEL: telemetry export

## Diagram 2: PoA Issuance Flow (Multi-Sig)
Sequence:
1. Client submits PoA request (scope, agent_type, action_class, sector).
2. Server constructs canonical JSON (version, weights, new taxonomy fields).
3. Signer set produces signatures (M-of-N).
4. Server verifies weighted threshold.
5. Persist PoA; respond with PoA ID + digest.
6. Auditor can fetch PoA + digest to verify offline.

## Diagram 3: Token Issue & Replay Protection
Sequence:
1. Client presents PoA ID, requests token.
2. Server generates JTI + exp.
3. WAL write (append record). Fsync or batch.
4. Return JWT.
5. Replay attempt: server checks WAL snapshot+in-memory index → rejects with `token_replay_detected`.

## Diagram 4: Revocation & Inclusion Proof
Sequence:
1. Admin revokes PoA.
2. Server updates Merkle tree, stores leaf (poa_id + reason + timestamp).
3. Client requests inclusion proof; server returns (path hashes, root, leaf).
4. Auditor verifies inclusion locally.
5. Consistency proof (Phase 2): server returns subtree evolution hashes.

## Diagram 5: Rotation Ledger & Anchoring
Sequence:
1. Periodic rotation triggered.
2. New public key set; compute new rotation hash (prev + key material).
3. Sign ledger entry (prev_hash + new_hash + timestamp).
4. Append to ledger store.
5. Anchor provider receives combined digest (rotation tip + capabilities hash) → returns receipt.
6. Receipt stored and retrievable via anchor endpoint.

## Diagram 6: Attestation Verification
Sequence:
1. Client sends attestation (nonce, sig_mode, signer list, signature bytes).
2. Server verifies nonce uniqueness via WAL (reuse → replay error taxonomy).
3. Canonical digest computed (domain separation).
4. Signature verified via agility registry.
5. Optional notarization fields cross-checked (provider vs success flag).
6. Response includes verification status.

## Visual Conventions
- Colors: Security (red), Governance (blue), Observability (purple), Storage (gray), External (orange).
- Notation: Digest objects as hex strings, signatures as base64 blocks.
- Error paths annotated with taxonomy codes in dashed red arrows.

## Mermaid Layer Hints
Use subgraphs: `subgraph Storage`, `subgraph Observability`, `subgraph External`.

## Future Extensions (Post-Beta)
- Delegation chain depth enforcement annotation.
- Capability diff endpoint sequence.

---
Update when new services or flows are added (RB2 taxonomy expansion impacts PoA Issuance diagram field list).
