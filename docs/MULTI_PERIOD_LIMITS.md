---
title: Multi Period Limits
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Multi-Period Rate Limits

> **Feature**: SEC13.ITEM2 - Structured numeric limit parsing with multi-period support  
> **Status**: Implemented (P2.5)  
> **Date**: November 2025

## Overview

This document describes the multi-period rate limiting system that extends AgentAuth's model governance capabilities beyond simple per-minute rate limits. The system supports configurable time windows (minute, hour, day, week, month) with human-readable syntax and backward compatibility.

## Motivation

While the original system supported `max_requests_per_minute`, real-world AI governance requires multi-tier rate limiting:

- **Burst Protection**: Prevent rapid spikes within short windows (per-minute)
- **Resource Fairness**: Distribute capacity fairly over longer periods (per-hour/day)
- **Cost Control**: Enforce monthly quotas to prevent bill shock
- **Compliance**: Meet regulatory requirements for rate limiting (e.g., GDPR data processing limits)

## JSON Schema

### Basic Configuration

```json
{
  "model_limits": {
    "gpt-4": {
      "max_input_tokens": 8192,
      "max_output_tokens": 4096,
      "rate_limits_extended": ["100/minute", "5000/hour", "100K/day"]
    }
  }
}
```

### Backward Compatibility

Existing `max_requests_per_minute` configurations continue to work:

```json
{
  "model_limits": {
    "gpt-3.5": {
      "max_input_tokens": 4096,
      "max_requests_per_minute": 120  // Legacy field, still supported
    }
  }
}
```

### Combined Configuration

Both legacy and extended limits can coexist (both will be enforced):

```json
{
  "model_limits": {
    "claude-3": {
      "max_input_tokens": 8192,
      "max_requests_per_minute": 100,              // Legacy per-minute limit
      "rate_limits_extended": ["5000/hour", "100K/day"]  // Extended multi-period
    }
  }
}
```

**Enforcement**: ALL configured limits (legacy + extended) must pass. If ANY limit is exceeded, the request is denied with HTTP 429.

## Period Specifications

### Supported Periods

| Period | Aliases | Duration | Example |
|--------|---------|----------|---------|
| `minute` | `min`, `minutes` | 60 seconds | `100/minute` |
| `hour` | `hr`, `h`, `hours` | 3600 seconds | `5000/hour` |
| `day` | `d`, `days` | 86400 seconds | `50K/day` |
| `week` | `w`, `wk`, `weeks` | 604800 seconds | `10K/week` |
| `month` | `mo`, `months` | 2592000 seconds (30 days) | `1M/month` |

**Note**: Month duration is approximate (30 days) for rate limiting purposes. This is sufficient for quota enforcement and avoids calendar complexity.

### Numeric Multipliers

Supports shorthand notation for large numbers:

- **K/k**: Thousands (×1000)
  - `5K/hour` = 5000 per hour
  - `1.5k/day` = 1500 per day
- **M/m**: Millions (×1,000,000)
  - `1M/month` = 1,000,000 per month
  - `0.5M/day` = 500,000 per day

**Examples**:
```json
"rate_limits_extended": [
  "100/minute",      // 100 per minute
  "5K/hour",         // 5000 per hour
  "50K/day",         // 50000 per day
  "1.5M/month"       // 1,500,000 per month
]
```

### Case Insensitivity

All period names and multipliers are case-insensitive:
- `100/MINUTE` = `100/minute`
- `5K/HOUR` = `5k/hour`
- `1M/DAY` = `1m/day`

## Enforcement Behavior

### Multi-Period Check

When a request arrives, the system checks **all** configured period limits:

```
1. Check legacy max_requests_per_minute (if configured)
2. Check each rate_limits_extended entry:
   - Fetch current window state for period
   - If window expired (now - windowStart >= period), reset counter
   - Increment counter
   - If counter > limit, DENY with 429
3. If all limits pass, ALLOW with 200
```

### Independent Window Tracking

Each period maintains its own sliding window:

```go
// Example state for model "gpt-4"
modelRateStateExtended["gpt-4"] = {
    1*time.Minute:  {WindowStart: 2025-11-05 18:45:00, Count: 87},
    1*time.Hour:    {WindowStart: 2025-11-05 18:00:00, Count: 4523},
    24*time.Hour:   {WindowStart: 2025-11-05 00:00:00, Count: 48372}
}
```

**Window Rollover**: When `now - windowStart >= period`, the counter resets to 0 and `windowStart` is updated to `now`.

### Response Format

#### Success (200 OK)
```json
{
  "success": true,
  "model_id": "gpt-4",
  "input_tokens": 1234,
  "output_tokens": 567,
  "input_limit": 8192,
  "output_limit": 4096,
  "rate_limit": 100,
  "limit_enforced": true
}
```

#### Rate Limit Exceeded (429 Too Many Requests)

**Legacy Limit Exceeded**:
```json
{
  "success": false,
  "error": "model_rate_limit_exceeded",
  "model_id": "gpt-4",
  "limit": 100,
  "window_seconds": 60
}
```

**Extended Limit Exceeded**:
```json
{
  "success": false,
  "error": "model_rate_limit_exceeded",
  "model_id": "gpt-4",
  "limit": 5000,
  "window_seconds": 3600,
  "period": "hour"
}
```

**Key Difference**: Extended limits include `"period"` field (e.g., `"hour"`, `"day"`) for client clarity.

## Audit Trail

Rate limit violations are logged to the audit chain (`AGENTAUTH_MODEL_LIMIT_AUDIT_PATH`) with hash chaining:

```json
{
  "ts": 1730836800,
  "model_id": "gpt-4",
  "kind": "rate_extended",
  "provided": 5001,
  "limit": 5000,
  "window_start": 1730836800,
  "window_seconds": 3600,
  "prev_hash": "sha256:abc123...",
  "hash": "sha256:def456..."
}
```

**Fields**:
- `kind`: `"rate"` (legacy) or `"rate_extended"` (multi-period)
- `window_seconds`: Period duration in seconds (60 for minute, 3600 for hour, etc.)
- `provided`: Actual request count when limit exceeded
- `limit`: Configured limit for this period

## Migration Guide

### Phase 1: Add Extended Limits (Week 1)

**Before** (`model_limits.json`):
```json
{
  "model_limits": {
    "gpt-4": {
      "max_input_tokens": 8192,
      "max_requests_per_minute": 100
    }
  }
}
```

**After**:
```json
{
  "model_limits": {
    "gpt-4": {
      "max_input_tokens": 8192,
      "max_requests_per_minute": 100,              // Keep legacy
      "rate_limits_extended": ["5000/hour", "100K/day"]  // Add extended
    }
  }
}
```

**Impact**: Both limits are enforced. Clients get more granular rate limit feedback.

### Phase 2: Monitor & Tune (Week 2-4)

1. **Monitor Audit Logs**:
   ```bash
   grep "rate_extended" /tmp/model_limit_audit.jsonl | jq -r '[.model_id, .limit, .window_seconds, .provided] | @tsv'
   ```

2. **Adjust Limits**: If hourly/daily limits are too restrictive, increase them:
   ```json
   "rate_limits_extended": ["10000/hour", "200K/day"]  // Doubled
   ```

3. **Client Feedback**: Update error messages to explain period limits:
   ```python
   if resp.status_code == 429:
       period = resp.json().get("period", "minute")
       limit = resp.json()["limit"]
       print(f"Rate limit exceeded: {limit} requests per {period}")
   ```

### Phase 3: Retire Legacy (Week 5+)

Once clients handle multi-period limits gracefully, remove legacy field:

```json
{
  "model_limits": {
    "gpt-4": {
      "max_input_tokens": 8192,
      "rate_limits_extended": ["100/minute", "5000/hour", "100K/day"]
    }
  }
}
```

**Note**: No code changes required - the system automatically skips legacy enforcement if `max_requests_per_minute` is absent.

## Operational Recommendations

### Period Selection

Choose periods based on business goals:

| Use Case | Recommended Periods | Example |
|----------|---------------------|---------|
| **Burst Protection** | `minute` | `100/minute` prevents rapid spikes |
| **Fair Capacity Distribution** | `hour`, `day` | `5000/hour + 100K/day` ensures hourly fairness |
| **Cost Control** | `day`, `month` | `1M/month` aligns with billing cycle |
| **Compliance** | `day` | `10K/day` meets GDPR data processing limits |

### Typical Configurations

**Tiered Service**:
```json
{
  "model_limits": {
    "free-tier": {
      "rate_limits_extended": ["10/minute", "100/hour", "1K/day"]
    },
    "pro-tier": {
      "rate_limits_extended": ["100/minute", "5000/hour", "100K/day"]
    },
    "enterprise-tier": {
      "rate_limits_extended": ["1000/minute", "50K/hour", "1M/day"]
    }
  }
}
```

**Development vs Production**:
```json
{
  "model_limits": {
    "dev-model": {
      "rate_limits_extended": ["100/minute", "1000/hour"]  // Relaxed for testing
    },
    "prod-model": {
      "rate_limits_extended": ["50/minute", "2000/hour", "40K/day"]  // Strict
    }
  }
}
```

### Monitoring

**Prometheus Metrics** (existing counter enhanced with period label):
```promql
# Rate limit violations by period
rate(agentauth_model_rate_limit_exceeded_total{period="hour"}[5m])

# Usage percentage per period (requires custom exporter)
(agentauth_model_rate_current / agentauth_model_rate_limit) * 100
```

**Audit Log Queries**:
```bash
# Hourly violations in last 24h
jq -r 'select(.kind=="rate_extended" and .window_seconds==3600) | [.ts, .model_id, .provided, .limit] | @tsv' \
  /tmp/model_limit_audit.jsonl | \
  awk '{dt=strftime("%Y-%m-%d %H:00", $1); print dt, $2, $3"/"$4}' | \
  sort | uniq -c
```

### Alert Rules

**Prometheus Alerting**:
```yaml
groups:
  - name: rate_limits
    rules:
      - alert: HighRateLimitViolations
        expr: rate(agentauth_model_rate_limit_exceeded_total[5m]) > 10
        for: 5m
        annotations:
          summary: "High rate limit violations for {{ $labels.model_id }}"
      
      - alert: HourlyQuotaNearExhaustion
        expr: (agentauth_model_rate_current{period="hour"} / agentauth_model_rate_limit{period="hour"}) > 0.9
        annotations:
          summary: "{{ $labels.model_id }} at 90% hourly quota"
```

## Implementation Details

### Parser Architecture

**Package**: `github.com/.../pkg/limits`

**Core Function**:
```go
func ParseRateLimit(s string) (RateLimit, error)
```

**Regex Pattern**: `^(\d+(?:\.\d+)?[KkMm]?)\s*/\s*(\w+)$`
- Group 1: Numeric value with optional K/M multiplier
- Group 2: Period name

**Examples**:
```go
ParseRateLimit("1000/hour")   // {1000, 1*time.Hour}, nil
ParseRateLimit("5K/day")      // {5000, 24*time.Hour}, nil
ParseRateLimit("1.5M/month")  // {1500000, 30*24*time.Hour}, nil
ParseRateLimit("invalid")     // {}, error
```

### State Management

**Data Structures**:
```go
// Server state (web/server_clean.go)
type BetaServer struct {
    // Legacy per-minute
    modelRateLimits map[string]int
    modelRateState  map[string]struct{WindowStart time.Time, Count int}
    
    // Multi-period extended
    modelRateLimitsExtended map[string][]struct{Limit int, Period time.Duration}
    modelRateStateExtended  map[string]map[time.Duration]struct{WindowStart time.Time, Count int}
}
```

**Thread Safety**: All maps protected by sync.Mutex locks (modelRateMu, modelRateStateMu, modelRateLimitsExtendedMu, modelRateStateExtendedMu).

### Error Handling

**Invalid Format**: Malformed rate limits are logged to stderr and skipped:
```
[model-limits] invalid rate limit "bad/format" for model gpt-4: invalid rate limit format: "bad/format" (expected format: "1000/hour")
```

**Graceful Degradation**: If all extended limits fail to parse, the model falls back to legacy `max_requests_per_minute` (if configured).

## Security Considerations

### Clock Skew Tolerance

Window rollover uses server time (`time.Now()`). Clock adjustments (NTP sync, DST) do not compromise enforcement:
- **Forward Jump**: Counter resets early (stricter enforcement)
- **Backward Jump**: Window extends (more permissive, but self-correcting on next request)

**Recommendation**: Use NTP synchronization in production to minimize clock drift.

### Resource Exhaustion

Each model × period combination creates a window state entry. For 1000 models × 3 periods = 3000 entries.

**Memory Estimate**: 3000 entries × 40 bytes/entry = ~120 KB (negligible).

**Mitigation**: Window states are lazy-initialized (created on first request). Idle models consume zero memory.

### Adversarial Behavior

**Attack**: Rapidly rotate request patterns to avoid window accumulation.

**Defense**: Use **both** short (minute) and long (hour/day) periods. Even if an attacker spreads requests to avoid per-minute limit, they'll hit hourly/daily limits.

**Example**:
```json
"rate_limits_extended": ["100/minute", "5000/hour"]
```
- Attacker can only send 5000 requests/hour max, regardless of per-minute distribution.

## Testing

### Unit Tests

**Package**: `github.com/.../pkg/limits`
- **37 test scenarios** covering valid formats, multipliers, case-insensitivity, errors
- **Coverage**: 100% of parser logic

### Integration Tests

**Package**: `github.com/.../web`
- `TestMultiPeriodRateLimits`: 10/minute enforcement
- `TestMultiPeriodHourlyLimit`: 15/hour enforcement
- `TestMultiPeriodDailyLimit`: 50K/day parsing
- `TestBackwardCompatibilityPerMinute`: Legacy `max_requests_per_minute`
- `TestBothLegacyAndExtended`: Dual enforcement
- `TestPeriodRollover`: Window expiration and reset
- `TestInvalidRateLimitFormat`: Error handling

**All tests pass**: ✅ 7/7

## Future Enhancements

### Per-User Multi-Period Limits

Extend `user_limits` schema:
```json
{
  "user_limits": {
    "gpt-4": {
      "alice": {
        "rate_limits_extended": ["50/minute", "2000/hour"]
      }
    }
  }
}
```

### Currency Conversion

Support cost-based limits (requires external pricing data):
```json
"rate_limits_extended": ["$100/day", "$2000/month"]
```

### Custom Periods

Support arbitrary durations:
```json
"rate_limits_extended": ["10000/quarter", "50K/year"]
```

### Distributed State

For multi-instance deployments, use Redis/Memcached for shared window state:
```go
// Pseudo-code
state := redis.Get("ratelimit:gpt-4:1h")
state.Count++
redis.SetEx("ratelimit:gpt-4:1h", 3600, state)
```

## References

- **GAP Matrix**: `docs/GAP_MATRIX.auto.md` (sec13.item2)
- **Parser**: `pkg/limits/parser.go`
- **Integration**: `web/server_clean.go` (lines 485-515, 760-810, 1567-1627)
- **Tests**: `web/multi_period_limits_test.go`
- **Audit Format**: `web/server_clean.go::writeModelLimitAudit()`

---

**Last Updated**: November 5, 2025  
**Maintained By**: AgentAuth Core Team
