# Backup & Restore Procedures

**Document Version:** 1.0  
**Last Updated:** November 10, 2025  
**Owner:** DevOps Team

---

## Overview

This document describes backup and restore procedures for all GAuth system components including PostgreSQL database, Redis cache, and monitoring data.

---

## 🗄️ PostgreSQL Backup & Restore

### Backup Strategy

**Schedule:**
- Full backup: Daily at 2:00 AM UTC
- Incremental backup: Every 6 hours
- WAL archiving: Continuous
- Retention: 30 days

**Backup Location:**
- Primary: `/backups/postgresql/`
- Remote: S3 bucket `s3://gauth-backups/postgresql/`

### Manual Backup

#### Full Backup

```bash
# Create backup directory
mkdir -p /backups/postgresql/$(date +%Y%m%d)

# Perform pg_dump
kubectl exec <postgres-pod> -n gauth-staging -- \
  pg_dump -U gauth -d gauth -F c -f /tmp/gauth_backup.dump

# Copy from pod
kubectl cp gauth-staging/<postgres-pod>:/tmp/gauth_backup.dump \
  /backups/postgresql/$(date +%Y%m%d)/gauth_backup.dump

# Compress backup
gzip /backups/postgresql/$(date +%Y%m%d)/gauth_backup.dump

# Upload to S3
aws s3 cp /backups/postgresql/$(date +%Y%m%d)/gauth_backup.dump.gz \
  s3://gauth-backups/postgresql/$(date +%Y%m%d)/
```

#### Backup Script

```bash
#!/bin/bash
# File: scripts/backup-postgresql.sh

set -e

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/postgresql/${DATE}"
S3_BUCKET="s3://gauth-backups/postgresql/"
NAMESPACE="gauth-staging"

echo "Starting PostgreSQL backup at $(date)"

# Create backup directory
mkdir -p "${BACKUP_DIR}"

# Get PostgreSQL pod name
POSTGRES_POD=$(kubectl get pod -n ${NAMESPACE} -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# Perform backup
echo "Backing up database from pod ${POSTGRES_POD}"
kubectl exec ${POSTGRES_POD} -n ${NAMESPACE} -- \
  pg_dump -U gauth -d gauth -F c -f /tmp/gauth_${DATE}.dump

# Copy backup from pod
echo "Copying backup from pod"
kubectl cp ${NAMESPACE}/${POSTGRES_POD}:/tmp/gauth_${DATE}.dump \
  ${BACKUP_DIR}/gauth_${DATE}.dump

# Compress backup
echo "Compressing backup"
gzip ${BACKUP_DIR}/gauth_${DATE}.dump

# Upload to S3
echo "Uploading to S3"
aws s3 cp ${BACKUP_DIR}/gauth_${DATE}.dump.gz ${S3_BUCKET}${DATE}/

# Cleanup old pod backup
kubectl exec ${POSTGRES_POD} -n ${NAMESPACE} -- rm /tmp/gauth_${DATE}.dump

# Cleanup local backups older than 7 days
find /backups/postgresql -type d -mtime +7 -exec rm -rf {} \;

echo "Backup completed at $(date)"
echo "Backup file: ${BACKUP_DIR}/gauth_${DATE}.dump.gz"
echo "S3 location: ${S3_BUCKET}${DATE}/gauth_${DATE}.dump.gz"
```

### Restore from Backup

#### Prerequisites

```bash
# Stop application pods to prevent connections
kubectl scale deployment gauth-blue -n gauth-staging --replicas=0

# Verify no active connections
kubectl exec <postgres-pod> -n gauth-staging -- \
  psql -U postgres -c "SELECT count(*) FROM pg_stat_activity WHERE datname='gauth'"
```

#### Restore Steps

```bash
# Download backup from S3
aws s3 cp s3://gauth-backups/postgresql/20251110/gauth_20251110_020000.dump.gz /tmp/

# Decompress backup
gunzip /tmp/gauth_20251110_020000.dump.gz

# Copy to PostgreSQL pod
kubectl cp /tmp/gauth_20251110_020000.dump \
  gauth-staging/<postgres-pod>:/tmp/restore.dump

# Drop existing database (CAUTION!)
kubectl exec <postgres-pod> -n gauth-staging -- \
  psql -U postgres -c "DROP DATABASE gauth"

# Create fresh database
kubectl exec <postgres-pod> -n gauth-staging -- \
  psql -U postgres -c "CREATE DATABASE gauth OWNER gauth"

# Restore from backup
kubectl exec <postgres-pod> -n gauth-staging -- \
  pg_restore -U gauth -d gauth -F c /tmp/restore.dump

# Verify restore
kubectl exec <postgres-pod> -n gauth-staging -- \
  psql -U gauth -d gauth -c "SELECT COUNT(*) FROM pg_tables WHERE schemaname='public'"

# Cleanup
kubectl exec <postgres-pod> -n gauth-staging -- rm /tmp/restore.dump

# Restart application
kubectl scale deployment gauth-blue -n gauth-staging --replicas=3
```

#### Restore Script

```bash
#!/bin/bash
# File: scripts/restore-postgresql.sh

set -e

if [ $# -ne 1 ]; then
  echo "Usage: $0 <backup-date>"
  echo "Example: $0 20251110_020000"
  exit 1
fi

BACKUP_DATE=$1
S3_BUCKET="s3://gauth-backups/postgresql/"
NAMESPACE="gauth-staging"
TEMP_DIR="/tmp/restore_${BACKUP_DATE}"

echo "Starting PostgreSQL restore from backup ${BACKUP_DATE}"

# Create temp directory
mkdir -p ${TEMP_DIR}

# Download backup
echo "Downloading backup from S3"
aws s3 cp ${S3_BUCKET}${BACKUP_DATE}/gauth_${BACKUP_DATE}.dump.gz ${TEMP_DIR}/

# Decompress
echo "Decompressing backup"
gunzip ${TEMP_DIR}/gauth_${BACKUP_DATE}.dump.gz

# Get pod name
POSTGRES_POD=$(kubectl get pod -n ${NAMESPACE} -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# Scale down application
echo "Scaling down application"
kubectl scale deployment gauth-blue -n ${NAMESPACE} --replicas=0
kubectl wait --for=delete pod -l app=gauth -n ${NAMESPACE} --timeout=60s

# Copy backup to pod
echo "Copying backup to pod ${POSTGRES_POD}"
kubectl cp ${TEMP_DIR}/gauth_${BACKUP_DATE}.dump \
  ${NAMESPACE}/${POSTGRES_POD}:/tmp/restore.dump

# Terminate active connections
echo "Terminating active connections"
kubectl exec ${POSTGRES_POD} -n ${NAMESPACE} -- \
  psql -U postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='gauth' AND pid <> pg_backend_pid()"

# Drop and recreate database
echo "Dropping and recreating database"
kubectl exec ${POSTGRES_POD} -n ${NAMESPACE} -- psql -U postgres -c "DROP DATABASE IF EXISTS gauth"
kubectl exec ${POSTGRES_POD} -n ${NAMESPACE} -- psql -U postgres -c "CREATE DATABASE gauth OWNER gauth"

# Restore
echo "Restoring database"
kubectl exec ${POSTGRES_POD} -n ${NAMESPACE} -- \
  pg_restore -U gauth -d gauth -F c /tmp/restore.dump

# Verify
echo "Verifying restore"
TABLE_COUNT=$(kubectl exec ${POSTGRES_POD} -n ${NAMESPACE} -- \
  psql -U gauth -d gauth -t -c "SELECT COUNT(*) FROM pg_tables WHERE schemaname='public'")
echo "Table count: ${TABLE_COUNT}"

# Cleanup pod
kubectl exec ${POSTGRES_POD} -n ${NAMESPACE} -- rm /tmp/restore.dump

# Cleanup local
rm -rf ${TEMP_DIR}

# Scale up application
echo "Scaling up application"
kubectl scale deployment gauth-blue -n ${NAMESPACE} --replicas=3
kubectl wait --for=condition=ready pod -l app=gauth -n ${NAMESPACE} --timeout=120s

echo "Restore completed successfully at $(date)"
```

---

## 📦 Redis Backup & Restore

### Backup Strategy

**Schedule:**
- RDB snapshot: Every 6 hours
- AOF: Continuous
- Retention: 14 days

**Backup Location:**
- Primary: `/backups/redis/`
- Remote: S3 bucket `s3://gauth-backups/redis/`

### Manual Backup

```bash
# Trigger RDB save
kubectl exec <redis-pod> -n gauth-staging -- redis-cli BGSAVE

# Wait for completion
kubectl exec <redis-pod> -n gauth-staging -- \
  redis-cli LASTSAVE

# Copy RDB file
kubectl cp gauth-staging/<redis-pod>:/data/dump.rdb \
  /backups/redis/$(date +%Y%m%d)/dump.rdb

# Copy AOF file
kubectl cp gauth-staging/<redis-pod>:/data/appendonly.aof \
  /backups/redis/$(date +%Y%m%d)/appendonly.aof

# Compress and upload
tar -czf /backups/redis/$(date +%Y%m%d)/redis_backup.tar.gz \
  -C /backups/redis/$(date +%Y%m%d) dump.rdb appendonly.aof

aws s3 cp /backups/redis/$(date +%Y%m%d)/redis_backup.tar.gz \
  s3://gauth-backups/redis/$(date +%Y%m%d)/
```

### Restore from Backup

```bash
# Scale down application
kubectl scale deployment gauth-blue -n gauth-staging --replicas=0

# Scale down Redis
kubectl scale statefulset redis -n gauth-staging --replicas=0

# Download backup
aws s3 cp s3://gauth-backups/redis/20251110/redis_backup.tar.gz /tmp/

# Extract backup
tar -xzf /tmp/redis_backup.tar.gz -C /tmp/

# Scale up Redis with one replica
kubectl scale statefulset redis -n gauth-staging --replicas=1
kubectl wait --for=condition=ready pod -l app=redis -n gauth-staging --timeout=60s

# Get Redis pod
REDIS_POD=$(kubectl get pod -n gauth-staging -l app=redis -o jsonpath='{.items[0].metadata.name}')

# Stop Redis temporarily
kubectl exec ${REDIS_POD} -n gauth-staging -- redis-cli SHUTDOWN NOSAVE

# Wait for shutdown
sleep 5

# Copy backup files
kubectl cp /tmp/dump.rdb gauth-staging/${REDIS_POD}:/data/dump.rdb
kubectl cp /tmp/appendonly.aof gauth-staging/${REDIS_POD}:/data/appendonly.aof

# Delete pod to restart
kubectl delete pod ${REDIS_POD} -n gauth-staging

# Wait for pod to be ready
kubectl wait --for=condition=ready pod -l app=redis -n gauth-staging --timeout=60s

# Verify data
kubectl exec ${REDIS_POD} -n gauth-staging -- redis-cli DBSIZE

# Scale up application
kubectl scale deployment gauth-blue -n gauth-staging --replicas=3
```

---

## 📊 Monitoring Data Backup

### Prometheus Data

**Backup Strategy:**
- Snapshot: Daily
- Retention: 30 days
- Location: Persistent volume with snapshots

```bash
# Create Prometheus snapshot
kubectl exec <prometheus-pod> -n gauth-staging -- \
  curl -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot

# Get snapshot name from response
SNAPSHOT_NAME="<snapshot-id>"

# Copy snapshot
kubectl cp gauth-staging/<prometheus-pod>:/prometheus/snapshots/${SNAPSHOT_NAME} \
  /backups/prometheus/$(date +%Y%m%d)

# Compress and upload
tar -czf /backups/prometheus/$(date +%Y%m%d)/prometheus_snapshot.tar.gz \
  -C /backups/prometheus/$(date +%Y%m%d) .

aws s3 cp /backups/prometheus/$(date +%Y%m%d)/prometheus_snapshot.tar.gz \
  s3://gauth-backups/prometheus/$(date +%Y%m%d)/
```

### Grafana Dashboards

Grafana dashboards are stored in Git and provisioned automatically.

```bash
# Export all dashboards
kubectl exec <grafana-pod> -n gauth-staging -- \
  grafana-cli admin export-dashboard

# Or backup Grafana database
kubectl exec <grafana-pod> -n gauth-staging -- \
  sqlite3 /var/lib/grafana/grafana.db .dump > grafana_backup.sql
```

---

## 🔄 Automated Backup with CronJob

### PostgreSQL Backup CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgresql-backup
  namespace: gauth-staging
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: postgres:14-alpine
              command:
                - /bin/sh
                - -c
                - |
                  pg_dump -h postgresql -U gauth -d gauth -F c > /backup/gauth_$(date +%Y%m%d_%H%M%S).dump
                  gzip /backup/gauth_*.dump
              volumeMounts:
                - name: backup
                  mountPath: /backup
          volumes:
            - name: backup
              persistentVolumeClaim:
                claimName: backup-pvc
          restartPolicy: OnFailure
```

---

## 🧪 Backup Testing

### Monthly Backup Test Procedure

1. **Schedule test window** (off-peak hours)
2. **Create test namespace**
3. **Restore from latest backup**
4. **Validate data integrity**
5. **Document results**
6. **Clean up test resources**

```bash
# Test restore script
./scripts/test-backup-restore.sh 20251110_020000 test-namespace
```

---

## 📋 Backup Verification Checklist

### Daily Checks
- [ ] Backup job completed successfully
- [ ] Backup file size reasonable (compare to previous)
- [ ] Backup uploaded to S3
- [ ] No errors in backup logs

### Weekly Checks
- [ ] Test restore in non-production environment
- [ ] Verify data integrity
- [ ] Check backup retention policy
- [ ] Review backup storage usage

### Monthly Checks
- [ ] Full disaster recovery test
- [ ] Update backup procedures if needed
- [ ] Review backup performance
- [ ] Audit backup access logs

---

## 🚨 Disaster Recovery RTO/RPO

| Component | RTO (Recovery Time) | RPO (Data Loss) |
|-----------|---------------------|-----------------|
| PostgreSQL | 15 minutes | 6 hours |
| Redis | 5 minutes | 6 hours |
| Monitoring | 30 minutes | 24 hours |
| Application | 10 minutes | N/A |

---

## 📞 Emergency Contacts

**Backup Issues:**
- DevOps On-Call: PagerDuty
- Database Admin: dba@example.com
- Infrastructure Team: #infrastructure

**Escalation Path:**
1. DevOps Engineer (0-15 min)
2. Senior DBA (15-30 min)
3. Infrastructure Lead (30-60 min)

---

**Document Owner:** DevOps Team  
**Review Schedule:** Quarterly  
**Next Review:** February 10, 2026
