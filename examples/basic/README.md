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
   # Basic GAuth CLI Example

   This example demonstrates the core GAuth flow using a simple CLI program. It shows how to:
   - Initialize a GAuth instance with typed configuration
   - Request an authorization grant
   - (Extendable) Issue and validate tokens, handle errors, and audit events

   ## Running the Example

   1. **Run the CLI Demo**
      ```bash
      go run main.go
       AuthServerURL: "http://localhost:8080",

   2. **Observe Output**
      - The program will print the steps of the authorization flow and the grant details.

   ## Extending the Example

   - Add token issuance and validation using the GAuth API.
   - Integrate audit logging and error handling.
   - See comments in `main.go` for extension points.

   ---
   For more, see the main project README and package documentation.
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