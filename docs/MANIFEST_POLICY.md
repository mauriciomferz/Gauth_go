---
title: Manifest Policy
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Signed Policy Manifest (RB4)

Endpoint: `GET /api/v1/policy/manifest`

Purpose: Provide a tamper‑evident snapshot of capability governance state for clients / auditors. Enables offline verification (CLI) and change monitoring via hash & signature. RB4 completes the transparency surface started in RB3 (discovery) by adding cryptographic integrity to governance metadata.

## Schema (v1)
Top‑level JSON object:

| Field | Type | Description |
|-------|------|-------------|
| `schema_version` | int | Manifest schema version (starts at 1). |
| `generated_at` | string (RFC3339Nano) | Timestamp of manifest generation (excluded from hash & signature input). |
| `capabilities` | []object | Ordered list of capability entries (sorted by `id`). |
| `action_matrix` | []object | Ordered list of action → required capabilities mappings. |
| `registry_hash` | string | Canonical capability registry hash (if file-backed) `sha256:<hex>` else empty. |
| `registry_prev_hash` | string | Previous registry hash (if available) for change detection. |
| `registry_last_changed_at` | string | RFC3339 timestamp of last semantic registry change; omitted when unknown. |
| `manifest_hash` | string | Canonical SHA256 over unsigned core fields (see below). Prefixed `sha256:`. |
| `signature` | string (base64url) | Ed25519 signature over domain‑separated canonical bytes. |
| `sig_kid` | string | Key ID of signing key (active EdDSA). |
| `sig_mode` | string | Always `eddsa`. |
| `capability_count` | int | Total capabilities. Convenience / monitoring. |
| `action_count` | int | Total distinct actions. |

### Capability Entry
| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Unique capability identifier. |
| `version` | string | Current active version. |
| `stable` | bool | Stability (true=beta/GA readiness). |
| `deprecated_after` | string (RFC3339, optional) | Planned deprecation start. |
| `sunset_after` | string (RFC3339, optional) | Hard removal date/time. |
| `versions` | []string (optional) | Additional supported versions (superset includes `version`). |

### Action Mapping Entry
| Field | Type | Notes |
|-------|------|-------|
| `action` | string | Action verb/name. |
| `required` | []string | Capability IDs required. Ordered lexicographically. |

## Canonical Ordering & Hash
1. Collect capabilities; sort by `id` ascending.
2. For each capability omit empty optional fields.
3. Collect actions from `action_capabilities` (server state); sort action names ascending; sort each required slice ascending.
4. Build interim struct excluding: `generated_at`, `signature`, `sig_kid`, `sig_mode`, `manifest_hash`.
5. JSON marshal with default encoder (Go) yielding `canonical_bytes`.
6. Compute `sha256(canonical_bytes)` → hex; store as `manifest_hash` prefixed `sha256:`.

## Signature Domain Separation
Signing payload: `AGENTAUTH_POLICY_MANIFEST:` + `canonical_bytes` (same bytes used for hash).
Signature: Ed25519 using active key from `crypto.GlobalEdDSARegistry` (sig_kid = active ID, sig_mode = `eddsa`).

## ETag & Caching
| Header | Value | Notes |
|--------|-------|-------|
| `ETag` | `W"<hex_of_sha256(canonical_bytes)>"` | Weak ETag; hex matches portion of `manifest_hash` without `sha256:` prefix. |
| `Cache-Control` | `max-age=60` | Clients may reuse payload for up to 60s to reduce load. |

Conditional Requests: Provide `If-None-Match: <ETag>`; server returns `304 Not Modified` with empty body if canonical bytes unchanged.

Mismatch Behavior: A different (stale or random) `If-None-Match` value yields a standard `200` response with a full body (verified via tests).

## Verification Steps (CLI / Client)
1. Fetch JSON via HTTPS.
2. Temporarily remove `generated_at`, `signature`, `sig_kid`, `sig_mode`, `manifest_hash` from a copy.
3. Re-marshal copy → `canonical_bytes`.
4. Compute local hash; compare with `manifest_hash` (prefix handling). Fail if mismatch.
5. Prepend `AGENTAUTH_POLICY_MANIFEST:` to `canonical_bytes`; verify Ed25519 signature using public key discovered through `/.well-known/agentauth-configuration` JWKS (or dedicated EdDSA key list endpoint if available). Fail if verification fails.
6. (Optional) Record `manifest_hash` for drift / change detection dashboards.
6. Use `ETag` for subsequent conditional requests.

## Error Cases
| HTTP Status | Code | Condition |
|-------------|------|-----------|
| 500 | `manifest_build_failed` | Unexpected marshaling or capability enumeration failure |
| 500 | `signing_unavailable` | EdDSA registry unavailable or no active key |
| 503 | `registry_unavailable` | Registry in transient invalid state (future) |

Error Payload Format:
```jsonc
{
  "success": false,
  "code": "signing_unavailable",
  "error": "signing_unavailable",
  "message": "active eddsa key unavailable",
  "rfc_ref": "AAP-001:policy_manifest"
}
```

## Example (Abbreviated)
```json
{
  "schema_version": 1,
  "generated_at": "2025-10-27T11:20:05.123456Z",
  "capabilities": [
    {"id":"cap.transfer","version":"v1","stable":true},
    {"id":"cap.issue","version":"v1","stable":true,"deprecated_after":"2025-12-31T00:00:00Z"}
  ],
  "action_matrix": [
    {"action":"transaction:execute","required":["cap.transfer"]},
    {"action":"transaction:issue","required":["cap.issue"]}
  ],
  "registry_hash": "sha256:ab12...",
  "manifest_hash": "sha256:deadbeef...",
  "capability_count": 2,
  "action_count": 2,
  "signature": "5Q0p...base64url...",
  "sig_kid": "eddsa-key-1",
  "sig_mode": "eddsa"
}
```

## Future Extensions (v2+ Candidates)
- Add `policy_chain_head` (hash chain of previous manifest hashes).
- Include `notarization` object (timestamping receipt).
- Optional `diff_since` endpoint for incremental changes.
- Add `integrity_status` field mirroring other chains (ok|mismatch|unconfigured).
- Embed multi-signature envelope referencing policy manifest hash for stronger federation scenarios.
- Introduce `policy_chain_head` anchored in external transparency log (RFC0150 ledger).

## Test Requirements
1. Deterministic hash: two successive builds with identical registry state must produce identical `manifest_hash` & signature (ignoring `generated_at`).
2. Signature verification positive/negative (tamper one capability → failure).
3. 304 behavior with `If-None-Match`.
4. Missing active key triggers `signing_unavailable` (simulate by unregistering registry temporarily).
5. Metrics counter increments (`IncPolicyManifestEmitted`) per 200 response (non-304) ensuring observability.

## Instrumentation
Metric Counter: `policy_manifest_emitted_total` (memory name: `policyManifestEmitted`). Incremented each time the endpoint serves a `200` response (not on `304`). Enables rate / availability tracking and alerting for signing regressions.

## CLI Verification Tool
Path: `cmd/verify-manifest`

Features:
- Fetch or read manifest JSON (HTTP or local file).
- Reconstruct canonical struct, recompute hash, compare with `manifest_hash`.
- Verify Ed25519 signature with domain separation.
- Output human or machine-readable JSON.

Usage Examples:
```bash
go run ./cmd/verify-manifest -public-key "$PUB" \
  -url http://localhost:8080/api/v1/policy/manifest -print-canonical

go run ./cmd/verify-manifest -file manifest.json -public-key-file ./pub.key -json
```

Exit Codes:
| Code | Meaning |
|------|---------|
| 0 | Verification success |
| 1 | Hash or signature mismatch / structural failure |
| 2 | Usage / input error (missing key, fetch failure) |

JSON Output (success):
```json
{"success":true,"status":"ok","kid":"abc123","manifest_hash":"sha256:...","capabilities":42,"actions":17}
```

JSON Output (failure example):
```json
{"success":false,"status":"signature_invalid","kid":"abc123"}
```

## Security Considerations
- Domain separation prevents cross‑protocol signature replay.
- Hash + signature over canonical subset excludes volatile fields, ensuring stability.
- Weak ETag sufficient: collision risk negligible for SHA256; primary integrity comes from signature.
- Public key discovery should be pinned / validated using existing trust anchor mechanisms (JWKS integrity, ledger anchoring in future RB). 
- Tamper detection validated in tests (altering capability version invalidates signature).

## RB4 Implementation Summary
RB4 introduces a cryptographically signed policy manifest consolidating capability governance and action matrix into a stable, verifiable artifact. The endpoint returns a canonical JSON structure, a SHA256 manifest hash (`manifest_hash`), and an Ed25519 signature with domain separation (`AGENTAUTH_POLICY_MANIFEST:` prefix). Caching is optimized using a weak ETag with 60 second TTL; conditional requests reduce recomputation. Error handling standardizes structured payloads via `respondError`. Instrumentation adds the `policyManifestEmitted` counter. A standalone CLI (`verify-manifest`) enables offline verification (hash + signature) supporting audits and drift monitoring. Tests cover determinism, signature verification & tamper detection, conditional caching (304), error path (`signing_unavailable`), and metrics emission.

---
RB4 schema design document.