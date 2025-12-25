---
title: Deprecation Timeline
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Deprecation Timeline (Educational -> Beta Endpoints)

> Last Updated: 2025-10-17
> Status: Active

Status Date: 2025-10-15

This document tracks the staged removal of legacy `educational` terminology, endpoints, and aliases after the migration to `beta` nomenclature.

## Scope
Legacy surfaces slated for removal:
| Category | Artifact / Pattern | Current Status | Removal Target |
|----------|--------------------|----------------|----------------|
| HTTP Endpoints | `/api/v1/educational/*` | Served, marked `deprecated: true` in OpenAPI | Next minor release (v0.4.0) |
| OpenAPI HTML | `educational-examples.html` | Banner added (deprecated) | Remove at v0.4.0 (retain YAML archival) |
| OpenAPI YAML | `educational-examples.yaml` | Title updated "(Deprecated)" | Archive (move to `docs/openapi/archive/`) at v0.4.0 |
| JS Fallbacks | Hardcoded fallback to educational paths | Present (commented removal note) | v0.4.0 (or earlier if no usage) |
| Generated Aliases | `NewEducationalServer`, `type EducationalServer =` | Required for backward compatibility | Review at v0.5.0; remove if consumers migrated |
| Terminology Allowlist | Remaining regex for generated aliases | Minimal | Remove once aliases dropped |
| Doc File | `EDUCATIONAL_DEMO.md` | Deprecated banner present | Replace with pointer or rename at v0.4.0 |

## Rationale
Maintains short transition window for any consumers while preventing confusion with production-grade claims.

## Milestones
| Version | Actions |
|---------|---------|
| v0.3.0 (current) | Introduced beta endpoints & specs; deprecated educational paths; added banners |
| v0.3.x patch window | Monitor for issues; communicate upcoming removal in CHANGELOG & README |
| v0.4.0 | Remove educational endpoints from server router; delete HTML; move YAML to archive; strip JS fallbacks; update allowlist to only alias lines |
| v0.5.0 | Remove generated aliases & final allowlist entries (subject to adoption review) |

## Communication Checklist
- [ ] CHANGELOG entry each release referencing upcoming removal until executed.
- [ ] README badge / note updated when endpoints removed.
- [ ] Terminology policy updated post-removal (trim alias allowlist).
- [ ] Issue created tracking alias removal for v0.5.0.

## Removal Readiness Gates
Before removing educational endpoints:
1. All docs & examples use `/api/v1/beta/*` exclusively.
2. No CI tests reference educational paths (grep validation).
3. JS/UI code path coverage confirmed for beta endpoints only.
4. OpenAPI clients (if any) regenerated against beta spec.

## Post-Removal Actions (v0.4.0)
1. 410/Gone (optional) or 404 responses for legacy paths for one further patch (consider cost/benefit).
2. Update SECURITY and DISCLAIMER docs confirming legacy naming cleaned.
3. Announce removal in release notes.

## Rollback Strategy
If unforeseen consumer usage is identified pre-v0.4.0 release candidate:
1. Retain endpoints but keep deprecated for one extended patch (v0.3.(x+1)).
2. Add telemetry counter (optional) for legacy path hits to guide safe removal.

## Ownership
Migration steward: Documentation / Governance maintainer (see CONTRIBUTORS.md)

---
End of document.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
