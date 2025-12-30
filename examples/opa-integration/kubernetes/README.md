---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Kubernetes Deployment Guide

This directory contains production-ready Kubernetes manifests for deploying AgentAuth with OPA sidecar.

## Architecture

```
┌─────────────────────────────────────┐
│         Kubernetes Pod              │
│                                     │
│  ┌──────────────┐  ┌─────────────┐  │
│  │    AgentAuth     │  │     OPA     │  │
│  │  Container   │←→│   Sidecar   │  │
│  │   :8080      │  │    :8181    │  │
│  └──────────────┘  └─────────────┘  │
│         ↑                ↑          │
│         │                │          │
└─────────┼────────────────┼──────────┘
          │                │
          │                │
    ┌─────┴────────┐  ┌────┴─────────┐
    │   Service    │  │  ConfigMap   │
    │   :80→:8080  │  │  (Policies)  │
    └──────────────┘  └──────────────┘
```

## Files

- `opa-configmap.yaml` - OPA Rego policies
- `deployment.yaml` - AgentAuth + OPA sidecar deployment
- `secrets-example.yaml` - Secret template (DO NOT commit real secrets)
- `monitoring.yaml` - Prometheus ServiceMonitor and alerting rules
- `hpa.yaml` - Horizontal Pod Autoscaler

## Prerequisites

1. **Kubernetes cluster** (v1.24+)
   ```bash
   kubectl version --short
   ```

2. **Namespace**
   ```bash
   kubectl create namespace gauth-system
   ```

3. **Secrets** (create before deploying)
   ```bash
   # JWT signing key
   kubectl create secret generic gauth-secrets \
     --from-literal=jwt-signing-key="$(openssl rand -base64 32)" \
     -n gauth-system

   # Database credentials
   kubectl create secret generic gauth-db-secrets \
     --from-literal=username=gauth_admin \
     --from-literal=password="YOUR_DB_PASSWORD" \
     -n gauth-system
   ```

## Deployment Steps

### 1. Deploy OPA Policies
```bash
kubectl apply -f opa-configmap.yaml
```

**Verify:**
```bash
kubectl get configmap opa-policies -n gauth-system
kubectl describe configmap opa-policies -n gauth-system
```

### 2. Deploy AgentAuth with OPA Sidecar
```bash
kubectl apply -f deployment.yaml
```

**Verify:**
```bash
# Check deployment status
kubectl get deployment gauth-with-opa -n gauth-system

# Check pods
kubectl get pods -n gauth-system -l app=gauth

# Check both containers are running
kubectl get pods -n gauth-system -o jsonpath='{.items[*].status.containerStatuses[*].name}'
```

### 3. Setup Monitoring (Optional)
```bash
kubectl apply -f monitoring.yaml
```

### 4. Setup Autoscaling (Optional)
```bash
kubectl apply -f hpa.yaml
```

## Verification

### 1. Check Pod Health
```bash
# All containers running
kubectl get pods -n gauth-system -l app=gauth

# Expected output:
# NAME                              READY   STATUS    RESTARTS
# gauth-with-opa-xxxxx-yyyyy        2/2     Running   0
```

### 2. Check OPA Health
```bash
# Port-forward to OPA
kubectl port-forward -n gauth-system \
  svc/gauth 8181:8181

# Health check
curl http://localhost:8181/health

# Expected: {"status":"ok"}
```

### 3. Test Policy Evaluation
```bash
# Port-forward if not already done
kubectl port-forward -n gauth-system svc/gauth 8181:8181

# Test scope validation
curl -X POST http://localhost:8181/v1/data/gauth/authz/allow \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "action": "validate_scope",
      "parent_scopes": ["users:*"],
      "child_scopes": ["users:read"]
    }
  }'

# Expected: {"result":true}
```

### 4. Check AgentAuth Application
```bash
# Port-forward to AgentAuth
kubectl port-forward -n gauth-system svc/gauth 8080:80

# Health check
curl http://localhost:8080/health

# Expected: {"status":"healthy","opa_enabled":true}
```

## Updating Policies

### Hot-Reload (No Downtime)
```bash
# Edit ConfigMap
kubectl edit configmap opa-policies -n gauth-system

# Trigger rolling update
kubectl rollout restart deployment gauth-with-opa -n gauth-system

# Watch rollout
kubectl rollout status deployment gauth-with-opa -n gauth-system
```

### Verify New Policy
```bash
# Check ConfigMap version
kubectl get configmap opa-policies -n gauth-system \
  -o jsonpath='{.metadata.resourceVersion}'

# Check if pods picked up new policy
kubectl logs -n gauth-system \
  -l app=gauth -c opa --tail=50 | grep "Bundle loaded"
```

## Scaling

### Manual Scaling
```bash
# Scale to 5 replicas
kubectl scale deployment gauth-with-opa \
  -n gauth-system --replicas=5

# Verify
kubectl get pods -n gauth-system -l app=gauth
```

### Auto-Scaling
```bash
# Apply HPA
kubectl apply -f hpa.yaml

# Check HPA status
kubectl get hpa gauth-hpa -n gauth-system

# Watch scaling events
kubectl get hpa gauth-hpa -n gauth-system -w
```

## Monitoring

### Logs

**AgentAuth container:**
```bash
kubectl logs -n gauth-system -l app=gauth -c gauth --tail=100 -f
```

**OPA container:**
```bash
kubectl logs -n gauth-system -l app=gauth -c opa --tail=100 -f
```

**Both containers:**
```bash
kubectl logs -n gauth-system -l app=gauth --all-containers=true -f
```

### Metrics

If Prometheus is installed:
```bash
# Port-forward to Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090

# Open browser
open http://localhost:9090

# Query OPA metrics:
# - opa_http_request_duration_seconds
# - opa_policy_evaluation_duration_seconds
```

### Events
```bash
kubectl get events -n gauth-system --sort-by='.lastTimestamp'
```

## Troubleshooting

### Pod Not Starting

**Check pod status:**
```bash
kubectl describe pod -n gauth-system -l app=gauth
```

**Common issues:**
- **ImagePullBackOff**: Check image name and registry access
- **CrashLoopBackOff**: Check container logs
- **Pending**: Check resource requests vs cluster capacity

### OPA Policy Errors

**Check OPA logs:**
```bash
kubectl logs -n gauth-system -l app=gauth -c opa
```

**Common issues:**
- **Syntax error in Rego**: Validate policy with `opa test` locally
- **Policy not loaded**: Check ConfigMap mount
- **Evaluation error**: Check input format matches policy expectations

### AgentAuth-OPA Connection Issues

**Test from AgentAuth container:**
```bash
kubectl exec -n gauth-system -it \
  $(kubectl get pod -n gauth-system -l app=gauth -o name | head -n1) \
  -c gauth -- sh

# Inside container:
wget -qO- http://localhost:8181/health
```

**Common issues:**
- **Connection refused**: OPA container not ready
- **Timeout**: Check network policy or resource limits
- **404**: Check OPA is running with `--server` flag

### Performance Issues

**Check resource usage:**
```bash
kubectl top pods -n gauth-system -l app=gauth
```

**Common issues:**
- **CPU throttling**: Increase CPU limits
- **OOM kills**: Increase memory limits
- **High latency**: Check OPA policy complexity, consider caching

### Policy Not Taking Effect

**Verify ConfigMap version:**
```bash
# Get ConfigMap revision
kubectl get configmap opa-policies -n gauth-system \
  -o jsonpath='{.metadata.resourceVersion}'

# Get pod's ConfigMap revision
kubectl get pod -n gauth-system -l app=gauth \
  -o jsonpath='{.items[0].spec.volumes[?(@.name=="opa-policies")].configMap.name}'
```

**Force restart:**
```bash
kubectl rollout restart deployment gauth-with-opa -n gauth-system
```

## Security Best Practices

1. **Use NetworkPolicy** to restrict pod communication:
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: gauth-netpol
     namespace: gauth-system
   spec:
     podSelector:
       matchLabels:
         app: gauth
     policyTypes:
     - Ingress
     - Egress
     ingress:
     - from:
       - podSelector: {}
       ports:
       - protocol: TCP
         port: 8080
     egress:
     - to:
       - podSelector: {}
       ports:
       - protocol: TCP
         port: 8181
     - to:
       - namespaceSelector:
           matchLabels:
             name: database
       ports:
       - protocol: TCP
         port: 5432
   ```

2. **Rotate secrets regularly**:
   ```bash
   kubectl create secret generic gauth-secrets \
     --from-literal=jwt-signing-key="$(openssl rand -base64 32)" \
     --dry-run=client -o yaml | kubectl apply -f -
   
   kubectl rollout restart deployment gauth-with-opa -n gauth-system
   ```

3. **Use PodSecurityPolicy** or **Pod Security Standards**:
   ```bash
   kubectl label namespace gauth-system \
     pod-security.kubernetes.io/enforce=restricted
   ```

4. **Enable audit logging**:
   ```yaml
   # In OPA container args:
   - "--set=decision_logs.console=true"
   ```

## Cleanup

### Remove deployment
```bash
kubectl delete -f deployment.yaml
kubectl delete -f opa-configmap.yaml
```

### Remove namespace (CAUTION: deletes all resources)
```bash
kubectl delete namespace gauth-system
```

## Production Checklist

- [ ] Secrets created and rotated regularly
- [ ] Resource requests/limits configured
- [ ] Liveness/readiness probes tested
- [ ] PodDisruptionBudget configured (minAvailable: 1)
- [ ] HPA configured for auto-scaling
- [ ] Monitoring and alerting setup
- [ ] NetworkPolicy applied
- [ ] Pod security standards enforced
- [ ] Backup and disaster recovery plan
- [ ] Load testing completed
- [ ] Rollback procedure documented

## Further Reading

- [Kubernetes OPA Best Practices](https://www.openpolicyagent.org/docs/latest/kubernetes-primer/)
- [OPA High Availability](https://www.openpolicyagent.org/docs/latest/management-high-availability/)
- [AgentAuth OPA Integration Guide](../../../docs/OPA_INTEGRATION_GUIDE.md)
