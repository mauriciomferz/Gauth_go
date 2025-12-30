---
title: Documentation Conventions
category: guide
status: active
lastUpdated: 2025-11-12
owners: documentation-team
refreshCadence: quarterly
source: governance-policy
---
# Documentation Conventions

> Version: 1.0 • Established: 2025-11-12 • Steward: Documentation Lead
> Purpose: Provide uniform structure, naming, metadata, and lifecycle rules for all Markdown assets.

## 1. Metadata Header
Every non-generated documentation file must start with:
```
---
title: <Human readable title>
category: <one-of: architecture | rfc | guide | operations | security | performance | api | adr | release | generated | roadmap | compliance | example | ui | audit | maintenance | org | misc>
status: draft | active | deprecated | superseded
lastUpdated: YYYY-MM-DD
owners: team-alias[,optional-secondary]
tags: [tag1, tag2, ...]
---
```
Generated files add:
```
generated: true
source: <tool or script>
refreshCadence: <event|manual|per-release>
```
If a file lacks this header, add it on next touch (do **not** retroactively bump `lastUpdated` unless content changes).

## 2. Naming Patterns
| Type | Pattern | Examples |
|------|---------|----------|
| ADR | `ADR-<slug>.md` | `ADR-key-rotation-scheduler-vault-integration.md` |
| Release Notes | `RELEASE_NOTES_<version>.md` or `.auto.md` | `RELEASE_NOTES_beta.md` |
| Generated | `*.auto.md` suffix | `GAP_MATRIX.auto.md` |
| Runbook | PascalCase service event | `RedisDown.md`, `AgentAuthServiceUnavailable.md` |
| Report (phase/session) | `PHASE<number>_COMPLETION_REPORT.md` | `PHASE6_COMPLETION_REPORT.md` |
| Roadmap | `ROADMAP_<scope>.md` | `ROADMAP_NEXT_SPRINT.md` |
| Compliance Matrix | `*_COMPLIANCE_MATRIX.md` | `rfc0111_compliance_matrix.md` |

Prefer descriptive prefixes (`SECURITY_`, `PERFORMANCE_`, etc.) when ambiguity exists.

## 3. Directory Assignment (Target State)
See `DOCUMENTATION_INDEX.md` taxonomy. New files must go straight to target directory—avoid creating new top-level categories without approval.

## 4. Lifecycle States
- `draft`: content incomplete; expect rapid iteration.
- `active`: authoritative and maintained.
- `deprecated`: superseded; keep for historical context, add pointer to replacement.
- `superseded`: replaced entirely; schedule archival.

## 5. Update Cadence Guidelines
| Category | Minimum Review Frequency |
|----------|-------------------------|
| security | monthly |
| operations (runbooks) | quarterly or after incident |
| adr | upon new decision |
| api reference | per release |
| generated | per tooling run |
| roadmap | per sprint |
| compliance | per audit cycle |

## 6. Generated Files Policy
- Must live in `docs/generated/` (or migrate gradually).
- Include metadata header with `generated: true`.
- Do not manually edit; changes made via source scripts.
- PRs touching generated files should only include regeneration diff.

## 7. Content Quality Checklist
Before merging a doc PR:
1. Metadata header present.
2. Clear purpose line under the H1.
3. No trailing whitespace; max line length ≈120.
4. Tables render correctly (verify locally).
5. Cross-links use relative paths (`../architecture/...`).
6. If code snippets: specify language fences (` ```go `, ` ```bash `).

## 8. Cross-Linking Standard
Use relative paths from current file. Example:
```
See [Architecture Overview](../architecture/ARCHITECTURE.md) for component relationships.
```
Avoid absolute GitHub URLs (break on forks).

## 9. Deprecation Procedure
1. Add `status: deprecated`.
2. Insert banner: `> Deprecated: replaced by <link>`.
3. Move file (optional) to `docs/archive/` during quarterly cleanup.

## 10. ADR Process Snapshot
- New ADR: copy template from `ADR_INDEX.md` references.
- Assign sequential ordering by merge date (implicit via git history).
- Link ADR in `ADR_INDEX.md` manually until automation added.

## 11. Tagging Guidance
Tags are free-form but prefer controlled set: `[authz, crypto, performance, security, compliance, operations, api, architecture, tooling, roadmap, audit]`.

## 12. Review Roles
| Role | Responsibility |
|------|----------------|
| Docs Steward | Enforces conventions; triage misc files |
| Security Team | Reviews security category changes |
| SRE | Validates runbooks |
| Compliance Lead | Audits compliance matrices & reports |
| Release Manager | Oversees change log & release notes |

## 13. Onboarding Quick Steps
1. Read `DOCUMENTATION_INDEX.md`.
2. Follow metadata header in first contribution.
3. Place file in correct target directory.
4. Request appropriate reviewer(s) based on table above.

## 14. Automation Roadmap
- Script: `scripts/docs_index.sh` (planned) to regenerate index & validate headers.
- Lint: Add CI step to fail if metadata header missing.
- Generated relocation: enforce path check.

## 15. Merge Hygiene
Combine large structural moves and content edits separately to reduce review noise.

## 16. Future Enhancements
- Add JSON manifest summarizing docs (for tooling).
- Adopt docs search index (lunr) for local offline search.

## 17. FAQ
Q: Can I skip metadata for small examples?  
A: No—apply minimal header (`status: active`).

Q: What if a file spans multiple categories?  
A: Choose the most specific; add secondary tags.

Q: How to retire an ADR?  
A: Mark `status: superseded` and link new ADR.

---
Maintained by Documentation Lead. Suggestions welcome via PR or issue.
