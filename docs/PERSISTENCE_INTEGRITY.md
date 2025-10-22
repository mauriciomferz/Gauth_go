# Persistence Integrity Status Gauges

This addendum documents the integrity verification status gauges for persisted violation and semantic counter snapshots.

## Gauges
| Gauge | Meaning | Value Mapping |
|-------|---------|---------------|
| `gauth_persistence_integrity_violation` | Hash-chain integrity of violation counters persistence file | ok=1, mismatch=0, legacy=-1, unconfigured=-2 |
| `gauth_persistence_integrity_semantic` | Hash-chain integrity of semantic counters persistence file | ok=1, mismatch=0, legacy=-1, unconfigured=-2 |

## Update Semantics
Gauges reflect the most recent explicit verification endpoint invocation; they do not auto-refresh. Call:
- `GET /api/v1/beta/metrics/violations/verify`
- `GET /api/v1/beta/metrics/poa/semantics/verify`

on a periodic cadence (e.g. every 5m) to keep gauges current.

## Prometheus Example
```
# HELP gauth_persistence_integrity_violation Current integrity status of violation persistence file (ok=1 mismatch=0 legacy=-1 unconfigured=-2)
# TYPE gauth_persistence_integrity_violation gauge
gauth_persistence_integrity_violation 1
# HELP gauth_persistence_integrity_semantic Current integrity status of semantic persistence file (ok=1 mismatch=0 legacy=-1 unconfigured=-2)
# TYPE gauth_persistence_integrity_semantic gauge
gauth_persistence_integrity_semantic 0
```

## OpenTelemetry
When `GAUTH_OTEL_METRICS_ENABLE=1`, the same integer values are emitted via `Int64ObservableGauge` instruments under identical names.

## Recommended PromQL
```promql
# Any mismatch persisting >2 scrapes
(gauth_persistence_integrity_violation == 0) OR (gauth_persistence_integrity_semantic == 0)

# Legacy snapshot lingering >30m (should rotate forward)
(avg_over_time(gauth_persistence_integrity_violation[30m]) == -1) OR (avg_over_time(gauth_persistence_integrity_semantic[30m]) == -1)

# Unconfigured unexpectedly
(gauth_persistence_integrity_violation == -2) OR (gauth_persistence_integrity_semantic == -2)
```

## Operational Guidance
1. `mismatch (0)`: Capture file; recompute hash offline (`sha256(prev_hash + payload_bytes)`); investigate unauthorized modification or partial write.
2. `legacy (-1)`: Trigger a persistence save or restart to enroll into wrapper chain.
3. `unconfigured (-2)`: Ensure `GAUTH_VIOLATION_PERSIST_PATH` / `GAUTH_SEMANTIC_PERSIST_PATH` env vars are set and writable; verify at least one snapshot has been written.

## SLO
Integrity mismatches SHOULD be zero over any rolling 30d window. Future enhancement may add mismatch counters for automated periodic verification results.

---
Owner: Observability Maintainers
Status: Alpha