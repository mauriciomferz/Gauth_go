# GAuth Development Guide

**Gimel Foundation RFC Implementation - Developer Documentation**

**Copyright (c) 2025 Gimel Foundation gGmbH i.G.**  
Licensed under Apache 2.0

**Gimel Foundation gGmbH i.G.**, www.GimelFoundation.com  
Operated by Gimel Technologies GmbH  
MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert  
Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com

Development guide for the GiFo-RFC-0111 and GiFo-RFC-0115 implementation, featuring complete RFC-0115 PoA-Definition compliance.

## ⚠️ **CRITICAL: NO SECURITY WARNING**

**This is a demonstration prototype with:**
- No real authentication or authorization
- All responses are mocked/hardcoded
- This is a development/demonstration implementation only
- For educational and demo purposes only

## 🚀 **Quick Start**

### **1. RFC-Compliant Authorization**

```go
import "github.com/Gimel-Foundation/gauth/pkg/auth"

// Create RFC service prototype
service, err := auth.NewRFCCompliantService("YourCompany", "ai-authorization")
if err != nil {
    log.Fatal(err)
}

// Create comprehensive PoA Definition (RFC 115)
poa := auth.PoADefinition{
    Principal: auth.Principal{
        Type:     auth.PrincipalTypeOrganization,
        Identity: "company-2025",
        Organization: &auth.Organization{
            Type:                auth.OrgTypeCommercial,
            Name:                "Your Company",
            RegisteredAuthority: true,
        },
    },
    // ... complete RFC 115 structure
}

// Authorize with full RFC validation
response, err := service.AuthorizeGAuth(ctx, auth.GAuthRequest{
    ClientID:      "ai-client-id",
    PoADefinition: poa,
    Jurisdiction:  "US",
})
```

### **2. Development JWT Foundation**

```go
import "github.com/Gimel-Foundation/gauth/pkg/auth"

// JWT service with RSA-256 signatures
jwtService, err := auth.NewProperJWTService("issuer", "audience")

// Features:
// - ⚠️ NO SECURITY (all crypto functions are stubbed)
// - Quantum-resistant cryptography support
// - Professional key management
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