# Testing Guide

## Overview

This guide explains how to effectively test GAuth components and extensions. It covers:
- Unit testing patterns
- Integration testing
- Performance benchmarking
- Mocking strategies

## Test Structure

### 1. Unit Tests

Each package has comprehensive unit tests:

```go
func TestAuthentication(t *testing.T) {
    // Setup
    auth := gauth.New(gauth.Config{
        AuthServerURL: "https://auth.example.com",
        ClientID:     "test-client",
    })

    // Test cases
    tests := []struct {
        name    string
        creds   gauth.Credentials
        wantErr error
    }{
        {
            name: "valid credentials",
            creds: gauth.Credentials{
                Username: "valid",
                Password: "valid",
            },
            wantErr: nil,
        },
        // Add more test cases...
    }

    // Run tests
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := auth.Authenticate(context.Background(), tt.creds)
            if err != tt.wantErr {
                t.Errorf("got error %v, want %v", err, tt.wantErr)
            }
        })
    }
}
```

### 2. Examples as Tests

Document usage patterns with testable examples:

```go
func ExampleRateLimiter() {
    limiter := rate.NewLimiter(&rate.Config{
        RequestsPerSecond: 100,
        WindowSize:       60,
    })

    err := limiter.Allow(context.Background(), "client-123")
    fmt.Println(err == nil)
    // Output: true
}
```

### 3. Benchmarks

Measure performance characteristics:

```go
func BenchmarkRateLimit(b *testing.B) {
    limiter := rate.NewLimiter(&rate.Config{
        RequestsPerSecond: 1000,
        WindowSize:       1,
    })

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _ = limiter.Allow(context.Background(), "client")
        }
    })
}
```

## Testing Tools

### 1. Mock Token Store

```go
type mockStore struct {
    tokens map[string]*token.Token
}

func (m *mockStore) Get(ctx context.Context, key string) (*token.Token, error) {
    if t, ok := m.tokens[key]; ok {
        return t, nil
    }
    return nil, token.ErrTokenNotFound
}

// Implement other Store methods...
```

### 2. Test Utilities

```go
// testConfig returns a config for testing
func testConfig() gauth.Config {
    return gauth.Config{
        AuthServerURL: "https://test.example.com",
        ClientID:     "test-client",
        RateLimit: gauth.RateLimitConfig{
            RequestsPerSecond: 100,
            WindowSize:       1,
        },
    }
}

// assertNoError fails the test if err is not nil
func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

## Test Categories

### 1. Authentication Tests

```go
func TestAuth_ValidCredentials(t *testing.T)
func TestAuth_InvalidCredentials(t *testing.T)
func TestAuth_ExpiredToken(t *testing.T)
func TestAuth_RevokedToken(t *testing.T)
```

### 2. Rate Limiting Tests

```go
func TestRateLimit_BasicFlow(t *testing.T)
func TestRateLimit_WindowSliding(t *testing.T)
func TestRateLimit_Concurrent(t *testing.T)
func TestRateLimit_BurstHandling(t *testing.T)
```

### 3. Token Management Tests

```go
func TestToken_Creation(t *testing.T)
func TestToken_Validation(t *testing.T)
func TestToken_Expiration(t *testing.T)
func TestToken_Revocation(t *testing.T)
```

## Common Test Patterns

### 1. Concurrent Testing

```go
func TestConcurrent(t *testing.T) {
    const workers = 10
    var wg sync.WaitGroup
    auth := gauth.New(testConfig())

    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Perform concurrent operations
        }()
    }
    wg.Wait()
}
```

### 2. Cleanup

```go
func TestWithCleanup(t *testing.T) {
    auth := gauth.New(testConfig())
    t.Cleanup(func() {
        // Cleanup resources
    })

    // Test logic
}
```

### 3. Timeouts

```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    // Test with timeout context
}
```

## Integration Testing

### 1. Setup

```go
func setupIntegrationTest(t *testing.T) *gauth.Auth {
    t.Helper()
    // Setup test environment
    return gauth.New(integrationConfig())
}
```

### 2. End-to-End Flow

```go
func TestIntegration_FullFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    auth := setupIntegrationTest(t)
    // Test complete authentication flow
}
```

## Performance Testing

### 1. Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkRateLimit ./...

# Run with memory profiling
go test -bench=. -memprofile=mem.prof ./...
```

### 2. Load Testing

```go
func BenchmarkLoad(b *testing.B) {
    auth := gauth.New(loadTestConfig())
    b.SetParallelism(100)

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            // Simulate load
        }
    })
}
```

## Best Practices

1. **Test Organization**:
   - Group related tests
   - Use descriptive names
   - Include test documentation

2. **Test Coverage**:
   - Aim for high coverage
   - Test edge cases
   - Include error conditions

3. **Performance Testing**:
   - Set realistic benchmarks
   - Test under load
   - Profile memory usage

4. **Mocking**:
   - Mock external dependencies
   - Use interfaces
   - Keep mocks simple

5. **Documentation**:
   - Document test patterns
   - Include example tests
   - Explain test utilities