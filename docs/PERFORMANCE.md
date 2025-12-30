---
title: Performance
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth Performance Guide

> Last Updated: 2025-10-17 (post Milestone 2B authenticity instrumentation)
> Status: Active

This guide provides tips and best practices for optimizing AgentAuth performance in development environments.

## Performance Patterns

### 1. Token Management

#### Caching
Implement token caching for frequently accessed tokens:

```go
type CachedTokenStore struct {
    cache     *cache.Cache
    store     token.Store
    hitMetric metrics.Counter
}

func (s *CachedTokenStore) Get(ctx context.Context, id string) (*token.Token, error) {
    // Try cache first
    if cached, ok := s.cache.Get(id); ok {
        s.hitMetric.Inc()
        return cached.(*token.Token), nil
    }
    
    // Fallback to store
    token, err := s.store.Get(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Cache for future
    s.cache.Set(id, token, token.TimeToExpiry())
    return token, nil
}
```

#### Batch Operations
Use batch operations for multiple tokens:

```go
type BatchTokenStore interface {
    GetMany(ctx context.Context, ids []string) ([]*token.Token, error)
    SaveMany(ctx context.Context, tokens []*token.Token) error
}

// Implementation
func (s *TokenStore) GetMany(ctx context.Context, ids []string) ([]*token.Token, error) {
    results := make([]*token.Token, 0, len(ids))
    pipe := s.redis.Pipeline()
    
    // Queue all gets
    for _, id := range ids {
        pipe.Get(ctx, id)
    }
    
    // Execute in single round trip
    cmds, err := pipe.Exec(ctx)
    if err != nil {
        return nil, err
    }
    
    // Process results
    for _, cmd := range cmds {
        // ... process each result
    }
    
    return results, nil
}
```

### 2. Rate Limiting

#### Distributed Rate Limiting
Use Redis for distributed rate limiting:

```go
type RedisRateLimiter struct {
    client  *redis.Client
    script  *redis.Script
    metrics *RateLimitMetrics
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string) error {
    start := time.Now()
    defer func() {
        l.metrics.LatencyHistogram.Observe(time.Since(start).Seconds())
    }()
    
    // Use Redis EVAL for atomic operations
    allowed, err := l.script.Run(ctx, l.client,
        []string{key},
        l.limit,
        l.window.Seconds(),
    ).Result()
    
    if err != nil {
        l.metrics.ErrorCounter.Inc()
        return err
    }
    
    if !allowed.(bool) {
        l.metrics.RejectionCounter.Inc()
        return ErrRateLimitExceeded
    }
    
    l.metrics.AllowedCounter.Inc()
    return nil
}
```

### 3. Token Validation

#### Fast Path Validation
Implement quick validation checks:

```go
type FastValidator struct {
    publicKeys sync.Map // cache of public keys
    algorithm  token.Algorithm
}
## Benchmarking & Regression Detection

### Current Crypto / Delegation Baseline (Apple M3 Pro, single run)

| Benchmark | ns/op | B/op | allocs/op | Notes |
|-----------|-------|------|-----------|-------|
| BenchmarkSignCanonicalPOA | ~16,500 | ~952 | 10 | Canonical JSON + Ed25519 sign |
| BenchmarkVerifyCanonicalPOA | ~35,600 | ~888 | 9 | Canonical JSON + Ed25519 verify |
| BenchmarkValidateDelegation (full service) | ~47,500 | (var) | (var) | Includes audit event logging & metrics |

Target: Signature verification adds <15% over pre‑signing validation baseline; current end‑to‑end delta acceptable. Re‑measure after revocation reason & integrity enhancements.

To run focused crypto benchmarks:
```bash
go test -run=NONE -bench=Benchmark(Sign|Verify) -benchmem ./pkg/aap001
```

Nightly benchmarks run via `.github/workflows/bench.yml` (02:00 UTC) and upload raw + parsed artifacts.

### Local Benchmark Run
Simple one-off:
```bash
go test -bench=Authorize -benchmem ./pkg/authz
```

Stable multi-run (captures multiple packages & aggregates):
```bash
./scripts/run_benchmarks.sh
```
Output directory: `build/bench/<timestamp>/` with `raw.txt` and `summary.txt`.

### Regression Comparison
Capture previous and new results, then:
```bash
scripts/compare-bench.sh old.txt new.txt 15
```
Threshold defaults to 10%; any ns/op, B/op or allocs/op increase beyond threshold flags `REGRESSION`.

### Interpreting Metrics
* `ns/op` – latency per operation (decision, validation, issuance, etc.).
* `B/op` – bytes allocated (heap pressure). Keep hot-path cache hits near zero allocations.
* `allocs/op` – allocation count; fewer reduces GC pressure.
* `p50/p90/p99` (internal in-memory metrics) – approximate latency percentiles for validation path, derived from a recent fixed-size reservoir (256 samples). These are for quick local regression hints; rely on external observability (Prometheus, OTEL) for production-grade histograms.

### Optimization Targets
1. Cache hit path: reuse decision structs to reduce allocations.
2. Miss path: pre-index policies, short-circuit denies.
3. Future regex evaluation: precompile & share compiled forms.

### Future Automation
Planned: CI gating on > X% degradation and historical trend chart generation.

Planned extension: Export internal percentile metrics via optional Prometheus adapter once histogram buckets are finalized.

See `CODE_STYLE.md` for benchmarking conventions and naming.

func (v *FastValidator) ValidateQuick(token string) error {
    // Quick format check
    if !v.isValidFormat(token) {
        return ErrInvalidFormat
    }
    
    // Header-only parsing
    header, err := v.parseHeader(token)
    if err != nil {
        return err
    }
    
    // Get cached key
    key, ok := v.publicKeys.Load(header.KeyID)
    if !ok {
        return ErrKeyNotFound
    }
    
    // Fast signature check
    return v.algorithm.FastVerify(token, key)
}
```

## Resource Management

### 1. Connection Pooling

```go
type PoolConfig struct {
    MaxIdleConns    int
    MaxOpenConns    int
    ConnMaxLifetime time.Duration
}

func NewConnectionPool(config PoolConfig) *sql.DB {
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    return db
}
```

### 2. Resource Cleanup

```go
type ResourceManager struct {
    cleanupInterval time.Duration
    lastCleanup    time.Time
    mu             sync.Mutex
}

func (m *ResourceManager) Cleanup(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if time.Since(m.lastCleanup) < m.cleanupInterval {
        return nil
    }
    
    // Perform cleanup
    if err := m.cleanupExpiredTokens(ctx); err != nil {
        return err
    }
    
    m.lastCleanup = time.Now()
    return nil
}
```

## Optimization Techniques

### 1. Memory Optimization

```go
type OptimizedToken struct {
    id        [16]byte    // Use fixed size arrays
    issuedAt  int64      // Use primitive types
    expiresAt int64
    flags     uint8      // Use bit flags
}

// Use object pool for frequent allocations
var tokenPool = sync.Pool{
    New: func() interface{} {
        return new(OptimizedToken)
    },
}

func GetToken() *OptimizedToken {
    return tokenPool.Get().(*OptimizedToken)
}

func PutToken(t *OptimizedToken) {
    // Reset fields
    *t = OptimizedToken{}
    tokenPool.Put(t)
}
```

### 2. Concurrent Processing

```go
type TokenProcessor struct {
    workers  int
    queue    chan tokenJob
    wg       sync.WaitGroup
}

func (p *TokenProcessor) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker()
    }
}

func (p *TokenProcessor) worker() {
    defer p.wg.Done()
    for job := range p.queue {
        // Process token job
        result := p.processToken(job.token)
        job.resultCh <- result
    }
}
```

## Performance Testing

### 1. Benchmarking

```go
func BenchmarkTokenValidation(b *testing.B) {
    validator := NewFastValidator()
    token := generateTestToken()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            validator.ValidateQuick(token)
        }
    })
}
```

### 2. Load Testing

```go
type LoadTest struct {
    rate      int
    duration  time.Duration
    validator *FastValidator
}

func (t *LoadTest) Run(ctx context.Context) *LoadTestResult {
    results := make(chan time.Duration, t.rate*int(t.duration.Seconds()))
    ticker := time.NewTicker(time.Second / time.Duration(t.rate))
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return t.analyzeResults(results)
        case <-ticker.C:
            go t.makeRequest(results)
        }
    }
}
```

## Monitoring

### 1. Performance Metrics

```go
type Metrics struct {
    TokenValidationDuration prometheus.Histogram
    CacheHitRate           prometheus.Gauge
    ActiveTokens           prometheus.Gauge
    ValidationErrors       prometheus.Counter
}

func NewMetrics() *Metrics {
    return &Metrics{
        TokenValidationDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "token_validation_duration_seconds",
            Help:    "Token validation duration in seconds",
            Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
        }),
        // ... other metrics
    }
}
```

### 2. Health Checks

```go
type HealthChecker struct {
    checks map[string]HealthCheck
}

func (h *HealthChecker) AddCheck(name string, check HealthCheck) {
    h.checks[name] = check
}

func (h *HealthChecker) RunChecks(ctx context.Context) *HealthReport {
    results := make(map[string]*HealthResult)
    for name, check := range h.checks {
        results[name] = check.Run(ctx)
    }
    return NewHealthReport(results)
}
```

## Configuration Examples

### 1. High-Performance Configuration

```go
config := &Config{
    TokenStore: &StoreConfig{
        CacheSize:        10000,
        CacheTTL:        5 * time.Minute,
        CleanupInterval: 10 * time.Minute,
    },
    RateLimit: &RateLimitConfig{
        Algorithm:      "token_bucket",
        Capacity:      1000,
        FillRate:      100,
        BatchSize:     10,
    },
    Validation: &ValidationConfig{
        FastValidation: true,
        CacheKeys:     true,
        BatchSize:     50,
    },
}
```

### 2. Resource-Optimized Configuration

```go
config := &Config{
    TokenStore: &StoreConfig{
        MaxTokens:       1000000,
        GCInterval:     1 * time.Hour,
        MaxBatchSize:   100,
    },
    Connection: &ConnectionConfig{
        MaxIdleConns:    10,
        MaxOpenConns:    100,
        ConnMaxLifetime: 1 * time.Hour,
    },
    WorkerPool: &WorkerConfig{
        Workers:        runtime.NumCPU(),
        QueueSize:      1000,
        BatchTimeout:   100 * time.Millisecond,
    },
}
```

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

## Revocation Consistency Proof V2 Optimization (2025-10-20)

[↔ See Transparency Details](REVOCATION_TRANSPARENCY.md#consistency-proof-v2-generation-optimization-2025-10-20)

### Diagram: Prefix Block Decomposition & Bridge Reduction
```mermaid
flowchart LR
    A1[Leaf0] --> A2[Leaf1]
    A1 --- A2
    subgraph Block1 (size=2)
        R1[(H01)]
    end
    B1[Leaf2] --> B2[Leaf3]
    B1 --- B2
    subgraph Block2 (size=2)
        R2[(H23)]
    end
    C1[Leaf4]
    subgraph Block3 (size=1)
        R3[(H4)]
    end
    R1 --> R2
    R2 --> R3
    subgraph Bridges
        BR1[(Merge R2,R3)]
        BR2[(Merge R1,BR1)]
    end
    R1 --> BR1
    BR1 --> BR2
    style BR2 fill:#ffd,stroke:#333,stroke-width:1px
```
Blocks (R1,R2,R3) formed by streaming; bridges (BR1,BR2) reduce right-to-left to final root.

### Glossary (Consistency Optimization)
<a id="glossary-consistency"></a>
| Term | Definition | Notes |
|------|------------|-------|
| `segment stack` | In-memory list of `(hash,size)` power-of-two subtrees produced by streaming leaf ingestion | Size ≤ `log2(n)+1` |
| `prefix_roots` | Ordered subtree root hashes for each power-of-two block covering historical prefix | Used for fast reconstruction & auditing |
| `prefix_sizes` | Length in leaves of each prefix block (power-of-two) | Sum equals `start_length` |
| `prefix_bridges` | Sequence of intermediate merge results from right-to-left reduction of prefix blocks | Deterministic; replay yields `start_root` |
| `fast reconstruction` | Algorithm reducing blocks via bridges to compute `start_root` without full tree rebuild | Guarded by `AGENTAUTH_CONSISTENCY_V2_FAST` |
| `temporary tree` | Full Merkle rebuild still used for path derivation (current phase) | Target for removal next phase |
| `interval-based path` | Planned method computing sibling hashes from streamed frontier without full rebuild | Future optimization |
| `frontier` | Emerging highest-level subtree roots after each leaf merge | Could underpin interval path algorithm |

> Architectural diagrams of the revocation transparency flow available in `REVOCATION_TRANSPARENCY.md#16-architecture-diagrams-revocation-transparency`.

### Interval-Based Streaming Consistency Path (Experimental)
Flag: `AGENTAUTH_CONSISTENCY_V2_INTERVAL_PATH=1`

Objective: Eliminate temporary full Merkle tree reconstruction during `GenerateConsistencyProofV2` path derivation.

Approach:
1. Reuse pre-computed leaf-domain digests (`LeafDigestForEventHash`).
2. Use largest power-of-two divisor (LSB) of current `oldSize` to identify maximal aligned block ending at `oldSize`.
3. Compute sibling block root for range `[oldSize, oldSize+lsb)` when fully contained in end size.
4. Emit sibling hash with position "R" (current block is left) and advance `oldSize`.
5. Repeat until `oldSize == newSize` or range crosses end boundary.

Complexity: Worst-case still O(k * log m) hashing where k blocks < log n; avoids materializing `levels` slice for all n.

Current Status: Prototype—verifier still performs canonical full-tree rebuild for path validation; next step is interval-aware verification.

Benchmark Placeholder (to populate after measurement):
| Size (n) | Legacy Path (µs) | Interval Path (µs) | Improvement |
|----------|------------------|--------------------|-------------|
| 512      | 262.6            | 197.7              | 1.33x faster |
| 1024     | 534.7            | 417.9              | 1.28x faster |
| 2048     | 1116.0           | 842.6              | 1.32x faster |
| 4096     | 2197.3           | 1646.0             | 1.33x faster |

Memory/Alloc Impact:
- 4096 leaves legacy: 4.77 MB alloc, 49,194 allocs
- 4096 leaves interval: 3.54 MB alloc, 36,870 allocs (~26% fewer allocs)

Environment: Apple M3 Pro (arm64), Go test benchmark mode, AGENTAUTH_CONSISTENCY_V2_INTERVAL_PATH toggled.

Risks / Edge Cases:
- Non-power-of-two growth segments may yield repeated small sibling extractions (diminishing returns).
- Path length must remain within logarithmic bound; guard present in verification.
- Future optimization: compute multiple consecutive sibling blocks by scanning bits of `(endSize - oldSize)`.

Fallback: If flag not set, legacy level traversal executes unchanged.


### Overview
The Consistency Proof V2 generation path was optimized to reduce allocations and latency when producing append-only proofs between historical and latest Signed Tree Heads. The previous implementation rebuilt all Merkle levels (O(n) memory) to derive:
1. Logarithmic consistency path (`path`, `positions`)
2. Prefix block decomposition (`prefix_roots`, `prefix_sizes`)

The new implementation streams leaves into a segment stack, merging adjacent equal-sized segments immediately. This produces a canonical power-of-two block decomposition without materializing every level. A right-to-left reduction over these blocks yields `prefix_bridges`, enabling fast deterministic reconstruction of the historical start root.

### Components
| Component | Before | After |
|----------|--------|-------|
| Prefix Decomposition | Full level scan | Streaming segment stack (O(k) space) |
| Start Root Reconstruction | Full Merkle rebuild | Reduction via `prefix_bridges` (fast path) |
| Consistency Path | Full level traversal | (Still full tree) – scheduled for next phase |
| Memory Footprint | All levels `[]merkleNode` | Stack of at most `log2(n)` segments |

### Data Structures
`segment{hash,size}` entries represent power-of-two sized subtrees. Adjacent equal sizes merge:
```
Leaf (size=1) + Leaf (size=1) -> Parent (size=2)
Parent (size=2) + Parent (size=2) -> Parent (size=4)
... until mismatch halts merge chain.
```
Final stack after processing first `start_length` leaves holds left→right prefix blocks used for fast reconstruction.

### Bridges Construction
Perform a right-to-left reduction:
```
Blocks: [B0, B1, B2, B3]
Merge(B2,B3) -> M1
Merge(B1,M1) -> M2 (possible consolidation if size(B1)==size(M1_before))
Merge(B0,M2) -> Root
```
Each intermediate merged digest is appended to `prefix_bridges`. Verification replays this sequence to reconstruct `start_root` and checks invariants.

### Benchmark Snapshot (Apple M3 Pro)
| Leaves | New Gen ns/op | Legacy Gen ns/op | New Alloc (B/op) | Legacy Alloc (B/op) | New allocs/op | Legacy allocs/op |
|--------|---------------|------------------|------------------|--------------------|---------------|------------------|
| 64     | 32,494        | 51,828           | 73,472           | 117,545            | 792           | 1,253            |
| 256    | 129,732       | 203,742          | 294,016          | 471,979            | 3,101         | 4,911            |
| 1024   | 530,838       | 837,270          | 1,196,560        | 1,930,798          | 12,325        | 19,516           |
| 4096   | 2,217,502     | 3,548,736        | 4,768,269        | 7,700,026          | 49,194        | 77,894           |
> Source: `BenchmarkConsistencyProofGeneration` (benchtime=0.2s). Values approximate; re-run for precise numbers.

### Fast Path Reconstruction Complexity
Let `b = len(prefix_roots)`. Reconstruction uses at most `b - 1` primary merges plus additional consolidation merges when adjacent merged sizes equal (`<= b-1`). Overall: `O(b)` time, `O(b)` space; with `b ≤ log2(start_length)+1`.

### Remaining Work (Planned)
1. Path-Only Generation: Replace temporary tree build for consistency path with interval-based sibling derivation using maintained frontier states (target: reduce generation memory & shave 10–20% latency for large trees).
2. Memoized Range Hashes: Cache subtree hashes during segment stack formation to eliminate repeated hashing in future path algorithm.
3. Allocation Tightening: Reuse segment slice via pool; avoid per-leaf temporary slice growth.
4. Auditor Telemetry Expansion: Capture allocation counts & GC pause deltas for fast vs legacy modes.
5. Concurrency: Explore parallel leaf hashing for large batch appends (prefetch next block while merging current). Requires deterministic ordering guarantee via buffered channel.

### Validation Invariants
During verification:
1. Sum(prefix_sizes) == start_length.
2. Each `prefix_sizes[i]` is power-of-two.
3. `prefix_bridges` reduction outputs canonical `start_root` (when fast flag enabled) matching full rebuild root.
4. Any mismatch aborts verification with explicit error (`fast_reconstruction_mismatch`).

### How To Re-Run Benchmarks
```bash
go test -run=NONE -bench=BenchmarkConsistencyProofGeneration -benchmem ./pkg/delegation
```
For a single size (e.g., 4096 leaves):
```bash
go test -run=NONE -bench=BenchmarkConsistencyProofGeneration/new_gen_4096 -benchmem ./pkg/delegation
```

### Troubleshooting
| Symptom | Cause | Action |
|---------|-------|--------|
| `fast_reconstruction_mismatch` | Non-deterministic merge order or altered hashing prefix | Confirm hashing domain strings unchanged; re-generate bridges. |
| Elevated allocs vs baseline | Accidental slice copy or forgotten pooling | Profile with `go test -cpuprofile / -memprofile` and inspect stack merges. |
| Path longer than bound | Incorrect sibling inclusion logic | Verify parity checks and convergence loop termination. |

### Security Considerations
Fast path never accepts a reconstructed root silently; it always cross-checks against canonical rebuild. This maintains integrity while allowing performance observation. Bridges are deterministic and replayable; they do not leak additional state beyond subtree hashes already inferable from prefix decomposition.

---
