# GAuth Production Deployment Guide

**Project**: GAuth Enterprise IAM Platform  
**Version**: 1.0.0  
**Date**: November 15, 2025  
**Status**: Production Ready ✅

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Environment Configuration](#environment-configuration)
4. [Deployment Methods](#deployment-methods)
5. [Monitoring and Observability](#monitoring-and-observability)
6. [Rollback Procedures](#rollback-procedures)
7. [Troubleshooting](#troubleshooting)
8. [Security Considerations](#security-considerations)
9. [Operational Runbook](#operational-runbook)

---

## Overview

### Architecture

GAuth consists of two main components:

1. **Backend Service** (Go)
   - REST API server on port 8080
   - Prometheus metrics endpoint
   - PostgreSQL database for persistence
   - Redis for caching

2. **Frontend Application** (React)
   - Static SPA served by Nginx on port 80/443
   - Proxies API requests to backend
   - Production-optimized bundle

### Deployment Options

- **Docker Compose**: Quick deployment on single host
- **Kubernetes**: Scalable production deployment
- **Manual**: Traditional server deployment

---

## Prerequisites

### Infrastructure Requirements

#### Minimum Specifications (Small Scale)
- **CPU**: 2 cores
- **RAM**: 4 GB
- **Disk**: 20 GB SSD
- **Network**: 100 Mbps

#### Recommended Specifications (Production)
- **CPU**: 4-8 cores
- **RAM**: 8-16 GB
- **Disk**: 50-100 GB SSD
- **Network**: 1 Gbps

### Software Requirements

- Docker 24+ or Kubernetes 1.28+
- PostgreSQL 16+
- Redis 7+
- TLS certificates (Let's Encrypt or commercial)

### Secrets Management

Required secrets:
```bash
GAUTH_JWT_SIGNING_KEY          # JWT signing key (256-bit min)
GAUTH_DB_PASSWORD              # PostgreSQL password
REDIS_PASSWORD                 # Redis password
GRAFANA_ADMIN_PASSWORD         # Grafana admin password
```

**⚠️ CRITICAL**: Never commit secrets to version control

---

## Environment Configuration

### 1. Backend Configuration

Copy and customize the production environment file:

```bash
cp .env.production .env
```

Edit `.env` with your production values:

```bash
# Core settings
GAUTH_ENV=production
GAUTH_CORS_ALLOW=https://your-domain.com

# Security (CHANGE THESE!)
GAUTH_JWT_SIGNING_KEY=your-secure-256-bit-key-here

# Database
GAUTH_DB_HOST=your-postgres-host
GAUTH_DB_USER=gauth_prod
GAUTH_DB_PASSWORD=your-secure-db-password
GAUTH_DB_NAME=gauth_production

# TLS
GAUTH_TLS_ENABLED=true
GAUTH_TLS_CERT_PATH=/etc/gauth/tls/cert.pem
GAUTH_TLS_KEY_PATH=/etc/gauth/tls/key.pem
```

### 2. Frontend Configuration

Edit `web/ui-react/.env.production`:

```bash
# API endpoint (your backend URL)
VITE_API_BASE_URL=https://api.your-domain.com/api/v1

# Environment
VITE_ENV=production

# Analytics (optional)
VITE_ANALYTICS_ENABLED=true
VITE_ANALYTICS_ID=your-analytics-id
```

### 3. Database Setup

Initialize PostgreSQL database:

```sql
CREATE USER gauth_prod WITH PASSWORD 'your-secure-password';
CREATE DATABASE gauth_production OWNER gauth_prod;
GRANT ALL PRIVILEGES ON DATABASE gauth_production TO gauth_prod;
```

Run migrations:

```bash
psql -U gauth_prod -d gauth_production -f schema/init.sql
```

---

## Deployment Methods

### Method 1: Docker Compose (Recommended for Single Host)

#### Step 1: Clone Repository

```bash
git clone https://github.com/mauriciomferz/Gauth_go.git
cd Gauth_go
```

#### Step 2: Configure Environment

```bash
# Backend
cp .env.production .env
# Edit .env with your values

# Frontend
cd web/ui-react
cp .env.production .env
# Edit .env with your values
cd ../..
```

#### Step 3: Build Images

```bash
# Build backend
docker build -f Dockerfile.production -t gauth-backend:latest .

# Build frontend
cd web/ui-react
docker build -f Dockerfile.production -t gauth-frontend:latest .
cd ../..
```

#### Step 4: Deploy Stack

```bash
# Create .env file for Docker Compose
cat > .env.docker <<EOF
GAUTH_JWT_SIGNING_KEY=your-key-here
GAUTH_DB_USER=gauth_prod
GAUTH_DB_PASSWORD=your-db-password
GAUTH_DB_NAME=gauth_production
REDIS_PASSWORD=your-redis-password
GRAFANA_ADMIN_PASSWORD=your-grafana-password
GAUTH_CORS_ALLOW=https://your-domain.com
VITE_API_BASE_URL=https://api.your-domain.com/api/v1
EOF

# Start services
docker-compose -f docker-compose.production.yml --env-file .env.docker up -d
```

#### Step 5: Verify Deployment

```bash
# Check services status
docker-compose -f docker-compose.production.yml ps

# Check backend health
curl http://localhost:8080/health

# Check frontend
curl http://localhost:3000/health

# View logs
docker-compose -f docker-compose.production.yml logs -f
```

---

### Method 2: Kubernetes Deployment

#### Step 1: Create Namespace

```bash
kubectl create namespace gauth-production
```

#### Step 2: Create Secrets

```bash
# Create secrets from files
kubectl create secret generic gauth-secrets \
  --from-literal=jwt-signing-key=your-key \
  --from-literal=db-password=your-db-password \
  --from-literal=redis-password=your-redis-password \
  -n gauth-production

# TLS certificates
kubectl create secret tls gauth-tls \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem \
  -n gauth-production
```

#### Step 3: Deploy Database

```bash
kubectl apply -f k8s-postgres.yaml -n gauth-production
```

#### Step 4: Deploy Redis

```bash
kubectl apply -f k8s-redis.yaml -n gauth-production
```

#### Step 5: Deploy Backend

```yaml
# k8s-backend.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gauth-backend
  namespace: gauth-production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gauth-backend
  template:
    metadata:
      labels:
        app: gauth-backend
    spec:
      containers:
      - name: backend
        image: ghcr.io/mauriciomferz/gauth_go-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: GAUTH_JWT_SIGNING_KEY
          valueFrom:
            secretKeyRef:
              name: gauth-secrets
              key: jwt-signing-key
        - name: GAUTH_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: gauth-secrets
              key: db-password
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          exec:
            command:
            - /app/web-server
            - -healthcheck
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - /app/web-server
            - -healthcheck
          initialDelaySeconds: 5
          periodSeconds: 5
```

```bash
kubectl apply -f k8s-backend.yaml
```

#### Step 6: Deploy Frontend

```bash
kubectl apply -f k8s-frontend.yaml
```

#### Step 7: Deploy Ingress

```yaml
# k8s-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gauth-ingress
  namespace: gauth-production
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - your-domain.com
    - api.your-domain.com
    secretName: gauth-tls
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: gauth-frontend
            port:
              number: 80
  - host: api.your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: gauth-backend
            port:
              number: 8080
```

```bash
kubectl apply -f k8s-ingress.yaml
```

---

## Monitoring and Observability

### Prometheus Metrics

Backend metrics available at: `http://backend:8080/api/v1/beta/metrics`

Key metrics:
- `gauth_http_requests_total` - Total HTTP requests
- `gauth_http_request_duration_seconds` - Request duration
- `gauth_token_creation_total` - Token creations
- `gauth_authorization_checks_total` - Authorization checks
- `gauth_cache_hits_total` - Cache hit rate

### Grafana Dashboards

Access Grafana: `http://localhost:3001` (default: admin/admin)

Import dashboards from `grafana-dashboards/`

### Logging

#### Backend Logs

```bash
# Docker Compose
docker-compose logs -f backend

# Kubernetes
kubectl logs -f deployment/gauth-backend -n gauth-production
```

Log levels: `debug`, `info`, `warn`, `error`

Production default: `warn` (JSON format)

#### Frontend Logs

```bash
# Docker Compose
docker-compose logs -f frontend

# Kubernetes
kubectl logs -f deployment/gauth-frontend -n gauth-production
```

### Health Checks

```bash
# Backend health
curl https://api.your-domain.com/health

# Frontend health
curl https://your-domain.com/health

# Metrics health
curl https://api.your-domain.com/api/v1/beta/metrics
```

---

## Rollback Procedures

### Docker Compose Rollback

```bash
# Step 1: Stop current deployment
docker-compose -f docker-compose.production.yml down

# Step 2: Pull previous image version
docker pull ghcr.io/mauriciomferz/gauth_go-backend:v1.0.0
docker pull ghcr.io/mauriciomferz/gauth_go-frontend:v1.0.0

# Step 3: Tag as latest
docker tag ghcr.io/mauriciomferz/gauth_go-backend:v1.0.0 gauth-backend:latest
docker tag ghcr.io/mauriciomferz/gauth_go-frontend:v1.0.0 gauth-frontend:latest

# Step 4: Start services
docker-compose -f docker-compose.production.yml up -d
```

### Kubernetes Rollback

```bash
# Rollback backend
kubectl rollout undo deployment/gauth-backend -n gauth-production

# Rollback frontend
kubectl rollout undo deployment/gauth-frontend -n gauth-production

# Check rollout status
kubectl rollout status deployment/gauth-backend -n gauth-production
kubectl rollout status deployment/gauth-frontend -n gauth-production

# Rollback to specific revision
kubectl rollout undo deployment/gauth-backend --to-revision=2 -n gauth-production
```

### Database Rollback

```bash
# Backup current database
pg_dump -U gauth_prod gauth_production > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore previous backup
psql -U gauth_prod -d gauth_production < backup_20251115_120000.sql
```

---

## Troubleshooting

### Common Issues

#### 1. Backend Won't Start

**Symptoms**: Container exits immediately

**Check**:
```bash
docker logs gauth-backend-prod
kubectl logs deployment/gauth-backend -n gauth-production
```

**Common causes**:
- Missing environment variables
- Database connection failure
- Invalid JWT signing key
- Port already in use

**Solutions**:
```bash
# Verify environment variables
docker exec gauth-backend-prod env | grep GAUTH

# Test database connection
docker exec gauth-backend-prod psql -h $GAUTH_DB_HOST -U $GAUTH_DB_USER -d $GAUTH_DB_NAME -c "SELECT 1"

# Check port availability
netstat -tuln | grep 8080
```

#### 2. Frontend Shows API Errors

**Symptoms**: "Network Error" or CORS errors

**Check**:
```bash
# Verify backend is accessible
curl -v http://backend:8080/health

# Check CORS configuration
docker exec gauth-backend-prod env | grep CORS
```

**Solutions**:
- Update `GAUTH_CORS_ALLOW` to include frontend domain
- Verify `VITE_API_BASE_URL` points to correct backend
- Check network connectivity between containers

#### 3. High Memory Usage

**Symptoms**: OOM kills, slow performance

**Check**:
```bash
# Docker
docker stats gauth-backend-prod

# Kubernetes
kubectl top pod -n gauth-production
```

**Solutions**:
- Increase memory limits
- Enable connection pooling
- Review slow queries
- Implement caching

#### 4. Database Connection Pool Exhausted

**Symptoms**: "too many connections" errors

**Check**:
```sql
SELECT count(*) FROM pg_stat_activity WHERE datname = 'gauth_production';
```

**Solutions**:
```bash
# Increase max connections
GAUTH_DB_MAX_CONNECTIONS=200

# Tune connection pool
GAUTH_DB_IDLE_CONNECTIONS=20
GAUTH_DB_CONNECTION_LIFETIME=3600
```

---

## Security Considerations

### 1. Secrets Management

**Do**:
- Use environment variables or secrets manager
- Rotate secrets regularly (weekly recommended)
- Use different secrets per environment
- Encrypt secrets at rest

**Don't**:
- Commit secrets to git
- Share secrets via email/chat
- Use default/example secrets
- Reuse secrets across services

### 2. Network Security

**Requirements**:
- TLS 1.3 for all external traffic
- Internal service mesh or VPN
- Firewall rules limiting port access
- DDoS protection at edge

**Implementation**:
```bash
# Enable TLS
GAUTH_TLS_ENABLED=true
GAUTH_TLS_MIN_VERSION=1.3

# Restrict CORS
GAUTH_CORS_ALLOW=https://your-domain.com

# Enable rate limiting
GAUTH_RATE_LIMIT_ENABLED=true
GAUTH_RATE_LIMIT_REQUESTS=100
```

### 3. Access Control

- Implement least privilege principle
- Use service accounts with minimal permissions
- Enable audit logging
- Regular access reviews

### 4. Monitoring Security Events

Alert on:
- High authentication failure rate
- Unusual traffic patterns
- Unexpected API errors
- Database connection spikes

---

## Operational Runbook

### Daily Operations

#### Morning Checks (5 minutes)
```bash
# 1. Check service status
docker-compose ps
# or
kubectl get pods -n gauth-production

# 2. Check for errors
docker-compose logs --tail=100 backend | grep ERROR
docker-compose logs --tail=100 frontend | grep ERROR

# 3. Verify metrics collection
curl http://localhost:9090/api/v1/query?query=up

# 4. Check disk space
df -h
```

#### Weekly Maintenance (30 minutes)
```bash
# 1. Review metrics and alerts
# - Check Grafana dashboards
# - Review alert history
# - Identify trends

# 2. Update dependencies
git pull origin main
docker-compose pull

# 3. Backup database
./scripts/backup-database.sh

# 4. Review logs for anomalies
# - High error rates
# - Slow queries
# - Memory leaks
```

#### Monthly Tasks (2 hours)
- Security patches
- Certificate renewal
- Performance review
- Capacity planning
- Access audit

### Emergency Procedures

#### Service Outage

1. **Assess Impact**
   ```bash
   # Check all services
   docker-compose ps
   kubectl get pods -n gauth-production
   ```

2. **Immediate Response**
   ```bash
   # Restart affected service
   docker-compose restart backend
   # or
   kubectl rollout restart deployment/gauth-backend -n gauth-production
   ```

3. **Investigate**
   ```bash
   # Check logs
   docker-compose logs --tail=500 backend
   
   # Check metrics
   # Visit Grafana dashboards
   ```

4. **Communicate**
   - Notify stakeholders
   - Update status page
   - Document incident

5. **Post-Mortem**
   - Root cause analysis
   - Preventive measures
   - Update runbook

---

## Performance Tuning

### Backend Optimization

```bash
# Connection pooling
GAUTH_DB_MAX_CONNECTIONS=100
GAUTH_DB_IDLE_CONNECTIONS=10

# Caching
GAUTH_CACHE_ENABLED=true
GAUTH_CACHE_TTL=300

# Rate limiting
GAUTH_RATE_LIMIT_REQUESTS=1000
GAUTH_RATE_LIMIT_WINDOW=60s
```

### Frontend Optimization

- Enable CDN for static assets
- Configure browser caching
- Enable gzip compression
- Optimize bundle size

### Database Optimization

```sql
-- Analyze query performance
EXPLAIN ANALYZE SELECT ...;

-- Create indexes
CREATE INDEX idx_subscriptions_client_id ON subscriptions(client_id);

-- Vacuum regularly
VACUUM ANALYZE;
```

---

## Backup and Recovery

### Automated Backups

```bash
#!/bin/bash
# scripts/backup-database.sh

BACKUP_DIR=/backups
DATE=$(date +%Y%m%d_%H%M%S)

# Backup PostgreSQL
pg_dump -U $GAUTH_DB_USER -h $GAUTH_DB_HOST $GAUTH_DB_NAME | \
  gzip > $BACKUP_DIR/gauth_db_$DATE.sql.gz

# Upload to S3
aws s3 cp $BACKUP_DIR/gauth_db_$DATE.sql.gz \
  s3://your-backup-bucket/gauth/

# Cleanup old backups (keep 30 days)
find $BACKUP_DIR -name "gauth_db_*.sql.gz" -mtime +30 -delete
```

### Recovery Procedure

```bash
# 1. Download backup
aws s3 cp s3://your-backup-bucket/gauth/gauth_db_20251115_120000.sql.gz .

# 2. Extract
gunzip gauth_db_20251115_120000.sql.gz

# 3. Restore
psql -U gauth_prod -d gauth_production < gauth_db_20251115_120000.sql

# 4. Verify
psql -U gauth_prod -d gauth_production -c "SELECT COUNT(*) FROM subscriptions;"
```

---

## Scaling Guidelines

### Horizontal Scaling

#### Backend
```bash
# Docker Compose (not recommended for production)
docker-compose up -d --scale backend=3

# Kubernetes
kubectl scale deployment gauth-backend --replicas=5 -n gauth-production
```

#### Auto-scaling (Kubernetes)
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gauth-backend-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gauth-backend
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Vertical Scaling

Increase resources based on metrics:
- **CPU**: If CPU usage > 70% sustained
- **Memory**: If memory usage > 80%
- **Disk**: If disk usage > 80%

---

## Support and Contacts

### Documentation
- **API Guide**: `API_INTEGRATION_GUIDE.md`
- **Phase 2 Summary**: `PHASE_2_COMPLETE_SUMMARY.md`
- **Architecture**: `ARCHITECTURE_SOLUTION.md`

### Repository
- **GitHub**: https://github.com/mauriciomferz/Gauth_go
- **Issues**: https://github.com/mauriciomferz/Gauth_go/issues

### Monitoring
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001
- **Backend Metrics**: http://localhost:8080/api/v1/beta/metrics

---

## Changelog

### Version 1.0.0 (November 15, 2025)
- Initial production deployment guide
- Docker Compose and Kubernetes deployment methods
- Monitoring and alerting setup
- Rollback procedures
- Security best practices
- Operational runbook

---

**Document Status**: ✅ Production Ready  
**Last Updated**: November 15, 2025  
**Maintained By**: GAuth DevOps Team
