---
title: "AgentAuth Pod Down Alert Runbook"
category: runbook
status: active
lastUpdated: 2025-11-12
owners: devops-team
refreshCadence: monthly
---
# Alert Runbook: AgentAuthPodDown

**Alert Name:** `AgentAuthPodDown`  
**Severity:** Critical  
**Component:** Pod  
**Category:** Availability

---

## Summary

One or more AgentAuth pods have been down for more than 2 minutes, indicating a pod failure or crash.

---

## Alert Details

**Trigger Condition:**
```promql
up{job="agentauth-service"} == 0
```

**Duration:** 2 minutes  
**Impact:** Reduced service capacity, potential service degradation

---

## Immediate Actions (First 5 Minutes)

1. **Check Pod Status**
   ```bash
   kubectl get pods -n agentauth-staging -l app=agentauth
   kubectl describe pod <pod-name> -n agentauth-staging
   ```

2. **Check Pod Logs**
   ```bash
   # Current logs
   kubectl logs <pod-name> -n agentauth-staging --tail=100
   
   # Previous container logs (if pod restarted)
   kubectl logs <pod-name> -n agentauth-staging --previous
   ```

3. **Check Events**
   ```bash
   kubectl get events -n agentauth-staging --sort-by='.lastTimestamp' | grep <pod-name>
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
kubectl get pods -n agentauth-staging -l app=agentauth -o wide

# Describe pod for detailed state
kubectl describe pod <pod-name> -n agentauth-staging

# Check resource usage
kubectl top pods -n agentauth-staging

# Check node status
kubectl get nodes
kubectl describe node <node-name>

# Check deployment status
kubectl get deployment agentauth-blue -n agentauth-staging
kubectl describe deployment agentauth-blue -n agentauth-staging
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
kubectl rollout undo deployment/agentauth-blue -n agentauth-staging
kubectl rollout status deployment/agentauth-blue -n agentauth-staging
```

### Scenario 2: Resource Exhaustion

1. Check if pod was OOMKilled
   ```bash
   kubectl get pod <pod-name> -n agentauth-staging -o jsonpath='{.status.containerStatuses[0].lastState.terminated.reason}'
   ```

2. Increase resource limits if needed
   ```bash
   kubectl edit deployment agentauth-blue -n agentauth-staging
   # Increase memory/CPU limits
   ```

3. Scale up if resource constraints are cluster-wide
   ```bash
   # Add more nodes or scale existing pods
   kubectl scale deployment agentauth-blue -n agentauth-staging --replicas=5
   ```

### Scenario 3: Failed Health Checks

1. Check health endpoint directly
   ```bash
   kubectl port-forward <pod-name> -n agentauth-staging 8080:8080
   curl http://localhost:8080/api/v1/beta/health
   ```

2. Review health check configuration
   ```bash
   kubectl get deployment agentauth-blue -n agentauth-staging -o yaml | grep -A 10 livenessProbe
   ```

3. Adjust probe timing if startup is slow
   - Increase `initialDelaySeconds`
   - Increase `timeoutSeconds`
   - Adjust `failureThreshold`

### Scenario 4: Image Pull Issues

1. Check image pull status
   ```bash
   kubectl describe pod <pod-name> -n agentauth-staging | grep -A 5 "Events:"
   ```

2. Verify image exists and is accessible
   ```bash
   docker pull ghcr.io/mauriciomferz/agentauth-staging:latest
   ```

3. Check image pull secrets
   ```bash
   kubectl get secrets -n agentauth-staging
   ```

---

## Validation

After resolution, verify:

1. **Pod is running**
   ```bash
   kubectl get pods -n agentauth-staging -l app=agentauth
   ```

2. **Health checks passing**
   ```bash
   kubectl exec <pod-name> -n agentauth-staging -- curl -f http://localhost:8080/api/v1/beta/health
   ```

3. **Service endpoints updated**
   ```bash
   kubectl get endpoints agentauth-service -n agentauth-staging
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
- Slack channel: #agentauth-incidents
- Email: oncall@example.com

---

## Related Alerts

- `AgentAuthServiceUnavailable` - All pods down
- `AgentAuthPodRestartLoop` - Pod restarting repeatedly
- `AgentAuthCriticalMemory` - Memory exhaustion
- `AgentAuthPodNotReady` - Pod not passing readiness checks

---

## Additional Resources

- [AgentAuth Deployment Guide](../PRODUCTION_DEPLOYMENT_GUIDE.md)
- [Kubernetes Debugging Guide](https://kubernetes.io/docs/tasks/debug/)
- [AgentAuth Architecture Documentation](../README.md)

---

**Last Updated:** November 10, 2025  
**Version:** 1.0  
**Owner:** DevOps Team
