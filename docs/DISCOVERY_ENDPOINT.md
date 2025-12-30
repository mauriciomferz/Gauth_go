---
title: Discovery Endpoint
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Discovery Endpoint (RB3)

Endpoint: `GET /api/v1/discovery`

Purpose: Provide a concise, cacheable snapshot of cryptographic + governance configuration for clients.

## Required Fields (Backlog Spec)
| Field | Type | Description |
|-------|------|-------------|
| `digest_domains` | `[]string` | Active canonical digest domain identifiers (`GAUTH_AAP-001_POA_V1`, `GAUTH_AAP-001_POA_V2`, `GAUTH_AAP-001_POA_V3|tax=1`). |
| `active_digest_domain` | `string` | Domain primarily used for newly issued single‑sig PoAs (V3 if taxonomy enabled, else V1). |
| `token_algorithms` | `[]string` | Supported token signature / JWT algorithms (e.g., `HS256`, `RS256`, `EdDSA`). |
| `replay_strict_mode` | `bool` | Whether durable replay protection (WAL + snapshot) is active (`GAUTH_REPLAY_WAL_PATH` set). |
| `poa_version_current` | `int` | Current issuance version for newly created PoAs (auto bumps to 3 when taxonomy fields present). |
| `capabilities_hash` | `string` | SHA256 hash of canonical capability registry if file‑backed (empty if static). |
| `rotation_tip_hash` | `string` | Head hash of rotation ledger (if configured). |
| `schema_version` | `int` | Discovery JSON schema version (starts at 1). |
| `generated_at` | `string` | RFC3339 timestamp of response generation. |
| `etag` | `string` | Weak ETag header (W/"<hex>") generated from stable canonical JSON (excludes `generated_at`). |
| `max_delegation_depth` | `int` (optional) | Present only when `GAUTH_MAX_DELEGATION_DEPTH` > 0; maximum allowed delegation chain length (root = depth 1). |

### Optional / Future Fields
| Field | Description |
|-------|-------------|
| `taxonomy_supported` | Boolean indicating RB2 taxonomy availability | 
| `multi_sig_supported` | True if threshold >1 configured | 
| `weights_total` | Sum of weight mapping when multi-signature weights active | 

## Caching
Header: `Cache-Control: max-age=30` (30 second client-side freshness).

Client MAY send `If-None-Match` with prior ETag to receive `304 Not Modified` when unchanged.

## Example Response
```json
{
  "schema_version": 1,
  "generated_at": "2025-10-27T00:15:10Z",
  "digest_domains": ["GAUTH_AAP-001_POA_V1","GAUTH_AAP-001_POA_V2","GAUTH_AAP-001_POA_V3|tax=1"],
  "active_digest_domain": "GAUTH_AAP-001_POA_V3|tax=1",
  "token_algorithms": ["HS256","RS256","EdDSA"],
  "replay_strict_mode": true,
  "poa_version_current": 3,
  "capabilities_hash": "sha256:ab12...",
  "rotation_tip_hash": "sha256:deadbeef...",
  "max_delegation_depth": 8
}
```

## ETag Computation
Canonical JSON built excluding `generated_at`, hashed with SHA256 hex → weak ETag: `W/"<hex>"`.

## Error Codes
| Status | Code | Reason |
|--------|------|--------|
| 500 | `discovery_internal_error` | Unexpected state (e.g., capability registry hash load failure) |

## Test Requirements
1. All fields present.
2. `Cache-Control` header equals `max-age=30`.
3. Second request with `If-None-Match` returns `304` and empty body.
4. `active_digest_domain` selection logic correct for taxonomy enabled vs disabled.
5. `max_delegation_depth` absent when env unset or set to 0; present and equals configured value when >0.

---
RB3 initial design document. Implementation will add handler and tests accordingly.