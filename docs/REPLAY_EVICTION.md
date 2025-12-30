---
title: Replay Eviction
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Durable Replay Protection with Configurable Eviction Policies

## Overview

This document describes the enhanced replay protection system (sec6.item1 P1) that provides fail-closed JTI (JWT ID) uniqueness enforcement with configurable eviction policies, durable persistence, and comprehensive metrics.

## Motivation

**Security Requirements:**
- **Replay Attack Prevention**: Prevent malicious actors from reusing captured tokens to gain unauthorized access
- **Fail-Closed Mode**: Reject tokens on any replay detection error (no false negatives)
- **Crash Recovery**: Maintain replay state across process restarts to prevent replay after crashes
- **Memory Management**: Prevent unbounded memory growth in long-running services

**Operational Requirements:**
- **Flexible Eviction**: Support different eviction strategies based on deployment needs
- **Observability**: Track cache performance (hit/miss rates), eviction events, store size
- **Configuration**: Environment variable-based configuration for deployment flexibility
- **Performance**: Minimize latency impact of replay checks (<1ms p99)

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                    AgentAuth Service                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ ReplayStore Interface                                 │   │
│  │  - CheckAndStore(jti string) error                   │   │
│  └────────────────┬─────────────────────────────────────┘   │
│                   │                                          │
│                   ▼                                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ DurableReplayStoreAdapter                            │   │
│  │  - Wraps DurableReplayStore                          │   │
│  │  - Implements fail-closed semantics                  │   │
│  │  - Maps Seen()+Record() → CheckAndStore()            │   │
│  └────────────────┬─────────────────────────────────────┘   │
└────────────────────┼─────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│          DurableReplayStore (pkg/replay)                     │
│                                                              │
│  State:                                                      │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ entries:     map[string]time.Time  (JTI → timestamp) │   │
│  │ accessTimes: map[string]time.Time  (JTI → access)    │   │
│  │ evictionPolicy: EvictionPolicy                       │   │
│  │ evictionStats:  EvictionStats                        │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  Operations:                                                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Seen(jti)                                            │   │
│  │  1. Lock state                                       │   │
│  │  2. applyEviction(now)  ← Trigger eviction           │   │
│  │  3. Check existence                                  │   │
│  │  4. Update accessTimes[jti] = now  (LRU tracking)    │   │
│  │  5. Report metrics (hit/miss, store size)            │   │
│  │  6. Return result                                    │   │
│  │                                                       │   │
│  │ Record(jti, ts)                                      │   │
│  │  1. Lock state                                       │   │
│  │  2. entries[jti] = ts                                │   │
│  │  3. accessTimes[jti] = ts                            │   │
│  │  4. AppendWAL(Record op)  ← Durability               │   │
│  │  5. Report metrics (latency, store size)             │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  Eviction:                                                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ applyEviction(now)                                   │   │
│  │  - TTL: Scan entries, delete if now-ts > TTL        │   │
│  │  - Size: Sort by timestamp, evict oldest if > max   │   │
│  │  - LRU: Sort by accessTime, evict LRU if > max      │   │
│  │  - Composite: Apply all policies (ANY match evicts) │   │
│  │  - Update evictionStats, report metrics             │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  Persistence:                                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ WALStore                                             │   │
│  │  - Append-only log of Record operations             │   │
│  │  - Snapshot() creates point-in-time checkpoint      │   │
│  │  - Rotate() truncates WAL after snapshot            │   │
│  │  - Recover() replays WAL on startup                 │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Eviction Policies

Four pluggable eviction strategies are supported:

#### 1. **TTL Eviction** (Time-Based)
- **Strategy**: Evict entries older than configured TTL
- **Use Case**: Token validity period-based cleanup (e.g., JTI valid for 15 minutes)
- **Implementation**: `TTLEvictionPolicy{TTL: duration}`
- **Complexity**: O(n) scan during eviction trigger
- **Memory**: Unbounded without size limit
- **Configuration**: `GAUTH_REPLAY_EVICTION_POLICY=ttl GAUTH_REPLAY_TTL_SEC=900`

#### 2. **Size-Based Eviction**
- **Strategy**: Evict oldest entries when count exceeds max size
- **Use Case**: Memory-constrained environments, fixed capacity
- **Implementation**: `SizeBasedEvictionPolicy{MaxSize: count}`
- **Complexity**: O(n log n) sort by timestamp when evicting
- **Memory**: Bounded by MaxSize
- **Configuration**: `GAUTH_REPLAY_EVICTION_POLICY=size GAUTH_REPLAY_EVICTION_MAX_SIZE=10000`

#### 3. **LRU Eviction** (Least Recently Used)
- **Strategy**: Evict least recently accessed entries when count exceeds max size
- **Use Case**: Hot JTI caching, active token protection
- **Implementation**: `LRUEvictionPolicy{MaxSize: count}` with accessTimes tracking
- **Complexity**: O(n log n) sort by access time when evicting
- **Memory**: Bounded by MaxSize + accessTimes overhead
- **Configuration**: `GAUTH_REPLAY_EVICTION_POLICY=lru GAUTH_REPLAY_EVICTION_MAX_SIZE=10000`

#### 4. **Composite Eviction**
- **Strategy**: Combine multiple policies (ANY match triggers eviction)
- **Use Case**: Hybrid scenarios (e.g., "TTL + size limit")
- **Implementation**: `CompositeEvictionPolicy{Policies: []EvictionPolicy}`
- **Complexity**: Sum of constituent policy complexities
- **Memory**: Constrained by strictest policy
- **Configuration**: `GAUTH_REPLAY_EVICTION_POLICY=ttl+size`

### Eviction Trigger Points

Eviction is triggered lazily during:
1. **Read Operations** (`Seen()`): Before checking JTI existence
2. **Snapshot Operations** (`Snapshot()`): Before writing checkpoint
3. **Manual Triggers**: Via SnapshotTrigger.Trigger() API

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GAUTH_REPLAY_WAL_PATH` | `./data/replay.wal` | Path to Write-Ahead Log file |
| `GAUTH_REPLAY_TTL_SEC` | `900` (15 min) | TTL for replay entries (seconds) |
| `GAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC` | `300` (5 min) | Automatic snapshot interval (seconds) |
| `GAUTH_REPLAY_EVICTION_POLICY` | `ttl` | Eviction policy: `ttl`, `lru`, `size`, `ttl+size` |
| `GAUTH_REPLAY_EVICTION_MAX_SIZE` | `10000` | Max entries for size/LRU policies |

### Eviction Policy Selection Guide

| Scenario | Recommended Policy | Rationale |
|----------|-------------------|-----------|
| **Token TTL = 15 min** | `ttl` | Natural expiration matches token lifetime |
| **Memory-constrained (e.g., 512MB)** | `size` (5000-10000) | Fixed memory footprint |
| **High-traffic API (1000 req/s)** | `lru` (50000) | Keep hot tokens, evict cold |
| **Compliance (90-day retention)** | `ttl` (7776000 sec) | Long retention with periodic cleanup |
| **Hybrid (TTL + capacity)** | `ttl+size` | Expiration + memory safety |

### Code Examples

#### Basic Integration (Manual Configuration)

```go
import (
    "github.com/.../pkg/gauth"
    "github.com/.../pkg/replay"
)

// Create DurableReplayStore with TTL eviction
config := replay.DurableReplayStoreConfig{
    WALPath:        "/var/lib/gauth/replay.wal",
    TTL:            15 * time.Minute,
    SnapshotInterval: 5 * time.Minute,
    EvictionPolicy: &replay.TTLEvictionPolicy{TTL: 15 * time.Minute},
    Metrics:        myMetricsImpl, // Implement DurableReplayMetrics
}

store, err := replay.NewDurableReplayStore(config)
if err != nil {
    log.Fatalf("Failed to create replay store: %v", err)
}
defer store.Close()

// Wrap for gauth ReplayStore interface
adapter := replay.NewDurableReplayStoreAdapter(store)

// Inject into gauth
gauthSvc, err := gauth.New(gauthConfig, gauth.WithReplayStore(adapter))
```

#### Environment Variable Configuration

```bash
# Set environment variables
export GAUTH_REPLAY_WAL_PATH="/var/lib/gauth/replay.wal"
export GAUTH_REPLAY_TTL_SEC=900
export GAUTH_REPLAY_EVICTION_POLICY="ttl+size"
export GAUTH_REPLAY_EVICTION_MAX_SIZE=50000

# Application code
```

```go
import (
    "github.com/.../pkg/gauth"
    "github.com/.../pkg/replay"
)

// Auto-configure from environment
store, err := replay.NewDurableReplayStoreFromEnv(myMetricsImpl)
if err != nil {
    log.Fatalf("Failed to create replay store from env: %v", err)
}
defer store.Close()

adapter := replay.NewDurableReplayStoreAdapter(store)
gauthSvc, err := gauth.New(gauthConfig, gauth.WithReplayStore(adapter))
```

#### Size-Based Eviction (Fixed Capacity)

```go
config := replay.DurableReplayStoreConfig{
    WALPath:        "/var/lib/gauth/replay.wal",
    TTL:            1 * time.Hour, // Long TTL (eviction by size, not time)
    EvictionPolicy: &replay.SizeBasedEvictionPolicy{MaxSize: 10000},
    MaxSize:        10000,
}
```

#### LRU Eviction (Hot Token Caching)

```go
config := replay.DurableReplayStoreConfig{
    WALPath:        "/var/lib/gauth/replay.wal",
    TTL:            30 * time.Minute,
    EvictionPolicy: &replay.LRUEvictionPolicy{MaxSize: 50000},
    MaxSize:        50000,
}
```

#### Composite Policy (TTL + Size Safety)

```go
config := replay.DurableReplayStoreConfig{
    WALPath:        "/var/lib/gauth/replay.wal",
    TTL:            15 * time.Minute,
    EvictionPolicy: &replay.CompositeEvictionPolicy{
        Policies: []replay.EvictionPolicy{
            &replay.TTLEvictionPolicy{TTL: 15 * time.Minute},
            &replay.SizeBasedEvictionPolicy{MaxSize: 100000},
        },
    },
}
```

## Operational Guide

### Deployment Configurations

#### Development Environment
```bash
GAUTH_REPLAY_WAL_PATH="./tmp/replay.wal"
GAUTH_REPLAY_TTL_SEC=600              # 10 minutes (short for dev)
GAUTH_REPLAY_EVICTION_POLICY=ttl
GAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC=60 # 1 minute (frequent snapshots)
```

#### Production (High Traffic)
```bash
GAUTH_REPLAY_WAL_PATH="/var/lib/gauth/replay.wal"
GAUTH_REPLAY_TTL_SEC=900                # 15 minutes
GAUTH_REPLAY_EVICTION_POLICY=lru
GAUTH_REPLAY_EVICTION_MAX_SIZE=100000   # 100K hot tokens
GAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC=300  # 5 minutes
```

#### Production (Memory-Constrained)
```bash
GAUTH_REPLAY_WAL_PATH="/var/lib/gauth/replay.wal"
GAUTH_REPLAY_TTL_SEC=900
GAUTH_REPLAY_EVICTION_POLICY=ttl+size
GAUTH_REPLAY_EVICTION_MAX_SIZE=10000    # Strict limit
GAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC=300
```

#### Compliance (Long Retention)
```bash
GAUTH_REPLAY_WAL_PATH="/mnt/audit/replay.wal"
GAUTH_REPLAY_TTL_SEC=7776000            # 90 days
GAUTH_REPLAY_EVICTION_POLICY=ttl
GAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC=3600 # Hourly snapshots
```

### Monitoring

#### Metrics to Track

**Performance Metrics:**
- `replay_cache_hit_total`: Cache hit count (existing JTI detected)
- `replay_cache_miss_total`: Cache miss count (new JTI)
- `replay_store_latency_seconds`: Time spent in Seen()/Record() operations
- `replay_wal_flush_latency_seconds`: WAL append latency

**Eviction Metrics:**
- `replay_evictions_total{reason="ttl|lru|size|composite"}`: Eviction events by policy
- `replay_store_size`: Current number of entries in store
- `replay_store_errors_total`: Errors during operations

**Persistence Metrics:**
- `replay_wal_pending`: Pending WAL entries before next snapshot
- `replay_wal_snapshot_duration_seconds`: Time to create snapshot

#### Prometheus Queries

**Cache Hit Rate:**
```promql
rate(replay_cache_hit_total[5m]) / 
(rate(replay_cache_hit_total[5m]) + rate(replay_cache_miss_total[5m]))
```

**Eviction Rate (per minute):**
```promql
rate(replay_evictions_total[1m]) * 60
```

**Store Size Trending:**
```promql
replay_store_size
```

**p99 Latency:**
```promql
histogram_quantile(0.99, rate(replay_store_latency_seconds_bucket[5m]))
```

#### Alert Rules

**High Eviction Rate:**
```yaml
- alert: HighReplayEvictionRate
  expr: rate(replay_evictions_total[5m]) > 100
  for: 10m
  annotations:
    summary: "Replay store evicting >100 entries/sec (may need capacity increase)"
```

**Store Size Near Limit:**
```yaml
- alert: ReplayStoreNearCapacity
  expr: replay_store_size > 0.9 * $MAX_SIZE
  for: 5m
  annotations:
    summary: "Replay store at 90% capacity (consider increasing max size)"
```

**High Latency:**
```yaml
- alert: HighReplayStoreLatency
  expr: histogram_quantile(0.99, rate(replay_store_latency_seconds_bucket[5m])) > 0.001
  for: 5m
  annotations:
    summary: "p99 replay store latency >1ms (performance degradation)"
```

### Maintenance

#### Manual Snapshot Trigger

```go
trigger := replay.NewSnapshotTrigger(store)
if err := trigger.Trigger(context.Background()); err != nil {
    log.Errorf("Manual snapshot failed: %v", err)
}
```

#### Graceful Shutdown

```go
// Store auto-creates final snapshot on Close()
if err := store.Close(); err != nil {
    log.Errorf("Failed to close replay store: %v", err)
}
```

#### WAL Rotation (Automatic)

Snapshots automatically trigger WAL rotation:
1. Create snapshot with active entries
2. Truncate WAL file
3. Re-append active entries to new WAL
4. Delete old WAL

#### Backup Strategy

```bash
# Snapshot file contains full state
cp /var/lib/gauth/replay.wal.snapshot /backup/replay-$(date +%Y%m%d).snapshot

# WAL contains recent changes
cp /var/lib/gauth/replay.wal /backup/replay-$(date +%Y%m%d).wal
```

## Migration Guide

### Phase 1: Add DurableReplayStore (Week 1)

**Goal**: Deploy alongside existing in-memory replay protection.

**Steps:**
1. Add environment variables to deployment config (start with TTL policy)
2. Initialize DurableReplayStore in application startup
3. Inject via `WithReplayStore()` option
4. Monitor metrics (cache hit rate, eviction rate, latency)
5. Verify crash recovery by restarting service and checking WAL replay

**Rollback**: Remove `WithReplayStore()` option, revert to in-memory `seenJTI` map

### Phase 2: Optimize Eviction Policy (Week 2-4)

**Goal**: Tune eviction policy based on production traffic.

**Steps:**
1. Analyze cache hit rate and store size trends
2. If store size unbounded: Switch to `size` or `lru` policy
3. If frequent evictions: Increase `GAUTH_REPLAY_EVICTION_MAX_SIZE`
4. If memory pressure: Switch to `ttl+size` composite policy
5. Adjust snapshot interval based on WAL size growth

**Metrics to Watch:**
- Cache hit rate (target: >95% for hot tokens)
- Eviction rate (target: <10% of request rate)
- p99 latency (target: <1ms)
- Store size (target: stable, not growing unbounded)

### Phase 3: Remove In-Memory Fallback (Week 5+)

**Goal**: Fully migrate to DurableReplayStore.

**Steps:**
1. Verify 99.99% uptime with DurableReplayStore for 2+ weeks
2. Confirm crash recovery tested in staging
3. Remove legacy `seenJTI` map from gauth.Service (code cleanup)
4. Document final configuration in runbooks

## Security Considerations

### Fail-Closed Semantics

**Guarantee**: Any error during replay check causes token rejection.

**Implementation:**
```go
func (a *DurableReplayStoreAdapter) CheckAndStore(jti string) error {
    seen, err := a.store.Seen(jti)
    if err != nil {
        return fmt.Errorf("replay store error: %w", err) // Fail-closed
    }
    if seen {
        return fmt.Errorf("replay detected: JTI %s already seen", jti)
    }
    // ... Record JTI ...
}
```

**Error Scenarios:**
- WAL append failure → Reject token
- File I/O error during snapshot → Reject token
- Lock contention timeout → Reject token

### Clock Skew Tolerance

**Issue**: TTL eviction depends on accurate system time.

**Mitigation:**
- Use NTP-synchronized clocks in production
- Record operations clamp future timestamps to `now`
- TTL buffer (e.g., TTL = token validity + 10%)

### Resource Exhaustion Defense

**Attack**: Flood with unique JTIs to exhaust memory.

**Mitigations:**
1. **Size-Based Eviction**: Hard cap on entry count
2. **Rate Limiting**: Limit tokens issued per minute (upstream)
3. **Monitoring**: Alert on rapid store size growth

### Adversarial Behavior

**Attack**: Request old JTI repeatedly to keep in LRU cache.

**Mitigation:**
- LRU updates access time on hit AND miss (defense: no access time update on miss in current impl)
- Composite policy (TTL + LRU) ensures eventual expiration regardless of access pattern

## Performance Characteristics

### Latency

| Operation | Complexity | Typical Latency | p99 Latency |
|-----------|-----------|----------------|-------------|
| Seen() (hit) | O(1) | 50-100μs | 200μs |
| Seen() (miss) | O(1) | 50-100μs | 200μs |
| Record() | O(1) + WAL append | 100-300μs | 500μs |
| Eviction (TTL) | O(n) scan | Varies | <1ms (amortized) |
| Eviction (Size/LRU) | O(n log n) sort | Varies | <5ms (triggered infrequently) |
| Snapshot() | O(n) serialize | 10-50ms (10K entries) | 100ms (100K entries) |

### Memory

| Policy | Memory Formula | Example (10K entries) |
|--------|---------------|----------------------|
| TTL | 16 * n + overhead | ~160KB + maps overhead |
| Size | 16 * n + overhead | ~160KB (bounded) |
| LRU | 16 * n + 16 * n (accessTimes) | ~320KB (bounded) |
| Composite | Max of constituent policies | Varies |

**Note**: `n` = number of active entries, overhead includes Go map allocation.

### Disk I/O

| Operation | I/O Pattern | Frequency |
|-----------|------------|-----------|
| WAL append | Sequential write (append-only) | Per token (buffered) |
| Snapshot | Sequential write + read | Every 5 minutes (configurable) |
| WAL rotation | Truncate + rewrite | After each snapshot |
| Recovery | Sequential read | At startup |

**Optimization**: WAL uses buffered I/O; snapshot triggers batched writes.

## Testing Summary

### Test Coverage

**Unit Tests (pkg/replay/eviction_test.go):**
- TestTTLEviction: TTL-based expiration ✅
- TestSizeBasedEviction: Oldest entry removal ✅
- TestLRUEviction: Least recently used eviction ✅
- TestCompositeEviction: Multi-policy enforcement ✅
- TestParseEvictionPolicy: Env var parsing (9 scenarios) ✅
- TestEvictionMetrics: Metrics instrumentation ✅
- TestEvictionConcurrency: Thread safety ✅

**Integration Tests (pkg/replay/gauth_integration_test.go):**
- TestDurableReplayStoreAgentAuthIntegration: CheckAndStore interface ✅
- TestDurableReplayStoreFromEnv: Environment configuration ✅
- TestDurableReplayStoreWithSizeEviction: Size policy with env vars ✅
- TestDurableReplayStoreCompositePolicy: TTL+size composite ✅
- TestDurableReplayStoreEnvDefaults: Default value validation ✅
- TestDurableReplayStorePersistence: Crash recovery ✅

**Total**: 13 test functions, all passing (100% pass rate)

### Manual Testing

**Crash Recovery:**
1. Start service with DurableReplayStore
2. Process 1000 tokens
3. Kill -9 service (simulate crash)
4. Restart service
5. Verify: Previous JTIs detected as replays, new JTIs accepted

**Eviction Verification:**
1. Configure `GAUTH_REPLAY_EVICTION_POLICY=size GAUTH_REPLAY_EVICTION_MAX_SIZE=100`
2. Issue 200 tokens
3. Verify: Store size stays ≤ 100, oldest evicted

**Metrics Validation:**
1. Configure Prometheus scraping
2. Process traffic for 1 hour
3. Query `replay_cache_hit_total`, `replay_evictions_total`, `replay_store_size`
4. Verify: Metrics align with expected behavior

## Future Enhancements

### Distributed Replay Store
- **Goal**: Share replay state across multiple service instances
- **Approach**: Redis-backed DurableReplayStore implementation
- **Benefit**: Horizontal scalability, consistent replay protection

### Advanced Eviction Policies
- **Probabilistic Eviction**: Bloom filter for approximate membership
- **Adaptive TTL**: Adjust TTL based on traffic patterns
- **Priority-Based**: Evict low-priority JTIs first

### External Anchoring
- **Goal**: Immutable audit trail of all JTIs
- **Approach**: Periodic Merkle root of JTI set → external log (e.g., blockchain, transparency log)
- **Benefit**: Non-repudiation, compliance

### Performance Optimizations
- **Sharding**: Partition JTI space across multiple stores
- **Async WAL**: Batch WAL appends to reduce I/O
- **mmap Snapshot**: Memory-mapped file for faster recovery

## References

- **sec6.item1 GAP**: "In-memory JTI map + optional ReplayStore reject duplicates/errors; missing durable persistence & eviction controls"
- **P2.3 (DurableReplayStore)**: Initial WAL-based persistence implementation
- **OWASP A08:2021**: Software and Data Integrity Failures (replay attack category)
- **RFC 7519 (JWT)**: Section 4.1.7 (jti claim for replay prevention)

---

**Document Version**: 1.0  
**Last Updated**: November 5, 2025  
**Maintainer**: AgentAuth Core Team
