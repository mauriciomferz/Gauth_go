# RFC Source Ingestion (Phase 0)

This directory will hold canonical (or near-canonical) text extracts for RFC 0111 and RFC 0115 sections.

## Goals (Phase 0)
- Provide stable text inputs for automated clause indexing.
- Extract normative statements (MUST, MUST NOT, SHOULD, SHOULD NOT, MAY, REQUIRED) for machine verification tooling.
- Emit a JSON index mapping: `rfc`, `section_id`, `title`, `normative_level`, `raw_text`, `hash`.

## Files
- `rfc0111.md` – Placeholder structured content for RFC 0111.
- `rfc0115.md` – Placeholder structured content for RFC 0115.
- `CLAUSE_INDEX_SPEC.md` – Schema definition for generated clause index.

## Not Canonical
Texts here are **derivative placeholders** pending permission and sourcing of official spec wording. Do not treat as authoritative; they exist to enable tooling scaffolds.

## Next Steps
1. Replace placeholder section text with actual specification language when available.
2. Run `make rfc_clause_index` to generate `docs/rfc/rfc_clause_index.json`.
3. Link each row of `docs/RFC_COMPLIANCE_MATRIX.md` and `docs/RFC_MAP.md` to real clause IDs.

