# Fail-Closed Replay Store Mode: Security vs Availability Advisory

**Document Status**: Active  
**Last Updated**: 2025-12-27  
**Audience**: Operations, Security Engineering, Compliance

---

## Executive Summary

The AgentAuth replay store supports a **fail-closed mode** that prioritizes security over availability during JTI (JWT ID) verification failures. This document outlines the operational implications, risk trade-offs, and recommended deployment strategies for this critical security feature.

> [!WARNING]
> Fail-closed mode can cause service denials during transient replay store outages. Operators must balance security requirements against availability SLAs.

---

## Overview

### What is Fail-Closed Mode?

When the replay store (Redis or other distributed backend) becomes unavailable:
- **Fail-Open**: Token validation proceeds without replay protection (availability-first)
- **Fail-Closed**: Token validation is rejected with an error (security-first)

### Current Implementation

Fail-closed behavior is implemented in [`pkg/gauth_rfc_001/rfc0111.go`](file:///Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/pkg/gauth_rfc_001/rfc0111.go) and triggered when the `CheckJTI` method encounters a replay store availability error.

**Key Metric**: `IncReplayStoreAvailabilityImpact()` is emitted each time a validation is denied due to replay store unavailability.
- **MCP Authorization**: MCP endpoints (Resource, Tool, Prompt) operate in a mandatory fail-closed mode; any authorization bridge failure or missing token identity results in an immediate 403/401 denial.

---

## Security vs Availability Trade-Off

### Fail-Closed (Security-First)

**Advantages**:
- ✅ Prevents replay attacks during infrastructure failures
- ✅ Maintains strict compliance with AAP-001 requirements
- ✅ No degradation of security posture during outages

**Disadvantages**:
- ❌ Service unavailable during replay store outages
- ❌ Can amplify cascading failures
- ❌ May violate availability SLAs (e.g., 99.95% uptime)

### Fail-Open (Availability-First)

**Advantages**:
- ✅ Service continues during infrastructure failures
- ✅ Better availability metrics
- ✅ Graceful degradation

**Disadvantages**:
- ❌ **Critical Security Risk**: Replay attacks possible during outages
- ❌ Compliance violations (AAP-001 mandates replay protection)
- ❌ Audit trail gaps

---

## Deployment Recommendations

### High-Security Environments

**Use fail-closed mode when**:
- Financial transactions or high-value operations
- Regulatory compliance requires strict replay protection
- Token replay poses existential business risk

**Mitigation strategies**:
1. Deploy Redis in HA configuration (Sentinel or Cluster mode)
2. Monitor `replay_store_availability_impact` metric and alert aggressively
3. Implement circuit breakers for upstream systems
4. Set conservative TTLs for replay store keys

### High-Availability Environments

**Consider fail-open mode when** (with caution):
- Service availability is mission-critical
- Replay attack risk is acceptable during brief outages
- Defense-in-depth layers exist (e.g., rate limiting, anomaly detection)

> [!CAUTION]
> Fail-open mode is **NOT recommended** for production systems handling sensitive operations. Use only with explicit risk acceptance from security leadership.

---

## Operational Runbook

### Monitoring

Monitor these metrics to detect replay store issues:

```promql
# Replay store availability impact (fail-closed rejections)
rate(gauth_replay_store_availability_impact_total[5m]) > 0

# Replay store error rate
rate(gauth_replay_store_errors_total[5m]) / rate(gauth_replay_checks_total[5m]) > 0.01
```

**Alert thresholds**:
- `replay_store_availability_impact` > 0: **P1 incident** (service degradation)
- Replay store error rate > 1%: **P2 incident** (degraded replay protection)

### Incident Response

**Symptoms**: Elevated `401 Unauthorized` errors with `replay_store_unavailable` error code

**Immediate actions**:
1. **Verify Redis health**: Check connectivity, CPU, memory
2. **Check network**: Confirm network path between AgentAuth and Redis
3. **Review logs**: Search for `replay store check failed` entries
4. **Consider temporary fail-open** (requires executive approval):
   ```bash
   # EMERGENCY ONLY - Requires security sign-off
   export GAUTH_REPLAY_FAILOPEN=1
   ```

**Post-incident**:
- Root cause analysis of replay store failure
- Review HA configuration
- Update runbooks with lessons learned

---

##Configuration

### Environment Variables

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| `GAUTH_REPLAY_FAILCLOSED` | `1` / `0` | `1` | Enable fail-closed mode |
| `GAUTH_REPLAY_STORE_TIMEOUT_MS` | Integer | `500` | Replay store operation timeout |
| `GAUTH_REPLAY_STORE_RETRIES` | Integer | `2` | Number of retry attempts |

### Redis HA Configuration

**Recommended setup** for fail-closed mode:

```yaml
# Redis Sentinel (minimum 3 nodes for quorum)
redis:
  mode: sentinel
  sentinels:
    - host: redis-sentinel-1:26379
    - host: redis-sentinel-2:26379
    - host: redis-sentinel-3:26379
  master_name: gauth-replay-master
  socket_timeout: 500ms
  command_timeout: 500ms
```

---

## Risk Register Impact

Implementing fail-closed mode addresses:
- **RR-006**: Replay store exhaustion mitigation
- **MR-001**: Token replay attack prevention (mitigated)

See [`RESIDUAL_RISKS.md`](file:///Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/docs/RESIDUAL_RISKS.md) for complete risk tracking.

---

## Testing Recommendations

### Chaos Engineering

Simulate replay store failures to validate behavior:

```bash
# Block Redis connectivity
iptables -A OUTPUT -p tcp --dport 6379 -j DROP

# Run validation tests
curl -H "Authorization: Bearer $TOKEN" https://gauth/api/v1/validate

# Expected: 401 with replay_store_unavailable error
```

### Load Testing

Measure fail-closed impact on throughput:
- Baseline: Validate normal ops/sec with healthy replay store
- Failure: Measure rejection rate with Redis offline
- Recovery: Time to restore service after Redis returns

---

## References

- [AAP-001 Specification](file:///Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/docs/Gifo_0111.md)
- [THREAT_MODEL.md](file:///Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/docs/THREAT_MODEL.md)
- [RESIDUAL_RISKS.md](file:///Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/docs/RESIDUAL_RISKS.md)
- [GAP_MATRIX.md](file:///Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/docs/GAP_MATRIX.md)

---

## Revision History

| Date | Version | Changes |
|------|---------|---------|
| 2025-12-27 | 1.0 | Initial advisory created for Phase 14 |
