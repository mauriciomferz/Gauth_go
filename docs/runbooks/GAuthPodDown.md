---
title: "GAuth Pod Down Alert Runbook"
category: runbook
status: active
lastUpdated: 2025-11-12
owners: devops-team
refreshCadence: monthly
---
# Alert Runbook: GAuthPodDown

**Alert Name:** `GAuthPodDown`  
**Severity:** Critical  
**Component:** Pod  
**Category:** Availability

---

## Summary

One or more GAuth pods have been down for more than 2 minutes, indicating a pod failure or crash.

---

## Alert Details

**Trigger Condition:**
```promql
up{job="gauth-service"} == 0
```

**Duration:** 2 minutes  
**Impact:** Reduced service capacity, potential service degradation

---

## Immediate Actions (First 5 Minutes)

1. **Check Pod Status**
   ```bash
   kubectl get pods -n gauth-staging -l app=gauth
   kubectl describe pod <pod-name> -n gauth-staging
   ```

2. **Check Pod Logs**
   ```bash
   # Current logs
   kubectl logs <pod-name> -n gauth-staging --tail=100
   
   # Previous container logs (if pod restarted)
   kubectl logs <pod-name> -n gauth-staging --previous
   ```

3. **Check Events**
   ```bash
   kubectl get events -n gauth-staging --sort-by='.lastTimestamp' | grep <pod-name>
   ```

---

## Diagnosis

### Common Causes

1. **Application Crash**
   - Check logs for panic, fatal errors, or exceptions
   - Look for OOM (Out of Memory) kills in events
   - Review recent code deployments

2. **Failed Health Checks**
   - Liveness probe failing
   - Readiness probe failing
   - Application startup timeout

3. **Resource Constraints**
   - Node resources exhausted (CPU/Memory)
   - Pod evicted due to resource pressure
   - Disk space issues

4. **Configuration Issues**
   - Invalid environment variables
   - Missing secrets or ConfigMaps
   - Invalid image pull (check ImagePullBackOff)

### Investigation Commands

```bash
# Check pod status and restarts
kubectl get pods -n gauth-staging -l app=gauth -o wide

# Describe pod for detailed state
kubectl describe pod <pod-name> -n gauth-staging

# Check resource usage
kubectl top pods -n gauth-staging

# Check node status
kubectl get nodes
kubectl describe node <node-name>

# Check deployment status
kubectl get deployment gauth-blue -n gauth-staging
kubectl describe deployment gauth-blue -n gauth-staging
```

---

## Resolution Steps

### Scenario 1: Application Crash

1. Review application logs for root cause
2. If known bug, deploy hotfix
3. If unknown, collect logs and escalate to development team
4. Consider rolling back to previous version if necessary

```bash
# Rollback deployment
kubectl rollout undo deployment/gauth-blue -n gauth-staging
kubectl rollout status deployment/gauth-blue -n gauth-staging
```

### Scenario 2: Resource Exhaustion

1. Check if pod was OOMKilled
   ```bash
   kubectl get pod <pod-name> -n gauth-staging -o jsonpath='{.status.containerStatuses[0].lastState.terminated.reason}'
   ```

2. Increase resource limits if needed
   ```bash
   kubectl edit deployment gauth-blue -n gauth-staging
   # Increase memory/CPU limits
   ```

3. Scale up if resource constraints are cluster-wide
   ```bash
   # Add more nodes or scale existing pods
   kubectl scale deployment gauth-blue -n gauth-staging --replicas=5
   ```

### Scenario 3: Failed Health Checks

1. Check health endpoint directly
   ```bash
   kubectl port-forward <pod-name> -n gauth-staging 8080:8080
   curl http://localhost:8080/api/v1/beta/health
   ```

2. Review health check configuration
   ```bash
   kubectl get deployment gauth-blue -n gauth-staging -o yaml | grep -A 10 livenessProbe
   ```

3. Adjust probe timing if startup is slow
   - Increase `initialDelaySeconds`
   - Increase `timeoutSeconds`
   - Adjust `failureThreshold`

### Scenario 4: Image Pull Issues

1. Check image pull status
   ```bash
   kubectl describe pod <pod-name> -n gauth-staging | grep -A 5 "Events:"
   ```

2. Verify image exists and is accessible
   ```bash
   docker pull ghcr.io/mauriciomferz/gauth-staging:latest
   ```

3. Check image pull secrets
   ```bash
   kubectl get secrets -n gauth-staging
   ```

---

## Validation

After resolution, verify:

1. **Pod is running**
   ```bash
   kubectl get pods -n gauth-staging -l app=gauth
   ```

2. **Health checks passing**
   ```bash
   kubectl exec <pod-name> -n gauth-staging -- curl -f http://localhost:8080/api/v1/beta/health
   ```

3. **Service endpoints updated**
   ```bash
   kubectl get endpoints gauth-service -n gauth-staging
   ```

4. **Metrics being collected**
   - Check Grafana dashboard for pod metrics
   - Verify Prometheus targets are up

---

## Prevention

1. **Improve monitoring**
   - Set up pre-alerts for high resource usage
   - Monitor application error logs proactively

2. **Enhance resilience**
   - Implement circuit breakers
   - Add retry logic with exponential backoff
   - Improve graceful shutdown handling

3. **Resource management**
   - Set appropriate resource requests and limits
   - Use HPA for automatic scaling
   - Monitor resource trends over time

4. **Testing**
   - Load test new deployments before production
   - Test failure scenarios in staging
   - Validate health check configurations

---

## Escalation

**Escalate if:**
- Pod continues to crash after multiple restart attempts
- Root cause cannot be identified within 15 minutes
- Multiple pods are affected (may indicate broader issue)
- Requires code changes or hotfix deployment

**Escalation Path:**
1. Senior DevOps Engineer
2. Development Team Lead
3. CTO / VP Engineering

**Contact Information:**
- On-call engineer: Check PagerDuty rotation
- Slack channel: #gauth-incidents
- Email: oncall@example.com

---

## Related Alerts

- `GAuthServiceUnavailable` - All pods down
- `GAuthPodRestartLoop` - Pod restarting repeatedly
- `GAuthCriticalMemory` - Memory exhaustion
- `GAuthPodNotReady` - Pod not passing readiness checks

---

## Additional Resources

- [GAuth Deployment Guide](../PRODUCTION_DEPLOYMENT_GUIDE.md)
- [Kubernetes Debugging Guide](https://kubernetes.io/docs/tasks/debug/)
- [GAuth Architecture Documentation](../README.md)

---

**Last Updated:** November 10, 2025  
**Version:** 1.0  
**Owner:** DevOps Team
