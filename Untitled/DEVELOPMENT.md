# GAuth Development Guide

## Quick Start

1. **Basic Authentication**

```go
import "github.com/Gimel-Foundation/gauth/pkg/gauth"

auth := gauth.New(gauth.Config{
    AuthServerURL: "https://auth.example.com",
    ClientID:     "client-123",
    ClientSecret: "secret-456",
})

// Authenticate a request
token, err := auth.Authenticate(ctx, gauth.Credentials{
    Username: "user",
    Password: "pass",
})
```

2. **Rate Limiting**

```go
import "github.com/Gimel-Foundation/gauth/internal/rate"

limiter := rate.NewLimiter(&rate.Config{
    RequestsPerSecond: 100,
    WindowSize:       60,
})

// Check if request is allowed
if err := limiter.Allow(ctx, "client-123"); err != nil {
    // Handle rate limit exceeded
}
```

## Common Use Cases

### 1. Token Management

```go
// Create a token store with TTL cleanup
store := token.NewMemoryStore(24 * time.Hour)

// Store a token
token := &token.Token{
    Value:     "jwt-token",
    Type:      token.AccessToken,
    ExpiresAt: time.Now().Add(time.Hour),
    Scopes:    []string{"read", "write"},
}
store.Save(ctx, "user-123", token)
```

### 2. Event Handling

```go
// Use typed events instead of strings
event := &events.Event{
    Type:      events.AuthSuccess,
    Timestamp: time.Now(),
    Details: &events.AuthEventDetails{
        ClientID:  "client-123",
        GrantType: "password",
        Scopes:    []string{"read"},
    },
}
```

### 3. Time-based Restrictions

```go
// Use strongly typed time ranges
timeRange := &restriction.TimeRange{
    Start: time.Now(),
    End:   time.Now().Add(24 * time.Hour),
}

allowed, msg := timeRange.IsAllowed(time.Now())
if !allowed {
    // Handle restriction
}
```

## Package Structure

### Public API (`pkg/gauth/`)
- Core authentication types and functions
- Stable, versioned interfaces
- Configuration types

### Internal Implementation (`internal/`)
- `rate/`: Rate limiting algorithms
- `token/`: Token storage and validation
- `events/`: Event types and handling
- `restriction/`: Access restrictions
- `resources/`: User-facing messages

### Examples (`examples/`)
- Basic authentication flows
- Rate limiting patterns
- Token management
- Event handling

## Extension Points

1. **Custom Token Storage**
```go
// Implement the Store interface
type Store interface {
    Save(ctx context.Context, key string, token *Token) error
    Get(ctx context.Context, key string) (*Token, error)
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, filter Filter) ([]*Token, error)
}
```

2. **Custom Rate Limiting**
```go
// Implement the Algorithm interface
type Algorithm interface {
    Allow(ctx context.Context, id string) error
    GetRemainingQuota(id string) int
    Reset(id string)
}
```

3. **Event Handling**
```go
// Add new event types and details
type CustomEventDetails struct {
    // Your custom fields
}

func (CustomEventDetails) isEventDetails() {}
```

## Best Practices

1. **Type Safety**
   - Use proper event types instead of strings
   - Avoid map[string]interface{}
   - Define clear interfaces

2. **Error Handling**
```go
var (
    ErrTokenNotFound = errors.New("token not found")
    ErrTokenExpired  = errors.New("token expired")
)
```

3. **Resource Management**
```go
// Always close resources
limiter := rate.NewLimiter(config)
defer limiter.Close()
```

4. **Thread Safety**
   - Use proper synchronization
   - Hide implementation details
   - Provide clean interfaces

## Testing

```go
func TestAuthentication(t *testing.T) {
    auth := gauth.New(gauth.Config{...})
    token, err := auth.Authenticate(ctx, credentials)
    if err != nil {
        t.Errorf("Authentication failed: %v", err)
    }
    // Add more assertions
}
```

## Common Patterns

1. **Resilient Authentication**
```go
auth := gauth.New(gauth.Config{
    RateLimit: gauth.RateLimitConfig{
        RequestsPerSecond: 100,
        WindowSize:       60,
    },
})
```

2. **Token Validation**
```go
if err := auth.ValidateToken(ctx, token); err != nil {
    switch err {
    case token.ErrTokenExpired:
        // Handle expired token
    case token.ErrInvalidToken:
        // Handle invalid token
    default:
        // Handle other errors
    }
}
```

3. **Event Processing**
```go
switch event.Type {
case events.AuthSuccess:
    details := event.Details.(*events.AuthEventDetails)
    // Process authentication success
case events.RateLimitExceeded:
    details := event.Details.(*events.RateLimitEventDetails)
    // Handle rate limit exceeded
}
```

## Troubleshooting

1. **Rate Limiting Issues**
   - Check window size configuration
   - Verify client ID consistency
   - Monitor remaining quota

2. **Token Validation Failures**
   - Verify token expiration
   - Check scope requirements
   - Validate token format

3. **Performance Optimization**
   - Use appropriate window sizes
   - Implement efficient storage
   - Monitor cleanup routines