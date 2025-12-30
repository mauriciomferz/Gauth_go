---
title: Terminology Policy
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Terminology Policy (Beta Demonstration)

> Last Updated: 2025-10-17
> Status: Active

This repository is a **Beta Demonstration** of AgentAuth concepts. To maintain consistent positioning and avoid accidental misrepresentation, the following terminology rules apply.

## Allowed Primary Term
| Concept | Canonical Term |
|---------|----------------|
| Project state | Beta Demonstration |
| Purpose | Learning / Evaluation / Experimentation |

## Replaced / Deprecated Terms
| Deprecated Word/Phrase | Replacement | Notes |
|------------------------|-------------|-------|
| Educational Implementation | Beta Demonstration | Use only in historical quotations (should fade out) |
| Educational (standalone adjective) | Beta (or omit) | Update phrasing for clarity |
| Production Ready (positive claim) | NOT Production Ready | Always keep disclaimer |

## Allowed Legacy Occurrences (Allowlist)
Minimal residual patterns intentionally retained (transition phase). The CI guard ignores only these:
```
GENERATED_API.md:.*NewEducationalServer
GENERATED_API.md:.*type EducationalServer =
web/static/js/.*educational/examples/run/.*/logs
```

Removed former allowlist entries (now migrated):
```
CHANGELOG.md:.*educational wording
web/README.md:.*legacy/educational endpoints have been fully removed
```

## Writing Guidance
1. Prefer “beta demonstration” over repeating “beta” excessively.
2. Disclaimers must include: `NOT production ready`.
3. Historical context may use past tense: “Previously referred to as the educational implementation…”. Limit to a single mention per document.
4. Example READMEs should not reintroduce the old wording in new commits.

## CI Enforcement
The workflow `.github/workflows/terminology-guard.yml` fails if disallowed terms appear outside the allowlist.

## Updating This Policy
Amend this file and adjust the regex allowlist in the workflow in the same PR to keep them synchronized. When migration checklist complete, remove all but generated alias patterns.

## Migration Completion Checklist
- [ ] All README/example headings converted
- [ ] Scripts use /api/v1/beta/* as primary paths
- [ ] Fallback educational endpoints isolated to comments or constants
- [ ] OpenAPI specs updated & HTML regenerated
- [ ] Allowlist trimmed to generated alias patterns only
- [ ] Legacy `EDUCATIONAL_DEMO.md` replaced or cross-linked and renamed

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
