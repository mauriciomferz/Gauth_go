# PDP/PEP Metrics Export - Implementation Guide

## Overview

This document describes the Prometheus metrics export implementation for the Power Decision Point (PDP) and Policy Enforcement Point (PEP) audit logging system.

## Exported Metrics

### Enforcement Counters

**Metric:** `gauth_rfc0111_pep_enforcements_total`  
**Type:** Counter (labeled)  
**Labels:**
- `allowed`: `"true"` or `"false"` - Whether the policy enforcement allowed the action
- `action_type`: Action type (e.g., `"read"`, `"write"`, `"delete"`, `"transaction"`)

**Description:** Total count of policy enforcement decisions, labeled by outcome and action type.

**Example Queries:**
```promql
# Total enforcements (all types)
sum(rate(gauth_rfc0111_pep_enforcements_total[5m]))

# Denied actions rate
sum(rate(gauth_rfc0111_pep_enforcements_total{allowed="false"}[5m]))

# Enforcement allow/deny ratio
sum(rate(gauth_rfc0111_pep_enforcements_total{allowed="true"}[5m])) / 
sum(rate(gauth_rfc0111_pep_enforcements_total[5m]))

# Per-action enforcement rates
sum(rate(gauth_rfc0111_pep_enforcements_total[5m])) by (action_type)
```

---

### Violation Counters

**Metric:** `gauth_rfc0111_pep_violations_total`  
**Type:** Counter (labeled)  
**Labels:**
- `violation_type`: Type of violation (e.g., `"policy_violation"`, `"resource_access_denied"`, `"rate_limit_exceeded"`)
- `severity`: Severity level (`"critical"`, `"high"`, `"medium"`, `"low"`)

**Description:** Total count of policy violations, labeled by type and severity.

**Example Queries:**
```promql
# Total violations rate
sum(rate(gauth_rfc0111_pep_violations_total[5m]))

# Critical violations rate (alerting threshold)
sum(rate(gauth_rfc0111_pep_violations_total{severity="critical"}[5m]))

# High-severity violations by type
sum(rate(gauth_rfc0111_pep_violations_total{severity=~"critical|high"}[5m])) by (violation_type)

# Violation severity distribution
sum(rate(gauth_rfc0111_pep_violations_total[5m])) by (severity)
```

---

### Enforcement Latency

**Metric:** `gauth_rfc0111_pep_enforcement_latency_seconds`  
**Type:** Histogram  
**Buckets:** Default latency buckets (0.001s to 10s)

**Description:** Histogram of policy enforcement decision latencies.

**Example Queries:**
```promql
# P50, P95, P99 enforcement latency
histogram_quantile(0.50, rate(gauth_rfc0111_pep_enforcement_latency_seconds_bucket[5m]))
histogram_quantile(0.95, rate(gauth_rfc0111_pep_enforcement_latency_seconds_bucket[5m]))
histogram_quantile(0.99, rate(gauth_rfc0111_pep_enforcement_latency_seconds_bucket[5m]))

# Average enforcement latency
rate(gauth_rfc0111_pep_enforcement_latency_seconds_sum[5m]) / 
rate(gauth_rfc0111_pep_enforcement_latency_seconds_count[5m]))
```

---

### Audit Buffer Gauges

**Metrics:**
- `gauth_rfc0111_pep_audit_buffer_enforcements`  
- `gauth_rfc0111_pep_audit_buffer_violations`

**Type:** Gauge  
**Description:** Current number of entries in the audit buffer (FIFO rotation).

**Example Queries:**
```promql
# Buffer utilization percentage (assuming 10k max)
(gauth_rfc0111_pep_audit_buffer_enforcements / 10000) * 100
(gauth_rfc0111_pep_audit_buffer_violations / 10000) * 100

# Total audit entries in memory
gauth_rfc0111_pep_audit_buffer_enforcements + gauth_rfc0111_pep_audit_buffer_violations
```

---

## Alerting Rules

### Critical Violations Alert

```yaml
groups:
  - name: pdp_violations
    rules:
      - alert: HighCriticalViolationRate
        expr: |
          sum(rate(gauth_rfc0111_pep_violations_total{severity="critical"}[5m])) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High rate of critical policy violations"
          description: "Critical violations rate is {{ $value }} per second (threshold: 0.1/s)"
```

### High Denial Rate Alert

```yaml
- alert: HighEnforcementDenialRate
  expr: |
    sum(rate(gauth_rfc0111_pep_enforcements_total{allowed="false"}[5m])) / 
    sum(rate(gauth_rfc0111_pep_enforcements_total[5m])) > 0.5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High policy enforcement denial rate"
    description: "{{ $value | humanizePercentage }} of enforcements are denials (threshold: 50%)"
```

### Enforcement Latency Alert

```yaml
- alert: HighEnforcementLatency
  expr: |
    histogram_quantile(0.95, rate(gauth_rfc0111_pep_enforcement_latency_seconds_bucket[5m])) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High PEP enforcement latency"
    description: "P95 enforcement latency is {{ $value }}s (threshold: 100ms)"
```

### Buffer Overflow Risk Alert

```yaml
- alert: AuditBufferNearFull
  expr: |
    (gauth_rfc0111_pep_audit_buffer_enforcements > 9000) or
    (gauth_rfc0111_pep_audit_buffer_violations > 9000)
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Audit buffer approaching capacity"
    description: "Audit buffer is at {{ $value }} entries (threshold: 9000/10000)"
```

---

## Integration Example

### Basic Setup

```go
package main

import (
    "github.com/.../internal/metrics"
    "github.com/.../pkg/gauth"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

func main() {
    // Create Prometheus registry
    registry := prometheus.NewRegistry()
    
    // Create metrics collector
    promMetrics := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{
        Namespace: "gauth",
        Subsystem: "rfc0111",
        Registry:  registry,
    })
    
    // Create PDP audit logger with metrics
    auditLogger := gauth.NewProductionPEPAuditLogger(
        10000,  // maxEntries
        true,   // enableConsole
        true,   // enableMetrics
    )
    auditLogger.SetMetrics(promMetrics)
    
    // Expose metrics endpoint
    http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
    http.ListenAndServe(":9090", nil)
}
```

### With SimplePDP

```go
// Create SimplePDP with audit logger
pdp := gauth.NewSimplePDP(
    policyStore,
    gauth.WithPDPAuditLogger(auditLogger),
)

// Enforce policies
decision, err := pdp.Enforce(ctx, &gauth.EnforcementRequest{
    Action:     "write",
    Resource:   "resource-123",
    Attributes: map[string]interface{}{"user": "alice"},
})
// Metrics automatically exported
```

---

## Grafana Dashboard

### Panel: Enforcement Rate

```json
{
  "title": "Policy Enforcement Rate",
  "targets": [
    {
      "expr": "sum(rate(gauth_rfc0111_pep_enforcements_total{allowed=\"true\"}[5m]))",
      "legendFormat": "Allowed"
    },
    {
      "expr": "sum(rate(gauth_rfc0111_pep_enforcements_total{allowed=\"false\"}[5m]))",
      "legendFormat": "Denied"
    }
  ],
  "yaxes": [{"format": "reqps"}]
}
```

### Panel: Violation Severity Heatmap

```json
{
  "title": "Violations by Severity",
  "targets": [
    {
      "expr": "sum(rate(gauth_rfc0111_pep_violations_total[5m])) by (severity)",
      "legendFormat": "{{ severity }}"
    }
  ],
  "type": "graph",
  "stack": true
}
```

### Panel: Enforcement Latency Percentiles

```json
{
  "title": "Enforcement Latency (P50/P95/P99)",
  "targets": [
    {
      "expr": "histogram_quantile(0.50, rate(gauth_rfc0111_pep_enforcement_latency_seconds_bucket[5m]))",
      "legendFormat": "P50"
    },
    {
      "expr": "histogram_quantile(0.95, rate(gauth_rfc0111_pep_enforcement_latency_seconds_bucket[5m]))",
      "legendFormat": "P95"
    },
    {
      "expr": "histogram_quantile(0.99, rate(gauth_rfc0111_pep_enforcement_latency_seconds_bucket[5m]))",
      "legendFormat": "P99"
    }
  ],
  "yaxes": [{"format": "s"}]
}
```

---

## Performance Considerations

### Cardinality Management

- **Enforcement counters**: 2 × N action types (low cardinality)
- **Violation counters**: M violation types × 4 severities (medium cardinality)
- **Latency histogram**: ~13 buckets (fixed cardinality)

**Estimated series:** ~50-100 time series for typical deployments

### Memory Overhead

- **Per counter**: ~2KB
- **Per histogram**: ~30KB
- **Total overhead**: ~5-10MB for all PDP metrics

### Scraping Recommendations

- **Scrape interval**: 15s (default)
- **Evaluation interval**: 15s
- **Retention**: 15d minimum for trend analysis

---

## Troubleshooting

### Metrics Not Appearing

1. Verify metrics are enabled:
   ```go
   logger := NewProductionPEPAuditLogger(10000, true, true) // enableMetrics = true
   ```

2. Ensure metrics collector is set:
   ```go
   logger.SetMetrics(promMetrics)
   ```

3. Check Prometheus scrape config:
   ```yaml
   scrape_configs:
     - job_name: 'gauth'
       static_configs:
         - targets: ['localhost:9090']
   ```

### High Cardinality Warning

If you see excessive cardinality:
- Limit action types to standard set (read/write/delete/transaction)
- Normalize violation types to prevent unbounded growth
- Consider aggregating low-frequency action types into "other"

### Missing Labels

Verify labels are being set correctly:
```promql
# Check label values
count(gauth_rfc0111_pep_enforcements_total) by (allowed, action_type)
count(gauth_rfc0111_pep_violations_total) by (violation_type, severity)
```

---

## Additional Resources

- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Grafana Dashboard JSON Export](https://grafana.com/docs/grafana/latest/dashboards/export-import/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [RFC-0111 Authorization Framework](../../docs/RFC_IMPLEMENTATION_COVERAGE.md)
