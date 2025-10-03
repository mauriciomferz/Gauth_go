# Rate Limiting Package

The `rate` package provides flexible and efficient rate limiting implementations for Go applications.

## Features

- Multiple algorithms:
  - Token Bucket
  - Sliding Window
  - Fixed Window
  - Leaky Bucket
  - Dynamic Rate Limiting

- Storage backends:
  - In-memory
  - Redis
  - Distributed

- Key features:
  - Thread-safe
  - Context support
  - Configurable windows
  - Burst handling
  - Quota management
  - Cleanup automation

## Installation

```bash
go get github.com/Gimel-Foundation/gauth/pkg/rate
```

## Quick Start

```go
import (
    "context"
    "time"
    "github.com/Gimel-Foundation/gauth/pkg/rate"
)

func main() {
    // Create a rate limiter
    limiter := rate.NewTokenBucket(rate.Config{
        Limit:     100,           // 100 requests
        Window:    time.Minute,   // per minute
        BurstSize: 10,           // allow bursts of 10
    })
    defer limiter.Close()

    // Use the limiter
    ctx := context.Background()
    quota, err := limiter.Allow(ctx, "client-123")
    if err == rate.ErrLimitExceeded {
        // Handle rate limit exceeded
        return
    }

    // Check remaining quota
    fmt.Printf("Remaining requests: %d\n", quota.Remaining)
}
```

## Available Algorithms

### Token Bucket
Best for APIs that need to handle burst traffic:

```go
limiter := rate.NewTokenBucket(rate.Config{
    Limit:     1000,
    Window:    time.Hour,
    BurstSize: 50,
})
```

### Sliding Window
Best for smooth traffic distribution:

```go
limiter := rate.NewSlidingWindow(rate.Config{
    Limit:     60,
    Window:    time.Minute,
    Precision: 6,  // 6 segments per window
})
```

### Fixed Window
Best for simple cases:

```go
limiter := rate.NewFixedWindow(rate.Config{
    Limit:  100,
    Window: time.Hour,
})
```

### Leaky Bucket
Best for constant outflow rate:

```go
limiter := rate.NewLeakyBucket(rate.Config{
    Limit:     60,
    Window:    time.Minute,
    LeakRate:  1,  // 1 request per second
})
```

### Dynamic Rate Limiting
Best for adaptive limits:

```go
limiter := rate.NewDynamic(rate.DynamicConfig{
    BaseLimit:    100,
    MaxLimit:     1000,
    ScaleFactor:  2.0,
    CooldownTime: time.Minute,
})
```

## Distributed Rate Limiting

For distributed environments:

```go
store := rate.NewRedisStore(rate.RedisConfig{
    Addrs: []string{"localhost:6379"},
})

limiter := rate.NewDistributed(rate.Config{
    Algorithm: rate.SlidingWindow,
    Store:     store,
    Limit:     1000,
    Window:    time.Hour,
})
```

## Best Practices

1. Choose the right algorithm:
   - Token Bucket: For APIs with burst traffic
   - Sliding Window: For smooth distribution
   - Fixed Window: For simple cases
   - Leaky Bucket: For constant outflow
   - Dynamic: For adaptive limits

2. Configure proper limits:
   ```go
   rate.Config{
       Limit:       60,     // 1 request per second
       Window:      time.Minute,
       BurstSize:   10,     // Allow bursts
       WaitTimeout: 5 * time.Second,  // Max wait time
   }
   ```

3. Handle rate limit errors:
   ```go
   quota, err := limiter.Allow(ctx, key)
   if err == rate.ErrLimitExceeded {
       // Add Retry-After header
       w.Header().Set("Retry-After", 
           quota.RetryAfter.String())
       http.Error(w, "Rate limit exceeded", 429)
       return
   }
   ```

4. Monitor rate limiting:
   ```go
   metrics := limiter.GetMetrics()
   log.Printf(
       "Requests: %d, Rejected: %d, Error Rate: %.2f%%",
       metrics.Requests,
       metrics.Rejected,
       metrics.ErrorRate * 100,
   )
   ```

5. Use distributed rate limiting in clusters:
   ```go
   limiter := rate.NewDistributed(rate.DistributedConfig{
       Store:     rate.NewRedisStore(redisAddrs),
       Algorithm: rate.SlidingWindow,
       Limit:     1000,
       Window:    time.Hour,
   })
   ```

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for details on how to contribute to this package.

## License

This package is part of the GAuth project and is licensed under the same terms.
See [LICENSE](../../LICENSE) for details.