---
title: "AgentAuth Service Unavailable Alert Runbook"
category: runbook
status: active
lastUpdated: 2025-11-12
owners: devops-team
refreshCadence: monthly
---
# Alert Runbook: AgentAuthServiceUnavailable

**Alert Name:** `AgentAuthServiceUnavailable`  
**Severity:** Critical  
**Component:** Service  
**Category:** Availability

---

## Summary

All AgentAuth pods are down, making the entire service unavailable. This is a **SEVERE OUTAGE** requiring immediate attention.

---

## Alert Details

**Trigger Condition:**
```promql
sum(up{job="agentauth-service"}) == 0
```

**Duration:** 1 minute  
**Impact:** Complete service outage, all users affected

---

## Immediate Actions (First 2 Minutes)

⚠️ **THIS IS A P0 INCIDENT - NOTIFY LEADERSHIP IMMEDIATELY**

1. **Declare Incident**
   ```bash
   # Post to Slack
   /incident declare "AgentAuth Complete Outage - All Pods Down"
   ```

2. **Quick Status Check**
   ```bash
   kubectl get pods -n agentauth-staging -l app=agentauth
   kubectl get deployment agentauth-blue -n agentauth-staging
   kubectl get nodes
   ```

3. **Check Service Status**
   ```bash
   kubectl get svc agentauth-service -n agentauth-staging
   kubectl get endpoints agentauth-service -n agentauth-staging
   ```

---

## Diagnosis

### Common Causes

1. **Deployment Failure**
   - Bad deployment rolled out
   - Image pull failure
   - Configuration error

2. **Cluster Issues**
   - Node failures
   - Network partition
   - Control plane issues

3. **Resource Exhaustion**
   - Cluster out of resources
   - Namespace quota exceeded
   - Storage issues

4. **External Dependencies**
   - Database down
   - Redis down
   - Critical API unavailable

### Investigation Commands

```bash
# Check all resources
kubectl get all -n agentauth-staging

# Check recent events
kubectl get events -n agentauth-staging --sort-by='.lastTimestamp' | head -20

# Check deployment rollout status
kubectl rollout status deployment/agentauth-blue -n agentauth-staging

# Check replica set
kubectl get rs -n agentauth-staging

# Check pod details
kubectl describe pods -n agentauth-staging -l app=agentauth

# Check node status
kubectl get nodes -o wide
kubectl top nodes
```

---

## Resolution Steps

### Priority 1: Quick Recovery (0-5 minutes)

**Option A: Rollback Deployment**
```bash
# Rollback to last known good version
kubectl rollout undo deployment/agentauth-blue -n agentauth-staging
kubectl rollout status deployment/agentauth-blue -n agentauth-staging
```

**Option B: Scale Up from Zero**
```bash
# Force scale up
kubectl scale deployment agentauth-blue -n agentauth-staging --replicas=3
kubectl get pods -n agentauth-staging -w
```

**Option C: Restart Deployment**
```bash
kubectl rollout restart deployment/agentauth-blue -n agentauth-staging
kubectl rollout status deployment/agentauth-blue -n agentauth-staging
```

### Priority 2: Emergency Fallback (5-10 minutes)

If quick recovery fails, deploy emergency fallback:

```bash
# Deploy minimal service with health check only
kubectl apply -f k8s-emergency-fallback.yaml

# Or use previous known-good image
kubectl set image deployment/agentauth-blue \
  agentauth=ghcr.io/mauriciomferz/agentauth-staging:<previous-tag> \
  -n agentauth-staging
```

### Priority 3: Root Cause Analysis (10+ minutes)

Once service is restored, investigate root cause:

```bash
# Review recent changes
git log --oneline -10

# Check application logs from failed pods
kubectl logs -l app=agentauth -n agentauth-staging --previous --tail=200

# Review Prometheus metrics before outage
# Open Grafana and check metrics 15 minutes before alert

# Check cluster events
kubectl get events --all-namespaces --sort-by='.lastTimestamp' | head -50
```

---

## Validation

1. **Pods Running**
   ```bash
   kubectl get pods -n agentauth-staging -l app=agentauth
   # Should show 3/3 running pods
   ```

2. **Service Endpoints**
   ```bash
   kubectl get endpoints agentauth-service -n agentauth-staging
   # Should show multiple IP addresses
   ```

3. **Health Check**
   ```bash
   kubectl run test-curl --rm -i --image=curlimages/curl --restart=Never \
     -n agentauth-staging -- curl -f http://agentauth-service/api/v1/beta/health
   ```

4. **Load Test**
   ```bash
   # Run quick load test to verify capacity
   kubectl run load-test --rm -i --image=williamyeh/wrk --restart=Never \
     -n agentauth-staging -- wrk -t2 -c10 -d30s http://agentauth-service/api/v1/beta/health
   ```

---

## Communication Template

### Initial Notification (Within 5 minutes)
```
🚨 INCIDENT: AgentAuth Complete Outage
Status: Investigating
Impact: All AgentAuth services unavailable
Start Time: [TIME]
ETA: TBD
Actions: [Current actions being taken]
```

### Update Notification (Every 15 minutes)
```
📊 UPDATE: AgentAuth Outage
Status: [Investigating/Mitigating/Resolved]
Progress: [What's been done]
Next Steps: [What's next]
ETA: [Updated ETA]
```

### Resolution Notification
```
✅ RESOLVED: AgentAuth Outage
Duration: [Total time]
Root Cause: [Brief explanation]
Resolution: [What fixed it]
Follow-up: [Post-mortem scheduled for DATE]
```

---

## Post-Incident Actions

1. **Schedule Post-Mortem**
   - Within 24-48 hours
   - Include all stakeholders
   - Use blameless post-mortem template

2. **Document Timeline**
   - Record all actions taken
   - Note decision points
   - Capture lessons learned

3. **Create Action Items**
   - Preventive measures
   - Detection improvements
   - Response procedure updates

4. **Update Runbook**
   - Add new findings
   - Update resolution steps
   - Improve validation procedures

---

## Prevention

1. **Deployment Safety**
   - Implement blue-green deployments
   - Use canary deployments for risky changes
   - Add pre-deployment validation gates
   - Require staging validation before production

2. **Monitoring Improvements**
   - Add deployment health checks
   - Monitor rollout progress
   - Alert on replica count drops
   - Track deployment success rate

3. **Resilience**
   - Implement PodDisruptionBudget (minAvailable=2)
   - Add anti-affinity rules
   - Use multiple availability zones
   - Maintain hot standby environment

4. **Testing**
   - Regular chaos engineering exercises
   - Test rollback procedures monthly
   - Validate disaster recovery plan
   - Load test all deployments

---

## Escalation

**IMMEDIATE ESCALATION REQUIRED**

This is a P0 incident - escalate immediately to:

1. **Incident Commander** (on-call DevOps lead)
2. **Development Team Lead**
3. **VP Engineering**
4. **CTO** (if outage > 15 minutes)

**Contact Methods:**
- PagerDuty: High urgency page
- Slack: #agentauth-incidents (mention @oncall)
- Phone: Use emergency contact list

---

## Related Alerts

- `AgentAuthPodDown` - Individual pod failures
- `AgentAuthPodRestartLoop` - Pods crash looping
- `PostgreSQLDown` - Database unavailable
- `RedisDown` - Cache unavailable

---

## Additional Resources

- [Incident Response Plan](../INCIDENT_RESPONSE_PLAN.md)
- [Disaster Recovery Guide](../DISASTER_RECOVERY_GUIDE.md)
- [Post-Mortem Template](../POST_MORTEM_TEMPLATE.md)

---

**Last Updated:** November 10, 2025  
**Version:** 1.0  
**Owner:** DevOps Team  
**Reviewed By:** CTO
