---
title: Demo Narrative
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Beta Demo Narrative

Goal: Showcase end-to-end governance & integrity for AI agent authorization and transparency under AAP-001/AAP-002.
Duration Target: ~8–10 minutes live.

## Cast
- Operator (human) driving CLI & HTTP requests.
- Agent A (Data Retrieval AI) requiring vault.read.
- Agent B (Ledger Append AI) requiring ledger.append.
- Auditor CLI (independent verification).

## Flow
1. Discovery & Capabilities
   - Call `/api/v1/discovery` (post RB3) to show: algorithm list, PoA version(s), replay strict mode.
   - Fetch `/api/v1/capabilities` hash & manifest.
2. PoA Issuance (Multi-Sig)
   - Issue PoA for Agent A with scope `vault.read` (Weights show threshold logic). Show canonical digest.
   - Second PoA for Agent B `ledger.append`. Demonstrate version/weights encoded.
3. Token Issuance & Replay Protection
   - Obtain access token using PoA ID (include JTI). Immediately replay same token → expect `token_replay_detected` (401) taxonomy response.
4. Vault Access Attempt
   - Valid token reaches protected endpoint `vault/read` (mock success). Show p95 latency metric snapshot.
5. Rotation & Ledger Anchoring
   - Trigger key rotation (if available) → show new rotation hash & appended ledger entry signature (post RB5).
   - Auditor CLI verifies chain integrity.
6. Revocation Transparency
   - Revoke PoA for Agent A; fetch Merkle inclusion proof for revoked entry; verify via auditor CLI.
   - Request consistency proof (Phase 2 after RB10) showing tree evolution.
7. Attestation Verification
   - Submit attestation with nonce; replay same nonce → expect nonce replay error taxonomy.
   - Show notarization fields & multi-signature verification stable across taxonomy expansion.
8. Observability
   - Display Prometheus counters (token issue count, replay detections) & OTEL trace span example (token.validate).
9. Policy Manifest Integrity
   - Fetch `/api/v1/policy/manifest` signed; locally verify signature matches discovery algorithm set.
10. Closing Risk Acknowledgment
    - Highlight remaining post-beta tasks (delegation depth, capability diff endpoint).

## Success Criteria
- Every negative path returns structured `ErrorResponse` with RFC references.
- Auditor CLI independently validates: PoA multi-signature, revocation inclusion, rotation chain, attestation signature.
- Discovery endpoint provides all cryptographic environment details needed for offline verification.

## Artifacts to Prepare
- Pre-generated sample PoAs & signatures (fallback if live issuance latency spikes).
- Synthetic revocation tree with at least 8 leaves for non-trivial proof shape.
- Prometheus snapshot script `scripts/demo_prom_metrics.sh` (to be created) for quick display.

## Contingency
- If WAL not complete: simulate durable restart via in-memory store flush (note limitation). Emphasize roadmap.
- If OTEL exporter unavailable: show trace JSON mock captured from dev run.

---
Iterate as implementation lands (RB1–RB10). PR links appended below when ready.
