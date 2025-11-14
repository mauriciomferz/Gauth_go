---
title: "Redis Down Alert Runbook"
category: runbook
status: active
lastUpdated: 2025-11-12
owners: devops-team
refreshCadence: monthly
---
# Alert Runbook: RedisDown

**Alert Name:** `RedisDown`  
**Severity:** Critical  
**Component:** Redis  
**Category:** Redis

---

## Summary

Redis instance is down or unreachable, causing cache unavailability and potential performance degradation.

---

## Alert Details

**Trigger Condition:**
```promql
redis_up{namespace="gauth-staging"} == 0
```

**Duration:** 2 minutes  
**Impact:** Cache unavailable, increased database load, slower response times

---

## Immediate Actions (First 5 Minutes)

1. **Check Redis Pod Status**
   ```bash
   kubectl get pods -n gauth-staging -l app=redis
   kubectl describe pod <redis-pod> -n gauth-staging
   ```

2. **Check Redis Logs**
   ```bash
   kubectl logs <redis-pod> -n gauth-staging --tail=100
   kubectl logs <redis-pod> -n gauth-staging --previous
   ```

3. **Check Service and Endpoints**
   ```bash
   kubectl get svc redis -n gauth-staging
   kubectl get endpoints redis -n gauth-staging
   ```

---

## Diagnosis

### Common Causes

1. **Pod Crash**
   - Redis process crashed
   - OOM kill
   - Failed health checks

2. **Memory Issues**
   - Out of memory
   - Memory limit reached
   - High eviction rate

3. **Persistence Problems**
   - AOF corruption
   - RDB snapshot failure
   - Disk full

4. **Configuration Issues**
   - Invalid redis.conf
   - Network policy blocking access
   - Authentication failures

### Investigation Commands

```bash
# Check pod events
kubectl get events -n gauth-staging --field-selector involvedObject.name=<redis-pod>

# Check persistent volume
kubectl get pvc -n gauth-staging -l app=redis
kubectl describe pvc <pvc-name> -n gauth-staging

# Check memory usage
kubectl exec <redis-pod> -n gauth-staging -- redis-cli INFO memory

# Check Redis status
kubectl exec <redis-pod> -n gauth-staging -- redis-cli PING

# Check connected clients
kubectl exec <redis-pod> -n gauth-staging -- redis-cli CLIENT LIST
```

---

## Resolution Steps

### Scenario 1: Pod Crashed

1. **Check restart count**
   ```bash
   kubectl get pod <redis-pod> -n gauth-staging -o jsonpath='{.status.containerStatuses[0].restartCount}'
   ```

2. **Review crash logs**
   ```bash
   kubectl logs <redis-pod> -n gauth-staging --previous | tail -50
   ```

3. **Restart pod**
   ```bash
   kubectl delete pod <redis-pod> -n gauth-staging
   kubectl get pods -n gauth-staging -w
   ```

### Scenario 2: Memory Exhaustion

1. **Check memory usage**
   ```bash
   kubectl exec <redis-pod> -n gauth-staging -- redis-cli INFO memory | grep used_memory_human
   ```

2. **Clear unnecessary keys** (if safe)
   ```bash
   # Get key statistics
   kubectl exec <redis-pod> -n gauth-staging -- redis-cli --bigkeys
   
   # Flush specific database (CAUTION)
   kubectl exec <redis-pod> -n gauth-staging -- redis-cli FLUSHDB
   ```

3. **Increase memory limit**
   ```bash
   kubectl edit statefulset redis -n gauth-staging
   # Increase memory limits and maxmemory config
   ```

### Scenario 3: AOF Corruption

1. **Check AOF status**
   ```bash
   kubectl exec <redis-pod> -n gauth-staging -- cat /data/appendonly.aof
   ```

2. **Fix AOF file**
   ```bash
   kubectl exec <redis-pod> -n gauth-staging -- redis-check-aof --fix /data/appendonly.aof
   ```

3. **Restart Redis**
   ```bash
   kubectl delete pod <redis-pod> -n gauth-staging
   ```

### Scenario 4: Restore from Snapshot

If data is corrupted and recent backup exists:

```bash
# Stop Redis
kubectl scale statefulset redis -n gauth-staging --replicas=0

# Restore RDB file
./scripts/restore-redis-snapshot.sh <snapshot-date>

# Start Redis
kubectl scale statefulset redis -n gauth-staging --replicas=1
```

---

## Validation

1. **Redis is running**
   ```bash
   kubectl get pod <redis-pod> -n gauth-staging
   ```

2. **Can connect to Redis**
   ```bash
   kubectl exec <redis-pod> -n gauth-staging -- redis-cli PING
   # Should return: PONG
   ```

3. **Check key count**
   ```bash
   kubectl exec <redis-pod> -n gauth-staging -- redis-cli DBSIZE
   ```

4. **GAuth can use cache**
   ```bash
   kubectl logs <gauth-pod> -n gauth-staging | grep -i "redis connection"
   ```

5. **Metrics being collected**
   - Check Grafana Redis dashboard
   - Verify `redis_up` metric is 1

---

## Impact Assessment

**Cache Miss Scenario:**
- Database load will increase
- Response times will be slower
- No data loss (stateless cache)
- Service remains functional

**Degraded Performance:**
- Monitor database connection pool
- Watch for increased query latency
- May need to scale database temporarily

---

## Prevention

1. **Resource Management**
   - Set appropriate memory limits
   - Configure maxmemory-policy (allkeys-lru)
   - Monitor memory usage trends
   - Set up alerts for memory thresholds

2. **High Availability**
   - Deploy Redis Sentinel for HA
   - Use Redis Cluster for distributed cache
   - Configure replication
   - Maintain hot standby

3. **Persistence & Backup**
   - Enable AOF persistence
   - Schedule RDB snapshots
   - Store backups remotely
   - Test restore procedures

4. **Monitoring**
   - Track cache hit rate
   - Monitor eviction rate
   - Alert on high memory usage
   - Track connection count

---

## Escalation

**Escalate if:**
- Cannot restore Redis within 10 minutes
- Data corruption suspected
- Affecting production traffic
- Requires cache architecture changes

**Escalation Path:**
1. Senior DevOps Engineer
2. Infrastructure Team Lead
3. VP Engineering

**Contact Information:**
- On-call: Check PagerDuty
- Slack: #gauth-redis-incidents
- Email: infra-oncall@example.com

---

## Related Alerts

- `RedisHighMemoryUsage` - Memory approaching limit
- `RedisCriticalMemoryUsage` - Memory critically high
- `RedisHighEvictionRate` - High eviction rate
- `RedisLowCacheHitRate` - Poor cache performance

---

## Additional Resources

- [Redis Backup & Restore Guide](../redis-backup-restore.md)
- [Cache Strategy Documentation](../cache-strategy.md)
- [Redis Documentation](https://redis.io/documentation)

---

**Last Updated:** November 10, 2025  
**Version:** 1.0  
**Owner:** Infrastructure Team
