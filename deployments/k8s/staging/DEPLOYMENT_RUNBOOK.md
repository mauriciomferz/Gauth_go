# GAuth Staging Deployment Runbook
# Week 4 Day 1: Environment Setup & Deployment Procedures

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Infrastructure Setup](#infrastructure-setup)
3. [Secrets Management](#secrets-management)
4. [Deployment Procedure](#deployment-procedure)
5. [Verification & Smoke Testing](#verification--smoke-testing)
6. [Rollback Procedure](#rollback-procedure)
7. [Troubleshooting](#troubleshooting)
8. [Monitoring & Alerts](#monitoring--alerts)

---

## Prerequisites

### Required Tools
- `kubectl` v1.28+ (Kubernetes CLI)
- `helm` v3.12+ (Kubernetes package manager)
- `docker` v24+ (Container runtime)
- `openssl` (for key generation)
- `jq` (for JSON processing)

### Required Access
- Kubernetes cluster with RBAC enabled
- Docker registry access (for pushing images)
- DNS management (for domain configuration)
- Slack webhook URL (for alerting)

### Cluster Requirements
- **Kubernetes Version**: 1.28+
- **Node Count**: Minimum 3 nodes (for HA)
- **Node Resources**: 4 CPU, 8GB RAM per node
- **Storage Class**: Support for dynamic provisioning (e.g., `gp3` on AWS)
- **Ingress Controller**: NGINX Ingress Controller installed
- **Cert Manager**: cert-manager v1.13+ for TLS certificates

---

## Infrastructure Setup

### Step 1: Install NGINX Ingress Controller
```bash
# Add NGINX Ingress Helm repository
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

# Install NGINX Ingress Controller
helm install nginx-ingress ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.service.type=LoadBalancer \
  --set controller.metrics.enabled=true \
  --set controller.metrics.serviceMonitor.enabled=true
```

### Step 2: Install cert-manager
```bash
# Add Jetstack Helm repository
helm repo add jetstack https://charts.jetstack.io
helm repo update

# Install cert-manager CRDs
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.crds.yaml

# Install cert-manager
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.13.0
```

### Step 3: Create ClusterIssuer for Let's Encrypt
```bash
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: YOUR_EMAIL@yourdomain.com  # REPLACE
    privateKeySecretRef:
      name: letsencrypt-staging
    solvers:
    - http01:
        ingress:
          class: nginx
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: YOUR_EMAIL@yourdomain.com  # REPLACE
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

### Step 4: Install Metrics Server (for HPA)
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

---

## Secrets Management

### Generate JWT RSA Keys
```bash
# Generate RSA private key (2048-bit)
openssl genrsa -out jwt-private-key.pem 2048

# Extract public key
openssl rsa -in jwt-private-key.pem -pubout -out jwt-public-key.pem

# Verify keys
openssl rsa -in jwt-private-key.pem -text -noout
```

### Generate Ed25519 Signing Keys
```bash
# Generate Ed25519 key pair
openssl genpkey -algorithm ED25519 -out ed25519-key1.pem

# Extract private key bytes (hex format)
PRIVATE_HEX=$(openssl pkey -in ed25519-key1.pem -text -noout | grep 'priv:' -A 3 | tail -n 3 | tr -d ' \n:')

# Format for GAUTH_ROTATIONS_V2_ED25519_KEYS: kid:hex_private_key
echo "demo-key-1:${PRIVATE_HEX}"
```

### Create Kubernetes Secrets
```bash
# Create namespace first
kubectl create namespace gauth-staging

# Create PostgreSQL secrets
kubectl create secret generic postgres-secrets \
  --namespace=gauth-staging \
  --from-literal=postgres-password='REPLACE_WITH_STRONG_PASSWORD' \
  --from-literal=app-password='REPLACE_WITH_APP_PASSWORD'

# Create Redis secrets
kubectl create secret generic redis-secrets \
  --namespace=gauth-staging \
  --from-literal=redis-password='REPLACE_WITH_REDIS_PASSWORD'

# Create GAuth secrets
kubectl create secret generic gauth-secrets \
  --namespace=gauth-staging \
  --from-literal=postgres-password='REPLACE_WITH_APP_PASSWORD' \
  --from-file=jwt-private-key=jwt-private-key.pem \
  --from-file=jwt-public-key=jwt-public-key.pem \
  --from-literal=ed25519-keys='demo-key-1:YOUR_HEX_PRIVATE_KEY' \
  --from-literal=redis-password='REPLACE_WITH_REDIS_PASSWORD' \
  --from-literal=slack-webhook-url='REPLACE_WITH_SLACK_WEBHOOK'

# Create basic auth for Grafana ingress
htpasswd -c auth admin
kubectl create secret generic grafana-basic-auth \
  --namespace=gauth-staging \
  --from-file=auth

# Create basic auth for Prometheus ingress
htpasswd -c auth admin
kubectl create secret generic prometheus-basic-auth \
  --namespace=gauth-staging \
  --from-file=auth

# Clean up local files
rm -f jwt-private-key.pem jwt-public-key.pem ed25519-key1.pem auth
```

---

## Deployment Procedure

### Step 1: Build and Push Docker Image
```bash
# Navigate to project root
cd /Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go

# Build Docker image (multi-stage, production-ready)
docker build -t gauth:staging -f Dockerfile .

# Tag for registry (adjust for your registry)
docker tag gauth:staging YOUR_REGISTRY/gauth:staging
docker tag gauth:staging YOUR_REGISTRY/gauth:$(git rev-parse --short HEAD)

# Push to registry
docker push YOUR_REGISTRY/gauth:staging
docker push YOUR_REGISTRY/gauth:$(git rev-parse --short HEAD)
```

### Step 2: Update Deployment Manifest
```bash
# Update image reference in deployment.yaml
sed -i '' "s|image: gauth:staging|image: YOUR_REGISTRY/gauth:staging|g" \
  deployments/k8s/staging/deployment.yaml

# Update domain names in ingress.yaml
sed -i '' "s|gauth-staging.yourdomain.com|gauth-staging.ACTUAL_DOMAIN.com|g" \
  deployments/k8s/staging/ingress.yaml
```

### Step 3: Deploy to Kubernetes
```bash
# Apply namespace, resource quotas, and limit ranges
kubectl apply -f deployments/k8s/staging/namespace.yaml

# Apply ConfigMaps
kubectl apply -f deployments/k8s/staging/configmap.yaml

# Apply Services (before deployments for DNS resolution)
kubectl apply -f deployments/k8s/staging/service.yaml

# Apply RBAC, NetworkPolicy, HPA, PDB
kubectl apply -f deployments/k8s/staging/hpa-pdb-rbac-netpol.yaml

# Apply deployments (GAuth, PostgreSQL, Redis, Prometheus, Grafana, AlertManager)
kubectl apply -f deployments/k8s/staging/deployment.yaml
kubectl apply -f deployments/k8s/staging/postgres-deployment.yaml
kubectl apply -f deployments/k8s/staging/redis-deployment.yaml
kubectl apply -f deployments/k8s/staging/prometheus-deployment.yaml

# Apply Ingress (last, after services are ready)
kubectl apply -f deployments/k8s/staging/ingress.yaml

# Wait for rollout completion
kubectl rollout status deployment/gauth-deployment -n gauth-staging --timeout=5m
```

### Step 4: Verify Deployment
```bash
# Check pod status
kubectl get pods -n gauth-staging

# Expected output:
# NAME                                READY   STATUS    RESTARTS   AGE
# gauth-deployment-xxxxxxxxxx-xxxxx   1/1     Running   0          2m
# gauth-deployment-xxxxxxxxxx-xxxxx   1/1     Running   0          2m
# gauth-deployment-xxxxxxxxxx-xxxxx   1/1     Running   0          2m
# postgres-deployment-xxxxxxxx-xxxxx  1/1     Running   0          2m
# redis-deployment-xxxxxxxxxx-xxxxx   1/1     Running   0          2m

# Check logs
kubectl logs -n gauth-staging deployment/gauth-deployment --tail=50

# Check services
kubectl get svc -n gauth-staging

# Check ingress
kubectl get ingress -n gauth-staging
```

---

## Verification & Smoke Testing

### Health Check Endpoint
```bash
# Get ingress URL
INGRESS_URL=$(kubectl get ingress gauth-ingress -n gauth-staging -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

# Test health endpoint
curl -k https://${INGRESS_URL}/healthz

# Expected response: {"status":"healthy"}

# Test beta health endpoint
curl -k https://${INGRESS_URL}/api/v1/beta/health

# Expected response: {"status":"ok","timestamp":"..."}
```

### Metrics Endpoint
```bash
# Test Prometheus metrics
curl -k https://${INGRESS_URL}/metrics | grep gauth_

# Expected: Prometheus metrics in text format
```

### Basic Authorization Flow
```bash
# Test token issuance (example)
curl -k -X POST https://${INGRESS_URL}/api/v1/tokens \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "user@example.com",
    "capabilities": ["read", "write"],
    "duration": "1h"
  }'

# Expected: JWT token in response
```

### Database Connectivity
```bash
# Port-forward to PostgreSQL
kubectl port-forward -n gauth-staging svc/postgres-service 5432:5432 &

# Test connection
PGPASSWORD='YOUR_APP_PASSWORD' psql -h localhost -U gauth_app -d gauth -c "SELECT 1;"

# Expected: (1 row)
```

### Redis Connectivity
```bash
# Port-forward to Redis
kubectl port-forward -n gauth-staging svc/redis-service 6379:6379 &

# Test connection
redis-cli -h localhost -p 6379 -a 'YOUR_REDIS_PASSWORD' PING

# Expected: PONG
```

---

## Rollback Procedure

### Quick Rollback (Previous Revision)
```bash
# Rollback deployment to previous revision
kubectl rollout undo deployment/gauth-deployment -n gauth-staging

# Check rollback status
kubectl rollout status deployment/gauth-deployment -n gauth-staging
```

### Rollback to Specific Revision
```bash
# List deployment history
kubectl rollout history deployment/gauth-deployment -n gauth-staging

# Rollback to specific revision
kubectl rollout undo deployment/gauth-deployment -n gauth-staging --to-revision=3

# Verify rollback
kubectl describe deployment gauth-deployment -n gauth-staging
```

### Complete Environment Teardown
```bash
# Delete all resources in staging namespace
kubectl delete namespace gauth-staging

# WARNING: This deletes all data, secrets, and configurations!
```

---

## Troubleshooting

### Pod Not Starting
```bash
# Describe pod
kubectl describe pod -n gauth-staging -l app=gauth

# Check events
kubectl get events -n gauth-staging --sort-by='.lastTimestamp'

# Check logs
kubectl logs -n gauth-staging -l app=gauth --tail=100 --follow
```

### Database Connection Issues
```bash
# Check PostgreSQL logs
kubectl logs -n gauth-staging -l app=postgres --tail=50

# Test connectivity from GAuth pod
kubectl exec -it -n gauth-staging deployment/gauth-deployment -- sh
# Inside pod:
nc -zv postgres-service 5432
```

### Ingress Not Working
```bash
# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller --tail=100

# Check certificate status
kubectl describe certificate -n gauth-staging gauth-tls

# Check ingress configuration
kubectl get ingress gauth-ingress -n gauth-staging -o yaml
```

### High Memory/CPU Usage
```bash
# Check resource usage
kubectl top pods -n gauth-staging

# Check HPA status
kubectl get hpa -n gauth-staging gauth-hpa

# Manually scale if needed
kubectl scale deployment gauth-deployment -n gauth-staging --replicas=5
```

---

## Monitoring & Alerts

### Access Grafana Dashboard
```bash
# Get Grafana URL
kubectl get ingress grafana-ingress -n gauth-staging

# Open in browser: https://grafana-staging.yourdomain.com
# Default credentials: admin / admin (change immediately)
```

### Access Prometheus
```bash
# Get Prometheus URL
kubectl get ingress prometheus-ingress -n gauth-staging

# Open in browser: https://prometheus-staging.yourdomain.com
```

### Key Metrics to Monitor
1. **Pod Health**:
   - `kube_pod_status_phase{namespace="gauth-staging"}`
   
2. **Request Rate**:
   - `rate(http_requests_total{namespace="gauth-staging"}[5m])`
   
3. **Error Rate**:
   - `rate(http_requests_total{namespace="gauth-staging",status=~"5.."}[5m])`
   
4. **Latency (p95)**:
   - `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
   
5. **Memory Usage**:
   - `container_memory_usage_bytes{namespace="gauth-staging",pod=~"gauth-.*"}`
   
6. **CPU Usage**:
   - `rate(container_cpu_usage_seconds_total{namespace="gauth-staging",pod=~"gauth-.*"}[5m])`

### Alert Triggers
1. **Pod Down**: Any GAuth pod unavailable > 2 minutes
2. **High Error Rate**: 5xx errors > 5% for 5 minutes
3. **High Latency**: p95 latency > 1s for 5 minutes
4. **High CPU**: CPU usage > 80% for 10 minutes
5. **High Memory**: Memory usage > 85% for 10 minutes
6. **Database Connection Failures**: > 10 failures in 5 minutes
7. **Envelope Digest Mismatch**: > 3 mismatches in 10 minutes (from Week 3 security controls)

---

## Next Steps (Week 4 Days 2-5)

1. **Day 2-3**: CI/CD Pipeline Setup
   - GitHub Actions workflow for automated builds
   - Automated testing before deployment
   - Blue-green deployment strategy
   
2. **Day 4**: Smoke Testing Suite
   - Automated health checks
   - Authorization flow validation
   - RFC 0111/0115 compliance tests
   
3. **Day 5-7**: Performance Validation
   - Load testing (1000 req/s baseline)
   - Latency profiling (p50, p95, p99)
   - Resource optimization
   
4. **Day 8-10**: Production Cutover Plan
   - Production deployment runbook
   - Rollback procedures
   - Post-deployment monitoring

---

## Contact & Escalation

- **Primary Contact**: Platform Engineering Team
- **Escalation Path**: DevOps Lead → Engineering Manager → CTO
- **On-Call Schedule**: PagerDuty rotation (configure)
- **Documentation**: Confluence page (link)
- **Slack Channels**: 
  - `#gauth-staging-alerts` (automated alerts)
  - `#gauth-support` (team communication)
  - `#gauth-critical` (critical incidents)

---

**Document Version**: 1.0  
**Last Updated**: Week 4 Day 1 (November 9, 2025)  
**Author**: GAuth Platform Engineering Team  
**Status**: Production-Ready for Staging Deployment
