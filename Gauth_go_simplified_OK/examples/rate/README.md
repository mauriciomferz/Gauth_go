# Rate Limiting Examples

This directory demonstrates various rate limiting patterns.

## Token Bucket Algorithm

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/Gimel-Foundation/gauth/pkg/rate"
)

func main() {
    // Create token bucket limiter
    limiter := rate.NewTokenBucket(rate.Config{
        Tokens:    100,    // Max tokens
        Interval:  time.Minute,  // Refill interval
        BurstSize: 10,     // Max burst
    })

    // Check rate limit
    ctx := context.Background()
    for i := 0; i < 120; i++ {
        err := limiter.Allow(ctx, "client1")
        if err != nil {
            if err == rate.ErrLimitExceeded {
                log.Printf("Request %d: Rate limit exceeded", i)
                time.Sleep(time.Second)
                continue
            }
            log.Fatal(err)
        }
        log.Printf("Request %d: Allowed", i)
    }
}
```

## Sliding Window

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/Gimel-Foundation/gauth/pkg/rate"
)

func main() {
    // Create sliding window limiter
    limiter := rate.NewSlidingWindow(rate.Config{
        Window:   time.Minute,
        MaxRequests: 100,
        Precision:   6,  // Number of segments per window
    })

    // Simulate requests
    ctx := context.Background()
    for i := 0; i < 150; i++ {
        quota, err := limiter.AllowN(ctx, "client1", 1)
        if err != nil {
            log.Printf("Request %d: %v", i, err)
            continue
        }
        log.Printf("Request %d: Allowed (Remaining: %d)", i, quota.Remaining)
        time.Sleep(100 * time.Millisecond)
    }
}
```

## Distributed Rate Limiting

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/Gimel-Foundation/gauth/pkg/rate"
)

func main() {
    // Create distributed limiter
    limiter := rate.NewDistributed(rate.DistributedConfig{
        Algorithm: rate.SlidingWindow,
        Store:     rate.NewRedisStore("localhost:6379"),
        Window:    time.Minute,
        Limit:     1000,
    })

    // Use limiter across services
    ctx := context.Background()
    for i := 0; i < 100; i++ {
        quota, err := limiter.Allow(ctx, "shared-resource")
        if err != nil {
            log.Printf("Request %d: %v", i, err)
            continue
        }
        log.Printf(
            "Request %d: Allowed (Remaining: %d, Reset: %v)",
            i,
            quota.Remaining,
            quota.ResetAt.Sub(time.Now()),
        )
    }
}
```

## Custom Rate Limiting

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/Gimel-Foundation/gauth/pkg/rate"
)

// Custom rate limiter implementation
type CustomLimiter struct {
    store  rate.Store
    config rate.Config
}

func (c *CustomLimiter) Allow(ctx context.Context, key string) error {
    // Custom rate limiting logic
    return nil
}

func main() {
    // Create custom limiter
    limiter := &CustomLimiter{
        store: rate.NewMemoryStore(),
        config: rate.Config{
            Window: time.Minute,
            Limit:  100,
        },
    }

    // Use limiter
    ctx := context.Background()
    if err := limiter.Allow(ctx, "test"); err != nil {
        log.Fatal(err)
    }
}
```

## Dynamic Rate Limiting

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/Gimel-Foundation/gauth/pkg/rate"
)

func main() {
    // Create dynamic limiter
    limiter := rate.NewDynamic(rate.DynamicConfig{
        BaseLimit:    100,
        MaxLimit:     1000,
        ScaleFactor:  2.0,
        CooldownTime: time.Minute,
    })

    // Monitor and adjust limits
    go func() {
        for {
            stats := limiter.GetStats()
            if stats.ErrorRate > 0.1 { // 10% errors
                limiter.DecreaseLimit()
            } else if stats.Usage > 0.8 { // 80% usage
                limiter.IncreaseLimit()
            }
            time.Sleep(10 * time.Second)
        }
    }()

    // Use limiter
    ctx := context.Background()
    for {
        err := limiter.Allow(ctx, "dynamic")
        if err != nil {
            log.Printf("Rate limit exceeded: %v", err)
            time.Sleep(time.Second)
            continue
        }
        // Process request
    }
}
```

## Best Practices

1. **Choose the Right Algorithm**
   - Token Bucket: For APIs with burst traffic
   - Sliding Window: For smooth traffic distribution
   - Fixed Window: For simple cases

2. **Configure Proper Limits**
```go
rate.Config{
    Window:      time.Minute,
    Limit:       60,     // 1 request per second
    BurstLimit:  10,     // Allow bursts
    WaitTimeout: 5 * time.Second,  // Max wait time
}
```

3. **Handle Rate Limit Errors**
```go
if err := limiter.Allow(ctx, key); err != nil {
    if err == rate.ErrLimitExceeded {
        // Add Retry-After header
        retryAfter := limiter.GetRetryAfter(key)
        w.Header().Set("Retry-After", retryAfter.String())
        http.Error(w, "Rate limit exceeded", 429)
        return
    }
    // Handle other errors
}
```

4. **Monitor Rate Limiting**
```go
metrics := limiter.GetMetrics()
log.Printf(
    "Requests: %d, Rejected: %d, Error Rate: %.2f%%",
    metrics.Requests,
    metrics.Rejected,
    metrics.ErrorRate * 100,
)
```

5. **Use Distributed Rate Limiting in Clusters**
```go
limiter := rate.NewDistributed(rate.DistributedConfig{
    Store:     rate.NewRedisStore(redisAddrs),
    Algorithm: rate.SlidingWindow,
    Limit:     1000,
    Window:    time.Minute,
})
```