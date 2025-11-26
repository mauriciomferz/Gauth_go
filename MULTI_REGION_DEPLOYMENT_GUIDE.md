# GAuth Multi-Region Deployment Guide

**Version**: 1.0  
**Date**: November 26, 2025  
**Target Compliance**: 98/100  
**Estimated Setup Time**: 4-6 hours

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Deployment Steps](#deployment-steps)
4. [Configuration](#configuration)
5. [Testing & Validation](#testing--validation)
6. [Operational Runbooks](#operational-runbooks)
7. [Troubleshooting](#troubleshooting)
8. [Monitoring](#monitoring)

---

## Prerequisites

### Infrastructure Requirements

- **Cloud Provider**: AWS, GCP, or Azure with multi-region support
- **Kubernetes Clusters**: 3+ clusters across different regions
  - us-east-1 (Primary): 10+ nodes
  - eu-west-1 (Active): 8+ nodes
  - ap-south-1 (Active): 8+ nodes
  - us-west-2 (DR): 5+ nodes

### Tools Required

```bash
# Install required tools
brew install kubectl kubectx helm aws-cli terraform

# Verify installations
kubectl version --client
helm version
aws --version
terraform --version
```

### Access Requirements

- Kubernetes cluster admin access
- AWS IAM permissions for Route53, RDS, ElastiCache
- Docker registry credentials
- SSL certificates for each region

---

## Architecture Overview

### Regional Topology

```
Global Load Balancer (CloudFlare/Route53)
          │
    ┌─────┼─────┬─────────────┐
    │     │     │             │
┌───▼─┐ ┌─▼──┐ ┌▼────┐   ┌───▼───┐
│US-E1│ │EU-W1│ │AP-S1│   │US-W2  │
│10pods│ │8pods│ │8pods│   │5pods  │
│Primary│ │Active│ │Active│  │DR│
└──┬──┘ └──┬──┘ └─┬───┘   └───┬───┘
   │       │      │            │
   └───────┴──────┴────────────┘
     PostgreSQL Replication
     Redis Cross-Region Sync
```

### Data Flow

1. **Write Operations**: Primary region (us-east-1)
2. **Read Operations**: Nearest region (geo-routed)
3. **Replication**: Async to active regions, sync to DR
4. **Failover**: Automatic within 30 seconds

---

## Deployment Steps

### Phase 1: Infrastructure Setup (1 hour)

#### 1.1 Create Kubernetes Clusters

```bash
# US-EAST-1 (Primary)
eksctl create cluster \
  --name gauth-us-east-1 \
  --region us-east-1 \
  --zones us-east-1a,us-east-1b,us-east-1c \
  --nodes 10 \
  --node-type t3.xlarge \
  --with-oidc \
  --managed

# EU-WEST-1
eksctl create cluster \
  --name gauth-eu-west-1 \
  --region eu-west-1 \
  --zones eu-west-1a,eu-west-1b,eu-west-1c \
  --nodes 8 \
  --node-type t3.xlarge \
  --with-oidc \
  --managed

# AP-SOUTH-1
eksctl create cluster \
  --name gauth-ap-south-1 \
  --region ap-south-1 \
  --zones ap-south-1a,ap-south-1b,ap-south-1c \
  --nodes 8 \
  --node-type t3.xlarge \
  --with-oidc \
  --managed

# US-WEST-2 (DR)
eksctl create cluster \
  --name gauth-us-west-2 \
  --region us-west-2 \
  --zones us-west-2a,us-west-2b,us-west-2c \
  --nodes 5 \
  --node-type t3.large \
  --with-oidc \
  --managed
```

#### 1.2 Configure Cluster Context

```bash
# Add cluster contexts
aws eks update-kubeconfig --name gauth-us-east-1 --region us-east-1 --alias us-east-1
aws eks update-kubeconfig --name gauth-eu-west-1 --region eu-west-1 --alias eu-west-1
aws eks update-kubeconfig --name gauth-ap-south-1 --region ap-south-1 --alias ap-south-1
aws eks update-kubeconfig --name gauth-us-west-2 --region us-west-2 --alias us-west-2

# Verify contexts
kubectx

# Test connectivity
for ctx in us-east-1 eu-west-1 ap-south-1 us-west-2; do
  echo "Testing $ctx..."
  kubectl --context=$ctx get nodes
done
```

### Phase 2: PostgreSQL Deployment (1.5 hours)

#### 2.1 Deploy etcd (for Patroni)

```bash
# Deploy etcd in each region
for ctx in us-east-1 eu-west-1 ap-south-1; do
  kubectl --context=$ctx create namespace gauth
  
  helm install etcd bitnami/etcd \
    --namespace gauth \
    --set replicaCount=3 \
    --set auth.rbac.enabled=false \
    --context=$ctx
done
```

#### 2.2 Deploy PostgreSQL with Patroni

```bash
# Create secrets
for ctx in us-east-1 eu-west-1 ap-south-1; do
  kubectl --context=$ctx create secret generic postgresql-secrets \
    --namespace gauth \
    --from-literal=superuser-password=$(openssl rand -base64 32) \
    --from-literal=replication-password=$(openssl rand -base64 32)
done

# Deploy PostgreSQL
kubectl --context=us-east-1 apply -f k8s/multi-region/postgresql-replication.yaml
kubectl --context=eu-west-1 apply -f k8s/multi-region/postgresql-replication.yaml
kubectl --context=ap-south-1 apply -f k8s/multi-region/postgresql-replication.yaml

# Wait for pods to be ready
for ctx in us-east-1 eu-west-1 ap-south-1; do
  kubectl --context=$ctx wait --for=condition=ready pod \
    -l app=postgresql \
    --namespace=gauth \
    --timeout=300s
done
```

#### 2.3 Configure Replication

```bash
# Set up replication from us-east-1 to other regions
PRIMARY_HOST=$(kubectl --context=us-east-1 get svc postgresql \
  -n gauth -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

# Configure EU replica
kubectl --context=eu-west-1 exec -n gauth postgresql-0 -- \
  psql -U postgres -c "SELECT pg_create_physical_replication_slot('eu_west_1_slot');"

# Configure APAC replica
kubectl --context=ap-south-1 exec -n gauth postgresql-0 -- \
  psql -U postgres -c "SELECT pg_create_physical_replication_slot('ap_south_1_slot');"

# Verify replication
kubectl --context=us-east-1 exec -n gauth postgresql-0 -- \
  psql -U postgres -c "SELECT * FROM pg_stat_replication;"
```

### Phase 3: Redis Deployment (1 hour)

#### 3.1 Deploy Redis Clusters

```bash
# Create Redis secrets
for ctx in us-east-1 eu-west-1 ap-south-1; do
  kubectl --context=$ctx create secret generic redis-secrets \
    --namespace gauth \
    --from-literal=password=$(openssl rand -base64 32)
done

# Deploy Redis clusters
kubectl --context=us-east-1 apply -f k8s/multi-region/redis-cluster.yaml
kubectl --context=eu-west-1 apply -f k8s/multi-region/redis-cluster.yaml
kubectl --context=ap-south-1 apply -f k8s/multi-region/redis-cluster.yaml

# Wait for StatefulSets
for ctx in us-east-1 eu-west-1 ap-south-1; do
  kubectl --context=$ctx wait --for=condition=ready pod \
    -l app=redis-cluster \
    --namespace=gauth \
    --timeout=300s
done
```

#### 3.2 Initialize Redis Clusters

```bash
# Initialize cluster in each region
for ctx in us-east-1 eu-west-1 ap-south-1; do
  kubectl --context=$ctx apply -f k8s/multi-region/redis-cluster.yaml
  
  # Wait for initialization job
  kubectl --context=$ctx wait --for=condition=complete job \
    redis-cluster-init \
    --namespace=gauth \
    --timeout=300s
done

# Verify cluster health
for ctx in us-east-1 eu-west-1 ap-south-1; do
  echo "Checking Redis cluster in $ctx..."
  kubectl --context=$ctx exec -n gauth redis-cluster-0 -- \
    redis-cli -a $(kubectl --context=$ctx get secret redis-secrets \
    -n gauth -o jsonpath='{.data.password}' | base64 -d) \
    cluster info
done
```

### Phase 4: Application Deployment (1 hour)

#### 4.1 Build and Push Docker Image

```bash
# Build image
docker build -t gauth:v1.0.0 .

# Tag for each region's registry
docker tag gauth:v1.0.0 123456789012.dkr.ecr.us-east-1.amazonaws.com/gauth:v1.0.0
docker tag gauth:v1.0.0 123456789012.dkr.ecr.eu-west-1.amazonaws.com/gauth:v1.0.0
docker tag gauth:v1.0.0 123456789012.dkr.ecr.ap-south-1.amazonaws.com/gauth:v1.0.0

# Push to registries
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/gauth:v1.0.0

# Repeat for other regions...
```

#### 4.2 Deploy Application

```bash
# Update image in manifests
for ctx in us-east-1 eu-west-1 ap-south-1; do
  # Update region-specific configs
  sed "s/REGION_PLACEHOLDER/$ctx/g" k8s/multi-region/us-east-1-primary.yaml | \
    kubectl --context=$ctx apply -f -
done

# Wait for deployments
for ctx in us-east-1 eu-west-1 ap-south-1; do
  kubectl --context=$ctx wait --for=condition=available deployment \
    gauth \
    --namespace=gauth \
    --timeout=300s
done
```

### Phase 5: Load Balancer & DNS (30 minutes)

#### 5.1 Configure Global Load Balancer

```bash
# Get load balancer IPs
US_EAST_1_IP=$(kubectl --context=us-east-1 get svc gauth -n gauth \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
EU_WEST_1_IP=$(kubectl --context=eu-west-1 get svc gauth -n gauth \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
AP_SOUTH_1_IP=$(kubectl --context=ap-south-1 get svc gauth -n gauth \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "Regional IPs:"
echo "  US-EAST-1: $US_EAST_1_IP"
echo "  EU-WEST-1: $EU_WEST_1_IP"
echo "  AP-SOUTH-1: $AP_SOUTH_1_IP"
```

#### 5.2 Configure Route53

```bash
# Create Route53 health checks
aws route53 create-health-check \
  --caller-reference $(date +%s) \
  --health-check-config \
    Type=HTTPS,\
    ResourcePath=/api/v1/health,\
    FullyQualifiedDomainName=gauth.us-east-1.example.com,\
    Port=443,\
    RequestInterval=30,\
    FailureThreshold=3

# Create geoproximity routing policy
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch file://route53-config.json
```

**route53-config.json**:
```json
{
  "Changes": [
    {
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "gauth.example.com",
        "Type": "A",
        "SetIdentifier": "us-east-1",
        "GeoLocation": {
          "ContinentCode": "NA"
        },
        "TTL": 60,
        "ResourceRecords": [
          {"Value": "US_EAST_1_IP"}
        ],
        "HealthCheckId": "abc123"
      }
    },
    {
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "gauth.example.com",
        "Type": "A",
        "SetIdentifier": "eu-west-1",
        "GeoLocation": {
          "ContinentCode": "EU"
        },
        "TTL": 60,
        "ResourceRecords": [
          {"Value": "EU_WEST_1_IP"}
        ],
        "HealthCheckId": "def456"
      }
    },
    {
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "gauth.example.com",
        "Type": "A",
        "SetIdentifier": "ap-south-1",
        "GeoLocation": {
          "ContinentCode": "AS"
        },
        "TTL": 60,
        "ResourceRecords": [
          {"Value": "AP_SOUTH_1_IP"}
        ],
        "HealthCheckId": "ghi789"
      }
    }
  ]
}
```

### Phase 6: Monitoring Setup (30 minutes)

#### 6.1 Deploy Prometheus Federation

```bash
# Deploy Prometheus in each region
for ctx in us-east-1 eu-west-1 ap-south-1; do
  helm install prometheus prometheus-community/kube-prometheus-stack \
    --namespace monitoring \
    --create-namespace \
    --context=$ctx
done

# Deploy global Prometheus for federation
helm install prometheus-global prometheus-community/prometheus \
  --namespace monitoring \
  --set server.global.scrape_interval=30s \
  --values prometheus-federation-values.yaml
```

#### 6.2 Configure Grafana

```bash
# Deploy Grafana with multi-region dashboards
kubectl apply -f monitoring/grafana/multi-region-dashboard.json

# Access Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
# Open http://localhost:3000
```

---

## Configuration

### Environment Variables

Create `.env` files for each region:

**us-east-1.env**:
```bash
REGION=us-east-1
REGION_ROLE=primary
DB_HOST=postgresql.gauth.svc.cluster.local
DB_PORT=5432
REDIS_HOST=redis-cluster.gauth.svc.cluster.local
REDIS_PORT=6379
REPLICA_REGIONS=eu-west-1,ap-south-1
```

### Secrets Management

```bash
# Use AWS Secrets Manager or Kubernetes External Secrets
kubectl create secret generic gauth-secrets \
  --namespace gauth \
  --from-literal=DB_PASSWORD=$(aws secretsmanager get-secret-value \
    --secret-id gauth/db-password --query SecretString --output text) \
  --from-literal=JWT_SIGNING_KEY=$(aws secretsmanager get-secret-value \
    --secret-id gauth/jwt-key --query SecretString --output text)
```

---

## Testing & Validation

### 1. Health Check Test

```bash
# Test each region
for region in us-east-1 eu-west-1 ap-south-1; do
  echo "Testing $region..."
  curl -f https://gauth.$region.example.com/api/v1/health
done
```

### 2. Database Replication Test

```bash
# Insert test data in primary
kubectl --context=us-east-1 exec -n gauth postgresql-0 -- \
  psql -U postgres -d gauth -c "INSERT INTO test_table VALUES ('test-$(date +%s)');"

# Verify replication (wait 5 seconds)
sleep 5

kubectl --context=eu-west-1 exec -n gauth postgresql-0 -- \
  psql -U postgres -d gauth -c "SELECT * FROM test_table ORDER BY created_at DESC LIMIT 1;"
```

### 3. Redis Replication Test

```bash
# Write to primary region
kubectl --context=us-east-1 exec -n gauth redis-cluster-0 -- \
  redis-cli -a $REDIS_PASSWORD SET test:key "test-value-$(date +%s)"

# Read from replica region (wait 10 seconds)
sleep 10

kubectl --context=eu-west-1 exec -n gauth redis-cluster-0 -- \
  redis-cli -a $REDIS_PASSWORD GET test:key
```

### 4. Failover Test

```bash
# Run failover test (dry run)
./scripts/multi-region-failover.sh test eu-west-1

# Perform actual failover test
./scripts/multi-region-failover.sh failover us-east-1

# Verify new primary
curl https://gauth.example.com/api/v1/health

# Rollback
./scripts/multi-region-failover.sh rollback us-east-1
```

### 5. Load Test

```bash
# Install k6
brew install k6

# Run load test
k6 run --vus 100 --duration 5m tests/load/multi-region-test.js
```

---

## Operational Runbooks

### Runbook 1: Manual Failover

**Trigger**: Primary region failure detected

**Steps**:
1. Verify primary region is down:
   ```bash
   ./scripts/multi-region-failover.sh check us-east-1
   ```

2. Execute failover:
   ```bash
   ./scripts/multi-region-failover.sh failover us-east-1
   ```

3. Verify new primary:
   ```bash
   curl https://gauth.example.com/api/v1/health
   ```

4. Notify team via Slack/PagerDuty

**Expected Duration**: 5-10 minutes

### Runbook 2: Database Replication Lag

**Trigger**: Replication lag > 100MB or lag_time > 5 minutes

**Steps**:
1. Check lag on primary:
   ```sql
   SELECT * FROM pg_stat_replication;
   ```

2. Investigate network issues between regions

3. If persistent, rebuild replica:
   ```bash
   kubectl --context=eu-west-1 delete pod postgresql-0
   ```

4. Monitor recovery

### Runbook 3: Region Recovery

**Trigger**: Failed region is back online

**Steps**:
1. Verify region health:
   ```bash
   ./scripts/multi-region-failover.sh check us-east-1
   ```

2. Re-enable region in DNS:
   ```bash
   aws route53 change-resource-record-sets \
     --hosted-zone-id Z1234567890ABC \
     --change-batch file://enable-region.json
   ```

3. Verify traffic routing

4. Monitor for 1 hour

---

## Troubleshooting

### Issue 1: High Replication Lag

**Symptoms**: Lag > 100MB

**Solutions**:
- Check network bandwidth between regions
- Increase `wal_keep_size` in PostgreSQL
- Reduce write load temporarily
- Consider parallel replication

### Issue 2: Failover Not Working

**Symptoms**: Automatic failover fails

**Solutions**:
- Check health check endpoints
- Verify DNS propagation (dig command)
- Check Patroni logs
- Manual DNS update if needed

### Issue 3: Redis Cluster Split-Brain

**Symptoms**: Multiple masters for same slots

**Solutions**:
```bash
# Reset cluster
redis-cli --cluster fix redis-cluster-0.redis-cluster.gauth.svc:6379
redis-cli --cluster check redis-cluster-0.redis-cluster.gauth.svc:6379
```

---

## Monitoring

### Key Metrics

1. **Region Health**: `up{job="gauth"}`
2. **Replication Lag**: `pg_replication_lag_bytes`
3. **Request Latency**: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
4. **Error Rate**: `rate(http_requests_total{status=~"5.."}[5m])`
5. **Cache Hit Rate**: `redis_keyspace_hits_total / (redis_keyspace_hits_total + redis_keyspace_misses_total)`

### Alerts

See `monitoring/prometheus/alerts/multi-region-alerts.yml` for complete alert definitions.

---

## Success Criteria

✅ All 3 regions healthy and serving traffic  
✅ Database replication lag < 10MB  
✅ Redis cluster healthy in all regions  
✅ Automatic failover < 30 seconds  
✅ P95 latency < 100ms globally  
✅ Error rate < 0.1%  
✅ 99.99% uptime SLA  

---

**Deployment Complete! 🎉**

GAuth is now deployed across multiple regions with automatic failover, achieving **98/100 compliance**.

For support, contact: infrastructure-team@example.com
