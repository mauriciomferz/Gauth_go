# Policy Engine Governance & Versioning

> Status: Experimental (In-Memory) – NOT production ready
> Last Updated: 2025-10-21

This document describes the experimental policy engine, its provenance hashing model, versioning & rollback semantics, evaluation behavior, and observability.

## Overview
Policies are grouped into immutable Bundles forming a linear hash chain. Each appended bundle is assigned a monotonically increasing integer `version` beginning at 1. Governance operations allow temporary rollback to a historical version without mutating history.

```
Genesis (v1) -> v2 -> v3 -> v4 -> ...
                  ^ (rollback to v2 sets headOverride)
```
A rollback sets an override pointer; appending any new bundle clears the override and resumes forward progression.

## Data Structures
### Bundle
```jsonc
{
  "id": "b3",
  "version": 3,
  "policies": [ /* ordered set */ ],
  "created": "2025-10-21T10:11:12Z",
  "prev_hash": "<hash-of-v2>",
  "hash": "<sha256(canonical-serialization)>"
}
```

### Policy
```jsonc
{
  "id": "rbac-basic",
  "subjects": ["alice@example.com", "bob@example.com"],
  "rules": [
    {
      "actions": ["read"],
      "resources": ["report:finance"],
      "expr": "department == 'finance' && time_between('09:00','18:00')",
      "effect": "allow",
      "meta": {"owner": "risk-team"}
    }
  ],
  "meta": {"classification": "internal"}
}
```

### Evaluation Decision Extension
`EvalDecision` now includes `policy_version` (effective governing version – respects rollback) plus provenance hashes.

## Hash Computation
Canonical hash input struct:
```go
struct {
  ID       string    `json:"id"`
  Version  int       `json:"version"`
  Policies []Policy  `json:"policies"`
  Created  time.Time `json:"created"`
  PrevHash string    `json:"prev_hash"`
}
```
Steps:
1. Sort `Policies` by `Policy.ID` ascending.
2. For each policy sort `Rules` by `Rule.Expr` (stable deterministic ordering; empty exprs group together).
3. Marshal to JSON.
4. `hash = sha256(jsonBytes)`.
Including `Version` makes provenance sensitive to tampering attempts that alter revision numbers (defense against retroactive renumbering).

## Versioning Semantics
| Operation      | Behavior |
|----------------|----------|
| Append Bundle  | Assigns `Version = lastVersion + 1` (or 1 for genesis). Clears any rollback override. Recomputes hash chain head. Increments revision counter metric. |
| Rollback       | Sets `headOverride` to bundle with requested version. Fails if version not found. Does not change historical bundles or their hashes. Updates active version gauge. |
| Evaluate       | Uses `headOverride` if present, else latest bundle. Returns `policy_version` in response. |

Rollback is idempotent (repeated rollback to same version yields same state). Appending a bundle after rollback always produces the next numeric version (no gap reuse) preserving total order.

## Endpoints
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/beta/policy/bundles` | POST | Append bundle (admin token required if `GAUTH_POLICY_ADMIN_TOKEN` set). |
| `/api/v1/beta/policy/evaluate` | POST | Evaluate subject/action/resource against effective head bundle. |
| `/api/v1/beta/policy/rollback?version=NN` | POST | Activate historical version as effective head. |
| `/api/v1/beta/policy/chain` | GET | Paginated hashes; now includes `versions` array + `active_version`. |
| `/api/v1/beta/policy/provenance` | GET | Current head hash & chain verification status. |
| `/api/v1/beta/policy/metrics` | GET | Evaluation counters & latency snapshot. |
| `/api/v1/beta/policy/metrics/prometheus` | GET | Prometheus metrics including revision governance. |

## Governance Metrics (Prometheus)
```
# HELP gauth_policy_revisions_total Total appended policy bundle revisions
# TYPE gauth_policy_revisions_total counter
# HELP gauth_policy_active_version Current effective policy bundle version (rollback aware)
# TYPE gauth_policy_active_version gauge
```
Usage:
- Detect rollback: `gauth_policy_active_version < max_over_time(gauth_policy_revisions_total[5m])`.
- Rollback ratio: `(gauth_policy_revisions_total - gauth_policy_active_version) / gauth_policy_revisions_total`.

## Evaluation Metrics
Latency histogram: `gauth_policy_eval_latency_ns_bucket{le="..."}` + `_count`, `_sum` (midpoint approximation) + `gauth_policy_eval_latency_ns_p99`.
Allow/Deny counters and last decision metadata also exposed via JSON snapshot.

## Audit Integration (Planned)
Current evaluation audit entries include `bundle_hash` & `chain_head`. Future enhancement will add:
```jsonc
"meta": {
  "bundle_hash": "...",
  "chain_head": "...",
  "policy_version": 7,
  "rollback_active": true
}
```
Plus a dedicated audit event for rollback operations capturing previous vs new effective version.

## Security & Integrity Considerations
- In-memory only: restart wipes chain; persistence & tamper detection across restarts not implemented.
- Rollback unguarded by RBAC: only append endpoint uses optional admin token; rollback should adopt same header requirement (TODO).
- No multi-tenant isolation: single global chain per process.
- Hash chain linear – no Merkle inclusion proofs; large policy sets scale linearly for verification.

## Limitations & Roadmap
| Gap | Planned Direction |
|-----|-------------------|
| Persistence | Disk / DB store with snapshot + hash chain continuity. |
| RBAC for rollback | Admin token / future signed management JWT. |
| Audit events | Explicit rollback audit entry + evaluation version tagging. |
| Conflict detection | Policy substitution / diff analysis tooling (partial tests exist for tamper). |
| Merkle proofs | Replace linear chain with Merkle root + inclusion proofs for scalability. |
| Multi-tenancy | Namespaced registries keyed by tenant. |
| Metrics Depth | Add `gauth_policy_rollbacks_total`, version-labeled latency histograms. |
| Policy Diff API | Endpoint returning semantic diff between revisions (subject/resource/action changes). |

## Testing Summary
- `web/policy_version_rollback_test.go` exercises: auto-increment, rollback to historical version, evaluation version tagging, clearing override on new append.
- Chain verification ensures hash mismatch detection if any field (including version) is altered retroactively.

## Example Rollback Flow
```
# Append three bundles (versions 1..3)
POST /policy/bundles {id:"b1", ...}
POST /policy/bundles {id:"b2", ...}
POST /policy/bundles {id:"b3", ...}

# Rollback to version 2
POST /policy/rollback?version=2 -> active_version=2

# Evaluate (uses version 2 policies)
POST /policy/evaluate {subject:"alice"...} -> policy_version=2

# Append new bundle (version 4)
POST /policy/bundles {id:"b4", ...} -> rollback cleared, active_version=4
```

## Operational Alerts (Examples)
```promql
ALERT PolicyRollbackActive
  IF (max_over_time(gauth_policy_revisions_total[5m]) - gauth_policy_active_version) > 0
  FOR 3m
  LABELS {severity="warning"}
  ANNOTATIONS {summary="Rollback active", description="Active policy version behind latest revision."}

ALERT PolicyRapidChurn
  IF increase(gauth_policy_revisions_total[30m]) > 10
  FOR 5m
  LABELS {severity="info"}
  ANNOTATIONS {summary="High policy churn", description=">10 revisions in last 30m"}
```

## CLI / Environment Variables
| Var | Purpose |
|-----|---------|
| `GAUTH_SEED_POLICY` | When not `0` seeds an initial demo bundle (disable for deterministic tests). |
| `GAUTH_POLICY_ADMIN_TOKEN` | If set, required via `X-Admin-Token` header for append (extend to rollback TODO). |

## Known Issues
- Lack of persistence makes rollback ephemeral across restarts.
- No concurrency safety for simultaneous rollback and append (demo assumption of serialized admin operations).
- Metrics approximate latency sum; not suitable for precise SLO calculations.

## Newly Added (Beta Governance Enhancements)

The following features have been implemented since last update:

### Diff Endpoint `/api/v1/beta/policy/diff`
Computes differences between two bundle versions. Query params:
- `from` (optional int, defaults to current active version)
- `to` (optional int, defaults to head version)
Returns JSON `{ success, diff }` where `diff` includes arrays: `added`, `removed`, `changed`, `unchanged` plus provenance (`from_hash`, `to_hash`, `policy_chain`, `chain_head`). Each changed entry carries `from` and `to` policy bodies for inspection.

### Compact Timeline Endpoint `/api/v1/beta/policy/timeline`
Provides chronological list of all bundle versions with creation timestamps, short hash (first 8 hex), and active indicator. Response shape:
```json
{
  "success": true,
  "active_version": 3,
  "rolled_back": false,
  "timeline": [
    {"version":1,"short_hash":"c0ffee12","created":"2025-10-21T12:00:00Z","active":false},
    {"version":2,...},
    {"version":3,...,"active":true}
  ]
}
```
If a rollback is active (`active_version` differs from last appended) `rolled_back` is `true` and the UI labels the row with an `R` badge.

### Rollback RBAC Hardening
`POST /api/v1/beta/policy/rollback?version=X` now requires `X-Admin-Token` header. Missing header → `403`. This aligns rollback control with bundle append governance.

### Chain Persistence (`POLICY_CHAIN_STATE_PATH`)
When set, the server persists the entire chain after each append and rollback to the specified file path. On startup, it loads bundles from file to restore state. Limitation: The internal hash function is not exported; load re-appends bundles trusting stored fields (future improvement will add strict hash verification on load).

### Rollback Audit Event
Successful rollback emits audit entry: `action=rollback`, `resource=policy_chain`, meta includes: `target_version`, `previous_active_version`, `head_hash`.

### Evaluation Provenance Version
`policy_version` added to evaluation response JSON so clients can correlate decisions to active bundle version, even under rollback.

### Verification Badge Styling
UI governance panel displays chain verification status with color-coded badge classes (`pg-badge-ok`, `pg-badge-fail`). Verification uses `Registry.VerifyChain()` to recompute deterministic hashes and validate link integrity.

### Metrics Extensions
`/api/v1/beta/policy/metrics` surface `revisions_total` (monotonic counter) and `active_version` (gauge). Useful PromQL patterns:
```
gauth_policy_revisions_total - gauth_policy_active_version > 0  # rollback active
rate(gauth_policy_revisions_total[5m])                          # churn velocity
```

### Example Usage
```
# Diff latest active vs head
curl -s '/api/v1/beta/policy/diff'

# Diff between versions 2 and 5
curl -s '/api/v1/beta/policy/diff?from=2&to=5'

# Timeline
curl -s '/api/v1/beta/policy/timeline'

# Rollback (admin token required)
curl -s -X POST -H 'X-Admin-Token: demo-admin' '/api/v1/beta/policy/rollback?version=2'
```

### Persistence File Format (Simplified)
```json
{
  "bundles": [
    {"id":"bundle-web","version":1,"policies":[...],"created":"2025-10-21T12:34:56Z","prev_hash":"","hash":"..."},
    {"id":"bundle-web","version":2,"policies":[...],"created":"2025-10-21T13:00:00Z","prev_hash":"<hash v1>","hash":"..."}
  ]
}
```

### Future Improvements Suggested
- Exported hash function for strict persistence verification.
- Pagination for timeline endpoint.
- Policy diff rule-level granularity & semantic classification.
- `rollbacks_total` counter metric & `rollback_age_seconds` gauge.


## Contributing
Please reference `.github/ISSUE_TEMPLATE/epic_policy_governance.md` for breaking down feature additions (RBAC, persistence, audit). Submit focused PRs with updated tests and docs sections.

---
**Disclaimer:** This subsystem is educational; do not rely on it for production authorization governance until persistence, RBAC protections, and audit instrumentation are implemented.
