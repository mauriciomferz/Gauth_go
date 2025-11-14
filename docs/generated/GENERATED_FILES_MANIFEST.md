# Generated Files Manifest

> Purpose: Track all auto-generated documentation artifacts. Do not edit manually.
> Last Refresh: 2025-11-12

## Generated Artifacts
| File | Source / Script | Refresh Cadence | Notes |
|------|-----------------|-----------------|-------|
| `docs/GAP_MATRIX.auto.md` | gap matrix generator | per compliance scan | Compliance gap matrix snapshot |
| `docs/RELEASE_NOTES.auto.md` | release notes script | per release | Aggregated commit & PR summaries |
| `docs/generated/CODE_TODO_REPORT.auto.md` | todo harvest tool | daily (CI) | Extracts TODO/FIXME from source |

## Conventions
- Suffix: `.auto.md` indicates regeneration safety.
- Header must include `generated: true` and `source:`.
- Generated files should reside under `docs/generated/` (migration in progress).

## Migration Backlog
| Action | Target Date | Owner |
|--------|-------------|-------|
| Move `GAP_MATRIX.auto.md` to `docs/generated/` | 2025-11-20 | Tooling |
| Move `RELEASE_NOTES.auto.md` to `docs/generated/` | 2025-11-20 | Release Manager |
| Rename & move `CODE_TODO_REPORT.md` to `docs/generated/CODE_TODO_REPORT.auto.md` | 2025-11-12 | Tooling |

## Verification
Run (planned script):
```
./scripts/docs_index.sh --validate-generated
```
To confirm each listed file has a metadata header.

---
Maintenance: Update after each addition or removal of generated docs.
