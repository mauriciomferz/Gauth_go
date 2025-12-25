---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth+ Monitoring Configuration

This directory contains all configuration files for monitoring GAuth+ services using Prometheus, Grafana, and AlertManager.

## Directory Structure

```
monitoring/
├── prometheus.yml              # Prometheus configuration
├── alert-rules.yml            # Prometheus alert rules
├── alertmanager.yml           # AlertManager configuration
└── grafana/
    ├── provisioning/
    │   ├── datasources/
    │   │   └── prometheus.yml # Prometheus datasource config
    │   └── dashboards/
    │       └── gauthplus.yml  # Dashboard provisioning
    └── dashboards/
        └── gauthplus-monitoring.json  # GAuth+ dashboard
```

## Quick Start

### 1. Start Monitoring Stack

From the repository root:

```bash
cd deployments/docker
docker compose up -d
```

This starts:
- **GAuth Service** (port 8080) - Exposes metrics at `/metrics`
- **Prometheus** (port 9090) - Scrapes and stores metrics
- **Grafana** (port 3000) - Visualizes metrics
- **AlertManager** (port 9093) - Manages alerts

### 2. Access Grafana

1. Open browser: http://localhost:3000
2. Login: admin / admin
3. Navigate to **Dashboards** → **GAuth+** → **GAuth+ Monitoring Dashboard**

### 3. Verify Metrics

Check that GAuth+ is exposing metrics:

```bash
curl http://localhost:8080/metrics | grep gauthplus
```

Check that Prometheus is scraping:

```bash
curl http://localhost:9090/api/v1/query?query=up{job="gauth-service"}
```

## Configuration Files

### prometheus.yml

Configures Prometheus scraping and alerting:

- **Scrape Interval**: 15 seconds
- **Targets**: gauth-service, node-exporter, grafana, alertmanager
- **Alert Rules**: Loaded from `alert-rules.yml`
- **AlertManager**: Sends alerts to alertmanager:9093

### alert-rules.yml

Defines 10 alert rules for GAuth+:

1. **GAuthPlusHighValidationFailureRate** - Validation failures > 10%
2. **GAuthPlusCacheHitRateLow** - Cache hit rate < 70%
3. **GAuthPlusHighPolicyViolationRate** - Policy violations > 1/sec
4. **GAuthPlusHighValidationLatency** - P95 latency > 100ms
5. **GAuthPlusExcessiveDelegationDepth** - P95 depth > 5
6. **GAuthPlusFrequentSuccessorActivations** - > 0.1/sec
7. **GAuthPlusCriticalFiduciaryViolations** - Any critical violations
8. **GAuthPlusDualControlFailures** - Rejection rate > 20%
9. **GAuthPlusServiceDown** - Service unavailable
10. **GAuthPlusCacheSizeExcessive** - Cache > 50k entries

### alertmanager.yml

Configures alert routing and notifications:

- **Group By**: alertname, severity
- **Routes**: Critical and warning alerts separated
- **Receivers**: Webhook endpoints (configure for production)

### grafana/provisioning/datasources/prometheus.yml

Auto-provisions Prometheus as Grafana datasource:

- **URL**: http://prometheus:9090
- **Access**: Proxy (server-side)
- **Default**: True

### grafana/provisioning/dashboards/gauthplus.yml

Auto-provisions GAuth+ dashboards:

- **Folder**: GAuth+
- **Path**: /var/lib/grafana/dashboards
- **Update Interval**: 10 seconds

### grafana/dashboards/gauthplus-monitoring.json

Complete GAuth+ monitoring dashboard with 12 panels:

1. **GAuth+ Validations Rate** - Real-time validation rates by feature
2. **Total Validation Rate** - Aggregated validation gauge
3. **P95 Validation Duration** - Latency gauge
4. **Cache Hit Rate** - Hit rate by cache type
5. **Cache Size** - Current cache entries
6. **Policy Violations** - Violations by type/severity (1h)
7. **Successor Activations** - Activation rate gauge
8. **P95 Delegation Depth** - Delegation depth gauge
9. **Dual Control Approvals** - Approval/rejection trends (1h)
10. **Fiduciary Violations** - Violations by duty type (1h)
11. **Agent Capability Levels** - Table of agent capabilities
12. **Validation Duration Percentiles** - P50/P95/P99 trends

## Metrics Reference

### GAuth+ Metrics Exposed

All metrics are prefixed with `gauthplus_`:

#### Validation Metrics
- `gauthplus_validations_total{feature, result}` - Counter
- `gauthplus_validation_duration_seconds{feature}` - Histogram

#### Cache Metrics
- `gauthplus_cache_hits_total{cache_type}` - Counter
- `gauthplus_cache_misses_total{cache_type}` - Counter
- `gauthplus_cache_size{cache_type}` - Gauge

#### Policy Metrics
- `gauthplus_policy_violations_total{policy_type, severity}` - Counter
- `gauthplus_successor_activations_total` - Counter
- `gauthplus_delegation_depth` - Histogram
- `gauthplus_dual_control_approvals_total{action_type, status}` - Counter
- `gauthplus_capability_level{agent_id}` - Gauge
- `gauthplus_fiduciary_violations_total{duty_type, severity}` - Counter

## Customization

### Adding New Metrics

1. Add metric to `pkg/metrics/prometheus.go`
2. Record metric in relevant code
3. Restart GAuth service
4. Create dashboard panel in Grafana

### Modifying Alert Thresholds

Edit `alert-rules.yml`:

```yaml
- alert: GAuthPlusCacheHitRateLow
  expr: |
    (rate(gauthplus_cache_hits_total[5m]) 
    / (rate(gauthplus_cache_hits_total[5m]) + rate(gauthplus_cache_misses_total[5m]))) < 0.8  # Changed from 0.7
  for: 10m
```

Reload Prometheus config:

```bash
docker compose exec prometheus kill -HUP 1
# or
curl -X POST http://localhost:9090/-/reload
```

### Adding Dashboard Panels

1. Open dashboard in Grafana
2. Click **Add Panel**
3. Configure PromQL query
4. Save dashboard
5. Export JSON to `grafana/dashboards/gauthplus-monitoring.json`

## Troubleshooting

### Prometheus Not Scraping

Check Prometheus targets:

```bash
curl http://localhost:9090/api/v1/targets
```

Or navigate to: http://localhost:9090/targets

If `gauth-service` shows as DOWN:
- Verify GAuth service is running: `docker compose ps gauth`
- Check metrics endpoint: `curl http://localhost:8080/metrics`
- Review network connectivity: `docker compose exec prometheus ping gauth`

### Grafana Dashboard Missing

Check provisioning logs:

```bash
docker compose logs grafana | grep -i provision
```

Manually verify files:

```bash
docker compose exec grafana ls -la /etc/grafana/provisioning/datasources/
docker compose exec grafana ls -la /etc/grafana/provisioning/dashboards/
docker compose exec grafana ls -la /var/lib/grafana/dashboards/
```

### Alerts Not Firing

Check AlertManager status:

```bash
curl http://localhost:9093/api/v2/status
```

View active alerts in Prometheus:

```bash
curl http://localhost:9090/api/v1/alerts
```

Or navigate to: http://localhost:9090/alerts

### No Data in Panels

1. Verify metric exists in Prometheus:
   ```bash
   curl 'http://localhost:9090/api/v1/query?query=gauthplus_validations_total'
   ```

2. Check time range in Grafana (top-right corner)

3. Verify scrape interval hasn't exceeded retention period

4. Check for label mismatches in queries

## Production Deployment

### Security Hardening

1. **Change default passwords**:
   ```yaml
   # docker-compose.yml
   environment:
     GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_ADMIN_PASSWORD}
   ```

2. **Enable authentication in Prometheus**:
   - Use reverse proxy (nginx, Traefik)
   - Configure basic auth or OAuth

3. **Enable TLS**:
   - Configure TLS in Prometheus and Grafana
   - Use valid certificates

4. **Restrict network access**:
   - Use firewall rules
   - Limit exposure to internal network

### High Availability

1. **Multiple Prometheus instances**:
   - Use federation or remote write
   - Share alert rules

2. **External Grafana database**:
   ```yaml
   environment:
     GF_DATABASE_TYPE: postgres
     GF_DATABASE_HOST: postgres:5432
   ```

3. **Persistent volumes**:
   - Already configured in docker-compose.yml
   - Ensure backup strategy

### Performance Tuning

1. **Adjust retention periods**:
   ```yaml
   # prometheus.yml command
   - '--storage.tsdb.retention.time=30d'  # Adjust as needed
   ```

2. **Optimize scrape intervals**:
   ```yaml
   # prometheus.yml
   global:
     scrape_interval: 30s  # Increase if needed
   ```

3. **Use recording rules for expensive queries**:
   ```yaml
   # alert-rules.yml
   groups:
     - name: recordings
       interval: 30s
       rules:
         - record: gauthplus:validation_rate:5m
           expr: rate(gauthplus_validations_total[5m])
   ```

## Monitoring Metrics

Track monitoring stack health:

- **Prometheus**: http://localhost:9090/metrics
- **Grafana**: http://localhost:3000/metrics
- **AlertManager**: http://localhost:9093/metrics

Key metrics to monitor:
- `prometheus_tsdb_storage_blocks_bytes` - Storage usage
- `prometheus_target_scrapes_exceeded_sample_limit_total` - Scrape issues
- `grafana_api_response_status_total` - API health

## Backup and Recovery

### Backup Grafana Configuration

```bash
# Backup dashboards
docker compose exec grafana grafana-cli admin export-dashboard > backup.json

# Backup database (if using SQLite)
docker compose cp grafana:/var/lib/grafana/grafana.db ./grafana-backup.db
```

### Backup Prometheus Data

```bash
# Create snapshot
curl -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot

# Copy data directory
docker compose exec prometheus tar czf /prometheus/backup.tar.gz /prometheus/
docker compose cp prometheus:/prometheus/backup.tar.gz ./
```

### Restore Grafana

```bash
# Restore database
docker compose cp ./grafana-backup.db grafana:/var/lib/grafana/grafana.db
docker compose restart grafana
```

## Additional Resources

- **Full Guide**: See `GAUTHPLUS_GRAFANA_DASHBOARD_GUIDE.md`
- **Prometheus Docs**: https://prometheus.io/docs/
- **Grafana Docs**: https://grafana.com/docs/
- **AlertManager Docs**: https://prometheus.io/docs/alerting/latest/alertmanager/

## Support

For issues or questions:
1. Check logs: `docker compose logs [service]`
2. Review documentation files
3. Verify configuration files
4. Test metrics endpoint: `curl http://localhost:8080/metrics`

---

**Version**: 1.0  
**Last Updated**: November 26, 2025  
**Maintainer**: GAuth+ Team
