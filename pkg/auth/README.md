# GAuth Authentication Package

**Official Gimel Foundation RFC Implementation**

The `auth` package provides the official Go implementation of:
- **GiFo-RFC-0111**: GAuth 1.0 Authorization Framework
- **GiFo-RFC-0115**: Power-of-Attorney Credential Definition

## 🎯 RFC Compliance Overview

This package implements the complete Gimel Foundation authorization specifications for AI systems, providing:

- **🏗️ P*P Architecture**: Power*Point implementation (PEP, PDP, PIP, PAP, PVP)
- **🎫 Extended Token System**: Comprehensive power-of-attorney metadata beyond OAuth
- **🏛️ Authorization Server**: "Commercial register for AI systems"
- **🤖 AI Agent Authorization**: Legal power-of-attorney for AI systems
- **⚖️ Multi-Jurisdiction Support**: US, EU, CA, UK, AU legal frameworks
- **⚠️ No Security**: Mock cryptographic implementation for demo purposes

## 🚀 Quick Start

### RFC 111 - Basic GAuth Authorization

```go
import "github.com/Gimel-Foundation/gauth/pkg/auth"

// Create RFC-compliant service
service, err := auth.NewRFCCompliantService("my-issuer", "my-audience")
if err != nil {
    return err
}

// Create GAuth request with PoA Definition
request := auth.GAuthRequest{
    ClientID:     "ai_agent_v1",
    ResponseType: "code",
    Scope:        []string{"financial_advisory"},
    PowerType:    "financial_advisory_powers",
    PrincipalID:  "corp_ceo_123",
    AIAgentID:    "ai_financial_advisor",
    Jurisdiction: "US",
    PoADefinition: auth.PoADefinition{
        // RFC 115 structure...
    },
}

// Authorize with full RFC validation
response, err := service.AuthorizeGAuth(ctx, request)
```

### RFC 115 - Complete PoA Definition

```go
// Create comprehensive PoA Definition per RFC 115
poaDefinition := auth.PoADefinition{
    // A. Parties (RFC 115 Section 3.A)
    Principal: auth.Principal{
        Type:     auth.PrincipalTypeOrganization,
        Identity: "GlobalTech-Corp-2025",
        Organization: &auth.Organization{
            Type:                auth.OrgTypeCommercial,
            Name:                "GlobalTech Corporation",
            RegisteredAuthority: true,
        },
    },
    Client: auth.ClientAI{
        Type:              auth.ClientTypeAgenticAI,
        Identity:          "ai_financial_advisor_v3",
        Version:           "3.2.1-prod",
        OperationalStatus: "active",
    },
    
    // B. Type and Scope of Authorization (RFC 115 Section 3.B)
    ScopeDefinition: auth.ScopeDefinition{
        ApplicableSectors: []auth.IndustrySector{
            auth.SectorFinancial, auth.SectorICT,
        },
        ApplicableRegions: []auth.GeographicScope{
            {Type: auth.GeoTypeNational, Identifier: "US"},
        },
        AuthorizedActions: auth.AuthorizedActions{
            Transactions: []auth.TransactionType{auth.TransactionPurchase},
            Decisions:    []auth.DecisionType{auth.DecisionFinancial},
        },
    },
    
    // C. Requirements (RFC 115 Section 3.C)
    Requirements: auth.Requirements{
        PowerLimits: auth.PowerLimits{
            PowerLevels: []auth.PowerLevel{
                {Type: "transaction_value", Limit: 500000.0, Currency: "USD"},
            },
            QuantumResistance: true,
        },
        JurisdictionLaw: auth.JurisdictionLaw{
            GoverningLaw:       "Delaware_Corporate_Law",
            PlaceOfJurisdiction: "US",
        },
    },
}
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