---
title: Alerting Configuration and Guidelines
category: observability-alerting-guide
status: active
lastUpdated: 2025-11-12
owners: platform-eng
source: internal
refreshCadence: quarterly
---

# Alerting Guide: Capability Anchor Freshness & Integrity

This guide provides sample Prometheus alerting rules and operational recommendations for monitoring the freshness and integrity of capability anchoring.

## Key Metrics

Primary gauges now exported directly (no recording rule required for age):

Metric: `gauth_rfc0111_capability_anchor_last_write_seconds`
Type: Gauge
Semantics: Unix epoch seconds of the last successful capability anchor artifact emission (unsigned or signed wrapper).

Metric: `capability_anchor_age_seconds`
Type: Gauge
Semantics: Computed age (seconds since last successful emission) maintained by background SLA monitor. Prefer this over `time() - last_write` for accuracy and lower Prometheus evaluation cost.

Metric: `capability_anchor_stale`
Type: Gauge (0|1)
Semantics: 1 when the anchor age exceeds internal stale threshold (`stale_threshold_seconds`), else 0. Threshold value exposed via status endpoint (`/api/v1/beta/capabilities/anchor/status`).

Counters (exposed via Prometheus exposition endpoint `/api/v1/beta/capabilities/anchor/metrics/prometheus`):
- `gauth_rfc0111_capability_anchor_emitted_total` – successful emission events.
- `gauth_rfc0111_capability_anchor_skipped_total` – skipped because registry unchanged.
- `gauth_rfc0111_capability_anchor_hash_changed_total` – hash changed events (indicates registry evolution or potential tamper if unexpected).
Histogram:
- `capability_anchor_emission_interval_seconds` – distribution of intervals between successful emissions (excluding skips) for jitter / stall analysis.
Gauge:
- `capability_anchor_emission_jitter_seconds` – rolling stddev of last N (20) emission intervals.

## Why Monitor Freshness?
Stale anchoring may indicate:
- Background emission loop failure.
- File system permissions or disk full conditions preventing writes.
- Logic regressions impacting canonical hash computation.
- Cryptographic signing failures (when `GAUTH_CAP_ANCHOR_SIGN=1`).

Timely detection ensures capability registry integrity and auditability.

## Optional Recording Rules (Fallback / Derived Metrics)
Use these only if the native gauges are temporarily unavailable or for derived aggregations.
```
# Fallback age derivation (not needed if capability_anchor_age_seconds exists)
- record: gauth_capability_anchor_age_seconds
  expr: time() - gauth_rfc0111_capability_anchor_last_write_seconds

# Hourly emission rate (requires emitted counter)
- record: gauth_capability_anchor_emissions_per_hour
  expr: rate(gauth_rfc0111_capability_anchor_emitted_total[1h]) * 3600
```

## Core Alerts (Using Native Gauges)
```
# 1. Anchor Age Warning
- alert: CapabilityAnchorAgeWarning
  expr: capability_anchor_age_seconds > 300
  for: 5m
  labels:
    severity: warning
    component: capability-anchoring
  annotations:
    summary: Capability anchor freshness degrading
    description: "Capability anchor age >5m (age={{ $value }}s). Investigate emission loop or hash stability."

# 2. Anchor Stale Critical (boolean gauge preferred)
- alert: CapabilityAnchorStaleCritical
  expr: capability_anchor_stale == 1
  for: 2m
  labels:
    severity: critical
    component: capability-anchoring
  annotations:
    summary: Capability anchoring inactive
    description: "Stale threshold exceeded (stale=1) for >2m. Emission loop likely failed or blocked. Immediate investigation required."

# 3. Emission Gap (no new anchors)
- alert: CapabilityAnchorEmissionGap
  expr: increase(gauth_rfc0111_capability_anchor_emitted_total[30m]) == 0
  for: 10m
  labels:
    severity: warning
    component: capability-anchoring
  annotations:
    summary: No capability anchor emissions in last 30m
    description: "Emission counter not increased in 30m. Check background writer and registry stability."

# 4. Unexpected High Hash Change Rate (possible churn or tamper)
- alert: CapabilityAnchorHashChurnHigh
  expr: rate(gauth_rfc0111_capability_anchor_hash_changed_total[15m]) > 5
  for: 10m
  labels:
    severity: warning
    component: capability-anchoring
  annotations:
    summary: Elevated capability registry hash churn
    description: ">5 hash changes per 15m. Validate expected capability updates vs. potential tampering."

# 5. Prolonged Staleness Auto-Remediation (placeholder – implement remediation trigger)
- alert: CapabilityAnchorAutoRemediationNeeded
  expr: capability_anchor_stale == 1 and capability_anchor_age_seconds > 3600
  for: 5m
  labels:
    severity: critical
    component: capability-anchoring
  annotations:
    summary: Anchor stale >1h; remediation recommended
    description: "Anchor stale for over 1h. Consider manual trigger or automated remediation to force emission and external notarization."

# 6. High Emission Jitter (environment instability)
- alert: CapabilityAnchorHighJitter
  expr: capability_anchor_emission_jitter_seconds > (avg_over_time(capability_anchor_emission_jitter_seconds[30m]) * 2)
  for: 15m
  labels:
    severity: warning
    component: capability-anchoring
  annotations:
    summary: Elevated capability anchor emission jitter
    description: "Emission interval jitter exceeds 2x 30m average, investigate scheduling delays or IO contention."
```

## Runbook: CapabilityAnchorStaleCritical
1. Check process health: is the service running? (`/healthz`).
2. Inspect logs for anchor emission errors (search for `[cap-anchor]`).
3. Validate file path configuration: `GAUTH_CAP_ANCHOR_FILE_PATH` exists and writable.
4. Confirm interval: `GAUTH_CAP_ANCHOR_WRITE_INTERVAL` set as expected.
5. If signing enabled (`GAUTH_CAP_ANCHOR_SIGN=1`), verify EdDSA key availability via `/api/v1/beta/keys/eddsa`.
6. Manually trigger reload: `POST /api/v1/beta/capabilities/reload`; observe if `last_write` updates.
7. If hash changed unexpectedly, compare canonical registry JSON ordering and recently merged capability changes.

## External Notarization (Prototype Metrics)
When `GAUTH_CAP_ANCHOR_NOTARIZE=1` and a notarizer is configured, additional metrics appear:

Metric: `gauth_capability_anchor_notarization_latency_seconds`
Type: Histogram
Semantics: Round-trip latency of external notarization submissions. Used for SLOs (e.g., p95 < 2s). Near-zero in memory prototype.

Metric: `gauth_capability_anchor_notarized_age_seconds`
Type: Gauge
Semantics: Seconds since last successful notarization receipt. Age resets to 0 if never successful; suppress alerts until first success.

Metric: `gauth_capability_anchor_notarization_failures_total`
Type: Counter
Semantics: Cumulative failures submitting hash to external notary. Combine with lack of latency histogram updates to detect total outage.

Example Alerts:
```
- alert: CapabilityAnchorHighNotarizationLatency
  expr: histogram_quantile(0.95, sum(rate(gauth_capability_anchor_notarization_latency_seconds_bucket[5m])) by (le)) > 2
  for: 10m
  labels: { severity: warning }
  annotations:
    summary: External notarization latency elevated
    description: "p95 notarization latency >2s for 10m. Check external service health."

- alert: CapabilityAnchorStaleExternalNotarization
  expr: gauth_capability_anchor_notarized_age_seconds > 1200
  for: 5m
  labels: { severity: critical }
  annotations:
    summary: External notarization stale
    description: "No successful notarization in >20m. Investigate emission loop and notary availability."

- alert: CapabilityAnchorNotarizationFailuresSurge
  expr: increase(gauth_capability_anchor_notarization_failures_total[10m]) > 5
  labels: { severity: warning }
  annotations:
    summary: Notarization failures surge
    description: ">5 failures in 10m. Inspect logs for [notary] errors and provider credentials."
```

## Future Enhancements
- Add histogram for emission interval distribution (`capability_anchor_emission_interval_seconds`) to detect jitter / latency spikes.
- Integrate external notarization / transparency log freshness gauge (`capability_anchor_notarized_age_seconds`).
- Automated remediation action (forced emission) when prolonged stale detected.
- Fuzz/property tests for canonical hash stability.

## Dashboard Suggestions
Display panels:
- Last anchor write (RFC3339).
- Anchor age (seconds & sparkline for last 24h).
- Emissions per hour (once counter exposed).
- Hash change events per day.
- Signing status (public key presence, signature verification health).

---
Maintainer: Capability Governance Observability
Version: 0.1.0
