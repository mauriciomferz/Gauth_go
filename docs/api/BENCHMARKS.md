# Benchmark Guide

This guide explains GAuth's benchmarks and how to use them effectively.

## Running Benchmarks

### Basic Usage
```bash
# Run all benchmarks
make bench

# Run specific benchmark
go test -bench=BenchmarkAuthFlow ./test/benchmarks/...

# Run with memory stats
go test -bench=. -benchmem ./test/benchmarks/...
```

### Profile Generation
```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./test/benchmarks/...

# Memory profiling
go test -memprofile=mem.prof -bench=. ./test/benchmarks/...

# Profile analysis
go tool pprof cpu.prof
go tool pprof mem.prof
```

## Core Benchmarks

### Authentication
```
BenchmarkAuthFlow_Basic        # Basic auth performance
BenchmarkAuthFlow_JWT         # JWT operations
BenchmarkAuthFlow_Parallel    # Concurrent auth
```

### Token Management
```
BenchmarkToken_Store          # Token storage operations
BenchmarkToken_Redis         # Redis storage performance
BenchmarkToken_Rotation      # Key rotation impact
```

### Rate Limiting
```
BenchmarkRateLimit_Allow     # Rate limit checks
BenchmarkRateLimit_Sliding   # Sliding window algorithm
BenchmarkRateLimit_Redis     # Distributed rate limiting
```

## Performance Guidelines

### Authentication Flow
- Token validation: < 1ms
- Basic auth: < 10ms
- OAuth flow: < 100ms

### Token Operations
- Generation: < 1ms
- Validation: < 0.5ms
- Storage: < 5ms

### Rate Limiting
- Local check: < 0.1ms
- Distributed: < 10ms

## Example Results

```
BenchmarkAuthFlow_Basic-8     	  100000	     12042 ns/op	    2048 B/op	      24 allocs/op
BenchmarkAuthFlow_JWT-8      	  200000	      6521 ns/op	    1024 B/op	      16 allocs/op
BenchmarkToken_Store-8       	 1000000	      1023 ns/op	     128 B/op	       2 allocs/op
```

## Performance Tips

1. **Token Management**
   - Use in-memory store for high performance
   - Cache validated tokens
   - Batch token operations

2. **Rate Limiting**
   - Use local cache for frequent checks
   - Optimize Redis connections
   - Consider sliding window size

3. **Authentication**
   - Cache user sessions
   - Use fast password hashing
   - Optimize JWT signing

## Benchmark Categories

### 1. Core Operations
- Token generation/validation
- Authentication flows
- Authorization checks

### 2. Storage Operations
- In-memory performance
- Redis operations
- Cache effectiveness

### 3. Concurrent Load
- Parallel requests
- Resource contention
- Lock performance

## Writing New Benchmarks

### Template
```go
func BenchmarkFeature(b *testing.B) {
    // Setup
    svc := setupService()

    // Reset timer after setup
    b.ResetTimer()

    // Run benchmark
    for i := 0; i < b.N; i++ {
        svc.Operation()
    }
}
```

### Best Practices
1. Clear benchmark names
2. Proper setup/teardown
3. Reset timer after setup
4. Report memory allocations
5. Test different loads

## Continuous Monitoring

### Metrics to Track
- Operation latency
- Memory usage
- Allocation counts
- Cache hit rates

### Integration
```go
func init() {
    metrics.Register("auth_latency", prometheus.NewHistogram(...))
}

func BenchmarkWithMetrics(b *testing.B) {
    for i := 0; i < b.N; i++ {
        start := time.Now()
        operation()
        metrics.Observe("auth_latency", time.Since(start))
    }
}
```

## Further Reading

1. Go Testing Package: https://golang.org/pkg/testing
2. Profiling Go Programs: https://blog.golang.org/pprof
3. Benchmark Examples: examples/benchmarks/