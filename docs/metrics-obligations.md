---
title: Metrics-Obligations
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Obligation Execution Metrics

This document describes the new obligation execution observability primitives added to GAuth.

## Overview
Obligations (and advice) are post-decision actions executed by the PDP engine. Two production concerns motivated new instrumentation:
1. Latency visibility per obligation execution (to identify slow hooks / external calls).
2. Mandatory obligation enforcement failures that flip an `allow` decision to `deny` when `WithObligationFailureDenies(true)` is configured.

## Metrics Surface
### Prometheus (Adapter)
| Metric | Type | Description |
|--------|------|-------------|
| `gauth_rfc0111_obligation_latency_seconds` | Histogram | Per-obligation execution latency. Buckets inherit adapter latency buckets (default fast path: 100µs..100ms). |
| `gauth_rfc0111_mandatory_obligation_failures_total` | Counter | Count of mandatory obligation failures that reversed an allow decision into a deny outcome. |
| `gauth_rfc0111_obligations_executed_total` | Counter | (Existing) Successful obligation/advice executions. |
| `gauth_rfc0111_obligations_failed_total` | Counter | (Existing) Failed obligation/advice executions (non-mandatory or mandatory). |

> Note: `obligations_failed_total` counts all failures; `mandatory_obligation_failures_total` is a subset representing failures of obligations marked `Mandatory` that also changed the final decision.

### In-Memory Snapshot Fields (`Memory.SnapshotEx()`)
| Field | Type | Notes |
|-------|------|-------|
| `obligations_executed_total` | uint64 | Successful executions. |
| `obligations_failed_total` | uint64 | Failed executions. |
| `obligation_latency_count` | uint64 | Number of latency samples recorded. |
| `obligation_latency_total_ns` | uint64 | Aggregate latency nanoseconds. Derive average: `total_ns / count`. |
| `obligation_latency_max_ns` | uint64 | Maximum observed latency. |
| `mandatory_obligation_failures_total` | uint64 | Mandatory failures that flipped allow->deny. |

## Instrumentation Semantics
- Latency is recorded after each obligation execution attempt (success or failure).
- Mandatory failure counter increments only when: obligation is marked `Mandatory` AND execution returned an error AND original decision was `allow` AND engine configured with `WithObligationFailureDenies(true)` causing reversal.
- Histogram buckets may be tuned via `PrometheusAdapterOptions.Buckets` if obligations are routinely slower than validation path.

## Example PromQL Queries
```promql
# 95th percentile obligation latency (5m window)
histogram_quantile(0.95, sum by (le) (rate(gauth_rfc0111_obligation_latency_seconds_bucket[5m])))

# Mandatory failure rate per 10 minutes
rate(gauth_rfc0111_mandatory_obligation_failures_total[10m])

# Failure ratio (all failures vs executes) last hour
sum(rate(gauth_rfc0111_obligations_failed_total[1h])) / sum(rate(gauth_rfc0111_obligations_executed_total[1h]))
```

## Suggested Alerting (Initial Baselines)
| Alert | Expression | Rationale |
|-------|------------|-----------|
| High obligation latency p95 | `histogram_quantile(0.95, sum by (le) (rate(gauth_rfc0111_obligation_latency_seconds_bucket[15m]))) > 0.025` | >25ms sustained may indicate downstream degradation. |
| Mandatory failure surge | `increase(gauth_rfc0111_mandatory_obligation_failures_total[30m]) > 0` | Any occurrence may warrant investigation (start as warning). |
| Failure ratio elevated | `sum(rate(gauth_rfc0111_obligations_failed_total[30m])) / sum(rate(gauth_rfc0111_obligations_executed_total[30m])) > 0.05` | >5% failure ratio suggests instability in obligation handlers. |

Tune thresholds after baseline collection (first week in production). Use multi-window (short + long) to reduce flapping.

## Operational Guidance
- Investigate mandatory failures first; they influence authorization outcomes.
- Correlate high latency buckets with application logs around obligation handler IDs.
- If latency drives decision timeouts, consider circuit-breaking or moving heavy obligation work async (non-mandatory advice pattern).

## Extensibility
Future enhancements may add:
- Per-obligation ID labeled histogram (opt-in; beware cardinality explosion).
- Separate counters for advice vs obligation failures.
- Error classification labels (transient vs permanent) for improved SLO tracking.

## Backward Compatibility
Existing metrics remain unchanged; new methods added to the `Metrics` interface were implemented for both `Memory` and `PrometheusMetrics`. No public API removals.

## Change Log
- Added interface methods: `ObserveObligationLatency(time.Duration)`, `IncMandatoryObligationFailures()`.
- Updated `PrometheusMetrics` adapter with histogram + counter.
- PDP engine instrumentation to record latency and mandatory failure path.

---
Last updated: 2025-10-24
