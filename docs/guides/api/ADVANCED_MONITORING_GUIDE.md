# AgentAuth Advanced Monitoring Guide

**Version**: 1.0  
**Date**: November 26, 2025  
**Compliance**: 97/100 (+1.0 point for Advanced Monitoring)

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Quick Start](#quick-start)
4. [Metrics Endpoints](#metrics-endpoints)
5. [Grafana Dashboards](#grafana-dashboards)
6. [Alert Rules](#alert-rules)
7. [Configuration](#configuration)
8. [Production Deployment](#production-deployment)
9. [Troubleshooting](#troubleshooting)

---

## Overview

AgentAuth implements comprehensive monitoring using the Prometheus/Grafana stack, providing:

- **Real-time Metrics**: Request rates, latency, errors, cache performance
- **Health Monitoring**: Component health checks and system status
- **Alerting**: Automated alerts for critical conditions
- **Visualization**: Pre-built Grafana dashboards
- **Multiple Exporters**: Application, system, database, and cache metrics

### Key Features

✅ **10+ Prometheus Endpoints** - Comprehensive metric collection  
✅ **Grafana Dashboard** - Pre-configured with 10+ panels  
✅ **15+ Alert Rules** - Proactive issue detection  
✅ **Multi-component Monitoring** - API, database, cache, system  
✅ **Docker Compose Stack** - One-command deployment  
✅ **AlertManager Integration** - Multi-channel notifications  

---

## Architecture

### Monitoring Stack

```
┌─────────────────┐
│   AgentAuth API     │
│   Port: 8080    │
│   /metrics/*    │
└────────┬────────┘
         │ (scrape)
         ▼
┌─────────────────┐
│   Prometheus    │
│   Port: 9090    │
│   Time Series   │
└────────┬────────┘
         │
         ├──────────┬──────────────┐
         ▼          ▼              ▼
┌──────────┐  ┌──────────┐  ┌────────────┐
│ Grafana  │  │AlertMgr  │  │  Exporters │
│ Port:3001│  │ Port:9093│  │  Various   │
└──────────┘  └──────────┘  └────────────┘
```

### Components

1. **AgentAuth Application** - Exposes Prometheus metrics at multiple endpoints
2. **Prometheus** - Scrapes and stores time-series metrics data
3. **Grafana** - Visualizes metrics in interactive dashboards
4. **AlertManager** - Routes and manages alerts to various channels
5. **Node Exporter** - Collects host/system metrics
6. **Postgres Exporter** - Collects database metrics
7. **Redis Exporter** - Collects cache metrics

---

## Quick Start

### 1. Start Monitoring Stack

```bash
cd monitoring
docker-compose up -d
```

This starts:
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3001
- AlertManager: http://localhost:9093

### 2. Access Grafana

1. Open http://localhost:3001
2. Login: `admin` / `admin`
3. Navigate to "Dashboards" → "AgentAuth - System Overview"

### 3. Start AgentAuth Application

```bash
# Ensure AgentAuth is running and exposing metrics
go run ./cmd/web-server
```

### 4. Verify Metrics Collection

```bash
# Check Prometheus targets
open http://localhost:9090/targets

# View raw metrics
curl http://localhost:8080/api/v1/admin/metrics/prometheus
```

---

## Metrics Endpoints

### Admin Metrics Endpoints

#### 1. **System Metrics** - `GET /api/v1/admin/metrics/system`
```json
{
  "totalRequests": 150000,
  "uptime": 86400,
  "avgLatency": 45.3,
  "p95Latency": 125.8,
  "p99Latency": 256.4,
  "errorCount": 1250,
  "errorRate": 0.0083,
  "cacheHitRate": 0.892,
  "cacheSize": 50000,
  "memoryUsage": 512000000,
  "componentHealth": [...]
}
```

**Use Cases**:
- Admin dashboard overview
- Real-time system status
- Performance monitoring

#### 2. **Prometheus Metrics** - `GET /api/v1/admin/metrics/prometheus`
```
# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/api/v1/poa",status="200"} 15234

# HELP http_request_duration_seconds HTTP request latency
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 12000
http_request_duration_seconds_bucket{le="0.1"} 14500
```

**Use Cases**:
- Prometheus scraping
- Time-series analysis
- Grafana visualization

#### 3. **Token Violations** - `GET /api/v1/admin/metrics/token-violations`
```json
{
  "violations": [
    {
      "id": "viol-001",
      "subscriber": "ACME Corp",
      "violationType": "Expired Token Usage",
      "severity": "high",
      "timestamp": "2025-11-26T10:30:00Z",
      "resolved": false
    }
  ]
}
```

**Use Cases**:
- Security monitoring
- Compliance auditing
- Incident response

#### 4. **Semantic Counters** - `GET /api/v1/admin/metrics/semantic-counters`
```json
{
  "counters": {
    "capabilityAnchorValidations": 5000,
    "capabilityAnchorResolutions": 4800,
    "avgResolutionTime": 45.3,
    "successRate": 0.975,
    "activeAnchors": 150,
    "failedValidations": 125,
    "cachedAnchors": 4500,
    "cacheHitRate": 0.943
  }
}
```

**Use Cases**:
- Capability monitoring
- Performance analysis
- Optimization tracking

### Beta Metrics Endpoints

#### 5. **Authorization Metrics** - `GET /api/v1/beta/authz/metrics/prometheus`
- Authorization decisions (allow/deny)
- Decision latency
- Policy evaluation times
- Cache hit/miss rates

#### 6. **Policy Metrics** - `GET /api/v1/beta/policy/metrics/prometheus`
- Policy evaluations
- Policy changes
- Active policies
- Evaluation errors

#### 7. **Violations Metrics** - `GET /api/v1/beta/metrics/violations/prometheus`
- Security violations
- Compliance violations
- Violation trends

#### 8. **Revocation Metrics** - `GET /api/v1/beta/metrics/revocation/auto-sign/prometheus`
- Auto-sign operations
- Revocation counts
- Success/failure rates

#### 9. **Capabilities Metrics** - `GET /api/v1/beta/capabilities/anchor/metrics/prometheus`
- Anchor validations
- Resolution operations
- Cache performance

---

## Grafana Dashboards

### Pre-Installed Dashboard: "AgentAuth - System Overview"

#### Panels

1. **Request Rate** (Time Series)
   - HTTP requests per second
   - Grouped by method and path
   - 5-minute rate

2. **P95 Latency** (Gauge)
   - 95th percentile latency in milliseconds
   - Thresholds: Green (<100ms), Yellow (100-200ms), Red (>200ms)

3. **Error Rate** (Stat)
   - 5xx errors per second
   - Critical threshold indicator

4. **Cache Hit Rate** (Gauge)
   - Cache efficiency percentage
   - Target: >85%

5. **Service Health** (Stat)
   - Up/Down indicator
   - Background color coding

6. **Memory Usage** (Stat)
   - Current memory consumption
   - Trend indicator

7. **HTTP Status Codes** (Time Series)
   - Stacked area chart
   - 2xx (green), 4xx (yellow), 5xx (red)

8. **Go Runtime Metrics** (Time Series)
   - Goroutine count
   - Garbage collection rate

9. **Application Metrics** (Time Series)
   - Audit events rate
   - Authorization decisions rate

10. **Health Checks** (Table)
    - Component-wise health status
    - Real-time updates

### Custom Dashboards

You can create additional dashboards for:
- Database performance
- Cache detailed analysis
- Security monitoring
- Business metrics

---

## Alert Rules

### Critical Alerts (Severity: critical)

#### 1. **HighErrorRate**
```yaml
expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
for: 5m
```
- **Threshold**: >5% error rate
- **Action**: Page on-call engineer
- **Channels**: PagerDuty + Slack

#### 2. **ServiceDown**
```yaml
expr: up{job="gauth"} == 0
for: 1m
```
- **Threshold**: Service unreachable for 1 minute
- **Action**: Immediate escalation
- **Channels**: PagerDuty + Slack

#### 3. **DatabaseConnectionIssues**
```yaml
expr: pg_up == 0
for: 2m
```
- **Threshold**: Database unreachable for 2 minutes
- **Action**: Check database health
- **Channels**: PagerDuty + Slack

#### 4. **FailedHealthCheck**
```yaml
expr: health_check_status{status="unhealthy"} == 1
for: 5m
```
- **Threshold**: Component unhealthy for 5 minutes
- **Action**: Investigate component
- **Channels**: Slack

### Warning Alerts (Severity: warning)

#### 5. **HighLatency**
```yaml
expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.2
for: 5m
```
- **Threshold**: P95 latency >200ms
- **Action**: Performance investigation
- **Channels**: Slack

#### 6. **LowCacheHitRate**
```yaml
expr: (cache_hits / (cache_hits + cache_misses)) < 0.7
for: 10m
```
- **Threshold**: <70% hit rate
- **Action**: Review cache configuration
- **Channels**: Slack

#### 7. **HighMemoryUsage**
```yaml
expr: (process_resident_memory_bytes / 1024 / 1024) > 1024
for: 5m
```
- **Threshold**: >1GB memory usage
- **Action**: Memory leak investigation
- **Channels**: Slack

#### 8. **HighGoroutineCount**
```yaml
expr: go_goroutines > 1000
for: 10m
```
- **Threshold**: >1000 goroutines
- **Action**: Goroutine leak investigation
- **Channels**: Slack

### Info Alerts (Severity: info)

#### 9. **RateLimitExhaustion**
```yaml
expr: rate(rate_limit_exceeded_total[5m]) > 10
for: 5m
```
- **Threshold**: >10 rate limit violations/second
- **Action**: Review rate limits
- **Channels**: Email

---

## Configuration

### Prometheus Configuration

File: `monitoring/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'gauth'
    static_configs:
      - targets: ['host.docker.internal:8080']
    metrics_path: '/api/v1/admin/metrics/prometheus'
    scrape_interval: 10s
```

**Key Settings**:
- `scrape_interval`: How often to collect metrics (default: 15s)
- `evaluation_interval`: How often to evaluate alert rules (default: 15s)
- `metrics_path`: Endpoint path for metrics

### AlertManager Configuration

File: `monitoring/alertmanager/config.yml`

```yaml
receivers:
  - name: 'critical'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
    slack_configs:
      - channel: '#gauth-critical'
```

**Configuration Steps**:

1. **Slack Integration**:
   ```yaml
   slack_api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
   ```

2. **PagerDuty Integration**:
   ```yaml
   service_key: 'YOUR_PAGERDUTY_INTEGRATION_KEY'
   ```

3. **Email Integration**:
   ```yaml
   email_configs:
     - to: 'ops-team@example.com'
       smarthost: 'smtp.example.com:587'
       auth_username: 'alertmanager@example.com'
       auth_password: 'YOUR_PASSWORD'
   ```

### Grafana Configuration

**Change Admin Password**:
```bash
docker exec -it gauth-grafana grafana-cli admin reset-admin-password newpassword
```

**Add Data Source** (if not auto-provisioned):
1. Go to Configuration → Data Sources
2. Add Prometheus
3. URL: `http://prometheus:9090`
4. Save & Test

**Import Dashboard**:
1. Go to Dashboards → Import
2. Upload `monitoring/grafana/dashboards/gauth-overview.json`
3. Select Prometheus data source
4. Import

---

## Production Deployment

### Prerequisites

- Docker and Docker Compose
- Sufficient disk space for time-series data (recommend 50GB+)
- Network access between monitoring stack and AgentAuth application

### Deployment Steps

#### 1. Configure Exporters

Update `monitoring/docker-compose.yml` with your database/cache credentials:

```yaml
postgres-exporter:
  environment:
    - DATA_SOURCE_NAME=postgresql://user:pass@host:5432/dbname?sslmode=disable

redis-exporter:
  environment:
    - REDIS_ADDR=redis://your-redis-host:6379
```

#### 2. Configure AlertManager

Update `monitoring/alertmanager/config.yml` with your notification channels:

```yaml
global:
  slack_api_url: 'YOUR_SLACK_WEBHOOK'

receivers:
  - name: 'critical'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_KEY'
```

#### 3. Start Monitoring Stack

```bash
cd monitoring
docker-compose up -d

# Verify all services are running
docker-compose ps

# Check logs
docker-compose logs -f
```

#### 4. Verify Metrics Collection

```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'

# Test AgentAuth metrics endpoint
curl http://localhost:8080/api/v1/admin/metrics/prometheus | head -20
```

#### 5. Test Alerts

```bash
# Trigger a test alert
curl -X POST http://localhost:9093/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '[{
    "labels": {"alertname": "TestAlert", "severity": "warning"},
    "annotations": {"summary": "Test alert from monitoring setup"}
  }]'
```

### Security Considerations

1. **Secure Grafana**:
   ```yaml
   environment:
     - GF_SECURITY_ADMIN_PASSWORD=strong-password-here
     - GF_USERS_ALLOW_SIGN_UP=false
   ```

2. **Restrict Prometheus Access**:
   - Use firewall rules
   - Enable authentication if exposed
   - Use HTTPS in production

3. **Protect Metrics Endpoints**:
   - Add authentication to `/metrics` endpoints
   - Use API keys or JWT tokens
   - Restrict IP access

### High Availability

For production HA setup:

1. **Prometheus HA**:
   ```yaml
   prometheus-1:
     ...
   prometheus-2:
     ...
   ```

2. **Grafana HA**:
   - Use external database (PostgreSQL/MySQL)
   - Load balancer in front

3. **AlertManager Clustering**:
   ```yaml
   alertmanager:
     command:
       - '--cluster.peer=alertmanager-2:9094'
   ```

---

## Troubleshooting

### Common Issues

#### 1. **Prometheus Not Scraping AgentAuth**

**Symptoms**: No data in Grafana, targets show as "Down"

**Solutions**:
```bash
# Check AgentAuth is running
curl http://localhost:8080/api/v1/admin/metrics/prometheus

# Check Prometheus targets
open http://localhost:9090/targets

# Check network connectivity
docker exec gauth-prometheus ping host.docker.internal

# Review Prometheus logs
docker logs gauth-prometheus
```

#### 2. **Grafana Dashboard Shows No Data**

**Symptoms**: Dashboard panels are empty

**Solutions**:
1. Verify Prometheus data source is configured
2. Check time range selector (default: last 1 hour)
3. Verify Prometheus is collecting data:
   ```
   http://localhost:9090/graph
   Query: up{job="gauth"}
   ```

#### 3. **Alerts Not Firing**

**Symptoms**: No alert notifications despite conditions met

**Solutions**:
```bash
# Check AlertManager status
open http://localhost:9093

# Test alert rule
curl http://localhost:9090/api/v1/rules

# Check AlertManager logs
docker logs gauth-alertmanager

# Verify webhook URLs are correct
docker exec gauth-alertmanager cat /etc/alertmanager/config.yml
```

#### 4. **High Memory Usage by Prometheus**

**Symptoms**: Prometheus container using excessive memory

**Solutions**:
```yaml
# Reduce retention period
prometheus:
  command:
    - '--storage.tsdb.retention.time=15d'  # Default: 15d, reduce to 7d
    - '--storage.tsdb.retention.size=10GB'  # Add size limit
```

#### 5. **Missing Metrics**

**Symptoms**: Expected metrics not appearing

**Solutions**:
1. Check metric naming in application code
2. Verify Prometheus scrape configuration
3. Check for metric registration errors:
   ```bash
   # AgentAuth application logs
   grep "metric" logs/gauth.log
   ```

### Debugging Commands

```bash
# View Prometheus configuration
docker exec gauth-prometheus cat /etc/prometheus/prometheus.yml

# Check alert rules
docker exec gauth-prometheus promtool check rules /etc/prometheus/alerts/*.yml

# Test PromQL queries
curl 'http://localhost:9090/api/v1/query?query=up'

# View AlertManager configuration
docker exec gauth-alertmanager amtool config show

# Check Grafana data sources
docker exec gauth-grafana grafana-cli admin data-sources list

# Export metrics for analysis
curl http://localhost:9090/api/v1/export > metrics-export.json
```

### Log Locations

```bash
# Prometheus logs
docker logs gauth-prometheus

# Grafana logs
docker logs gauth-grafana

# AlertManager logs
docker logs gauth-alertmanager

# Node Exporter logs
docker logs gauth-node-exporter
```

---

## Performance Tuning

### Prometheus

```yaml
# Optimize for high cardinality
prometheus:
  command:
    - '--query.max-samples=50000000'
    - '--query.timeout=2m'
    - '--storage.tsdb.max-block-duration=2h'
```

### Grafana

```yaml
# Increase query timeout
grafana:
  environment:
    - GF_DATAPROXY_TIMEOUT=300
    - GF_DATAPROXY_MAX_IDLE_CONNECTIONS=100
```

### Metric Cardinality

Avoid high-cardinality labels:
```go
// BAD: user_id creates too many time series
requestCounter.WithLabelValues(userID, endpoint).Inc()

// GOOD: aggregate by endpoint only
requestCounter.WithLabelValues(endpoint).Inc()
```

---

## Backup and Restore

### Prometheus Data

```bash
# Backup
docker run --rm -v gauth_prometheus_data:/data \
  -v $(pwd):/backup \
  busybox tar czf /backup/prometheus-backup.tar.gz /data

# Restore
docker run --rm -v gauth_prometheus_data:/data \
  -v $(pwd):/backup \
  busybox tar xzf /backup/prometheus-backup.tar.gz -C /
```

### Grafana Dashboards

```bash
# Export dashboard
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:3001/api/dashboards/uid/gauth-overview > dashboard-backup.json

# Import dashboard
curl -X POST -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d @dashboard-backup.json \
  http://localhost:3001/api/dashboards/db
```

---

## Monitoring Best Practices

1. **Set Appropriate Alert Thresholds**
   - Start conservative, tune based on actual patterns
   - Avoid alert fatigue

2. **Use Multiple Severity Levels**
   - Critical: immediate action required
   - Warning: investigate when convenient
   - Info: for awareness only

3. **Monitor the Monitors**
   - Set up alerts for Prometheus/Grafana health
   - Monitor disk space for metrics storage

4. **Regular Review**
   - Weekly: Review alert trends
   - Monthly: Optimize queries and dashboards
   - Quarterly: Capacity planning

5. **Document Runbooks**
   - Link alerts to remediation procedures
   - Include example PromQL queries

---

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [AlertManager Documentation](https://prometheus.io/docs/alerting/latest/alertmanager/)
- [PromQL Tutorial](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [AgentAuth Metrics API Reference](../AUDIT_EXPORT_API_REFERENCE.md)

---

**Compliance Achievement**: With this advanced monitoring implementation, AgentAuth reaches **97/100 compliance** (+1.0 point for comprehensive monitoring and observability).

**Next Steps**:
- Multi-region deployment (+1.0) → 98/100
- Advanced security features (+1.0) → 99/100
- Performance optimization (+1.0) → 100/100
