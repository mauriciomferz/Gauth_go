---
title: Disaster Recovery Guide
category: disaster-recovery-guide
status: active
lastUpdated: 2025-11-12
owners: platform-eng
source: internal
refreshCadence: annually
---

# Disaster Recovery Guide

**Document Version:** 1.0  
**Last Updated:** November 10, 2025  
**Owner:** DevOps Team / Infrastructure Team

---

## Executive Summary

This document outlines disaster recovery procedures for the GAuth application, covering complete system recovery scenarios including cluster failures, data center outages, and catastrophic data loss.

**Recovery Objectives:**
- **RTO (Recovery Time Objective):** 30 minutes
- **RPO (Recovery Point Objective):** 6 hours
- **Availability Target:** 99.9% uptime

---

## 🎯 Disaster Scenarios

### Scenario 1: Complete Pod Failure (All GAuth Pods Down)
- **Likelihood:** Medium
- **Impact:** High
- **RTO:** 10 minutes
- **Runbook:** [GAuthServiceUnavailable](./runbooks/GAuthServiceUnavailable.md)

### Scenario 2: Database Failure (PostgreSQL Down)
- **Likelihood:** Low
- **Impact:** Critical
- **RTO:** 15 minutes
- **Runbook:** [PostgreSQLDown](./runbooks/PostgreSQLDown.md)

### Scenario 3: Cache Failure (Redis Down)
- **Likelihood:** Medium
- **Impact:** Medium
- **RTO:** 5 minutes
- **Runbook:** [RedisDown](./runbooks/RedisDown.md)

### Scenario 4: Complete Cluster Failure
- **Likelihood:** Very Low
- **Impact:** Critical
- **RTO:** 30 minutes
- **Procedure:** This document (below)

### Scenario 5: Data Corruption or Loss
- **Likelihood:** Very Low
- **Impact:** Critical
- **RTO:** 60 minutes
- **Procedure:** This document (below)

---

## 🚨 Disaster Declaration Criteria

Declare a disaster when:

1. **Complete cluster unavailability** > 15 minutes
2. **Multiple critical components** failing simultaneously
3. **Data corruption** detected across multiple systems
4. **Regional outage** affecting primary infrastructure
5. **Security breach** requiring complete system rebuild

### Declaration Process

```bash
# 1. Notify leadership via Slack
/incident declare "Disaster Recovery - [SCENARIO] - P0"

# 2. Page incident commander
# Use PagerDuty high-urgency page

# 3. Activate war room
# Zoom link: https://example.zoom.us/disaster-recovery

# 4. Start incident log
# Document: https://docs.google.com/disaster-incidents/
```

---

## 🔄 Complete Cluster Recovery

### Prerequisites

- [ ] Access to backup cluster or fresh Kubernetes cluster
- [ ] Access to S3 backup bucket: `s3://gauth-backups/`
- [ ] Latest configuration files from Git
- [ ] AWS credentials configured
- [ ] kubectl configured for target cluster

### Step 1: Verify Backup Availability

```bash
# Check latest backups
aws s3 ls s3://gauth-backups/postgresql/ --recursive | tail -5
aws s3 ls s3://gauth-backups/redis/ --recursive | tail -5

# Verify backup integrity
LATEST_DB_BACKUP=$(aws s3 ls s3://gauth-backups/postgresql/ --recursive | tail -1 | awk '{print $4}')
aws s3 cp s3://gauth-backups/${LATEST_DB_BACKUP} /tmp/test_backup.dump.gz
gunzip -t /tmp/test_backup.dump.gz && echo "Backup OK" || echo "Backup CORRUPTED"
```

### Step 2: Prepare Target Cluster

```bash
# Create namespace
kubectl create namespace gauth-staging

# Create secrets
kubectl create secret generic gauth-secrets \
  --from-literal=database-url="postgresql://gauth:password@postgresql:5432/gauth" \
  --from-literal=redis-url="redis://redis:6379" \
  -n gauth-staging

# Apply network policies
kubectl apply -f k8s-network-policies.yaml
```

### Step 3: Deploy PostgreSQL

```bash
# Deploy PostgreSQL StatefulSet
kubectl apply -f k8s-postgres.yaml

# Wait for pod to be ready
kubectl wait --for=condition=ready pod -l app=postgresql -n gauth-staging --timeout=300s

# Verify PostgreSQL is running
kubectl exec -n gauth-staging $(kubectl get pod -n gauth-staging -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U postgres -c "SELECT version()"
```

### Step 4: Restore Database

```bash
# Use restore script
./scripts/restore-postgresql.sh <latest-backup-date>

# Verify database content
kubectl exec -n gauth-staging <postgres-pod> -- \
  psql -U gauth -d gauth -c "SELECT COUNT(*) FROM pg_tables WHERE schemaname='public'"
```

### Step 5: Deploy Redis

```bash
# Deploy Redis StatefulSet
kubectl apply -f k8s-redis.yaml

# Wait for pod to be ready
kubectl wait --for=condition=ready pod -l app=redis -n gauth-staging --timeout=300s

# Restore Redis data (if needed)
./scripts/restore-redis.sh <latest-backup-date>

# Verify Redis
kubectl exec -n gauth-staging <redis-pod> -- redis-cli PING
```

### Step 6: Deploy GAuth Application

```bash
# Apply deployment manifest
kubectl apply -f k8s-test-blue.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=gauth -n gauth-staging --timeout=300s

# Verify health
kubectl run test-health --rm -i --image=curlimages/curl --restart=Never \
  -n gauth-staging -- curl -f http://gauth-service/api/v1/beta/health
```

### Step 7: Deploy Monitoring Stack

```bash
# Deploy Prometheus, Grafana, AlertManager
kubectl apply -f k8s-monitoring-stack.yaml
kubectl apply -f k8s-alertmanager.yaml
kubectl apply -f k8s-prometheus-alerts-enhanced.yaml

# Wait for monitoring pods
kubectl wait --for=condition=ready pod -l app=prometheus -n gauth-staging --timeout=300s
kubectl wait --for=condition=ready pod -l app=grafana -n gauth-staging --timeout=300s

# Verify Prometheus targets
kubectl port-forward -n gauth-staging svc/prometheus 9090:9090 &
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job, health}'
```

### Step 8: Validation

```bash
# Run validation tests
./scripts/validate-deployment.sh

# Check all pods
kubectl get pods -n gauth-staging

# Run smoke tests
kubectl run smoke-test --rm -i --image=curlimages/curl --restart=Never \
  -n gauth-staging -- sh -c '
  for i in $(seq 1 10); do
    curl -s http://gauth-service/api/v1/beta/health && echo " - OK"
    sleep 1
  done'

# Check metrics
kubectl port-forward -n gauth-staging svc/grafana 3000:3000 &
# Open http://localhost:3000 and verify dashboards
```

---

## 📊 Data Recovery Procedures

### PostgreSQL Data Corruption

```bash
# 1. Identify corruption
kubectl exec <postgres-pod> -n gauth-staging -- \
  psql -U postgres -d gauth -c "SELECT pg_database.datname FROM pg_database"

# 2. Attempt REINDEX
kubectl exec <postgres-pod> -n gauth-staging -- \
  psql -U postgres -d gauth -c "REINDEX DATABASE gauth"

# 3. If corruption persists, restore from backup
./scripts/restore-postgresql.sh <backup-date>

# 4. Verify data integrity
./scripts/verify-database-integrity.sh
```

### Redis Data Loss

```bash
# Redis is a cache - data loss is acceptable
# Rebuild cache from database queries

# 1. Clear Redis
kubectl exec <redis-pod> -n gauth-staging -- redis-cli FLUSHALL

# 2. Restart application to rebuild cache
kubectl rollout restart deployment/gauth-blue -n gauth-staging

# 3. Monitor cache hit rate
kubectl port-forward -n gauth-staging svc/grafana 3000:3000
# Check Redis dashboard - hit rate will improve over time
```

---

## 🏗️ Infrastructure Recovery

### Kubernetes Cluster Recreation

If entire cluster must be rebuilt:

```bash
# 1. Create new kind cluster (for local testing)
kind create cluster --name gauth-dr --config kind-cluster-config.yaml

# 2. Configure kubectl context
kubectl config use-context kind-gauth-dr

# 3. Follow "Complete Cluster Recovery" procedure above

# For production: Use cloud provider to provision new cluster
# AWS EKS: eksctl create cluster --name gauth-prod-dr --region us-east-1
# GCP GKE: gcloud container clusters create gauth-prod-dr --region us-central1
# Azure AKS: az aks create --resource-group gauth-rg --name gauth-prod-dr
```

### Storage Recovery

```bash
# If PVCs are corrupted or lost

# 1. Delete existing PVCs
kubectl delete pvc -n gauth-staging --all

# 2. Recreate PVCs
kubectl apply -f k8s-postgres.yaml
kubectl apply -f k8s-redis.yaml

# 3. Restore data from backups
./scripts/restore-postgresql.sh <backup-date>
./scripts/restore-redis.sh <backup-date>
```

---

## 🧪 Disaster Recovery Testing

### Monthly DR Drill

**Schedule:** First Saturday of each month, 10:00 AM UTC

**Procedure:**
1. Create test namespace `gauth-dr-test`
2. Execute complete cluster recovery
3. Validate all services
4. Document timing and issues
5. Clean up test namespace
6. Update procedures based on findings

```bash
# DR drill script
./scripts/dr-drill.sh

# This script:
# - Creates test namespace
# - Deploys all components from backups
# - Runs validation tests
# - Generates timing report
# - Cleans up resources
```

### Validation Checklist

After recovery, verify:

- [ ] All pods running (3 GAuth, 1 PostgreSQL, 1 Redis)
- [ ] Database accessible and data intact
- [ ] Redis operational (data optional)
- [ ] Health checks passing
- [ ] Metrics being collected
- [ ] Alerts configured and active
- [ ] Dashboards accessible
- [ ] Service responding to requests
- [ ] Load test passing (1000 req/sec)
- [ ] No errors in application logs

---

## 📞 Emergency Contacts

### Incident Commander
- Name: [Lead DevOps Engineer]
- Phone: [Number]
- PagerDuty: @oncall-devops

### Database Administrator
- Name: [Senior DBA]
- Phone: [Number]
- Email: dba@example.com

### Infrastructure Lead
- Name: [Infrastructure Manager]
- Phone: [Number]
- Slack: @infra-lead

### Executive Team
- CTO: [Name] - [Phone]
- VP Engineering: [Name] - [Phone]

---

## 📋 Post-Disaster Actions

### Immediate (Within 24 hours)

1. **Document timeline** of events
2. **Collect all logs** from failed systems
3. **Preserve evidence** for analysis
4. **Notify stakeholders** of resolution
5. **Schedule post-mortem** meeting

### Short-term (Within 1 week)

1. **Conduct post-mortem** (blameless)
2. **Identify root cause**
3. **Create action items** for prevention
4. **Update procedures** based on learnings
5. **Test new procedures**

### Long-term (Within 1 month)

1. **Implement preventive measures**
2. **Enhance monitoring**
3. **Update disaster recovery plan**
4. **Train team on new procedures**
5. **Conduct follow-up DR drill**

---

## 🛠️ Recovery Tools & Scripts

All scripts located in `scripts/`:

- `restore-postgresql.sh` - Database restore
- `restore-redis.sh` - Redis restore
- `validate-deployment.sh` - Post-recovery validation
- `dr-drill.sh` - DR testing automation
- `backup-all.sh` - Complete backup
- `verify-database-integrity.sh` - Data validation

---

## 📚 Related Documentation

- [Backup & Restore Procedures](./BACKUP_RESTORE_PROCEDURES.md)
- [Operational Runbooks](./runbooks/README.md)
- [Incident Response Plan](./INCIDENT_RESPONSE_PLAN.md)
- [Production Deployment Guide](./PRODUCTION_DEPLOYMENT_GUIDE.md)

---

## 🔄 Document Maintenance

**Review Schedule:** Quarterly  
**Next Review:** February 10, 2026  
**Version History:**
- v1.0 - November 10, 2025 - Initial version

**Change Approval:**
- DevOps Team Lead
- Infrastructure Manager
- CTO

---

**⚠️ CRITICAL: This document must be accessible offline and printed. Store physical copies in multiple secure locations.**

---

**Document Classification:** Internal - Confidential  
**Last Tested:** [Date of last DR drill]  
**Test Results:** [Link to test report]
