---
title: Documentation Directory Mapping
category: documentation-index
status: active
lastUpdated: 2025-11-13
owners: docs-governance
refreshCadence: quarterly
source: repo-reorg
---

# Documentation Directory Mapping

Defines canonical mapping of `category` front matter values to physical directories under `docs/`.
Validator enhancement will enforce these placements.

## Mapping Rules

| Category | Target Directory | Rationale |
|----------|------------------|-----------|
| compliance-report | docs/compliance/ | Audit & compliance status / final reports |
| gap-report | docs/gap/ | Gap closure, remediation, progress tracking |
| performance-report | docs/performance/ | Load, performance, benchmarking reports |
| architecture | docs/architecture/ | High-level architecture descriptions |
| architecture-spec | docs/architecture/ | Detailed specifications & diagrams |
| audit-log | docs/audit/ | Chronological audit / activity logs |
| implementation-status | docs/status/ | Implementation progress snapshots |
| roadmap | docs/roadmap/ | Forward-looking planning artifacts |
| security-report | docs/security/ | Security assessments & cryptographic reviews |
| security-design | docs/security/ | Security design documents |
| adr | docs/architecture/adr/ | Architecture decision records |
| runbook | docs/runbooks/ | Ops procedures & runbooks |
| documentation-index | docs/ | Index / taxonomy / meta-docs |
| generated | docs/generated/ | Auto-generated content (indexes, summaries) |

## Placement Guidelines

1. Files must reside in the directory corresponding to their `category` value.
2. Cross-category aggregation files (e.g., closure summaries) use category `documentation-index` and remain in `docs/` root.
3. Auto-generated files end in `.auto.md` and use category `generated`.
4. Legacy root-level markdown should be migrated; originals removed after verification.
5. If a document logically spans multiple categories, choose the most specific primary category and reference related docs via links.

## Pending Directories To Create

If absent, the following directories should be created during migration:
- `docs/performance/`
- `docs/audit/`
- `docs/status/`
- `docs/roadmap/`
- `docs/security/`
- `docs/architecture/adr/`
- `docs/runbooks/`
- `docs/generated/`

## Validator Enforcement (Planned)

Enhancements to `tools/docvalidate`:
- Verify `category` belongs to whitelist.
- Check physical path matches mapping table.
- Report mismatch as ERROR (currently only front matter schema enforced).
- Option `--fix-paths` (future) to auto-move misplaced files.

## Migration Progress Tracking

| Category | Total (est) | Migrated | Remaining |
|----------|-------------|----------|-----------|
| compliance-report | 14 | 1 | 13 |
| gap-report | 2 | 2 | 0 |
| performance-report | 4 | 0 | 4 |
| architecture (+spec) | 5 | 0 | 5 |
| audit-log | 16 | 0 | 16 |
| others | (TBD) | - | - |

_Update counts will be refreshed after each migration batch._

## Next Steps

1. Complete compliance-report migration (root & artifacts → docs/compliance/).
2. Create missing directories and migrate performance-report files.
3. Migrate architecture & architecture-spec documents.
4. Migrate audit-log artifacts.
5. Implement validator directory enforcement.
6. Regenerate taxonomy index (`make docs-meta-index`).

---

*This file is authoritative for directory placement logic. Update when new categories are introduced.*
