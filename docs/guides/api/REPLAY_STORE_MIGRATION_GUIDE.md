# Replay Store Migration Guide

**Version:** 1.0  
**Date:** November 30, 2025  
**Security:** Addresses CV-2025-005 (BoltDB Ephemeral Storage Vulnerability)

---

## Executive Summary

This guide provides step-by-step instructions for migrating from BoltDB to Redis for replay protection in production deployments, particularly in containerized environments (Docker, Kubernetes, Podman).

**CRITICAL SECURITY ISSUE:** BoltDB is **UNSAFE** for containerized deployments due to ephemeral storage vulnerability (CV-2025-005). Container restarts wipe ephemeral storage, allowing replay attacks.

**Required Action:** All production deployments MUST migrate to Redis or use persistent volumes.

---

## Table of Contents

1. [Why Migrate?](#why-migrate)
2. [Migration Options](#migration-options)
3. [Option 1: Migrate to Redis (Recommended)](#option-1-migrate-to-redis-recommended)
4. [Option 2: BoltDB with Persistent Volumes](#option-2-boltdb-with-persistent-volumes)
5. [Kubernetes Deployment Examples](#kubernetes-deployment-examples)
6. [Docker Compose Examples](#docker-compose-examples)
7. [Verification and Testing](#verification-and-testing)
8. [Rollback Procedures](#rollback-procedures)
9. [FAQ](#faq)

---

## Why Migrate?

### The Vulnerability (CV-2025-005)

BoltDB stores data in a local file. In containerized environments, this file is typically stored in ephemeral storage that is **wiped on container restart**:

```
Timeline of Attack:
T=0     Container starts, BoltDB creates /tmp/replay.db
T=60s   User authenticates, JTI recorded in BoltDB
T=120s  Attacker captures authentication token (JTI: abc-123)
T=180s  Container restarts (auto-scaling, update, node failure)
T=181s  New container starts, /tmp/replay.db DOES NOT EXIST
T=200s  Attacker replays captured token → BoltDB accepts (no record)
        ❌ AUTHENTICATION BYPASS
```

### Impact

- **Authentication bypass** after any container restart
- **No replay protection** in cloud-native deployments
- **Data loss** on pod rescheduling
- **Security audit failures** in production

### Who Is Affected?

- ✅ **Kubernetes deployments** (pods restart frequently)
- ✅ **Docker containers** with auto-restart policies
- ✅ **Cloud-native applications** (ECS, Cloud Run, App Service)
- ✅ **Auto-scaling deployments** (horizontal pod autoscaling)
- ❌ **Bare metal/VM deployments** (not affected, but Redis still recommended)

---

## Migration Options

### Option 1: Migrate to Redis (Recommended) ⭐

**Pros:**
- ✅ Distributed replay protection across multiple instances
- ✅ Automatic TTL expiration (no manual cleanup)
- ✅ Production-ready high availability
- ✅ No persistent volume management needed
- ✅ Works seamlessly in containerized environments

**Cons:**
- ⚠️ Requires Redis infrastructure
- ⚠️ Additional operational complexity

**Use Cases:**
- Production deployments
- Multi-instance deployments
- Cloud-native applications
- High-availability requirements

---

### Option 2: BoltDB with Persistent Volumes

**Pros:**
- ✅ No external dependencies
- ✅ Simple configuration
- ✅ Lower operational overhead

**Cons:**
- ⚠️ Single instance only (file locking)
- ⚠️ Requires persistent volume management
- ⚠️ Manual backup/restore procedures needed
- ⚠️ Not suitable for horizontal scaling

**Use Cases:**
- Development/testing
- Single-instance deployments
- Internal/experimental systems

---

## Option 1: Migrate to Redis (Recommended)

### Step 1: Deploy Redis

#### Using Managed Redis (Recommended for Production)

**AWS ElastiCache:**
```bash
# Create Redis cluster
aws elasticache create-replication-group \
  --replication-group-id gauth-replay-store \
  --replication-group-description "AgentAuth replay protection" \
  --engine redis \
  --cache-node-type cache.t3.micro \
  --num-cache-clusters 2 \
  --automatic-failover-enabled
```

**Azure Cache for Redis:**
```bash
# Create Redis instance
az redis create \
  --name gauth-replay-store \
  --resource-group gauth-production \
  --location eastus \
  --sku Basic \
  --vm-size c0
```

**Google Cloud Memorystore:**
```bash
# Create Redis instance
gcloud redis instances create gauth-replay-store \
  --size=1 \
  --region=us-central1 \
  --tier=basic
```

#### Using Self-Hosted Redis in Kubernetes

```yaml
# redis-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        volumeMounts:
        - name: redis-data
          mountPath: /data
      volumes:
      - name: redis-data
        persistentVolumeClaim:
          claimName: redis-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: redis
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: redis-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

Apply the configuration:
```bash
kubectl apply -f redis-deployment.yaml
```

---

### Step 2: Configure Application to Use Redis

#### Environment Variables

Set the following environment variables in your deployment:

```bash
# Redis connection
REDIS_HOST=redis.default.svc.cluster.local  # For Kubernetes
REDIS_PORT=6379
REDIS_PASSWORD=your-secure-password         # If authentication enabled
REDIS_DB=0                                   # Database number (0-15)

# Replay store configuration
GAUTH_REPLAY_STORE=redis                    # Use Redis instead of BoltDB
GAUTH_REPLAY_TTL=3600                       # TTL in seconds (1 hour default)
```

#### For Managed Redis Services

**AWS ElastiCache:**
```bash
REDIS_HOST=gauth-replay-store.abc123.0001.use1.cache.amazonaws.com
REDIS_PORT=6379
REDIS_TLS=true  # ElastiCache supports TLS
```

**Azure Cache for Redis:**
```bash
REDIS_HOST=gauth-replay-store.redis.cache.windows.net
REDIS_PORT=6380
REDIS_PASSWORD=your-access-key
REDIS_TLS=true  # Azure Redis requires TLS
```

**Google Cloud Memorystore:**
```bash
REDIS_HOST=10.0.0.3  # Internal IP from Memorystore console
REDIS_PORT=6379
```

---

### Step 3: Update Application Code (If Needed)

If your application currently instantiates BoltDB directly, update to use Redis:

**Before (BoltDB):**
```go
import "github.com/mauriciomferz/Gauth_go/pkg/gauth"

// Old code - UNSAFE in containers
replayStore, err := gauth.NewBoltReplayStore("/tmp/replay.db", time.Hour)
if err != nil {
    log.Fatal(err)
}
```

**After (Redis):**
```go
import (
    "github.com/go-redis/redis/v8"
    "github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001"
)

// New code - Safe for containers
redisClient := redis.NewClient(&redis.Options{
    Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
    Password: os.Getenv("REDIS_PASSWORD"),
    DB:       0,
})

replayStore, err := gauth_rfc_001.NewRedisReplayStore(
    redisClient,
    "gauth:replay",  // Key prefix
    time.Hour,       // TTL
)
if err != nil {
    log.Fatal(err)
}
```

---

### Step 4: Deploy and Verify

1. **Deploy updated configuration:**
   ```bash
   kubectl apply -f k8s/gauth-deployment.yaml
   ```

2. **Verify Redis connectivity:**
   ```bash
   kubectl exec -it gauth-pod -- /bin/sh
   
   # Test Redis connection
   redis-cli -h redis -p 6379 ping
   # Expected: PONG
   ```

3. **Check application logs:**
   ```bash
   kubectl logs -f deployment/gauth
   
   # Look for:
   # [SECURITY] Replay store: Redis (distributed)
   # [SECURITY] Redis connection: OK
   ```

4. **Test replay protection:**
   ```bash
   # Authenticate and capture token
   TOKEN=$(curl -X POST http://gauth:8080/api/v1/auth -d '...')
   
   # Use token (should succeed)
   curl -H "Authorization: Bearer $TOKEN" http://gauth:8080/api/v1/protected
   
   # Try to replay token (should fail)
   curl -H "Authorization: Bearer $TOKEN" http://gauth:8080/api/v1/protected
   # Expected: 401 Unauthorized (replay detected)
   ```

---

## Option 2: BoltDB with Persistent Volumes

⚠️ **WARNING:** This option is **NOT RECOMMENDED** for production. Use only for development/testing or single-instance deployments.

### Kubernetes with PersistentVolumeClaim

```yaml
# gauth-deployment-boltdb.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: gauth-replay-store
spec:
  accessModes:
    - ReadWriteOnce  # Single instance only
  resources:
    requests:
      storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gauth
spec:
  replicas: 1  # MUST be 1 (BoltDB file locking)
  selector:
    matchLabels:
      app: gauth
  template:
    metadata:
      labels:
        app: gauth
    spec:
      containers:
      - name: gauth
        image: gauth:latest
        env:
        - name: GAUTH_REPLAY_STORE
          value: "bolt"
        - name: GAUTH_REPLAY_STORE_PATH
          value: "/data/replay.db"  # Persistent path, NOT /tmp
        - name: GAUTH_ALLOW_UNSAFE_BOLTDB
          value: "1"  # Required to bypass container safety check
        volumeMounts:
        - name: replay-data
          mountPath: /data
      volumes:
      - name: replay-data
        persistentVolumeClaim:
          claimName: gauth-replay-store
```

Apply:
```bash
kubectl apply -f gauth-deployment-boltdb.yaml
```

### Docker with Named Volume

```yaml
# docker-compose.yml
version: '3.8'
services:
  gauth:
    image: gauth:latest
    environment:
      - GAUTH_REPLAY_STORE=bolt
      - GAUTH_REPLAY_STORE_PATH=/data/replay.db
      - GAUTH_ALLOW_UNSAFE_BOLTDB=1
    volumes:
      - gauth-replay-data:/data  # Named volume (persistent)
    ports:
      - "8080:8080"

volumes:
  gauth-replay-data:  # Persistent volume
```

---

## Kubernetes Deployment Examples

### Complete Production Deployment (Redis)

```yaml
# production-deployment.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: gauth-production
---
# Redis Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: gauth-production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "200m"
        volumeMounts:
        - name: redis-data
          mountPath: /data
      volumes:
      - name: redis-data
        persistentVolumeClaim:
          claimName: redis-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: gauth-production
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
  type: ClusterIP
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: redis-pvc
  namespace: gauth-production
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  storageClassName: standard  # Or your preferred storage class
---
# AgentAuth Application Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gauth
  namespace: gauth-production
spec:
  replicas: 3  # Horizontal scaling supported with Redis
  selector:
    matchLabels:
      app: gauth
  template:
    metadata:
      labels:
        app: gauth
    spec:
      containers:
      - name: gauth
        image: gauth:latest
        ports:
        - containerPort: 8080
        env:
        - name: GAUTH_ENV
          value: "production"
        - name: REDIS_HOST
          value: "redis.gauth-production.svc.cluster.local"
        - name: REDIS_PORT
          value: "6379"
        - name: GAUTH_REPLAY_STORE
          value: "redis"
        - name: GAUTH_JWT_SIGNING_KEY
          valueFrom:
            secretKeyRef:
              name: gauth-secrets
              key: jwt-signing-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /api/v1/beta/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/v1/beta/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: gauth
  namespace: gauth-production
spec:
  selector:
    app: gauth
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
---
# Secret (create separately with real values)
apiVersion: v1
kind: Secret
metadata:
  name: gauth-secrets
  namespace: gauth-production
type: Opaque
stringData:
  jwt-signing-key: "CHANGE-ME-TO-RANDOM-32-BYTE-KEY"
```

Deploy:
```bash
kubectl apply -f production-deployment.yaml
```

---

## Docker Compose Examples

### Development with Redis

```yaml
# docker-compose.yml
version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes

  gauth:
    image: gauth:latest
    ports:
      - "8080:8080"
    environment:
      - GAUTH_ENV=development
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - GAUTH_REPLAY_STORE=redis
      - GAUTH_JWT_SIGNING_KEY=dev-secret-change-in-production
      - GAUTH_DEV_INDEX=1
    depends_on:
      - redis

volumes:
  redis-data:
```

Run:
```bash
docker-compose up -d
```

---

## Verification and Testing

### 1. Container Restart Test

Test that replay protection survives container restarts:

```bash
# Step 1: Authenticate
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}' | jq -r '.token')

echo "Token: $TOKEN"

# Step 2: Use token (should succeed)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/protected
# Expected: 200 OK

# Step 3: Restart container
kubectl rollout restart deployment/gauth
# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=gauth --timeout=60s

# Step 4: Try to replay token (should fail)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/protected
# Expected: 401 Unauthorized (replay detected)

# ✅ PASS: Replay protection survived restart
# ❌ FAIL: If returns 200, replay store is ephemeral
```

### 2. Multi-Instance Test (Redis Only)

Test replay protection across multiple instances:

```bash
# Scale to 3 replicas
kubectl scale deployment/gauth --replicas=3

# Authenticate via instance 1
TOKEN=$(curl http://instance-1:8080/api/v1/auth -d '...' | jq -r '.token')

# Try to use same token via instance 2 (should fail)
curl -H "Authorization: Bearer $TOKEN" http://instance-2:8080/api/v1/protected
# Expected: 401 Unauthorized (replay detected across instances)
```

### 3. TTL Expiration Test

Verify that old tokens expire correctly:

```bash
# Set short TTL for testing
export GAUTH_REPLAY_TTL=60  # 60 seconds

# Authenticate
TOKEN=$(curl http://localhost:8080/api/v1/auth -d '...' | jq -r '.token')

# Use token immediately (should succeed)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/protected
# Expected: 200 OK

# Wait for TTL expiration
sleep 70

# Try again - should succeed (token no longer in replay store)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/protected
# Expected: 200 OK (replay record expired, token can be used again)
```

---

## Rollback Procedures

### If Migration Fails

1. **Immediate Rollback (Kubernetes):**
   ```bash
   # Rollback to previous deployment
   kubectl rollout undo deployment/gauth
   
   # Verify rollback
   kubectl rollout status deployment/gauth
   ```

2. **Restore BoltDB (with persistent volume):**
   ```bash
   # Revert environment variables
   kubectl set env deployment/gauth \
     GAUTH_REPLAY_STORE=bolt \
     GAUTH_REPLAY_STORE_PATH=/data/replay.db \
     GAUTH_ALLOW_UNSAFE_BOLTDB=1
   ```

3. **Emergency Recovery:**
   ```bash
   # If Redis is down, temporarily disable replay protection
   kubectl set env deployment/gauth \
     GAUTH_REPLAY_DISABLED=1
   
   # ⚠️ WARNING: This disables replay protection entirely
   # Only use as last resort for critical outages
   # Restore proper replay protection ASAP
   ```

---

## FAQ

### Q: Can I use BoltDB with a persistent volume in Kubernetes?

**A:** Yes, but **NOT RECOMMENDED** for production:
- ✅ Development/testing: Acceptable
- ⚠️ Production single-instance: Acceptable with caveats
- ❌ Production multi-instance: NOT SUPPORTED (file locking)

Use Redis for production deployments.

---

### Q: What happens if Redis goes down?

**A:** Authentication will fail if replay protection cannot be verified. Options:

1. **High Availability Redis** (Recommended):
   - Use Redis Sentinel or Cluster mode
   - Managed services have automatic failover

2. **Fallback to In-Memory** (Not Recommended):
   ```bash
   GAUTH_REPLAY_FALLBACK=memory
   ```
   This allows authentication to continue but loses replay protection.

3. **Circuit Breaker** (Best Practice):
   Implement circuit breaker pattern to temporarily allow authentication during Redis outages, with enhanced logging and post-outage review.

---

### Q: How much storage does Redis need?

**A:** Calculation:

```
Storage = (number of active tokens) × (token size)

Example:
- 1 million authentications/day
- 1-hour TTL (tokens active for 1 hour)
- Token size: ~100 bytes (JTI + metadata)

Active tokens = 1,000,000 / 24 = 41,667
Storage = 41,667 × 100 bytes = 4.16 MB

Recommendation: 
- Start with 256 MB Redis instance
- Monitor actual usage
- Scale up if needed (rare)
```

---

### Q: Can I migrate without downtime?

**A:** Yes, using blue-green deployment:

1. Deploy new version (green) with Redis alongside old version (blue)
2. Gradually shift traffic to green using service mesh or load balancer
3. Once green is stable, decommission blue

Note: Tokens issued by blue won't be replay-protected by green (different stores). Acceptable for short migration window.

---

### Q: What about multi-region deployments?

**A:** Options:

1. **Redis Cluster** (Recommended):
   - Use Redis Cluster with replicas in each region
   - Active-active replication

2. **Regional Redis Instances**:
   - Separate Redis per region
   - Accept that replay protection is regional only
   - Document limitation in security documentation

3. **Global Redis Service**:
   - AWS Global Datastore
   - Azure Cache for Redis with geo-replication

---

### Q: How do I monitor replay store health?

**A:** Prometheus metrics:

```prometheus
# Replay store operations
gauth_replay_store_checks_total
gauth_replay_store_hits_total
gauth_replay_store_misses_total

# Replay store latency
gauth_replay_store_latency_seconds

# Redis connection status
gauth_redis_connection_status
```

Alert on:
- High latency (> 100ms)
- Connection failures
- Increased error rate

---

### Q: What if I can't use Redis?

**A:** Alternatives:

1. **PostgreSQL** (Excellent choice):
   - Implement replay table with TTL cleanup
   - Better for regulated industries
   - Can leverage existing database infrastructure

2. **DynamoDB** (AWS):
   - Use TTL feature for automatic expiration
   - Serverless scaling

3. **Memcached**:
   - Simpler than Redis
   - Built-in TTL support
   - No persistence (acceptable for replay protection)

---

## Security Contact

For security issues related to replay protection:

- **Email:** security@gimel.foundation
- **Report:** See SECURITY.md
- **Reference:** CV-2025-005 (BoltDB Ephemeral Storage Vulnerability)

---

## References

- **Security Audit:** SECURITY_AUDIT_CRITICAL_REVIEW.md
- **Vulnerability Assessment:** SECURITY_VULNERABILITY_ASSESSMENT_2025.md
- **Executive Briefing:** SECURITY_EXECUTIVE_BRIEFING.md
- **Redis Documentation:** https://redis.io/docs/
- **Kubernetes PVC Guide:** https://kubernetes.io/docs/concepts/storage/persistent-volumes/

---

**Document Version:** 1.0  
**Last Updated:** November 30, 2025  
**Next Review:** January 31, 2026
