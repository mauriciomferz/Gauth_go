---
title: "PostgreSQL Down Alert Runbook"
category: runbook
status: active
lastUpdated: 2025-11-12
owners: devops-team
refreshCadence: monthly
---
# Alert Runbook: PostgreSQLDown

**Alert Name:** `PostgreSQLDown`  
**Severity:** Critical  
**Component:** Database  
**Category:** Database

---

## Summary

PostgreSQL database instance is down or unreachable, preventing all database operations.

---

## Alert Details

**Trigger Condition:**
```promql
pg_up{namespace="gauth-staging"} == 0
```

**Duration:** 2 minutes  
**Impact:** Complete data access failure, application may be unable to function

---

## Immediate Actions (First 5 Minutes)

1. **Check PostgreSQL Pod Status**
   ```bash
   kubectl get pods -n gauth-staging -l app=postgresql
   kubectl describe pod <postgres-pod> -n gauth-staging
   ```

2. **Check PostgreSQL Logs**
   ```bash
   kubectl logs <postgres-pod> -n gauth-staging --tail=100
   kubectl logs <postgres-pod> -n gauth-staging --previous
   ```

3. **Check Service and Endpoints**
   ```bash
   kubectl get svc postgresql -n gauth-staging
   kubectl get endpoints postgresql -n gauth-staging
   ```

---

## Diagnosis

### Common Causes

1. **Pod Crash or Restart**
   - PostgreSQL process crashed
   - OOM kill
   - Failed health checks

2. **Data Corruption**
   - Corrupted data files
   - WAL corruption
   - Transaction log issues

3. **Resource Issues**
   - Disk full
   - Out of memory
   - Connection exhaustion

4. **Configuration Problems**
   - Invalid postgresql.conf
   - Permission issues
   - Network policy blocking access

### Investigation Commands

```bash
# Check pod events
kubectl get events -n gauth-staging --field-selector involvedObject.name=<postgres-pod>

# Check persistent volume
kubectl get pvc -n gauth-staging -l app=postgresql
kubectl describe pvc <pvc-name> -n gauth-staging

# Check disk usage
kubectl exec <postgres-pod> -n gauth-staging -- df -h

# Check PostgreSQL process
kubectl exec <postgres-pod> -n gauth-staging -- ps aux | grep postgres

# Try connecting directly
kubectl exec <postgres-pod> -n gauth-staging -- psql -U gauth -d gauth -c "SELECT 1"
```

---

## Resolution Steps

### Scenario 1: Pod Crashed

1. **Check for recent restarts**
   ```bash
   kubectl get pod <postgres-pod> -n gauth-staging -o jsonpath='{.status.containerStatuses[0].restartCount}'
   ```

2. **Review crash logs**
   ```bash
   kubectl logs <postgres-pod> -n gauth-staging --previous
   ```

3. **Restart pod if needed**
   ```bash
   kubectl delete pod <postgres-pod> -n gauth-staging
   # StatefulSet will recreate it
   kubectl get pods -n gauth-staging -w
   ```

### Scenario 2: Disk Full

1. **Check disk usage**
   ```bash
   kubectl exec <postgres-pod> -n gauth-staging -- du -sh /var/lib/postgresql/data
   ```

2. **Clean up old WAL files** (if safe)
   ```bash
   kubectl exec <postgres-pod> -n gauth-staging -- \
     find /var/lib/postgresql/data/pg_wal -name "*.old" -mtime +7 -delete
   ```

3. **Expand PVC if needed**
   ```bash
   kubectl edit pvc postgresql-data -n gauth-staging
   # Increase storage size
   ```

### Scenario 3: Data Corruption

1. **Attempt recovery**
   ```bash
   kubectl exec <postgres-pod> -n gauth-staging -- \
     psql -U postgres -d gauth -c "REINDEX DATABASE gauth"
   ```

2. **Check for corruption**
   ```bash
   kubectl exec <postgres-pod> -n gauth-staging -- \
     psql -U postgres -d gauth -c "SELECT pg_database.datname, pg_size_pretty(pg_database_size(pg_database.datname)) FROM pg_database"
   ```

3. **Restore from backup** (if corruption severe)
   ```bash
   # See DISASTER_RECOVERY_GUIDE.md for full restore procedure
   ./scripts/restore-postgresql-backup.sh <backup-date>
   ```

### Scenario 4: Connection Issues

1. **Check service configuration**
   ```bash
   kubectl get svc postgresql -n gauth-staging -o yaml
   ```

2. **Test connectivity from AgentAuth pod**
   ```bash
   kubectl exec <gauth-pod> -n gauth-staging -- \
     nc -zv postgresql 5432
   ```

3. **Check network policies**
   ```bash
   kubectl get networkpolicies -n gauth-staging
   ```

---

## Validation

1. **Database is accessible**
   ```bash
   kubectl exec <postgres-pod> -n gauth-staging -- \
     psql -U gauth -d gauth -c "SELECT version()"
   ```

2. **Can query data**
   ```bash
   kubectl exec <postgres-pod> -n gauth-staging -- \
     psql -U gauth -d gauth -c "SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public'"
   ```

3. **AgentAuth can connect**
   ```bash
   kubectl logs <gauth-pod> -n gauth-staging | grep -i "database connection"
   ```

4. **Prometheus scraping metrics**
   - Check Grafana PostgreSQL dashboard
   - Verify `pg_up` metric is 1

---

## Prevention

1. **Resource Management**
   - Set appropriate disk size (20Gi minimum)
   - Monitor disk usage proactively
   - Set up automatic PVC expansion
   - Configure connection pooling

2. **High Availability**
   - Deploy PostgreSQL with replication
   - Use PostgreSQL Operator for HA
   - Configure automatic failover
   - Maintain hot standby

3. **Backup & Recovery**
   - Daily automated backups
   - Test restore procedures monthly
   - Store backups in remote location
   - Document recovery procedures

4. **Monitoring**
   - Monitor connection count
   - Track query performance
   - Alert on disk usage trends
   - Monitor replication lag (if HA)

---

## Escalation

**Escalate if:**
- Cannot restore database within 15 minutes
- Data corruption detected
- Backup restore required
- Requires database administrator expertise

**Escalation Path:**
1. Senior Database Administrator
2. DevOps Team Lead
3. VP Engineering

**Contact Information:**
- DBA on-call: Check PagerDuty
- Slack: #gauth-database-incidents
- Email: dba-oncall@example.com

---

## Related Alerts

- `PostgreSQLHighConnectionPoolUsage` - Connection pool nearly full
- `PostgreSQLCriticalConnectionPoolUsage` - Connection pool exhausted
- `PostgreSQLSlowQueries` - Query performance issues
- `PostgreSQLHighDiskUsage` - Disk space issues

---

## Additional Resources

- [PostgreSQL Backup & Restore Guide](../postgresql-backup-restore.md)
- [Database Disaster Recovery](../DISASTER_RECOVERY_GUIDE.md)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)

---

**Last Updated:** November 10, 2025  
**Version:** 1.0  
**Owner:** Database Team
