---
title: ADR Index & Cryptographic Enhancements Overview
category: adr-index
status: active
lastUpdated: 2025-11-12
owners: architecture-team
source: internal
refreshCadence: on-change
---
# Cryptographic Enhancements (2025-10-24)

- Aggregated signature schemes (BLS, batch) and multi-algorithm support added to signature abstraction.
- Registry and interfaces updated for extensibility; placeholders for BLS/batch logic present.
- Compliance with AAP-001/0115 strengthened; future cryptographic schemes can be integrated with minimal changes.

# ADR Index

This index lists Architecture Decision Records for the AgentAuth beta-refactor branch.

Related Design Docs (non-ADR):
- `DISCOVERY_ENDPOINT.md` (RB3) – JSON schema, caching, ETag semantics for `/api/v1/discovery` transparency surface.

| ADR | Status | Summary |
|-----|--------|---------|
| `ADR-capability-governance.md` | Proposed (Implemented) | Capability registry governance: schema versioning, canonical hash anchoring, transactional loader, audit pagination & discovery metadata expansion. |
| `ADR-capability-deprecation-sunset.md` | Proposed | Capability lifecycle: deprecation and sunset phases, enforcement flags, audit chain integration, anchoring, and rollback plan. |
| `ADR-envelope-v1-sunset.md` | Draft | Envelope V1 deprecation and sunset plan: metrics-driven multi-phase migration, operational playbooks, rollback, and communication matrix. |
| `ADR-external-notarization-integration.md` | Proposed | External notarization for capability registry and audit chain: TSA and transparency log integration, metrics, verification endpoints, and security considerations. |
| `ADR-key-rotation-scheduler-vault-integration.md` | Proposed | Automated key rotation scheduler, secure secret storage (Vault/KMS), rotation log, metrics, and discovery endpoint extensions. |
| `ADR-multi-signature-threshold-enforcement.md` | Proposed | Multi-signature/threshold enforcement for Power of Attorney: N-of-M signing, verification API, metrics, and audit log integration. |
| `ADR-tracing-sampling-semantics.md` | Proposed (Implemented) | RB9 tracing sampling semantics: ratio<=0 always-sample, future inversion plan, minimal span tags. |
| `ADR-ledger-entry-signatures.md` | Proposed | Ledger entry signatures: integration of cryptographic signatures for ledger entries, ensuring integrity and authenticity. |
| `ADR-revocation-consistency-proofs.md` | Proposed (Implemented) | Revocation Merkle subtree progression consistency proofs (logarithmic update verification, tamper detection, future notarization path). |

## Conventions
- File naming: `ADR-<short-topic>.md`.
- Sections recommended: Context, Decision Drivers, Options, Decision Outcome, Data Structures, API Changes, Security/Integrity, Operations, Migration, Future Work, Acceptance Criteria, References.
- Status values: Proposed | Accepted | Deprecated | Superseded.

## Adding a New ADR
1. Create file in `docs/` with name `ADR-<topic>.md`.
2. Add entry to table above with initial Status (usually Proposed).
3. Reference related GAP Matrix sections and any test files.
4. Link ADR from relevant documentation pages (e.g., `ARCHITECTURE.md`).
5. Update status when merged/stabilized.

## Planned ADRs (Draft Not Yet Added)
- Ledger external anchoring & signature extension.
- Multi-signature delegation issuance & verification.
- Capability hash external publication & transparency log integration.
- Policy obligations & advice execution layer.
- Metrics export taxonomy & extended labeling.

---
