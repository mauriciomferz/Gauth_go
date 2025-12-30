---
title: Capability Enforcement
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Capability Enforcement & Anchoring

This document summarizes the runtime capability matrix governance features: registry loading, action mapping enforcement, lifecycle metadata (deprecated_after / sunset_after), anchoring & notarization, audit chain, and observability metrics.

## Overview
A capability represents an atomic permission or feature token (e.g. `cap.delegation.create`). The server enforces that specific API actions present the required capabilities when `AGENTAUTH_CAPABILITY_ENFORCE=1`.

Key components:
- Registry (in-memory or file-backed) with schema_version and canonical hash.
- Action -> required capabilities mapping (`requiredActionCaps` in `BetaServer`).
- Lifecycle metadata: `DeprecatedAfter`, `SunsetAfter` fields drive optional strict enforcement (`AGENTAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE=1`).
- Anchoring: Periodic emission of signed anchor artifact capturing current registry hash & previous hash.
- External anchoring & notarization (prototype) providers for transparency (memory / TSA stub).
- Audit hash chain persistence for capability-related actions (create, revoke, enforcement denial).
- Metrics: emission counters, SLA freshness gauges, enforcement allow/deny counters.

## Environment Flags
| Flag | Purpose |
|------|---------|
| `AGENTAUTH_CAPABILITY_ENFORCE=1` | Enable runtime enforcement on delegation endpoints. |
| `AGENTAUTH_CAPABILITIES_PATH=/path/to/capabilities.json` | Load capabilities + action mappings from file instead of static seed. |
| `AGENTAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE=1` | Treat capabilities past `sunset_after` as missing (denial). |
| `AGENTAUTH_CAP_LIFECYCLE_STRICT=1` | Exclude deprecated versions from negotiation results. |
| `AGENTAUTH_CAP_ANCHOR_FILE_PATH=/path/anchor.json` | Persist latest anchor artifact. |
| `AGENTAUTH_CAP_ANCHOR_WRITE_INTERVAL=5m` | Interval between anchor emissions (default 5m). |
| `AGENTAUTH_CAP_ANCHOR_NOTARIZE=1` | Enable prototype external notarization of registry hash. |
| `AGENTAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS=600` | SLA threshold for staleness gauge. |
| `AGENTAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER=memory|tsa_stub` | External provider for anchoring receipts. |
| `AGENTAUTH_CAP_EXTERNAL_ANCHOR_RETRIES=N` | Retry attempts for external anchoring. |

## File-backed Capability Registry
Example JSON structure:
```json
{
  "schema_version": 1,
  "capabilities": [
    {"id": "cap.delegation.create", "version": "1.0", "stable": true},
    {"id": "cap.delegation.revoke", "version": "1.0", "stable": true}
  ],
  "action_mappings": {
    "delegation:create": ["cap.delegation.create"],
    "delegation:revoke": ["cap.delegation.revoke"]
  }
}
```
On load the server computes a canonical JSON, sorts capability IDs, and SHA256 hashes the blob (stored as `capabilityRegistryHash`). Hash transitions update `capabilityPrevRegistryHash`, `capabilityRegistryChangeAt`, increment metrics, and trigger anchor emission logic.

## Enforcement Flow
1. Client calls endpoint (e.g. `/api/v1/delegation/create`).
2. Server extracts capability claims from JSON body under `claims.cap`.
3. `enforceCapabilities(action, claims)` validates all required capabilities present and not sunset.
4. Denied path:
   - Returns `403 {"error":"capability_denied","missing":[...]}`.
   - Increments generic violation (`capability_denied`) plus dedicated counter `capability_enforce_denied_total`.
   - Appends audit entry with lifecycle phase metadata.
5. Allowed path:
   - Performs action (create or revoke) and returns success JSON.
   - Increments `capability_enforce_allowed_total`.
   - Appends audit entry with provided capability set.

## Metrics
Prometheus / in-memory names:
| Metric | Type | Description |
|--------|------|-------------|
| `capability_anchor_emitted_total` | counter | Successful anchor artifact emissions. |
| `capability_anchor_skipped_total` | counter | Emission attempts skipped due to interval throttle. |
| `capability_registry_hash_changed_total` | counter | Canonical registry hash transitions (semantic changes). |
| `capability_anchor_last_write_seconds` | gauge | Unix seconds of last emission. |
| `capability_anchor_age_seconds` | gauge | Age since last emission. |
| `capability_anchor_stale` | gauge | 1 if age > SLA threshold. |
| `capability_enforce_allowed_total` | counter | Enforcement decisions that passed (required caps present & not sunset). |
| `capability_enforce_denied_total` | counter | Enforcement decisions that failed (missing or sunset caps). |
| `external_anchor_attempts_total` | counter | External anchoring attempts. |
| `external_anchor_failures_total` | counter | External anchoring failures. |
| `external_anchor_latency_seconds` | histogram | Latency of external anchoring. |
| `capability_anchor_notarization_latency_seconds` | histogram | External notarization latency. |
| `capability_anchor_notarized_age_seconds` | gauge | Age since last notarization receipt. |

### Ratios & Alerting
Suggested derived ratios for dashboards:
- Enforcement denial ratio = `capability_enforce_denied_total / (capability_enforce_allowed_total + capability_enforce_denied_total)`.
- Registry churn rate = `capability_registry_hash_changed_total / time_window_hours`.
- Anchor freshness SLA: Alert if `capability_anchor_age_seconds > AGENTAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS` for >2 intervals.

## Audit Chain
Capability-related audit entries (create, revoke, enforce denial) are persisted with a hash-chain wrapper when `AGENTAUTH_CAP_AUDIT_PERSIST_PATH` is set. Each persistence write includes `prev_hash` and `hash` for tamper-evident sequencing.

## External Anchoring & Notarization (Prototype)
- External anchoring provider invoked with current registry hash (immediate attempt on startup + subsequent on changes).
- Notarization optional via `AGENTAUTH_CAP_ANCHOR_NOTARIZE=1` with latency and age metrics.
- Forced failure simulation for test determinism using `AGENTAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS`.

## Testing
- `web/capability_delegation_test.go` validates deny/allow flows.
- `web/capability_enforcement_metrics_test.go` asserts allow/deny counters increment correctly.

## Future Improvements
- Add property & fuzz tests for capability file loader canonicalization & hash stability.
- Introduce per-action labeled allow/deny counters (action label dimension) for granular ratios.
- Integrate production-grade transparency / timestamp authority.
- Formal ADRs for lifecycle deprecation & rotation scheduling.
- Add negotiation endpoint tests covering lifecycle strict filtering.

## Quick Demo
Issue a create without capability (denial) then with capability (allow):
```bash
export AGENTAUTH_CAPABILITY_ENFORCE=1
# Start server (example)
make run-web
# Denial
curl -s -X POST localhost:8080/api/v1/delegation/create -d '{"delegation_id":"demo1","subject":"alice","delegate":"bob"}' -H 'Content-Type: application/json'
# Allow
curl -s -X POST localhost:8080/api/v1/delegation/create -d '{"delegation_id":"demo2","subject":"alice","delegate":"bob","claims":{"cap":["cap.delegation.create"]}}' -H 'Content-Type: application/json'
```

## References
- Implementation: `web/server_clean.go` (enforcement handlers, anchoring logic)
- Metrics: `internal/metrics/metrics.go`, `internal/metrics/prometheus_adapter.go`
- Tests: `web/capability_delegation_test.go`, `web/capability_enforcement_metrics_test.go`
