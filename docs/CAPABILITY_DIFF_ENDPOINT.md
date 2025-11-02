# Capability Diff Endpoint (RB13)

Endpoint: `GET /api/v1/capabilities/diff?since=<hash>`

Purpose: Allow clients to fetch only changes in capability registry since a known anchored hash, reducing payload size and enabling efficient synchronization.

## Status
Phase 2 implementation:
- In-memory snapshot ring (configurable via `GAUTH_CAPABILITY_DIFF_HISTORY`, default 32) retains recent registry states keyed by hash.
- Real diff computation for added / removed / modified capability entries.
- 404 returned for unknown baseline hashes (not in retention window) or when retention disabled.
Pending: signed diff artifact, pagination, hash domain expansion.

## Request
Query Parameters:
- `since` (optional, string): Previously observed capability registry hash (e.g. `sha256:<hex>`). If omitted or equals current hash, empty diff returned.

## Response (Schema Version 1)
```json
{
  "schema_version": 1,
  "base_hash": "sha256:...",
  "current_hash": "sha256:...",
  "added": [ {"id":"cap.x","version":"1.0","stable":true} ],
  "removed": [ {"id":"cap.y","version":"1.0","stable":false} ],
  "modified": [ {"id":"cap.z","before":{"id":"cap.z","version":"1.0","stable":true},"after":{"id":"cap.z","version":"1.1","stable":true}} ],
  "generated_at": "2025-10-27T12:00:00Z"
}
```

Field Definitions:
- `base_hash`: Hash provided by client or resolved baseline.
- `current_hash`: Current canonical registry hash.
- `added`: Capabilities present now but absent in baseline.
- `removed`: Capabilities absent now but present in baseline.
- `modified`: Capabilities with metadata differences (version / stability / lifecycle timestamps / versions slice).
- `generated_at`: RFC3339 timestamp.

## Errors
| Status | Code | Condition |
|--------|------|-----------|
| 404 | `capability_version_not_found` | `since` provided but historical snapshot not retained (future: unknown hash). |
| 400 | `capability_diff_empty_request` | (Reserved) Potential future use when `since` equals current and client opts explicit error. |

## Metrics
- `capability_diff_requests_total` (counter): Increments per request.
- `capability_diff_latency_seconds` (histogram): Latency of diff computation (hash + compare). Excludes JSON serialization time.

Future Additions:
- `capability_diff_size_bucket` label (small|medium|large) on requests counter.
- `capability_diff_modified_count` gauge/histogram for change cardinality distribution.
- Signed diff artifact (`manifest_diff_signature`).
- Pagination: `?limit=` and `?cursor=` for large diffs.
- ETag support (hash of base+current+counts) + `If-None-Match`.

## Hash Computation
`sha256` over sorted lines `ID|Version|StableFlag` where `StableFlag` is `1` or `0`.

Future: Expand domain to include `DeprecatedAfter|SunsetAfter|Versions[]` for stronger semantic integrity. Hash function lives in `internal/capability/RegistryHash` facilitating future adjustments without breaking endpoint contract (hash prefix `sha256:` retained).

## Snapshot Retention
Implemented:
- In-memory ring buffer (capacity `GAUTH_CAPABILITY_DIFF_HISTORY`, default 32).
- Each diff request stores the current snapshot if new hash not already present.
- Eviction: oldest snapshot removed when capacity exceeded; index rebuilt (small cost, low cardinality).

Future Enhancements:
- Persistence to JSONL file for cold start historical diff continuity.
- Periodic snapshot creation only on semantic change (skip identical states to reduce memory churn).

## Modified Detection Rules
Implemented rules (Phase 2):
- Differences in `Version`, `Stable`, `DeprecatedAfter`, `SunsetAfter`, or length/content of `Versions` slice mark entry as modified.
Returned `modified` array objects only include simplified `before` / `after` (id, version, stable) for now.
Future: emit extended lifecycle fields or a `fields_changed` array for granular client UX.

## Security Considerations
- Reject unknown `since` values (404) to avoid diff against untrusted baseline.
- Hash includes stability flag to prevent misleading representation.
- Signed diff artifact will later allow offline verification against policy manifest.

## Rollout Strategy
1. (Complete) Skeleton: hash exposure + empty diff.
2. (Complete) Snapshot ring + real added/removed/modified diff.
3. (Pending) Field-level change reporting & hash domain expansion.
4. (Pending) Signed diff artifact for offline integrity.
5. (Pending) Pagination for large registries.

## Alert Examples (PromQL)
```yaml
ALERT CapabilityDiffLatencyHigh
  IF histogram_quantile(0.95, sum(rate(gauth_rfc0111_capability_diff_latency_seconds_bucket[5m])) by (le)) > 0.050
  FOR 10m
  LABELS { severity="warning" }
  ANNOTATIONS { summary="Capability diff p95 latency high", description="p95 >50ms over 10m" }
```

## Open Questions
- Should `removed` include lifecycle timestamps of removed entries? (Likely yes for audit context.)
- Diff compression via patch semantics vs full objects? (Defer until large objects appear.)
- Combine with policy manifest deltas or keep distinct endpoints? (Separate for now.)
