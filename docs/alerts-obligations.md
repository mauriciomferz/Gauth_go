---
title: Alerts-Obligations
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Obligation Metrics Alerting Guidance

This document proposes initial Prometheus / Alertmanager rules for the obligation execution metrics:
- `agentauth_aap001_obligation_latency_seconds` (histogram)
- `agentauth_aap001_obligations_executed_total` (counter)
- `agentauth_aap001_obligations_failed_total` (counter)
- `agentauth_aap001_mandatory_obligation_failures_total` (counter)

## Design Principles
1. Mandatory failures are rare and high impact — alert immediately.
2. Latency alerts use percentile + multi-window to reduce noise.
3. Failure ratio compared to executions gives stability insight.
4. Provide staging (Warning / Critical) thresholds to enable gradual tuning.

## Service Level Indicators (SLIs)
| SLI | Definition | Goal (Initial) |
|-----|------------|----------------|
| Obligation p95 latency | 95th percentile over 15m window | < 25ms |
| Mandatory failure occurrences | Any increase over 30m | 0 |
| Obligation failure ratio | failed / executed (30m) | < 5% |
| Obligation p99 latency | 99th percentile over 1h window | < 50ms |

## PromQL Building Blocks
```promql
# p95 obligation latency (15m)
histogram_quantile(0.95, sum by (le) (rate(agentauth_aap001_obligation_latency_seconds_bucket[15m])))

# p99 obligation latency (1h)
histogram_quantile(0.99, sum by (le) (rate(agentauth_aap001_obligation_latency_seconds_bucket[1h])))

# Mandatory failures in last 30m
increase(agentauth_aap001_mandatory_obligation_failures_total[30m])

# Failure ratio (30m)
sum(rate(agentauth_aap001_obligations_failed_total[30m]))
  /
  sum(rate(agentauth_aap001_obligations_executed_total[30m]))
```

## Suggested Alert Rules (YAML Snippets)
```yaml
# obligations_alerts.yaml
groups:
  - name: agentauth_obligations
    rules:
      - alert: AgentAuthObligationLatencyHighP95Warning
        expr: histogram_quantile(0.95, sum by (le) (rate(agentauth_aap001_obligation_latency_seconds_bucket[15m])) > 0.025
        for: 10m
        labels:
          severity: warning
          component: agentauth_pdp
        annotations:
          summary: "Obligation latency p95 elevated (>25ms)"
          description: |
            p95 obligation execution latency exceeded 25ms for 10m.
            Investigate slow obligation handlers or downstream services.

      - alert: AgentAuthObligationLatencyHighP95Critical
        expr: histogram_quantile(0.95, sum by (le) (rate(agentauth_aap001_obligation_latency_seconds_bucket[15m])) > 0.040
        for: 5m
        labels:
          severity: critical
          component: agentauth_pdp
        annotations:
          summary: "Obligation latency p95 critical (>40ms)"
          description: "Sustained high latency may impact decision times or timeouts."

      - alert: AgentAuthObligationMandatoryFailure
        expr: increase(agentauth_aap001_mandatory_obligation_failures_total[30m]) > 0
        for: 1m
        labels:
          severity: critical
          component: agentauth_pdp
        annotations:
          summary: "Mandatory obligation failure detected"
          description: |
            At least one mandatory obligation failed and flipped an allow decision to deny in the last 30 minutes.
            Examine obligation audit logs and handler errors immediately.

      - alert: AgentAuthObligationFailureRatioWarning
        expr: (sum(rate(agentauth_aap001_obligations_failed_total[30m]) / sum(rate(agentauth_aap001_obligations_executed_total[30m])) > 0.05
        for: 15m
        labels:
          severity: warning
          component: agentauth_pdp
        annotations:
          summary: "Obligation failure ratio >5%"
          description: |
            Elevated obligation failures. Check recent deployment changes or dependency health.

      - alert: AgentAuthObligationFailureRatioCritical
        expr: (sum(rate(agentauth_aap001_obligations_failed_total[30m]) / sum(rate(agentauth_aap001_obligations_executed_total[30m])) > 0.10
        for: 10m
        labels:
          severity: critical
          component: agentauth_pdp
        annotations:
          summary: "Obligation failure ratio >10%"
          description: "High failure rate — obligations may not be completing reliably."

      - alert: AgentAuthObligationLatencyHighP99
        expr: histogram_quantile(0.99, sum by (le) (rate(agentauth_aap001_obligation_latency_seconds_bucket[1h])) > 0.050
        for: 15m
        labels:
          severity: warning
          component: agentauth_pdp
        annotations:
          summary: "p99 obligation latency >50ms"
          description: "Long-tail latency degradation detected — assess heavy handlers or outliers."
```

## Tuning Guidance
| Area | Adjustment Strategy |
|------|---------------------|
| Latency thresholds | Start conservative (25ms p95) — refine after baseline week. If median <2ms, consider tightening. |
| Failure ratio | If normal failure ratio <1%, reduce warning threshold to 3%. |
| Mandatory failures | If expected during controlled chaos tests, add label filter (e.g. environment="prod") to rule. |
| Windows | Use shorter window (5m) for fast detection paired with longer window (30m) dashboards to confirm sustained trend. |

## Runbook Starters
| Alert | Immediate Checks |
|-------|------------------|
| Mandatory Failure | Inspect audit file (`obligations` JSONL) for failed obligation IDs; correlate with deployment logs. |
| High Latency | Identify top 5 slow obligations via audit `duration_ms`; verify downstream service health & network. |
| Failure Ratio | Classify failures (transient vs logic); roll back recent handler changes if spike correlates with deploy. |

## Dashboard Suggestions
1. Time-series: p50/p95/p99 obligation latency.
2. Counters: mandatory failures (singlestat) with 24h sparkline.
3. Ratio panel: failed vs executed obligations (stacked area).
4. Top obligation IDs by average latency (requires future labeled histogram or audit aggregation).

## Future Enhancements
- Label obligations with `obligation_id` in histogram (guard cardinality via allowlist).
- Separate counters for advice vs obligation to isolate critical paths.
- Error classification labeling (e.g., `type="timeout"`, `type="dependency"`).

## FAQ
**Q: Why histogram over summary?**
Histogram enables global aggregation (multi-instance PDP) and percentile calculation from raw buckets.

**Q: Why multi-window alerts?**
Short windows catch fast regressions; longer windows inform tuning and reduce false positives.

**Q: Can mandatory failures alert be silenced for known maintenance?**
Yes, route the alert through Alertmanager matching labels (e.g. `severity=critical, alertname=AgentAuthObligationMandatoryFailure`).

---
Last updated: 2025-10-24
