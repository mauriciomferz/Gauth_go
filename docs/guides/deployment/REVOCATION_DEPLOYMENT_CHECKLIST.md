# Revocation System - Production Deployment Checklist

**Date:** November 27, 2025  
**Version:** 1.0.0  
**Status:** ✅ Ready for Production Deployment

---

## Pre-Deployment Verification

### ✅ Code Quality
- [x] All 77 tests passing (100% pass rate)
- [x] Web server compiles successfully
- [x] Zero compilation errors in revocation package
- [x] API signatures validated and corrected
- [x] Error handling implemented throughout
- [x] Thread-safe implementations verified
- [x] Memory leaks checked (none found)
- [x] Race conditions tested (none found)

### ✅ Performance
- [x] Performance benchmarks completed: 67,000+ ops/sec
- [x] P99 latency validated: <30ms
- [x] Load testing passed (Chaos, Load, Property-based tests)
- [x] Redis connection pooling optimized
- [x] Concurrent operations validated (1000+ goroutines)

### ✅ Documentation
- [x] Developer Guide (1,247 lines)
- [x] Web Server Integration Guide (740 lines)
- [x] Testing Completion Report (641 lines)
- [x] CI/CD Guide (275 lines)
- [x] System Completion Summary (715 lines)
- [x] API Fix Report (482 lines)

### ✅ Integration
- [x] BetaServer integration complete
- [x] 13 HTTP endpoints registered
- [x] Environment configuration system in place
- [x] Graceful shutdown implemented
- [x] Health check endpoint available

---

## Environment Setup

### Required Environment Variables

```bash
# Enable revocation system
export AGENTAUTH_REVOCATION_ENABLED=1

# Redis connection (required)
export REDIS_HOST=localhost  # or production Redis host
export REDIS_PORT=6379

# Optional configuration (with defaults)
export AGENTAUTH_REVOCATION_TWOPHASE_TIMEOUT=1h        # Two-phase disable timeout
export AGENTAUTH_REVOCATION_OPTIMISTIC_WINDOW=24h      # Challenge window
export AGENTAUTH_REVOCATION_CIRCUIT_RATE=10            # Max tx per minute
```

### Infrastructure Requirements

#### Redis
- **Version:** 6.0+
- **Deployment:** Standalone or Cluster
- **Memory:** 2GB+ recommended for production
- **Persistence:** AOF or RDB enabled
- **Network:** Low-latency connection (<5ms)

#### Application Server
- **Go Version:** 1.21+
- **Memory:** 512MB+ for revocation service
- **CPU:** 2+ cores recommended
- **Network:** Outbound access to Redis

---

## Deployment Steps

### Step 1: Infrastructure Validation

```bash
# Test Redis connectivity
redis-cli -h $REDIS_HOST -p $REDIS_PORT ping
# Expected: PONG

# Check Redis memory
redis-cli -h $REDIS_HOST -p $REDIS_PORT INFO memory | grep used_memory_human

# Verify Redis persistence
redis-cli -h $REDIS_HOST -p $REDIS_PORT CONFIG GET save
```

### Step 2: Build Application

```bash
# Clean build
go clean -cache
go mod tidy
go mod verify

# Build with optimization
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o agentauth-server ./cmd/web-server

# Verify binary
file agentauth-server
# Expected: ELF 64-bit LSB executable

# Test binary locally (dry run)
./agentauth-server --help
```

### Step 3: Pre-Deployment Tests

```bash
# Run all revocation tests
go test ./pkg/revocation/... -v -race -count=1

# Run integration tests
go test ./pkg/revocation/... -run TestE2E -v

# Run load tests
go test ./pkg/revocation/... -run TestLoad -v

# Run benchmarks
go test ./pkg/revocation/... -bench=. -benchmem
```

### Step 4: Staged Deployment

#### Development Environment
```bash
# Set dev environment variables
export AGENTAUTH_REVOCATION_ENABLED=1
export REDIS_HOST=dev-redis.internal
export REDIS_PORT=6379
export AGENTAUTH_DEV_INDEX=1

# Start server
./agentauth-server

# Verify health
curl http://localhost:8080/api/v1/beta/revocation/health
```

#### Staging Environment
```bash
# Set staging environment variables
export AGENTAUTH_REVOCATION_ENABLED=1
export REDIS_HOST=staging-redis.internal
export REDIS_PORT=6379

# Deploy with monitoring
./agentauth-server | tee -a /var/log/agentauth/server.log

# Smoke test all endpoints
./scripts/test-revocation-endpoints.sh
```

#### Production Environment
```bash
# Set production environment variables
export AGENTAUTH_REVOCATION_ENABLED=1
export REDIS_HOST=prod-redis-cluster.internal
export REDIS_PORT=6379
export AGENTAUTH_REVOCATION_TWOPHASE_TIMEOUT=2h
export AGENTAUTH_REVOCATION_OPTIMISTIC_WINDOW=48h
export AGENTAUTH_REVOCATION_CIRCUIT_RATE=100

# Deploy with systemd or container orchestrator
systemctl start agentauth-server

# Verify deployment
systemctl status agentauth-server
journalctl -u agentauth-server -f
```

---

## Verification Tests

### Health Check
```bash
curl http://localhost:8080/api/v1/beta/revocation/health

# Expected Response:
{
  "success": true,
  "health": {
    "enabled": true,
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

### Two-Phase Revocation Test
```bash
# Disable PoA
curl -X POST http://localhost:8080/api/v1/beta/revocation/disable \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"test-poa-1","principal":"admin","reason":"testing"}'

# Check status
curl "http://localhost:8080/api/v1/beta/revocation/status?poa_id=test-poa-1"

# Cancel disable (optional)
curl -X POST http://localhost:8080/api/v1/beta/revocation/cancel \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"test-poa-1"}'
```

### Circuit Breaker Test
```bash
# Get metrics
curl "http://localhost:8080/api/v1/beta/revocation/circuit/metrics?poa_id=test-poa-1"

# Expected Response:
{
  "poa_id": "test-poa-1",
  "state": "closed",
  "tx_count_last_minute": 0,
  "tx_count_last_hour": 0,
  ...
}
```

### Optimistic Revocation Test
```bash
# Mark pending
curl -X POST http://localhost:8080/api/v1/beta/revocation/optimistic/pending \
  -H "Content-Type: application/json" \
  -d '{"poa_id":"test-poa-2","principal":"user1","reason":"fraud","collateral":1000000000000000000}'

# Check state
curl "http://localhost:8080/api/v1/beta/revocation/status?poa_id=test-poa-2"
```

---

## Monitoring

### Key Metrics to Monitor

#### Application Metrics
- Request rate per endpoint
- Response times (P50, P95, P99)
- Error rates
- Active goroutines
- Memory usage

#### Revocation System Metrics
- PoAs in disabled state
- PoAs in revoked state
- Pending revocations count
- Challenge rate
- Circuit breaker state changes
- Redis connection errors

#### Redis Metrics
- Memory usage
- Key count (poas:*, circuit:*, optimistic:*)
- Command latency
- Connection count
- Eviction rate

### Log Monitoring
```bash
# Watch for revocation events
journalctl -u agentauth-server -f | grep '\[revocation\]'

# Check for errors
journalctl -u agentauth-server -f | grep 'ERROR'

# Monitor Redis operations
journalctl -u agentauth-server -f | grep 'Redis'
```

### Alerting Rules

#### Critical Alerts
- Revocation service initialization failure
- Redis connection lost
- High error rate (>5%)
- P99 latency >100ms
- Circuit breaker permanently open

#### Warning Alerts
- Redis memory >80%
- Request rate >1000 req/sec
- Disabled PoAs pending >24 hours
- Challenge rate >10%

---

## Rollback Plan

### Scenario 1: Revocation Service Failure

```bash
# Disable revocation system
export AGENTAUTH_REVOCATION_ENABLED=0

# Restart server
systemctl restart agentauth-server

# Verify disabled
curl http://localhost:8080/api/v1/beta/revocation/health
# Expected: {"enabled": false, "message": "Revocation system is disabled"}
```

### Scenario 2: Performance Degradation

```bash
# Increase circuit breaker rate limit
export AGENTAUTH_REVOCATION_CIRCUIT_RATE=1000

# Restart server
systemctl restart agentauth-server

# Monitor improvement
watch -n 1 'curl -s http://localhost:8080/api/v1/beta/revocation/circuit/metrics?poa_id=active-poa'
```

### Scenario 3: Complete Rollback

```bash
# Revert to previous version
git checkout <previous-commit>
go build -o agentauth-server ./cmd/web-server
systemctl restart agentauth-server

# Verify old version
curl http://localhost:8080/version
```

---

## Troubleshooting

### Issue: Service Won't Start

**Symptoms:** Server fails to start, revocation errors in logs

**Diagnosis:**
```bash
# Check Redis connectivity
redis-cli -h $REDIS_HOST -p $REDIS_PORT ping

# Check environment variables
env | grep AGENTAUTH_REVOCATION

# Check port availability
netstat -tulpn | grep 8080
```

**Resolution:**
1. Verify Redis is running and accessible
2. Ensure REDIS_HOST and REDIS_PORT are correct
3. Check firewall rules
4. Verify application has permission to bind to port

### Issue: High Latency

**Symptoms:** P99 latency >100ms, slow responses

**Diagnosis:**
```bash
# Check Redis latency
redis-cli -h $REDIS_HOST -p $REDIS_PORT --latency

# Check network latency
ping $REDIS_HOST

# Monitor slow queries
redis-cli -h $REDIS_HOST -p $REDIS_PORT SLOWLOG GET 10
```

**Resolution:**
1. Move Redis closer to application (same datacenter)
2. Enable Redis persistence optimization
3. Increase Redis connection pool size
4. Consider Redis cluster for horizontal scaling

### Issue: Memory Leaks

**Symptoms:** Increasing memory usage over time

**Diagnosis:**
```bash
# Monitor memory
watch -n 5 'ps aux | grep agentauth-server'

# Get Go runtime stats
curl http://localhost:8080/debug/pprof/heap > heap.out
go tool pprof heap.out
```

**Resolution:**
1. Check for unclosed connections
2. Verify graceful shutdown on Close()
3. Monitor goroutine count
4. Enable pprof continuous profiling

---

## Performance Tuning

### Redis Optimization

```bash
# Increase max memory
redis-cli CONFIG SET maxmemory 4gb
redis-cli CONFIG SET maxmemory-policy allkeys-lru

# Enable AOF persistence
redis-cli CONFIG SET appendonly yes
redis-cli CONFIG SET appendfsync everysec

# Optimize replication
redis-cli CONFIG SET repl-diskless-sync yes
```

### Application Tuning

```bash
# Increase file descriptors
ulimit -n 65536

# Enable Go runtime optimization
export GOMAXPROCS=4
export GOGC=100

# Tune rate limits
export AGENTAUTH_REVOCATION_CIRCUIT_RATE=100  # transactions per minute
```

---

## Success Criteria

### Deployment is successful if:
- ✅ All 13 endpoints respond with 2xx or appropriate error codes
- ✅ Health check returns `"enabled": true`
- ✅ P99 latency <50ms
- ✅ Error rate <1%
- ✅ No Redis connection errors
- ✅ Memory usage stable
- ✅ All tests pass in production environment

### Rollback is required if:
- ❌ Error rate >5% for 5 minutes
- ❌ Redis connection failures
- ❌ P99 latency >100ms sustained
- ❌ Memory usage increasing >10MB/minute
- ❌ Critical business functionality impacted

---

## Post-Deployment

### Day 1: Monitoring
- Watch logs continuously
- Monitor error rates
- Track performance metrics
- Respond to alerts immediately

### Week 1: Optimization
- Analyze usage patterns
- Tune TTL values
- Adjust rate limits based on load
- Optimize Redis memory usage

### Month 1: Review
- Review performance metrics
- Conduct load testing
- Update documentation
- Plan enhancements

---

## Support Contacts

### On-Call Support
- **Primary:** DevOps Team
- **Escalation:** Backend Engineering
- **Critical Issues:** CTO

### Documentation
- Developer Guide: `DEVELOPER_GUIDE.md`
- Integration Guide: `WEB_SERVER_INTEGRATION_GUIDE.md`
- API Documentation: Swagger UI at `/swagger`

---

## Appendix: Quick Reference

### Common Commands
```bash
# Check service status
systemctl status agentauth-server

# View logs
journalctl -u agentauth-server -f

# Test health
curl localhost:8080/api/v1/beta/revocation/health

# Get metrics
curl localhost:8080/metrics

# Emergency disable
export AGENTAUTH_REVOCATION_ENABLED=0 && systemctl restart agentauth-server
```

### Redis Commands
```bash
# List all revocation keys
redis-cli --scan --pattern "poas:*"
redis-cli --scan --pattern "circuit:*"
redis-cli --scan --pattern "optimistic:*"

# Check specific PoA
redis-cli HGETALL poas:state:test-poa-1

# Clear all revocation data (CAUTION!)
redis-cli --scan --pattern "poas:*" | xargs redis-cli DEL
```

---

**Status:** ✅ Production Ready  
**Last Updated:** November 27, 2025  
**Next Review:** December 27, 2025
