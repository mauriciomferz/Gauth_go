---
title: Gauthplus Grafana Dashboard Guide
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth+ Grafana Dashboard Guide

## Overview

This guide provides instructions for setting up, using, and customizing the AgentAuth+ Grafana monitoring dashboard.

## Dashboard Components

### 1. Validation Metrics
- **AgentAuth+ Validations Rate**: Real-time rate of all AgentAuth+ feature validations (successor, delegation, capability, fiduciary)
- **Total Validation Rate**: Aggregated validation rate across all features
- **P95 Validation Duration**: 95th percentile latency for validations (threshold: 100ms)

### 2. Cache Performance
- **Cache Hit Rate**: Percentage of cache hits for capability and delegation caches (target: 80%+)
- **Cache Size**: Current number of entries in each cache type

### 3. Policy & Violations
- **Policy Violations**: Count of policy violations by type and severity over the last hour
- **Fiduciary Violations**: Fiduciary duty violations by duty type and severity

### 4. Operational Metrics
- **Successor Activations**: Rate of AI successor takeovers (5-minute window)
- **P95 Delegation Depth**: 95th percentile delegation chain depth (warning threshold: 5)
- **Dual Control Approvals**: Approval/rejection counts for dual control actions
- **Agent Capability Levels**: Table showing current capability level for each AI agent

### 5. Performance Analysis
- **Validation Duration Percentiles**: P50, P95, and P99 latencies by feature (color-coded)

## Setup Instructions

### Step 1: Start the Stack

```bash
cd /Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/deployments/docker

# Start all services including Grafana
docker compose up -d
```

### Step 2: Access Grafana

1. Open browser: `http://localhost:3000`
2. Default credentials:
   - Username: `admin`
   - Password: `admin`
3. Change password on first login (optional in dev)

### Step 3: Verify Dashboard

The AgentAuth+ Monitoring Dashboard should be automatically provisioned:

1. Navigate to **Dashboards** → **Browse**
2. Look for folder: **AgentAuth+**
3. Open: **AgentAuth+ Monitoring Dashboard**

### Step 4: Verify Data Source

1. Navigate to **Configuration** → **Data Sources**
2. Verify **Prometheus** is listed and working
3. Test connection: Should show "Data source is working"

## Using the Dashboard

### Monitoring Cache Performance

**Objective**: Ensure cache hit rate stays above 70% for optimal performance.

**Key Metrics**:
- **Cache Hit Rate** panel: Shows hit rate by cache type (capability, delegation)
- **Cache Size** panel: Monitor cache growth

**Actions**:
- If hit rate drops below 70%: Alert triggered, consider increasing TTL
- If cache size grows unexpectedly: Check for memory leaks or adjust cleanup interval

### Detecting Performance Issues

**Objective**: Identify validation latency problems early.

**Key Metrics**:
- **P95 Validation Duration** gauge: Real-time latency
- **Validation Duration Percentiles** graph: Trend analysis

**Thresholds**:
- 🟢 Green: < 50ms (excellent)
- 🟡 Yellow: 50-100ms (acceptable)
- 🔴 Red: > 100ms (investigate)

**Actions**:
- Check cache hit rates if latency increases
- Review database query performance
- Consider scaling services if sustained high latency

### Tracking Policy Violations

**Objective**: Monitor policy violations and respond to critical issues.

**Key Metrics**:
- **Policy Violations** panel: Shows violations by type/severity
- **Fiduciary Violations** panel: Fiduciary duty breaches

**Alert Levels**:
- **Warning**: > 1 violation/second for 5 minutes
- **Critical**: Any critical-severity fiduciary violations

**Actions**:
- Investigate patterns in violation types
- Review policy configurations if violations spike
- Audit affected agents/actions

### Monitoring Delegation Chains

**Objective**: Detect excessive delegation depths that may indicate issues.

**Key Metrics**:
- **P95 Delegation Depth** gauge: Current depth distribution
- Threshold: > 5 levels triggers warning

**Actions**:
- Review delegation configurations if depth consistently high
- Identify agents creating deep chains
- Consider policy adjustments

### Analyzing Dual Control Operations

**Objective**: Track approval/rejection rates for critical actions.

**Key Metrics**:
- **Dual Control Approvals** panel: Status breakdown by action type

**Actions**:
- Investigate high rejection rates (> 20%)
- Review approval workflows
- Identify problematic action types

## Alerts

The dashboard integrates with Prometheus AlertManager for proactive monitoring.

### Configured Alerts

1. **AgentAuthPlusHighValidationFailureRate**
   - Trigger: > 10% validation failures for 5 minutes
   - Severity: Warning
   - Action: Check logs for error patterns

2. **AgentAuthPlusCacheHitRateLow**
   - Trigger: < 70% hit rate for 10 minutes
   - Severity: Warning
   - Action: Increase TTL or review cache invalidation

3. **AgentAuthPlusHighPolicyViolationRate**
   - Trigger: > 1 violation/second for 5 minutes
   - Severity: Critical
   - Action: Immediate investigation required

4. **AgentAuthPlusHighValidationLatency**
   - Trigger: P95 > 100ms for 5 minutes
   - Severity: Warning
   - Action: Check cache and database performance

5. **AgentAuthPlusExcessiveDelegationDepth**
   - Trigger: P95 depth > 5 for 5 minutes
   - Severity: Warning
   - Action: Review delegation policies

6. **AgentAuthPlusFrequentSuccessorActivations**
   - Trigger: > 0.1 activations/second for 10 minutes
   - Severity: Warning
   - Action: Investigate agent incapacitations

7. **AgentAuthPlusCriticalFiduciaryViolations**
   - Trigger: Any critical fiduciary violation
   - Severity: Critical
   - Action: Immediate review and remediation

8. **AgentAuthPlusDualControlFailures**
   - Trigger: > 20% rejection rate for 5 minutes
   - Severity: Warning
   - Action: Review approval workflow

9. **AgentAuthPlusServiceDown**
   - Trigger: Service unavailable for 2 minutes
   - Severity: Critical
   - Action: Check service health and logs

10. **AgentAuthPlusCacheSizeExcessive**
    - Trigger: Cache > 50,000 entries for 10 minutes
    - Severity: Warning
    - Action: Review cache configuration and cleanup

### Viewing Alerts

1. Navigate to **Alerting** → **Alert Rules**
2. Filter by: `gauthplus`
3. View active/firing alerts
4. Check AlertManager: `http://localhost:9093`

## Customization

### Adding New Panels

1. Click **Add Panel** (top right)
2. Select **Add a new panel**
3. Choose visualization type
4. Configure query:
   ```promql
   # Example: Custom metric
   rate(gauthplus_custom_metric[5m])
   ```
5. Set thresholds, units, and legend
6. Click **Apply** to save

### Modifying Queries

All panels use PromQL (Prometheus Query Language):

**Common Query Patterns**:

```promql
# Rate of change over time
rate(metric_name[5m])

# Histogram quantiles
histogram_quantile(0.95, rate(metric_bucket[5m]))

# Sum across labels
sum(rate(metric_name[5m])) by (label_name)

# Ratio calculation
rate(numerator[5m]) / rate(denominator[5m])
```

### Adjusting Time Ranges

- Default: Last 1 hour
- Click time picker (top right) to change
- Options: 5m, 15m, 1h, 6h, 24h, 7d, 30d, custom
- Auto-refresh: 10 seconds (configurable)

### Creating Alert Rules

1. Navigate to **Alerting** → **Alert Rules**
2. Click **New Alert Rule**
3. Configure query and threshold
4. Set evaluation interval
5. Choose notification channel
6. Save and test

## Prometheus Queries Reference

### Cache Performance
```promql
# Hit rate by cache type
rate(gauthplus_cache_hits_total[5m]) / 
(rate(gauthplus_cache_hits_total[5m]) + rate(gauthplus_cache_misses_total[5m]))

# Cache operations per second
rate(gauthplus_cache_hits_total[5m]) + rate(gauthplus_cache_misses_total[5m])

# Current cache size
gauthplus_cache_size
```

### Validation Metrics
```promql
# Validation rate by feature
rate(gauthplus_validations_total[5m])

# Success rate
rate(gauthplus_validations_total{result="success"}[5m]) /
rate(gauthplus_validations_total[5m])

# P95 validation duration
histogram_quantile(0.95, rate(gauthplus_validation_duration_seconds_bucket[5m]))
```

### Policy & Violations
```promql
# Violations by severity
sum(rate(gauthplus_policy_violations_total[5m])) by (severity)

# Critical violations only
rate(gauthplus_policy_violations_total{severity="critical"}[5m])
```

### Operational
```promql
# Successor activation rate
rate(gauthplus_successor_activations_total[5m])

# Average delegation depth
rate(gauthplus_delegation_depth_sum[5m]) / 
rate(gauthplus_delegation_depth_count[5m])

# Dual control rejection rate
rate(gauthplus_dual_control_approvals_total{status="rejected"}[5m]) /
rate(gauthplus_dual_control_approvals_total[5m])
```

## Troubleshooting

### Dashboard Not Loading

**Symptoms**: Grafana starts but dashboard missing

**Solutions**:
1. Check provisioning logs:
   ```bash
   docker compose logs grafana | grep provisioning
   ```

2. Verify files exist:
   ```bash
   ls -la deployments/docker/monitoring/grafana/provisioning/
   ls -la deployments/docker/monitoring/grafana/dashboards/
   ```

3. Manually import dashboard:
   - Copy content of `gauthplus-monitoring.json`
   - Navigate to **Dashboards** → **Import**
   - Paste JSON and click **Load**

### No Data Showing

**Symptoms**: Dashboard loads but panels show "No data"

**Solutions**:
1. Verify AgentAuth service is running and exposing metrics:
   ```bash
   curl http://localhost:8080/metrics | grep gauthplus
   ```

2. Check Prometheus is scraping:
   - Navigate to Prometheus: `http://localhost:9090`
   - Go to **Status** → **Targets**
   - Verify `gauth-service` target is UP

3. Test query in Prometheus:
   - Navigate to **Graph** tab
   - Enter query: `gauthplus_validations_total`
   - Click **Execute**
   - Should see data if available

4. Check Prometheus logs:
   ```bash
   docker compose logs prometheus | grep error
   ```

### Incorrect Metrics Values

**Symptoms**: Metrics showing unexpected values

**Solutions**:
1. Verify metric labels match queries:
   ```bash
   curl -s http://localhost:8080/metrics | grep 'gauthplus_cache_hits{cache_type'
   ```

2. Check scrape interval alignment:
   - Prometheus scrape: 15s
   - Dashboard refresh: 10s
   - Query range: [5m]

3. Review rate calculation windows:
   ```promql
   # Use appropriate range for rate
   rate(metric[1m])  # Short-term, sensitive
   rate(metric[5m])  # Medium-term, balanced
   rate(metric[15m]) # Long-term, smooth
   ```

### High Memory Usage

**Symptoms**: Grafana or Prometheus using excessive memory

**Solutions**:
1. Reduce retention time in `prometheus.yml`:
   ```yaml
   --storage.tsdb.retention.time=7d  # Reduce from 200h
   ```

2. Limit dashboard query ranges:
   - Use shorter time windows
   - Reduce auto-refresh frequency

3. Monitor Prometheus storage:
   ```bash
   docker compose exec prometheus du -sh /prometheus
   ```

## Performance Best Practices

### Query Optimization

1. **Use appropriate rate windows**:
   - Short queries (< 1h): `[1m]` or `[5m]`
   - Long queries (> 1h): `[15m]` or `[30m]`

2. **Limit label cardinality**:
   - Avoid high-cardinality labels (e.g., unique IDs)
   - Use `agent_id` sparingly in aggregations

3. **Use recording rules for expensive queries**:
   ```yaml
   # prometheus.yml
   groups:
     - name: gauthplus_recordings
       interval: 30s
       rules:
         - record: gauthplus:cache_hit_rate:5m
           expr: rate(gauthplus_cache_hits_total[5m]) / 
                 (rate(gauthplus_cache_hits_total[5m]) + 
                  rate(gauthplus_cache_misses_total[5m]))
   ```

### Dashboard Efficiency

1. **Limit panel queries**:
   - Max 3-5 queries per panel
   - Use variables for dynamic filtering

2. **Set appropriate refresh intervals**:
   - Real-time monitoring: 5-10s
   - Historical analysis: 1-5m

3. **Use query caching**:
   - Grafana caches query results
   - Consistent time ranges improve cache hits

## Integration with AgentAuth+ Services

### Enabling Metrics in AgentAuth+

Metrics are automatically collected when the service starts. No additional configuration needed.

**Verify metrics are exposed**:
```bash
# Check metrics endpoint
curl http://localhost:8080/metrics | head -20

# Filter AgentAuth+ metrics
curl http://localhost:8080/metrics | grep gauthplus_ | head -10
```

**Expected output**:
```
# HELP gauthplus_validations_total Total number of AgentAuth+ validations performed
# TYPE gauthplus_validations_total counter
gauthplus_validations_total{feature="successor",result="success"} 42
gauthplus_validations_total{feature="delegation",result="success"} 128
...
```

### Cache Integration

The caching layer automatically records metrics:

```go
// Automatic metric recording in cache.go
metrics.RecordAgentAuthPlusCacheOperation("capability", hit)
metrics.UpdateAgentAuthPlusCacheSize("capability", len(c.cache))
```

No manual instrumentation required in application code.

### Validation Timing

Validation methods automatically track duration:

```go
// Automatic timing in gauthplus_integration.go
start := time.Now()
defer func() {
    metrics.RecordAgentAuthPlusValidation("successor", "checked", time.Since(start).Seconds())
}()
```

## Export and Sharing

### Export Dashboard JSON

1. Navigate to dashboard
2. Click **Dashboard settings** (gear icon)
3. Select **JSON Model** tab
4. Copy JSON
5. Save to file or share

### Create Dashboard Snapshots

1. Click **Share** icon (top right)
2. Select **Snapshot** tab
3. Set expiration time
4. Click **Publish to snapshot.raintank.io**
5. Share generated URL

### Import to Another Grafana Instance

1. Export dashboard JSON
2. On target instance: **Dashboards** → **Import**
3. Paste JSON or upload file
4. Select Prometheus data source
5. Click **Import**

## Production Considerations

### Security

1. **Change default credentials**:
   ```bash
   # Set via environment variables
   export GF_SECURITY_ADMIN_PASSWORD=secure-password
   docker compose up -d grafana
   ```

2. **Enable HTTPS**:
   - Use reverse proxy (nginx, Traefik)
   - Configure TLS certificates
   - Update `docker-compose.yml` ports

3. **Restrict access**:
   - Configure authentication (LDAP, OAuth)
   - Set up role-based access control
   - Limit dashboard editing permissions

### High Availability

1. **External database for Grafana**:
   ```yaml
   # docker-compose.yml
   environment:
     GF_DATABASE_TYPE: postgres
     GF_DATABASE_HOST: postgres:5432
     GF_DATABASE_NAME: grafana
   ```

2. **Persistent storage**:
   ```yaml
   volumes:
     - grafana_data:/var/lib/grafana
     - ./grafana-backups:/var/lib/grafana/backups
   ```

3. **Load balancing**:
   - Run multiple Grafana instances
   - Use load balancer (HAProxy, nginx)
   - Share configuration via provisioning

### Monitoring Grafana Itself

Add Grafana self-monitoring:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'grafana'
    static_configs:
      - targets: ['grafana:3000']
    metrics_path: '/metrics'
```

**Key Grafana metrics**:
- `grafana_stat_totals_dashboard`
- `grafana_api_response_status_total`
- `grafana_datasource_request_duration_seconds`

## Additional Resources

- **Prometheus Documentation**: https://prometheus.io/docs/
- **Grafana Documentation**: https://grafana.com/docs/
- **PromQL Guide**: https://prometheus.io/docs/prometheus/latest/querying/basics/
- **AgentAuth+ Implementation**: See `GAUTHPLUS_CACHING_IMPLEMENTATION.md`
- **Metrics Reference**: See `pkg/metrics/prometheus.go`

## Quick Reference

### Access URLs (Docker Compose)
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090
- **AlertManager**: http://localhost:9093
- **AgentAuth Metrics**: http://localhost:8080/metrics

### Default Credentials
- **Grafana**: admin / admin
- **Prometheus**: No authentication
- **AlertManager**: No authentication

### Key File Locations
- **Dashboard**: `deployments/docker/monitoring/grafana/dashboards/gauthplus-monitoring.json`
- **Datasource**: `deployments/docker/monitoring/grafana/provisioning/datasources/prometheus.yml`
- **Prometheus Config**: `deployments/docker/monitoring/prometheus.yml`
- **Alert Rules**: `deployments/docker/monitoring/alert-rules.yml`
- **AlertManager Config**: `deployments/docker/monitoring/alertmanager.yml`

### Useful Commands
```bash
# Start monitoring stack
docker compose up -d

# View logs
docker compose logs -f grafana
docker compose logs -f prometheus

# Restart services
docker compose restart grafana prometheus

# Check metrics
curl http://localhost:8080/metrics | grep gauthplus

# Validate Prometheus config
docker compose exec prometheus promtool check config /etc/prometheus/prometheus.yml
```

---

**Last Updated**: November 26, 2025  
**Dashboard Version**: 1.0  
**Grafana Version**: Latest (9.0+)  
**Prometheus Version**: Latest
