---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Blue-Green Deployment Strategy for AgentAuth Staging

This directory contains manifests for implementing a blue-green deployment strategy, enabling zero-downtime deployments with instant rollback capability.

## Overview

Blue-green deployment maintains two identical production environments (blue and green). Only one serves live traffic at a time. When deploying a new version:

1. Deploy new version to inactive environment (e.g., green)
2. Run smoke tests against green environment
3. Switch traffic from blue to green
4. Keep blue environment as instant rollback target

## Architecture

```
                        ┌─────────────────┐
                        │  Ingress/LB     │
                        │  (Traffic       │
                        │   Switching)    │
                        └────────┬────────┘
                                 │
                 ┌───────────────┼───────────────┐
                 │               │               │
        ┌────────▼────────┐     │      ┌────────▼────────┐
        │  Blue Service   │     │      │ Green Service   │
        │  (selector:     │     │      │  (selector:     │
        │   version=blue) │     │      │   version=green)│
        └────────┬────────┘     │      └────────┬────────┘
                 │               │               │
        ┌────────▼────────┐     │      ┌────────▼────────┐
        │ Blue Deployment │     │      │Green Deployment │
        │  (3 replicas)   │     │      │  (3 replicas)   │
        │  version: v1.0  │     │      │  version: v1.1  │
        └─────────────────┘     │      └─────────────────┘
                                │
                       Active: Blue
                       Inactive: Green
```

## Files

1. **agentauth-deployment-blue.yaml**: Blue environment deployment
2. **agentauth-deployment-green.yaml**: Green environment deployment
3. **agentauth-service-blue.yaml**: Service for blue environment
4. **agentauth-service-green.yaml**: Service for green environment
5. **agentauth-ingress-bluegreen.yaml**: Ingress with traffic switching capability
6. **switch-traffic.sh**: Script to switch traffic between blue/green

## Deployment Procedure

### Initial Setup (First Time)
```bash
# 1. Deploy blue environment
kubectl apply -f agentauth-deployment-blue.yaml
kubectl apply -f agentauth-service-blue.yaml

# 2. Wait for blue to be ready
kubectl rollout status deployment/agentauth-deployment-blue -n agentauth-staging

# 3. Deploy ingress pointing to blue
kubectl apply -f agentauth-ingress-bluegreen.yaml

# 4. Verify blue is serving traffic
curl https://agentauth-staging.yourdomain.com/healthz
```

### Deploying New Version
```bash
# 1. Deploy to green environment (inactive)
kubectl apply -f agentauth-deployment-green.yaml
kubectl apply -f agentauth-service-green.yaml

# 2. Wait for green to be ready
kubectl rollout status deployment/agentauth-deployment-green -n agentauth-staging

# 3. Run smoke tests against green
./smoke-tests.sh green

# 4. Switch traffic to green
./switch-traffic.sh green

# 5. Monitor green environment
kubectl logs -n agentauth-staging -l app=agentauth,version=green --tail=100 -f

# 6. If successful, keep blue as rollback target
# If issues arise, run: ./switch-traffic.sh blue
```

### Rollback (Instant)
```bash
# Switch back to blue environment
./switch-traffic.sh blue

# Verify blue is serving traffic
curl https://agentauth-staging.yourdomain.com/healthz
```

### Cleanup Old Environment
```bash
# After green has been stable for 24+ hours, clean up blue
kubectl delete -f agentauth-deployment-blue.yaml
kubectl delete -f agentauth-service-blue.yaml

# Rename green to blue for next cycle
# (This makes green the new "stable" environment)
```

## Traffic Switching Methods

### Method 1: Ingress Annotation (Recommended)
Update ingress annotation to switch backend service:
```yaml
nginx.ingress.kubernetes.io/service-upstream: "agentauth-service-green"
```

### Method 2: Service Selector Update
Update main service selector to point to desired version:
```yaml
selector:
  app: agentauth
  version: green  # Change from 'blue' to 'green'
```

### Method 3: DNS/Load Balancer
Update DNS record or load balancer to point to green service endpoint.

## Monitoring During Switchover

```bash
# Watch pod status
watch kubectl get pods -n agentauth-staging -l app=agentauth

# Monitor error rate
kubectl logs -n agentauth-staging -l app=agentauth,version=green --tail=100 | grep ERROR

# Check metrics
curl https://agentauth-staging.yourdomain.com/metrics | grep agentauth_http_requests_total
```

## Advantages

1. **Zero Downtime**: New version deployed while old version serves traffic
2. **Instant Rollback**: Switch back to old version immediately
3. **Smoke Testing**: Test new version before exposing to users
4. **Risk Mitigation**: Old version remains running during switchover
5. **Simple Process**: Single traffic switch operation

## Disadvantages

1. **Resource Cost**: Requires 2x infrastructure (both environments running)
2. **Database Migrations**: Complex with schema changes (requires backward compatibility)
3. **Stateful Apps**: Session state may be lost during switch
4. **Cost**: Double the compute resources during deployment window

## Best Practices

1. **Always Test Green**: Run comprehensive smoke tests before switching
2. **Monitor Metrics**: Watch error rates and latency during switchover
3. **Gradual Switch**: Consider canary-style gradual traffic shift (10% → 50% → 100%)
4. **Database Compatibility**: Ensure new version compatible with old schema
5. **Keep Blue Running**: Maintain blue environment for at least 1 hour after switch
6. **Automated Health Checks**: Use liveness/readiness probes to validate green
7. **Rollback Plan**: Document rollback procedure and test regularly

## Security Considerations

1. Both environments use same secrets (mounted from Kubernetes Secrets)
2. Both environments share same PostgreSQL and Redis instances
3. Network policies apply to both blue and green deployments
4. Ingress TLS certificates shared between environments

## Integration with CI/CD

GitHub Actions workflow (`deploy-staging.yml`) can be extended to:
1. Deploy to inactive environment (blue or green)
2. Run smoke tests
3. Wait for manual approval
4. Switch traffic
5. Monitor for errors
6. Auto-rollback if error rate > threshold

---

**Status**: Ready for implementation  
**Updated**: Week 4 Day 2 (November 9, 2025)  
**Next Steps**: Implement in GitHub Actions workflow
