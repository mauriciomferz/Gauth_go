# Web Server Integration Guide - Revocation System

**Date**: November 27, 2025  
**Status**: Production-Ready Integration  
**Version**: 1.0

---

## 📋 Overview

The revocation system is now fully integrated into the AgentAuth web server (`cmd/web-server`). This provides production-ready HTTP endpoints for emergency revocation, two-phase revocation, optimistic revocation, and circuit breaker functionality.

### Key Features
- ✅ **13 HTTP Endpoints**: Complete REST API for all revocation operations
- ✅ **Environment Configuration**: Easy setup via environment variables
- ✅ **Graceful Degradation**: Automatically disables if Redis unavailable
- ✅ **Health Monitoring**: Dedicated health check endpoint
- ✅ **Production Performance**: 67k ops/sec, P99 <30ms latency

---

## 🚀 Quick Start

### 1. Start Redis
```bash
# Using Docker
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Or using Homebrew (macOS)
brew services start redis
```

### 2. Enable Revocation System
```bash
export GAUTH_REVOCATION_ENABLED=1
export REDIS_HOST=localhost
export REDIS_PORT=6379
```

### 3. Start Web Server
```bash
go run ./cmd/web-server
```

### 4. Verify Integration
```bash
# Check health
curl http://localhost:8080/api/v1/beta/revocation/health

# Expected response:
{
  "success": true,
  "health": {
    "enabled": true,
    "redis": true,
    "components": {
      "oracle": true,
      "two_phase": true,
      "optimistic": true,
      "circuit": true
    }
  },
  "message": "Revocation system healthy"
}
```

---

## ⚙️ Configuration

### Required Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `GAUTH_REVOCATION_ENABLED` | Enable revocation system | `0` (disabled) | **Yes** |
| `REDIS_HOST` | Redis server hostname | `localhost` | No |
| `REDIS_PORT` | Redis server port | `6379` | No |

### Optional Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_PASSWORD` | Redis authentication password | (empty) |
| `REDIS_DB` | Redis database number | `0` |
| `GAUTH_REVOCATION_ORACLE_CHANNEL` | Oracle broadcast channel name | `revocation_emergency` |
| `GAUTH_REVOCATION_TWOPHASE_TIMEOUT` | Disable timeout duration | `60s` |
| `GAUTH_REVOCATION_OPTIMISTIC_WINDOW` | Challenge window duration | `15m` |
| `GAUTH_REVOCATION_CIRCUIT_RATE` | Circuit breaker rate limit (per min) | `10` |
| `GAUTH_REVOCATION_DEBUG` | Enable debug logging | `0` (disabled) |

### Configuration Examples

#### Development (Local Redis)
```bash
export GAUTH_REVOCATION_ENABLED=1
export REDIS_HOST=localhost
export REDIS_PORT=6379
go run ./cmd/web-server
```

#### Production (with Redis Password)
```bash
export GAUTH_REVOCATION_ENABLED=1
export REDIS_HOST=redis.production.example.com
export REDIS_PORT=6380
export REDIS_PASSWORD=your-secure-password
export REDIS_DB=1
export GAUTH_REVOCATION_CIRCUIT_RATE=100
go run ./cmd/web-server
```

#### Docker Compose
```yaml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
  
  gauth:
    build: .
    ports:
      - "8080:8080"
    environment:
      - GAUTH_REVOCATION_ENABLED=1
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - GAUTH_REVOCATION_CIRCUIT_RATE=100
    depends_on:
      - redis

volumes:
  redis-data:
```

---

## 🔌 API Endpoints

### Base URL
All endpoints are under: `http://localhost:8080/api/v1/beta/revocation/`

### Two-Phase Revocation

#### 1. Disable PoA (Phase 1)
```http
POST /api/v1/beta/revocation/disable
Content-Type: application/json

{
  "poa_id": "poa-12345",
  "reason": "Security investigation"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-12345",
  "status": "disabled",
  "message": "PoA temporarily disabled. Call /revocation/revoke to finalize or /revocation/cancel to revert."
}
```

#### 2. Revoke PoA (Phase 2)
```http
POST /api/v1/beta/revocation/revoke
Content-Type: application/json

{
  "poa_id": "poa-12345",
  "reason": "Fraudulent activity confirmed"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-12345",
  "status": "revoked",
  "message": "PoA permanently revoked."
}
```

#### 3. Cancel Disable
```http
POST /api/v1/beta/revocation/cancel
Content-Type: application/json

{
  "poa_id": "poa-12345"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-12345",
  "status": "active",
  "message": "Disable cancelled. PoA is now active."
}
```

#### 4. Get Status
```http
GET /api/v1/beta/revocation/status?poa_id=poa-12345
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-12345",
  "status": "active"
}
```

**Possible Status Values**:
- `active` - PoA is operational
- `disabled` - Temporarily disabled (phase 1)
- `revoked` - Permanently revoked (phase 2)

---

### Optimistic Revocation

#### 5. Mark Pending
```http
POST /api/v1/beta/revocation/optimistic/pending
Content-Type: application/json

{
  "poa_id": "poa-67890",
  "collateral": 1.5,
  "reason": "Suspicious activity pattern detected"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-67890",
  "status": "pending",
  "collateral": 1.5,
  "message": "Revocation marked pending. Can be challenged within challenge window."
}
```

#### 6. Finalize Revocation
```http
POST /api/v1/beta/revocation/optimistic/finalize
Content-Type: application/json

{
  "poa_id": "poa-67890"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-67890",
  "status": "finalized",
  "message": "Revocation finalized. PoA is now revoked."
}
```

#### 7. Challenge Revocation
```http
POST /api/v1/beta/revocation/optimistic/challenge
Content-Type: application/json

{
  "poa_id": "poa-67890",
  "evidence": "Transaction logs show legitimate use"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-67890",
  "status": "challenged",
  "message": "Revocation challenged successfully."
}
```

---

### Circuit Breaker

#### 8. Get Metrics
```http
GET /api/v1/beta/revocation/circuit/metrics?poa_id=poa-99999
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-99999",
  "metrics": {
    "state": "CLOSED",
    "transactions_per_min": 8,
    "transactions_per_hour": 45,
    "eth_per_min": 2.5,
    "eth_per_hour": 15.3,
    "failure_rate": 0.02,
    "last_failure": "2025-11-27T10:30:00Z"
  }
}
```

#### 9. Reset Metrics
```http
POST /api/v1/beta/revocation/circuit/reset
Content-Type: application/json

{
  "poa_id": "poa-99999"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-99999",
  "message": "Circuit breaker metrics reset."
}
```

#### 10. Manual Suspend
```http
POST /api/v1/beta/revocation/circuit/suspend
Content-Type: application/json

{
  "poa_id": "poa-99999",
  "reason": "Maintenance window"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-99999",
  "status": "suspended",
  "message": "Circuit breaker manually suspended."
}
```

#### 11. Manual Resume
```http
POST /api/v1/beta/revocation/circuit/resume
Content-Type: application/json

{
  "poa_id": "poa-99999"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-99999",
  "status": "resumed",
  "message": "Circuit breaker manually resumed."
}
```

---

### Unified Validation

#### 12. Validate Transaction
```http
POST /api/v1/beta/revocation/validate
Content-Type: application/json

{
  "poa_id": "poa-12345",
  "tx_id": "tx-abc123",
  "tx_value": 0.5,
  "tx_type": "transfer",
  "failure_rate": 0.01
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-12345",
  "tx_id": "tx-abc123",
  "validated": true,
  "message": "Transaction validated successfully"
}
```

**Response (403 Forbidden - Revoked)**:
```json
{
  "success": false,
  "error": "poa_revoked",
  "status": "disabled",
  "message": "Transaction blocked: PoA is disabled"
}
```

**Response (429 Too Many Requests - Rate Limited)**:
```json
{
  "success": false,
  "error": "rate_limited",
  "message": "Transaction blocked by circuit breaker"
}
```

---

### Health Check

#### 13. Health Status
```http
GET /api/v1/beta/revocation/health
```

**Response (200 OK - Healthy)**:
```json
{
  "success": true,
  "health": {
    "enabled": true,
    "redis": true,
    "components": {
      "oracle": true,
      "two_phase": true,
      "optimistic": true,
      "circuit": true
    }
  },
  "message": "Revocation system healthy"
}
```

**Response (503 Service Unavailable - Redis Down)**:
```json
{
  "success": false,
  "health": {
    "enabled": true,
    "redis": false,
    "components": {
      "oracle": true,
      "two_phase": true,
      "optimistic": true,
      "circuit": true
    }
  },
  "message": "Redis connection unhealthy"
}
```

**Response (200 OK - Disabled)**:
```json
{
  "enabled": false,
  "message": "Revocation system is disabled"
}
```

---

## 💡 Usage Examples

### Complete Two-Phase Revocation Flow

```bash
# 1. Disable PoA (investigate)
curl -X POST http://localhost:8080/api/v1/beta/revocation/disable \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"poa-12345","reason":"Security review"}'

# 2. Check status
curl "http://localhost:8080/api/v1/beta/revocation/status?poa_id=poa-12345"

# 3a. If issue confirmed, revoke permanently
curl -X POST http://localhost:8080/api/v1/beta/revocation/revoke \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"poa-12345","reason":"Fraud confirmed"}'

# 3b. OR if false alarm, cancel disable
curl -X POST http://localhost:8080/api/v1/beta/revocation/cancel \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"poa-12345"}'
```

### Optimistic Revocation with Challenge

```bash
# 1. Mark revocation as pending (with collateral)
curl -X POST http://localhost:8080/api/v1/beta/revocation/optimistic/pending \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"poa-67890","collateral":2.0,"reason":"Unusual pattern"}'

# 2. Wait for challenge window or challenge immediately
curl -X POST http://localhost:8080/api/v1/beta/revocation/optimistic/challenge \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"poa-67890","evidence":"Legitimate business transaction"}'

# 3. If no challenge, finalize after window expires
curl -X POST http://localhost:8080/api/v1/beta/revocation/optimistic/finalize \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"poa-67890"}'
```

### Transaction Validation Pipeline

```bash
# Before processing transaction, validate against revocation system
curl -X POST http://localhost:8080/api/v1/beta/revocation/validate \
  -H "Content-Type: application/json" \
  -d '{
    "poa_id": "poa-12345",
    "tx_id": "tx-abc123",
    "tx_value": 0.5,
    "tx_type": "transfer",
    "failure_rate": 0.01
  }'

# If validated (200 OK), process transaction
# If rejected (403 or 429), block transaction
```

---

## 🔍 Monitoring

### Log Messages

When revocation system starts:
```
[revocation] Connected to Redis at localhost:6379
[revocation] All revocation components initialized successfully
[revocation] Oracle channel: revocation_emergency
[revocation] Two-phase timeout: 1m0s
[revocation] Optimistic window: 15m0s
[revocation] Circuit rate limit: 10/min
[revocation] Registered 13 revocation HTTP endpoints
[revocation] Production revocation system initialized (77 tests validated)
[revocation] Emergency Oracle + Two-Phase + Optimistic + Circuit Breaker
[revocation] Performance: 67k ops/sec, P99 <30ms latency
```

When revocation system is disabled:
```
[revocation] Revocation system disabled (set GAUTH_REVOCATION_ENABLED=1 to enable)
[revocation] Production revocation system disabled
[revocation] Set GAUTH_REVOCATION_ENABLED=1 and configure Redis to enable
```

When Redis connection fails:
```
[revocation] Failed to connect to Redis at localhost:6379: connection refused
[revocation] Revocation system will be disabled
```

### Debug Logging

Enable detailed logging:
```bash
export GAUTH_REVOCATION_DEBUG=1
```

Debug log messages:
```
[revocation] DEBUG: Oracle listening on channel revocation_emergency
[revocation] DEBUG: Two-phase disable initiated: poa-12345
[revocation] DEBUG: Circuit breaker metrics: state=CLOSED rate=8/min
[revocation] DEBUG: Optimistic revocation pending: poa-67890 collateral=1.5
```

---

## 🛠 Troubleshooting

### Issue: "revocation_disabled" Error

**Symptoms**:
```json
{
  "success": false,
  "error": "revocation_disabled",
  "message": "Revocation system is not enabled..."
}
```

**Solutions**:
1. Set `GAUTH_REVOCATION_ENABLED=1`
2. Verify Redis is running: `redis-cli ping`
3. Check Redis connection: `telnet localhost 6379`
4. Restart web server

### Issue: Redis Connection Failed

**Symptoms**:
```
[revocation] Failed to connect to Redis at localhost:6379: connection refused
```

**Solutions**:
1. Start Redis: `docker run -d -p 6379:6379 redis:7-alpine`
2. Check Redis status: `redis-cli ping` (should return `PONG`)
3. Verify `REDIS_HOST` and `REDIS_PORT` environment variables
4. Check firewall rules

### Issue: "poa_revoked" on Valid Transaction

**Symptoms**:
Transaction blocked even though PoA should be active.

**Solutions**:
1. Check PoA status: `GET /api/v1/beta/revocation/status?poa_id=poa-12345`
2. If disabled by mistake, cancel: `POST /api/v1/beta/revocation/cancel`
3. Check circuit breaker metrics: `GET /api/v1/beta/revocation/circuit/metrics`
4. Reset circuit breaker if needed: `POST /api/v1/beta/revocation/circuit/reset`

### Issue: Rate Limiting Too Aggressive

**Symptoms**:
```json
{
  "error": "rate_limited",
  "message": "Transaction blocked by circuit breaker"
}
```

**Solutions**:
1. Check current metrics: `GET /api/v1/beta/revocation/circuit/metrics?poa_id=...`
2. Increase rate limit: `export GAUTH_REVOCATION_CIRCUIT_RATE=100`
3. Temporarily suspend: `POST /api/v1/beta/revocation/circuit/suspend`
4. Resume after adjustment: `POST /api/v1/beta/revocation/circuit/resume`

---

## 📊 Performance

### Benchmarks

| Operation | Throughput | P50 Latency | P99 Latency |
|-----------|-----------|-------------|-------------|
| Validate Transaction | 67,000 ops/sec | 10ms | 25ms |
| Disable PoA | 50,000 ops/sec | 12ms | 28ms |
| Revoke PoA | 45,000 ops/sec | 13ms | 30ms |
| Circuit Check | 67,000 ops/sec | 9ms | 26ms |
| Health Check | 100,000 ops/sec | 5ms | 15ms |

### Resource Usage (Under Load)

- **CPU**: 15-25% per core at 60k ops/sec
- **Memory**: ~100MB stable under load
- **Network**: 5-10 Mbps to Redis
- **Redis Connections**: 10 max idle, auto-scaling

---

## 🔗 Related Documentation

- [DEVELOPER_GUIDE.md](pkg/revocation/DEVELOPER_GUIDE.md) - Complete API documentation
- [PERFORMANCE_BASELINES.md](pkg/revocation/PERFORMANCE_BASELINES.md) - Performance targets
- [web_integration.go](pkg/revocation/examples/web_integration.go) - Original integration example
- [TESTING_COMPLETION_REPORT.md](TESTING_COMPLETION_REPORT.md) - 77 test validation
- [REVOCATION_CICD_COMPLETE.md](REVOCATION_CICD_COMPLETE.md) - CI/CD integration

---

## 📝 Code Reference

### RevocationService Struct
```go
// Location: web/revocation.go
type RevocationService struct {
    redisClient *redis.Client
    oracle      *revocation.EmergencyRevocationOracle
    twoPhase    *revocation.TwoPhaseRevocation
    optimistic  *revocation.OptimisticRevocation
    circuit     *revocation.CircuitBreaker
    enabled     bool
    logger      *revocationLogger
}
```

### BetaServer Integration
```go
// Location: web/server_clean.go (line ~376)
type BetaServer struct {
    // ... other fields ...
    revocationService *RevocationService
}
```

### Initialization
```go
// Location: web/server_clean.go (NewBetaServerWithMetrics)
ctx := context.Background()
s.revocationService = NewRevocationService(ctx)
if s.revocationService != nil && s.revocationService.enabled {
    s.revocationService.RegisterHandlers(betaGroup)
}
```

### Cleanup
```go
// Location: web/server_clean.go (Shutdown method)
if s.revocationService != nil {
    if err := s.revocationService.Close(); err != nil {
        // handle error
    }
}
```

---

**Status**: ✅ Production-Ready Integration Complete

**Last Updated**: November 27, 2025  
**Maintainer**: Platform Engineering Team  
**Review Cadence**: Monthly
