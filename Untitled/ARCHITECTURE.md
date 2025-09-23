# GAuth Architecture Guide

## Overview

GAuth is designed with a modular, layered architecture that prioritizes:
- Type safety
- Clear separation of concerns
- Extensibility
- Security best practices


## Architecture Layers

**Note:**
- All public APIs are type-safe (no public map[string]interface{}).
- Rate limiting is enforced per user (OwnerID) and per client, using the OwnerID field of the token as the subject for rate limiting.

```
┌─────────────────────────────────────────┐
│            Public API (pkg/)            │
├─────────────┬──────────────┬────────-───┤
│  Token API  │   Auth API   │ Resource   │
│             │              │   API      │
└─────────────┴──────────────┴───────────-┘
          │            │          │
┌─────────────────────────────────────────┐
│        Core Services (internal/)        │
├─────────┬─────────┬─────────┬──────────-┤
│  Token  │  Auth   │  Rate   │  Event    │
│ Service │ Service │ Limiter │ System    │
└─────────┴─────────┴─────────┴─────────-─┘
          │            │          │
┌─────────────────────────────────────────┐
│          Storage & Integration          │
├─────────┬─────────┬─────────┬──────────-┤
│  Token  │  User   │ Metrics │  Audit    │
│  Store  │  Store  │  Store  │   Log     │
└─────────┴─────────┴─────────┴──────────-┘
```

## Key Components

### 1. Public API Layer

The public API provides clean, type-safe interfaces for:

```go
// Token Management
type TokenManager interface {
    Generate(ctx context.Context, claims Claims) (*Token, error)
    Validate(ctx context.Context, token string) (*Claims, error)
    Revoke(ctx context.Context, token string) error
}

// Authentication
type Authenticator interface {
    Authenticate(ctx context.Context, creds Credentials) (*Token, error)
    Authorize(ctx context.Context, token string, resource string) error
}

// Resource Management
type ResourceManager interface {
    Register(ctx context.Context, resource Resource) error
    Grant(ctx context.Context, resource, subject string) error
}
```

### 2. Core Services Layer

Internal implementation of core functionality:

```go
// Token Service
type tokenService struct {
    store  TokenStore
    crypto CryptoProvider
    events EventEmitter
}

// Auth Service
type authService struct {
    tokens    TokenManager
    users     UserStore
    rateLimit RateLimiter
}

// Rate Limiter
type rateLimiter struct {
    algorithm Algorithm
    window    time.Duration
    limit     int64
}
```

### 3. Storage Layer

Pluggable storage backends:

```go
// Token Store
type TokenStore interface {
    Save(ctx context.Context, token *Token) error
    Get(ctx context.Context, id string) (*Token, error)
    Delete(ctx context.Context, id string) error
}

// User Store
type UserStore interface {
    FindUser(ctx context.Context, username string) (*User, error)
    SaveUser(ctx context.Context, user *User) error
}

// Metrics Store
type MetricsStore interface {
    RecordMetric(ctx context.Context, name string, value float64)
    GetMetrics(ctx context.Context) []Metric
}
```

## Data Flow

1. **Token Generation Flow**
```mermaid
sequenceDiagram
    Client->>+API: Request Token
    API->>+AuthService: Authenticate
    AuthService->>+UserStore: Validate User
    UserStore-->>-AuthService: User Valid
    AuthService->>+TokenService: Generate Token
    TokenService->>+TokenStore: Save Token
    TokenStore-->>-TokenService: Token Saved
    TokenService-->>-AuthService: Token
    AuthService-->>-API: Token
    API-->>-Client: Token Response
```

2. **Resource Access Flow**
```mermaid
sequenceDiagram
    Client->>+API: Access Resource
    API->>+AuthService: Validate Token
    AuthService->>+TokenStore: Get Token
    TokenStore-->>-AuthService: Token Info
    AuthService->>+ResourceManager: Check Access
    ResourceManager-->>-AuthService: Access Granted
    AuthService-->>-API: Authorized
    API-->>-Client: Resource Data
```

## Type Safety

GAuth uses strong typing throughout:

1. **Token Types**
```go
type TokenType string

const (
    AccessToken  TokenType = "access_token"
    RefreshToken TokenType = "refresh_token"
    IDToken      TokenType = "id_token"
)
```

2. **Claims**
```go
type Claims struct {
    Subject   string
    Issuer    string
    Audience  []string
    ExpiresAt time.Time
    Scopes    []string
}
```

3. **Metadata**
```go
type Metadata struct {
    Device     *DeviceInfo
    AppID      string
    AppVersion string
    Labels     map[string]string
}
```

## Extension Points

### 1. Storage Backends

Implement custom storage:
```go
type CustomTokenStore struct {
    db *sql.DB
}

func (s *CustomTokenStore) Save(ctx context.Context, token *Token) error {
    // Custom implementation
}
```

### 2. Authentication Methods

Add new auth methods:
```go
type CustomAuthenticator struct {
    client *CustomAuthClient
}

func (a *CustomAuthenticator) Authenticate(ctx context.Context) (*Token, error) {
    // Custom implementation
}
```

### 3. Rate Limiting

Custom rate limit algorithms:
```go
type CustomRateLimiter struct {
    cache *redis.Client
}

func (l *CustomRateLimiter) Allow(ctx context.Context) error {
    // Custom implementation
}
```

## Performance Considerations

1. **Caching**
```go
type CachedTokenStore struct {
    cache  *redis.Client
    store  TokenStore
    ttl    time.Duration
}
```

2. **Bulk Operations**
```go
type BulkTokenStore interface {
    SaveMany(ctx context.Context, tokens []*Token) error
    GetMany(ctx context.Context, ids []string) ([]*Token, error)
}
```

3. **Efficient Validation**
```go
type FastValidator struct {
    publicKeys map[string]*rsa.PublicKey
    cache      *sync.Map
}
```

## Security

1. **Token Security**
- Secure token generation
- Proper key management
- Regular key rotation

2. **Access Control**
- Fine-grained permissions
- Scope validation
- Resource isolation

3. **Audit Logging**
- Detailed event logging
- Security event tracking
- Compliance reporting

## Monitoring

1. **Metrics**
- Token operations
- Authentication attempts
- Rate limit hits

2. **Health Checks**
- Storage connectivity
- Service health
- Resource usage

3. **Alerts**
- Security events
- Performance issues
- Error thresholds

## Best Practices

1. **Token Management**
- Set appropriate TTLs
- Implement token rotation
- Use refresh tokens

2. **Error Handling**
- Clear error types
- Proper logging
- User-friendly messages

3. **Resource Management**
- Proper cleanup
- Resource pooling
- Connection management