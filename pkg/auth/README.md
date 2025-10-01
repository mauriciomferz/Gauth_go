# Authentication Package

The `auth` package provides secure, flexible authentication for Go applications.

## Overview

This package offers:
- Token-based authentication (JWT, PASETO)
- Multiple authentication methods
- Secure token storage
- Session management
- Token revocation
- Distributed support

## Getting Started

### Basic Usage

```go
import "github.com/Gimel-Foundation/gauth/pkg/auth"

// Create authenticator
auth := auth.New(auth.Config{
    TokenType: auth.JWT,
    Secret:   []byte("your-secret"),
    TTL:      24 * time.Hour,
})

// Authenticate user
token, err := auth.Authenticate(ctx, "username", "password")
if err != nil {
    // Handle error
}

// Validate token
claims, err := auth.ValidateToken(ctx, token)
if err != nil {
    // Handle error
}
```

### Available Token Types

1. JWT (JSON Web Tokens)
```go
auth := auth.New(auth.Config{
    TokenType: auth.JWT,
    Secret:   jwtKey,
    TTL:      time.Hour,
})
```

2. PASETO (Platform-Agnostic Security Tokens)
```go
auth := auth.New(auth.Config{
    TokenType: auth.PASETO,
    Secret:   pasetoKey,
    TTL:      time.Hour,
})
```

### Session Management

```go
// Create session manager
sessions := auth.NewSessionManager(auth.SessionConfig{
    Store:     sessionStore,
    MaxActive: 5, // Max concurrent sessions
})

// Create session
session, err := sessions.Create(ctx, userID, deviceInfo)
if err != nil {
    // Handle error
}

// Validate session
valid, err := sessions.Validate(ctx, sessionID)
if err != nil {
    // Handle error
}

// Revoke session
err = sessions.Revoke(ctx, sessionID)
if err != nil {
    // Handle error
}
```

### Token Storage

```go
// Create Redis token store
store := auth.NewRedisStore(auth.RedisConfig{
    Addrs: []string{"localhost:6379"},
})

// Create authenticator with store
auth := auth.New(auth.Config{
    TokenType: auth.JWT,
    Store:    store,
})
```

### Multi-Factor Authentication

```go
// Configure MFA
mfa := auth.NewMFAProvider(auth.MFAConfig{
    Type:     auth.TOTP,
    Issuer:   "YourApp",
    Digits:   6,
})

// Generate secret for user
secret, err := mfa.GenerateSecret(userID)
if err != nil {
    // Handle error
}

// Validate MFA code
valid, err := mfa.ValidateCode(userID, code)
if err != nil {
    // Handle error
}
```

## Security Best Practices

1. Token Management
```go
// Use secure token settings
auth := auth.New(auth.Config{
    TokenType:    auth.PASETO,
    Secret:      secureKey,
    TTL:         time.Hour,
    MaxTokens:   1000,
    EnableAudit: true,
})
```

2. Session Security
```go
// Configure secure sessions
sessions := auth.NewSessionManager(auth.SessionConfig{
    Store:        store,
    MaxActive:    5,
    IdleTimeout:  30 * time.Minute,
    MaxLifetime:  24 * time.Hour,
    RequiresMFA:  true,
})
```

3. Token Revocation
```go
// Revoke all user tokens
err := auth.RevokeAllTokens(ctx, userID)
if err != nil {
    // Handle error
}
```

4. Rate Limiting
```go
// Add rate limiting
limiter := rate.NewTokenBucket(rate.Config{
    Limit:  10,
    Window: time.Minute,
})

auth := auth.New(auth.Config{
    RateLimiter: limiter,
})
```

## Advanced Features

### Custom Authentication Methods

```go
// Implement custom authenticator
type CustomAuth struct {
    // ...
}

func (c *CustomAuth) Authenticate(ctx context.Context, creds interface{}) (bool, error) {
    // Custom authentication logic
    return true, nil
}

// Use custom authenticator
auth := auth.New(auth.Config{
    Authenticator: &CustomAuth{},
})
```

### Event Handling

```go
// Configure event handlers
auth := auth.New(auth.Config{
    Events: auth.EventConfig{
        OnAuthentication: func(ctx context.Context, e *auth.Event) {
            // Handle authentication event
        },
        OnTokenIssued: func(ctx context.Context, e *auth.Event) {
            // Handle token issued event
        },
        OnTokenRevoked: func(ctx context.Context, e *auth.Event) {
            // Handle token revoked event
        },
    },
})
```

### Distributed Authentication

```go
// Configure distributed auth
auth := auth.New(auth.Config{
    Store: auth.NewRedisStore(redisAddrs),
    Distributed: auth.DistributedConfig{
        Enabled:     true,
        SyncPeriod: time.Minute,
        Consistency: auth.StrongConsistency,
    },
})
```

## Error Handling

```go
// Handle specific auth errors
token, err := auth.Authenticate(ctx, username, password)
switch {
case errors.Is(err, auth.ErrInvalidCredentials):
    // Handle invalid credentials
case errors.Is(err, auth.ErrAccountLocked):
    // Handle locked account
case errors.Is(err, auth.ErrMFARequired):
    // Handle MFA requirement
default:
    // Handle other errors
}
```

## Monitoring

```go
// Configure monitoring
auth := auth.New(auth.Config{
    Monitoring: auth.MonitoringConfig{
        Enabled:      true,
        MetricsAddr: ":9090",
        TracingEnabled: true,
    },
})

// Get metrics
metrics := auth.GetMetrics()
log.Printf("Auth requests: %d", metrics.Requests)
log.Printf("Success rate: %.2f%%", metrics.SuccessRate)
```

## Testing

The package includes comprehensive test helpers:

```go
// Use test utilities
import "github.com/Gimel-Foundation/gauth/pkg/auth/testing"

// Create test authenticator
auth := testing.NewTestAuth()

// Mock authentication
testing.MockAuthentication(auth, "user123", true)

// Verify auth calls
testing.VerifyAuthCalls(t, auth, 1)
```

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for details on contributing to this package.

## License

This package is part of the GAuth project and is licensed under the same terms.
See [LICENSE](../../LICENSE) for details.