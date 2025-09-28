# GAuth Production Deployment Guide

This guide provides comprehensive instructions for deploying the GAuth Power-of-Attorney Protocol implementation to production environments.

## 📋 Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [CI/CD Pipeline](#cicd-pipeline)
- [Monitoring Stack](#monitoring-stack)
- [Database Configuration](#database-configuration)
- [Backup & Disaster Recovery](#backup--disaster-recovery)
- [Security Considerations](#security-considerations)
- [Performance Tuning](#performance-tuning)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

- **Kubernetes Cluster**: v1.25+ with at least 3 nodes
- **Storage**: 200GB+ of persistent storage with fast SSDs
- **Memory**: 8GB+ RAM available for GAuth services
- **CPU**: 4+ cores recommended for production workloads
- **Network**: Load balancer with SSL termination

### Required Tools

```bash
# Install required CLI tools
brew install kubectl helm docker
pip install awscli

# Verify installations
kubectl version --client
helm version
docker --version
aws --version
```

### Domain & SSL

- Domain name for the service (e.g., `gauth.company.com`)
- SSL certificate (Let's Encrypt or purchased)
- DNS management access

## Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/Gimel-Foundation/gauth.git
cd gauth

# Build Docker image
docker build -t ghcr.io/your-org/gauth:latest .

# Push to registry
docker push ghcr.io/your-org/gauth:latest
```

### 2. Configure Secrets

```bash
# Create namespace
kubectl create namespace gauth-production

# Generate and apply secrets
kubectl create secret generic gauth-secrets \
  --from-literal=vault-token="your-vault-token" \
  --from-literal=jwt-signing-key="$(openssl genrsa 2048 | base64 -w 0)" \
  -n gauth-production

kubectl create secret generic postgres-secrets \
  --from-literal=postgres-password="$(openssl rand -base64 32)" \
  --from-literal=gauth-password="$(openssl rand -base64 32)" \
  -n gauth-production

kubectl create secret generic redis-secrets \
  --from-literal=redis-password="$(openssl rand -base64 32)" \
  -n gauth-production
```

### 3. Deploy Infrastructure

```bash
# Deploy PostgreSQL
kubectl apply -f k8s/production/postgres-deployment.yaml

# Deploy Redis
kubectl apply -f k8s/production/redis-deployment.yaml

# Wait for databases to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n gauth-production --timeout=300s
kubectl wait --for=condition=ready pod -l app=redis -n gauth-production --timeout=300s

# Deploy GAuth application
kubectl apply -f k8s/production/gauth-deployment.yaml

# Wait for application to be ready
kubectl wait --for=condition=ready pod -l app=gauth -n gauth-production --timeout=300s
```

### 4. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n gauth-production

# Check services
kubectl get services -n gauth-production

# Test health endpoint
kubectl port-forward svc/gauth-service 8080:80 -n gauth-production &
curl http://localhost:8080/health
```

## CI/CD Pipeline

The project includes a comprehensive GitHub Actions workflow for automated testing and deployment.

### Setup GitHub Actions

1. **Configure Secrets** in your GitHub repository:
   ```
   GITHUB_TOKEN (automatic)
   DOCKERHUB_USERNAME
   DOCKERHUB_TOKEN
   KUBECONFIG (base64 encoded)
   SLACK_WEBHOOK (optional)
   ```

2. **Environment Configuration**:
   - Create `staging` and `production` environments in GitHub
   - Configure environment protection rules
   - Set up required reviewers for production deployments

3. **Workflow Features**:
   - Automated testing on all PRs
   - Security scanning with Gosec and Trivy
   - Docker image building and publishing
   - Staged deployments (staging → production)
   - Slack notifications

### Manual Deployment

```bash
# Deploy to staging
git push origin develop

# Deploy to production
git push origin main
```

## Monitoring Stack

The deployment includes a comprehensive monitoring stack with Prometheus, Grafana, and Jaeger.

### Deploy Monitoring

```bash
# Start monitoring stack
docker-compose up -d prometheus grafana jaeger alertmanager node-exporter

# Access dashboards
open http://localhost:3000  # Grafana (admin/admin)
open http://localhost:9090  # Prometheus
open http://localhost:16686 # Jaeger
```

### Key Metrics to Monitor

- **Application Metrics**:
  - Request rate and latency
  - Error rates by endpoint
  - Token issuance/revocation rates
  - Active sessions

- **Infrastructure Metrics**:
  - CPU and memory usage
  - Database connections and query performance
  - Redis cache hit rates
  - Network I/O

- **Security Metrics**:
  - Failed authentication attempts
  - Suspicious token usage patterns
  - Rate limiting triggers

### Alerting Rules

Create custom alerting rules in `monitoring/rules/`:

```yaml
# Example: High error rate alert
groups:
- name: gauth.rules
  rules:
  - alert: GAuthHighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "GAuth high error rate detected"
      description: "Error rate is {{ $value }} errors per second"
```

## Database Configuration

### PostgreSQL Production Setup

The PostgreSQL deployment includes:
- Persistent storage with fast SSDs
- Optimized configuration for OLTP workloads
- Automated backups
- Connection pooling
- SSL encryption

### Redis Configuration

The Redis deployment provides:
- Persistence with AOF and RDB
- Memory optimization
- Security hardening
- Performance tuning

### Connection Pooling

Consider using PgBouncer for PostgreSQL connection pooling:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pgbouncer
spec:
  template:
    spec:
      containers:
      - name: pgbouncer
        image: pgbouncer/pgbouncer:latest
        # Configuration here
```

## Backup & Disaster Recovery

### Automated Backups

The `scripts/backup-restore.sh` script provides comprehensive backup functionality:

```bash
# Perform full backup
./scripts/backup-restore.sh backup

# Test disaster recovery
./scripts/backup-restore.sh test

# Restore from backup
./scripts/backup-restore.sh restore-postgres backups/postgres/backup_file.sql.gz
```

### Backup Schedule

Set up automated backups with a Kubernetes CronJob:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: gauth-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: your-registry/gauth-backup:latest
            command: ["/scripts/backup-restore.sh", "backup"]
```

### Recovery Procedures

1. **Database Recovery**:
   ```bash
   # Stop application
   kubectl scale deployment gauth-deployment --replicas=0 -n gauth-production
   
   # Restore database
   ./scripts/backup-restore.sh restore-postgres latest_backup.sql.gz
   
   # Restart application
   kubectl scale deployment gauth-deployment --replicas=3 -n gauth-production
   ```

2. **Redis Recovery**:
   ```bash
   # Redis automatically recovers from AOF/RDB files
   kubectl delete pod -l app=redis -n gauth-production
   ```

3. **Full Disaster Recovery**:
   - Deploy infrastructure in new region
   - Restore databases from backups
   - Update DNS to point to new deployment
   - Verify all services are operational

## Security Considerations

### Network Security

- All inter-service communication uses TLS
- Network policies restrict pod-to-pod communication
- Ingress controller handles SSL termination
- Rate limiting at multiple layers

### Secret Management

- All secrets stored in Kubernetes secrets
- Integration with HashiCorp Vault for enhanced security
- Regular secret rotation
- Audit logging for secret access

### Security Scanning

The CI/CD pipeline includes:
- Static code analysis with Gosec
- Vulnerability scanning with Trivy
- Container image scanning
- Dependency checking

### Compliance

- GDPR compliance for user data
- SOC 2 Type II controls
- PCI DSS for payment data (if applicable)
- Regular security assessments

## Performance Tuning

### Application Tuning

```yaml
# GAuth deployment resource limits
resources:
  requests:
    memory: "256Mi"
    cpu: "200m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

### Database Tuning

PostgreSQL configuration optimizations:
- `shared_buffers = 256MB`
- `effective_cache_size = 1GB`
- `max_connections = 200`
- `work_mem = 4MB`

### Redis Tuning

Redis configuration optimizations:
- `maxmemory-policy allkeys-lru`
- `save 900 1` (persistence tuning)
- Connection pooling

### Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gauth-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gauth-deployment
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Troubleshooting

### Common Issues

1. **Pod Startup Issues**:
   ```bash
   kubectl describe pod <pod-name> -n gauth-production
   kubectl logs <pod-name> -n gauth-production
   ```

2. **Database Connection Issues**:
   ```bash
   kubectl exec -it <postgres-pod> -n gauth-production -- psql -U postgres -d gauth
   ```

3. **Redis Connection Issues**:
   ```bash
   kubectl exec -it <redis-pod> -n gauth-production -- redis-cli ping
   ```

### Health Checks

All services include comprehensive health checks:
- `/health` - Basic health check
- `/ready` - Readiness probe
- `/metrics` - Prometheus metrics

### Log Analysis

Centralized logging with structured JSON logs:
```bash
# View application logs
kubectl logs -f deployment/gauth-deployment -n gauth-production

# View audit logs
kubectl logs -f deployment/gauth-deployment -n gauth-production | grep "audit"
```

### Performance Monitoring

Use Grafana dashboards to monitor:
- Request latency percentiles
- Throughput metrics
- Error rates
- Resource utilization

## Support and Maintenance

### Regular Maintenance Tasks

- [ ] Weekly security updates
- [ ] Monthly performance reviews
- [ ] Quarterly disaster recovery tests
- [ ] Annual security audits

### Monitoring Checklist

- [ ] All services responding to health checks
- [ ] Database performance within SLA
- [ ] Error rates below threshold
- [ ] Backup completion verification
- [ ] Security alert review

### Escalation Procedures

1. **Critical Issues**: Page on-call engineer
2. **Major Issues**: Create incident ticket
3. **Minor Issues**: Standard support queue

For additional support, contact the GAuth team or refer to the project documentation.