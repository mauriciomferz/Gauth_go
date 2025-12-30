---
title: Demo Architecture & Two-Agent Interaction
 category: architecture-spec
 status: active
 lastUpdated: 2025-11-12
 owners: architecture-team
 refreshCadence: on-change
 source: design-session
 ---
# Demo Architecture & Two-Agent Interaction (AgentAuth Beta)
Generated: 2025-10-26

## Overview
The live demo illustrates:
1. Governance transparency (rotation summary multi-sig, attestation integrity + replay, revocation proofs).
2. Proof-of-Authorization (PoA) issuance & validation between two AI agents.
3. Reactive controls (semantic anomaly throttle).
4. External verifiability (auditor CLI + Merkle proofs).

## Components
| Component | Responsibility | Key Artifacts |
|-----------|---------------|---------------|
| Rotation Ledger | Key continuity & multi-sig integrity | Rotation summary JSON (threshold, signatures[]) |
| Attestation Service | Model limits governance snapshot | Signed attestation (nonce, combined_hash) |
| Revocation Chain | Transparency for revocations (tokens/PoA events) | Merkle root, inclusion & consistency proofs |
| PoA Service | Issue, validate, revoke proofs of authorization | PoA JSON (subject, resource, scope, delegation) |
| Auditor CLI (planned) | Independent verification | Commands: rotation-verify, attestation-verify, revocation-proof |
| Metrics Exposition | Observability of integrity + latency | Prometheus metrics (latency histograms, failure counters) |

## Request Flow Diagram (Mermaid)
```mermaid
sequenceDiagram
    participant A as Agent A
    participant B as Agent B
    participant S as AgentAuth Server
    A->>S: GET /api/v1/beta/rotations/summary
    S-->>A: Rotation summary (multi-sig)
    A->>S: PoA issuance request
    S-->>A: PoA (canonical digest + signature)
    A->>B: Present PoA + request action
    B->>S: Validate PoA
    S-->>B: Validation OK
    B->>S: GET /api/v1/model/limits/attestation
    S-->>B: Signed attestation (nonce)
    B->>S: POST /api/v1/model/limits/attestation/verify
    S-->>B: {valid:true, combined_hash, latency}
    A->>S: Trigger throttled sequence (simulate surge)
    S-->>A: 429 semantic throttle error (rfc111:model_limits)
    A->>S: Revoke PoA / token
    S-->>A: Revocation event appended
    A->>S: GET /api/v1/token/revocation/proof?id=<rev_event>
    S-->>A: Inclusion proof + merkle_root
```

## Artifact Integrity Diagram
```mermaid
graph LR
  ROT[Rotation Chain] --> SUM[Rotation Summary]
  SUM --> SIGS[Signatures[]]
  LIMITS[Model Limits Snapshot] --> ATTEST[Attestation]
  ATTEST --> NONCE[Nonce]
  ATTEST --> COMB[Combined Hash]
  REVOKE[Revocation Events] --> MERKLE[Merkle Tree Root]
  MERKLE --> PROOF[Inclusion Proof]
```

## Metrics to Highlight
| Metric | Story | Why It Matters |
|--------|-------|----------------|
| agentauth_rotation_signature_verify_latency_seconds | Efficiency of multi-sig verification | Scales with number of keys |
| agentauth_attestation_verify_latency_seconds | Attestation validation performance | Replay + signature overhead demonstration |
| agentauth_attestation_verify_failures_total | Integrity failure surface | Shows protection against tampering/replay |
| agentauth_attestation_nonce_cache_size | Replay defense footprint | Transparent memory usage / TTL pruning |

## Demo Script (Condensed)
1. Fetch rotation summary; show threshold & signatures.
2. Issue PoA (single-sig now; multi-sig planned) – print digest.
3. Validate PoA from Agent B – show success structure.
4. Fetch attestation; inspect nonce & signature fields.
5. Verify attestation (success); re-verify -> replay failure (409).
6. Trigger semantic throttle (burst requests) – observe 429 error & counter increment.
7. Revoke PoA or token; fetch revocation proof & verify root.
8. (Optional) Run auditor CLI to re-verify summary + attestation offline.

## External Verification (Planned)
- Auditor CLI fetches JSON artifacts, reconstructs canonical unsigned payloads, verifies signatures, recomputes Merkle proof, outputs machine-readable JSON.

## OpenAPI & Discovery (Planned)
Expose `/api/openapi.json` and `/.well-known/agentauth/config` containing:
```jsonc
{
  "supported_algorithms": ["EdDSA"],
  "attestation_prefix": "AGENTAUTH_MODEL_LIMIT_ATTEST:",
  "poa_digest_prefix": "AGENTAUTH_POA_DEF:",
  "replay_nonce_ttl_seconds": 3600,
  "rotation_multisig": true
}
```

## Risks & Mitigations (Demo Scope)
| Risk | Mitigation |
|------|------------|
| Replay after restart | Implement durable nonce store (JSONL + load) |
| Signature tampering | Strict canonical reconstruction + domain prefixes |
| Misleading multi-sig (weights) | Clarify equal-weight semantics; roadmap weighted enforcement |
| Proof generation failure | Graceful error codes (`revocation_proof_generation_failed`) |

## Success Criteria
- All integrity endpoints respond successfully (200) with valid artifacts.
- Replay attempt demonstrably blocked (409).
- Revocation proof root matches freshly computed root.
- Metrics show non-zero latency samples and failure counters for deliberate negative test.
- Diagrams referenced live in presentation.

---
Update after each remediation milestone; link to final slide deck resources.
