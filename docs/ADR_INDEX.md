# Cryptographic Enhancements (2025-10-24)

- Aggregated signature schemes (BLS, batch) and multi-algorithm support added to signature abstraction.
- Registry and interfaces updated for extensibility; placeholders for BLS/batch logic present.
- Compliance with RFC 0111/0115 strengthened; future cryptographic schemes can be integrated with minimal changes.

# ADR Index

This index lists Architecture Decision Records for the GAuth beta-refactor branch.

| ADR | Status | Summary |
|-----|--------|---------|
| `ADR-capability-governance.md` | Proposed (Implemented) | Capability registry governance: schema versioning, canonical hash anchoring, transactional loader, audit pagination & discovery metadata expansion. |
| `ADR-capability-deprecation-sunset.md` | Proposed | Capability lifecycle: deprecation and sunset phases, enforcement flags, audit chain integration, anchoring, and rollback plan. |
| `ADR-envelope-v1-sunset.md` | Draft | Envelope V1 deprecation and sunset plan: metrics-driven multi-phase migration, operational playbooks, rollback, and communication matrix. |
| `ADR-external-notarization-integration.md` | Proposed | External notarization for capability registry and audit chain: TSA and transparency log integration, metrics, verification endpoints, and security considerations. |
| `ADR-key-rotation-scheduler-vault-integration.md` | Proposed | Automated key rotation scheduler, secure secret storage (Vault/KMS), rotation log, metrics, and discovery endpoint extensions. |
| `ADR-multi-signature-threshold-enforcement.md` | Proposed | Multi-signature/threshold enforcement for Power of Attorney: N-of-M signing, verification API, metrics, and audit log integration. |

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
