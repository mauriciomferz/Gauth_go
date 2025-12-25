---
title: Gauthplus Monitoring Quickstart
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth+ Monitoring Stack - Quick Start

## Overview

Complete monitoring solution for GAuth+ with Grafana dashboard, Prometheus metrics, and AlertManager alerts.

## Quick Start (3 Minutes)

### Step 1: Start the Stack

```bash
cd /Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/deployments/docker
docker compose up -d
```

**What this does**:
- Starts GAuth service (port 8080) with metrics enabled
- Starts Prometheus (port 9090) to collect metrics
- Starts Grafana (port 3000) with pre-configured dashboard
- Starts AlertManager (port 9093) for alert management
- Auto-provisions datasources and dashboards

### Step 2: Access Grafana Dashboard

1. Open browser: **http://localhost:3000**
2. Login:
   - Username: `admin`
   - Password: `admin`
3. Navigate to: **Dashboards** → **Browse** → **GAuth+** folder
4. Open: **GAuth+ Monitoring Dashboard**

### Step 3: Generate Some Traffic

```bash
# Make a few authorization requests to generate metrics
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/rfc0111/authorize \
    -H "Content-Type: application/json" \
    -d '{
      "client_id": "agent-001",
      "requested_actions": ["read"],
      "poa_id": "550e8400-e29b-41d4-a716-446655440001"
    }'
  sleep 1
done
```

### Step 4: Watch the Dashboard Update

The dashboard auto-refreshes every 10 seconds. You should see:
- Validation rate increasing
- Cache hit/miss metrics appearing
- Duration percentiles updating

## What You Get

### 12 Dashboard Panels

1. **GAuth+ Validations Rate** - Real-time validation metrics by feature
2. **Total Validation Rate** - Aggregated gauge
3. **P95 Validation Duration** - Latency monitoring
4. **Cache Hit Rate** - Performance optimization tracking
5. **Cache Size** - Memory usage monitoring
6. **Policy Violations** - Compliance tracking
7. **Successor Activations** - AI takeover events
8. **P95 Delegation Depth** - Chain depth analysis
9. **Dual Control Approvals** - Approval workflow metrics
10. **Fiduciary Violations** - Duty breach tracking
11. **Agent Capability Levels** - Real-time capability table
12. **Validation Duration Percentiles** - P50/P95/P99 trends

### 10 Alert Rules

Automatically configured in Prometheus:

- ⚠️ High validation failure rate (>10%)
- ⚠️ Low cache hit rate (<70%)
- 🔴 High policy violation rate (>1/sec)
- ⚠️ High validation latency (P95 >100ms)
- ⚠️ Excessive delegation depth (>5)
- ⚠️ Frequent successor activations
- 🔴 Critical fiduciary violations
- ⚠️ Dual control failures (>20%)
- 🔴 Service down (>2min)
- ⚠️ Excessive cache size (>50k)

### 3 Monitoring Services

- **Grafana** (3000): Visualization and dashboards
- **Prometheus** (9090): Metrics collection and storage
- **AlertManager** (9093): Alert routing and management

## Verify Everything Works

### Check GAuth Metrics Endpoint

```bash
curl http://localhost:8080/metrics | grep gauthplus | head -20
```

**Expected output**:
```
# HELP gauthplus_validations_total Total number of GAuth+ validations performed
# TYPE gauthplus_validations_total counter
gauthplus_validations_total{feature="successor",result="success"} 0
gauthplus_validations_total{feature="delegation",result="success"} 0
gauthplus_cache_hits_total{cache_type="capability"} 0
gauthplus_cache_misses_total{cache_type="capability"} 0
...
```

### Check Prometheus Targets

```bash
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job=="gauth-service")'
```

**Or visit**: http://localhost:9090/targets

Should show `gauth-service` with status **UP**.

### Check Grafana Provisioning

```bash
docker compose logs grafana | grep -i provision
```

**Expected output**:
```
Provisioning dashboards from configuration
Dashboard "GAuth+ Monitoring Dashboard" provisioned successfully
```

## Access URLs

| Service | URL | Credentials |
|---------|-----|-------------|
| **Grafana** | http://localhost:3000 | admin / admin |
| **Prometheus** | http://localhost:9090 | None |
| **AlertManager** | http://localhost:9093 | None |
| **GAuth Metrics** | http://localhost:8080/metrics | None |
| **GAuth Health** | http://localhost:8080/health | None |

## Common Tasks

### View Active Alerts

**In Prometheus**:
- Navigate to: http://localhost:9090/alerts
- Filter by: `gauthplus`

**In AlertManager**:
- Navigate to: http://localhost:9093
- View firing alerts

**In Grafana**:
- Navigate to: **Alerting** → **Alert Rules**
- Filter: `component=gauthplus`

### Export Dashboard

1. Open dashboard in Grafana
2. Click **Dashboard settings** (⚙️ icon)
3. Select **JSON Model** tab
4. Copy JSON
5. Save to file

### Reload Prometheus Configuration

```bash
# Method 1: HUP signal
docker compose exec prometheus kill -HUP 1

# Method 2: HTTP reload (requires --web.enable-lifecycle flag)
curl -X POST http://localhost:9090/-/reload
```

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f grafana
docker compose logs -f prometheus
docker compose logs -f gauth
```

### Stop the Stack

```bash
docker compose down
```

**To remove data volumes**:
```bash
docker compose down -v
```

## Troubleshooting

### Dashboard Not Showing Data

**Check if metrics are being exposed**:
```bash
curl http://localhost:8080/metrics | grep gauthplus
```

**Check if Prometheus is scraping**:
```bash
curl http://localhost:9090/api/v1/query?query=up{job=\"gauth-service\"}
```

**Check scrape errors in Prometheus**:
- Navigate to: http://localhost:9090/targets
- Look for errors in `gauth-service` target

### Grafana Can't Connect to Prometheus

**Check datasource**:
1. Grafana → **Configuration** → **Data Sources**
2. Click **Prometheus**
3. Click **Test** button
4. Should show: "Data source is working"

**Check network connectivity**:
```bash
docker compose exec grafana ping prometheus
```

### Alerts Not Firing

**Check alert rules in Prometheus**:
```bash
curl http://localhost:9090/api/v1/rules | jq '.data.groups[] | select(.name=="gauthplus_alerts")'
```

**Verify AlertManager is connected**:
- Navigate to: http://localhost:9090/status/runtimeinfo
- Check: `alertmanagers` section

### High Memory Usage

**Reduce Prometheus retention**:

Edit `deployments/docker/monitoring/prometheus.yml`:
```yaml
command:
  - '--storage.tsdb.retention.time=7d'  # Change from 200h
```

Then restart:
```bash
docker compose restart prometheus
```

## Performance Notes

### Resource Usage (Typical)

- **GAuth**: ~50-100MB RAM
- **Prometheus**: ~200-500MB RAM (depends on retention)
- **Grafana**: ~50-100MB RAM
- **AlertManager**: ~20-50MB RAM

**Total**: ~320-750MB RAM for full stack

### Scrape Intervals

- **Prometheus scrapes GAuth**: Every 15 seconds
- **Dashboard auto-refresh**: Every 10 seconds
- **Alert evaluation**: Every 30 seconds

### Data Retention

- **Prometheus**: 200 hours (8.3 days) by default
- **Grafana**: Indefinite (uses Prometheus as source)
- **AlertManager**: 24 hours for resolved alerts

## Next Steps

### 1. Review the Full Guide

See `GAUTHPLUS_GRAFANA_DASHBOARD_GUIDE.md` for:
- Detailed panel descriptions
- PromQL query examples
- Alert customization
- Production deployment tips

### 2. Customize Alerts

Edit `deployments/docker/monitoring/alert-rules.yml`:
- Adjust thresholds
- Add new rules
- Modify notification routing

### 3. Add Custom Panels

- Open dashboard in Grafana
- Click **Add Panel**
- Write PromQL queries
- Save and export JSON

### 4. Set Up Notifications

Configure AlertManager to send notifications:
- Email (SMTP)
- Slack webhooks
- PagerDuty
- Custom webhooks

See `deployments/docker/monitoring/alertmanager.yml`

## Documentation Files

- **Complete Guide**: `GAUTHPLUS_GRAFANA_DASHBOARD_GUIDE.md` (700+ lines)
- **Monitoring Setup**: `deployments/docker/monitoring/README.md`
- **Roadmap**: `GAUTHPLUS_NEXT_STEPS.md`
- **Dashboard JSON**: `deployments/docker/monitoring/grafana/dashboards/gauthplus-monitoring.json`

## Support

If you encounter issues:

1. **Check logs**: `docker compose logs [service]`
2. **Verify metrics**: `curl http://localhost:8080/metrics`
3. **Test connectivity**: `docker compose exec grafana ping prometheus`
4. **Review targets**: http://localhost:9090/targets
5. **Check provisioning**: `docker compose logs grafana | grep provision`

## Summary

✅ **Complete monitoring stack in 3 minutes**  
✅ **12 pre-configured dashboard panels**  
✅ **10 automatic alert rules**  
✅ **Auto-provisioned datasources**  
✅ **Production-ready configuration**  
✅ **Comprehensive documentation**

**Start now**:
```bash
cd deployments/docker
docker compose up -d
# Open http://localhost:3000
```

---

**Created**: November 26, 2025  
**Version**: 1.0  
**Status**: Production Ready
