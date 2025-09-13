# Basic Authentication Example

This example demonstrates basic authentication using GAuth, including:
- Token-based authentication
- Simple rate limiting
- Basic error handling

## Features

1. **Authentication Flow**
   - Username/password validation
   - JWT token generation
   - Token validation

2. **Rate Limiting**
   - Basic sliding window
   - Per-client limits
   - Burst allowance

3. **Error Handling**
   - Authentication errors
   - Rate limit errors
   - Token validation errors

## Running the Example

1. **Start the Server**
   ```bash
   go run main.go
   ```

2. **Get a Token**
   ```bash
   curl -X POST http://localhost:8080/auth \
     -H "Content-Type: application/json" \
     -d '{"username":"test","password":"test123"}'
   ```

3. **Use the Token**
   ```bash
   curl -H "Authorization: Bearer <token>" \
     http://localhost:8080/protected
   ```

## Code Structure

```
basic/
  ├── main.go              # Server setup
  ├── handlers/
  │   └── auth.go         # Auth handlers
  └── middleware/
      └── auth.go         # Auth middleware
```

## Implementation Details

1. **Authentication Setup**
   ```go
   auth := gauth.New(gauth.Config{
       AuthServerURL: "http://localhost:8080",
       ClientID:     "example-client",
       ClientSecret: "example-secret",
   })
   ```

2. **Rate Limiting**
   ```go
   limiter := rate.NewSlidingWindow(rate.Config{
       RequestsPerSecond: 10,
       BurstSize:        5,
   })
   ```

3. **Token Validation**
   ```go
   func validateToken(token string) (*auth.Claims, error) {
       return auth.ValidateToken(token)
   }
   ```

## Testing

1. **Unit Tests**
   ```bash
   go test ./...
   ```

2. **Manual Testing**
   ```bash
   # Test rate limiting
   for i in {1..20}; do
     curl -H "Authorization: Bearer <token>" \
       http://localhost:8080/protected
   done
   ```

## Error Cases

1. **Invalid Credentials**
   ```bash
   curl -X POST http://localhost:8080/auth \
     -H "Content-Type: application/json" \
     -d '{"username":"wrong","password":"wrong"}'
   ```

2. **Invalid Token**
   ```bash
   curl -H "Authorization: Bearer invalid-token" \
     http://localhost:8080/protected
   ```

3. **Rate Limit Exceeded**
   ```bash
   # Make requests faster than the rate limit
   while true; do
     curl -H "Authorization: Bearer <token>" \
       http://localhost:8080/protected
   done
   ```

## Configuration

The example can be configured through environment variables:

```bash
export AUTH_SERVER_URL=http://localhost:8080
export CLIENT_ID=example-client
export CLIENT_SECRET=example-secret
export RATE_LIMIT=10
export BURST_SIZE=5
```

## Metrics

Basic metrics are available at `/metrics`:
- Request counts
- Error rates
- Token operations
- Rate limit stats

## Next Steps

1. Add persistent storage
2. Implement refresh tokens
3. Add user roles
4. Enhance rate limiting