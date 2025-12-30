---
title: Clause Index Spec
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Clause Index JSON Schema (Draft)

Each clause entry object:
```
{
  "rfc": "AAP-001" | "AAP-002",
  "section_id": "1.2" ,
  "fragment_id": "1.2.3" ,
  "title": "Delegation Chain Integrity",
  "normative_statements": [
     {
       "level": "MUST" | "SHOULD" | "MAY" | "MUST_NOT" | "SHOULD_NOT" | "REQUIRED",
       "text": "The implementation MUST link each delegation to its parent via a cryptographic hash.",
       "line": 42
     }
  ],
  "raw_block": "Full original markdown snippet for the section.",
  "block_hash": "sha256-...",
  "source_file": "aap001.md"
}
```

Files aggregated into top-level array:
```
{
  "generated_at": "2025-10-18T00:00:00Z",
  "clauses": [ ... ]
}
```

## Hashing
Use SHA256 over the raw_block string (normalized LF line endings). Domain separate prefix: `AGENTAUTH-RFC-BLOCK:`.

## Normalization Rules
- Trim trailing whitespace per line.
- Preserve internal blank lines.
- Do not alter capitalization.

## Normative Keyword Detection
Regex: `\b(MUST(?: NOT)?|SHOULD(?: NOT)?|MAY|REQUIRED)\b` (case sensitive).

## Open Questions
- Whether to include informational (non-normative) statements for context.
- Multi-language support out of scope in Phase 0.

