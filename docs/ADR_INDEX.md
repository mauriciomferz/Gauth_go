# ADR Index

This index lists Architecture Decision Records for the GAuth beta-refactor branch.

| ADR | Status | Summary |
|-----|--------|---------|
| `ADR-capability-governance.md` | Proposed (Implemented) | Capability registry governance: schema versioning, canonical hash anchoring, transactional loader, audit pagination & discovery metadata expansion. |

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
