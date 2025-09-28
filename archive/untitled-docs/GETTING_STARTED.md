# Getting Started with GAuth

## Introduction

GAuth is a modern authentication library for Go applications. This guide will help you get started with GAuth and explore its features.

## Installation

```bash
go get github.com/Gimel-Foundation/gauth
```

## First Steps

1. **Create a Simple Authentication Server**

Try the basic example in `examples/basic/main.go`:
```bash
cd examples/basic
go run main.go
```

This starts a server with:
- Basic authentication
- Rate limiting
- Token management

2. **Test the Rate Limiter**

Run the rate limiting example:
```bash
cd examples/rate
go run main.go
```

Watch how different patterns affect the rate limits:
- Burst requests
- Steady traffic
- Multiple clients

3. **Explore Token Management**

Try the token management example:
```bash
cd examples/token
go run main.go
```

See how tokens are:
- Created and validated
- Stored and retrieved
- Automatically cleaned up

## Manual Testing

1. **Authentication Flow**
```bash
# Request a token
curl -X POST http://localhost:8080/auth \
  -d '{"username": "test", "password": "test123"}'

# Use the token
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/protected
```

2. **Rate Limiting**
```bash
# Make rapid requests to see rate limiting
for i in {1..10}; do
  curl -H "Authorization: Bearer <token>" \
    http://localhost:8080/protected
done
```

3. **Token Management**
```bash
# Create a token
curl -X POST http://localhost:8080/token/create

# Validate a token
curl -X POST http://localhost:8080/token/validate \
  -d '{"token": "<token>"}'

# Revoke a token
curl -X POST http://localhost:8080/token/revoke \
  -d '{"token": "<token>"}'
```

## Configuration Examples

1. **Basic Setup**
```go
auth := gauth.New(gauth.Config{
    AuthServerURL: "https://auth.example.com",
    ClientID:     "client-123",
    ClientSecret: "secret-456",
})
```

2. **With Rate Limiting**
```go
auth := gauth.New(gauth.Config{
    // ... basic config ...
    RateLimit: gauth.RateLimitConfig{
        RequestsPerSecond: 100,
        WindowSize:       60,
        BurstSize:       10,
    },
})
```

3. **With Custom Token Store**
```go
auth := gauth.New(gauth.Config{
    // ... basic config ...
    TokenStore: myCustomStore,
})
```

## Monitoring

1. **Check Rate Limit Status**
```bash
curl http://localhost:8080/metrics | grep rate_limit
```

2. **View Token Statistics**
```bash
curl http://localhost:8080/metrics | grep token
```

3. **Monitor Authentication Events**
```bash
curl http://localhost:8080/metrics | grep auth
```

## Troubleshooting

1. **Rate Limit Issues**
- Check the current rate limit status
- Verify client identification
- Review window size settings

2. **Token Problems**
- Verify token format
- Check expiration times
- Confirm scope configuration

3. **Authentication Failures**
- Review credentials
- Check server connectivity
- Verify client configuration

## Next Steps

1. Read the [Development Guide](DEVELOPMENT.md) for implementation details
2. Explore the [API Documentation](pkg/gauth/doc.go)
3. Try the [Advanced Examples](examples/advanced/)

## Community Resources

- GitHub Issues: Report bugs and request features
- Discussions: Ask questions and share ideas
- Wiki: Additional documentation and guides