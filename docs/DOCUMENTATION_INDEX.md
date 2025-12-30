---
title: Documentation Index
category: documentation-index
status: active
lastUpdated: 2025-11-12
owners: docs-steward
generated: true
source: manual-curation
refreshCadence: weekly
---
# Documentation Index

> Generated: 2025-11-12 (manual curation). Purpose: Provide a navigable map of the documentation corpus (~900+ Markdown assets). Keep concise; link representative files per category. Update weekly or after major structural changes.

## Category Overview

| Category | Purpose | Representative Files (subset) | Location / Pattern | Steward |
|----------|---------|--------------------------------|--------------------|---------|
| Architecture & Design | High-level system, components, flows | `ARCHITECTURE_SOLUTION.md`, `docs/ARCHITECTURE.md`, `docs/ARCHITECTURE_DIAGRAM_SPEC.md`, `docs/STRUCTURE.md`, `docs/DEMO_ARCHITECTURE.md` | root + `docs/*ARCHITECTURE*` | Lead Architect |
| RFC & Standards Compliance | AAP-001/002 specs, matrices, gap analysis | `AAP001_README.md`, `docs/AAP-001.md`, `docs/AAP-002.md`, `docs/rfc_endpoint_mapping.md`, `docs/AAPCOMPLIANCE_MATRIX.md` | `docs/`, `docs/*AAP*` | Compliance Lead |
| Implementation Guides | How to run, integrate, quick start | `README.md`, `QUICK_START_GUIDE.md`, `docs/GETTING_STARTED.md`, `docs/RUNNING_SERVER.md`, `docs/DEV_RUN_GUIDE.md` | root + `docs/*RUN*` | Developer Experience |
| Operations & Runbooks | Incident response & procedures | `docs/runbooks/README.md`, `docs/runbooks/PostgreSQLDown.md`, `docs/runbooks/RedisDown.md`, `docs/DISASTER_RECOVERY_GUIDE.md`, `docs/BACKUP_RESTORE_PROCEDURES.md` | `docs/runbooks/`, ops guides | SRE |
| Security & Hardening | Threat model, hardening, crypto | `SECURITY.md`, `docs/HARDENING_ROADMAP.md`, `docs/THREAT_MODEL.md`, `docs/SECURITY_SETUP_GUIDE.md`, `docs/CRYPTOGRAPHY_IMPLEMENTATION.md` | root + `docs/*SECURITY*` | Security Team |
| Performance & Testing | Benchmarks, test plans, reports | `PERFORMANCE_TEST_REPORT.md`, `docs/BENCHMARKS.md`, `docs/FUZZ_TEST_PLAN.md`, `docs/PROPERTY_TESTING.md`, `docs/TESTING_GUIDE.md` | mixed | QA / Perf |
| API & Reference | API surfaces & generated outputs | `api/README.md`, `docs/API_REFERENCE.md`, `docs/COMPLETE_API_REFERENCE.md`, `docs/G_AGENT_API.md`, `docs/GENERATED_API.md` | `api/`, `docs/*API*` | API Owner |
| ADRs (Architecture Decisions) | Formal decision records | `docs/ADR_INDEX.md`, `docs/ADR-ledger-entry-signatures.md`, `docs/ADR-key-rotation-scheduler-vault-integration.md`, `docs/ADR-capability-governance.md`, `docs/ADR-envelope-v1-sunset.md` | `docs/ADR*` | Architecture |
| Release Notes & Changelog | Version and milestone history | `CHANGELOG.md`, `docs/CHANGELOG.md`, `docs/RELEASE_NOTES.auto.md`, `docs/RELEASE_NOTES_beta.md`, `docs/RELEASE_NOTES_2025-10-25.md` | root + `docs/RELEASE*` | Release Manager |
| Generated Artifacts | Auto-produced matrices/reports | `docs/GAP_MATRIX.auto.md`, `docs/CODE_TODO_REPORT.md`, `docs/RELEASE_NOTES.auto.md` | `docs/*auto.md` | Tooling |
| Roadmaps & Planning | Future work, sprint, phase plans | `WEEK6_ROADMAP.md`, `docs/ROADMAP_NEXT_SPRINT.md`, `docs/PHASE9_ROADMAP.md`, `docs/MILESTONE_2B_PLAN.md`, `docs/SPRINT3_PLAN.md` | root + `docs/*ROADMAP*` | PM |
| Compliance Reports & Assessments | Formal compliance status docs | `COMPLIANCE_PROGRESS_UPDATE_NOV_12_2025.md`, `docs/COMPLIANCE_IMPLEMENTATION.md`, `docs/COMPLIANCE_SUMMARY.md`, `docs/COMPLIANCE_ASSESSMENT.md`, `docs/COMPLIANCE_AAP-001_AAP-002_REPORT.md` | varied | Compliance |
| Examples & Demos | Example usage & demos | `examples/README.md`, `examples/token/README.md`, `examples/authz/README.md`, `examples/token_management/README.md`, `examples/cascade/docs/GETTING_STARTED.md` | `examples/` | DevRel |
| UI / Web | Frontend project docs | `web/ui-react/README.md`, `web/ui-react/START_HERE.md`, `web/ui-react/INTEGRATION_GUIDE.md`, `web/ui-react/STATUS_REPORT.md`, `web/ui-react/STRUCTURE.md` | `web/ui-react/` | Frontend Lead |
| Artifacts / Audits | Historical audit logs & progress | `docs/audit/AUDIT_LOG_INDEX.md`, `docs/audit/preproduction_audit_week1_day1.md`, `docs/audit/preproduction_audit_week2_day3.md`, `artifacts/conformance_report.md` | `docs/audit/`, legacy `artifacts/` | Audit |
| Maintenance & Quality | Hygiene, style, debt | `MAINTENANCE.md`, `TECH_DEBT.md`, `LINTER_EXCEPTIONS.md`, `CODE_QUALITY_ROADMAP.md`, `docs/CODE_STYLE.md` | mixed | Maintainers |
| Organizational / Exec | Executive summaries & org docs | `EXECUTIVE_SUMMARY_GAP_REMEDIATION.md`, `EXECUTIVE_SUMMARY_QA_COMPLIANCE.md`, `ORGANIZATION.md` | root | Leadership |
| Misc / Uncategorized | Pending classification | (Use triage process) | any | Docs Steward |

## Classification Rules

1. Primary key = directory path + filename pattern.
2. If a file matches multiple categories (e.g. security + architecture) pick the most specific (security first).
3. Generated artifacts must include a header marker: `generated: true` and `.auto.md` suffix where practical.
4. ADR file names start with `ADR-` or reside in ADR index; if missing, rename.
5. Runbooks belong ONLY under `docs/runbooks/`.

## Maintenance Cadence
- Weekly sweep: ensure new `.md` files have metadata header (see conventions).
- Release cycle: update Release Notes, Changelog, and Roadmap simultaneously.
- Quarterly: prune stale audit artifacts (older than 6 months) after archival.

## Quick Gap Detection
Use `grep -i "^#" -r docs/` to spot missing headers. Any file without metadata header is flagged.

## Proposed Folder Taxonomy (Target State)
```
docs/
  architecture/
  compliance/
  guides/
  operations/      (runbooks, DR, backup)
  security/
  performance/     (benchmarks, fuzz, property testing)
  adr/             (ADR records + index)
  reports/         (phase/session/completion summaries)
  generated/       (*.auto.md)
  reference/       (API, schemas)
  roadmap/         (plans, milestones)
  examples/        (if not kept under root examples/)
```
Migration is incremental; start with `architecture/`, `security/`, `operations/`, `adr/`, `generated/`.

## Triage Checklist
- Does filename express scope? If not, rename with prefix (e.g. `SECURITY_`, `PERF_`, `ADR-`).
- Is it superseded? Add `status: deprecated` in metadata.
- Is it auto-generated? Move to `docs/generated/` and ensure `.auto.md` suffix.

## Metrics (Optional)
Track count per category weekly to detect uncontrolled growth (e.g., artifacts ballooning).

## Next Steps
1. Implement directory moves for high-priority categories.
2. Apply metadata headers to uncovered files.
3. Add automation to regenerate index (`scripts/docs_index.sh`).
4. Complete migration of any remaining audit artifacts from `artifacts/` → `docs/audit/` (index now authoritative: see `docs/audit/AUDIT_LOG_INDEX.md`).

## Contributing
Open a PR for structural moves; avoid mixing code changes with large doc relocations. Update this index if adding new top-level category.
