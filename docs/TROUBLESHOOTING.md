---
title: Troubleshooting
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth Troubleshooting Guide

> Last Updated: 2025-10-17
> Status: Active

This guide helps you diagnose and fix common issues in GAuth.

## Common Issues

### 1. Token Validation Failures

#### Symptoms
- "token validation failed" errors
- Unexpected token rejections
- Token expiration issues

#### Diagnostic Steps
1. Check token format:
```go
import "github.com/Gimel-Foundation/gauth/pkg/token"

// Inspect token details
token, err := tokenManager.Inspect(ctx, tokenString)
if err != nil {
    log.Printf("Token inspection failed: %v", err)
    // Check specific error type
    switch {
    case errors.Is(err, token.ErrMalformed):
        // Handle malformed token
    case errors.Is(err, token.ErrExpired):
        // Handle expired token
    }
}
```

2. Verify token claims:
```go
// Check token metadata
if token.Metadata != nil {
    log.Printf("Token metadata: %+v", token.Metadata)
}

// Verify scopes
if !token.HasScope("required:scope") {
    log.Printf("Missing required scope")
}
```

3. Check token store:
```go
// Verify token in store
stored, err := tokenStore.Get(ctx, token.ID)
if err != nil {
    log.Printf("Token store error: %v", err)
}
```

#### Solutions
1. For malformed tokens:
   - Regenerate token
   - Check token creation parameters
   - Verify signing keys

2. For expired tokens:
   - Implement token refresh
   - Adjust token lifetime
   - Check clock synchronization

3. For scope issues:
   - Review scope configuration
   - Update token permissions
   - Check scope inheritance

### 2. Rate Limiting Issues

#### Symptoms
- Too many rate limit hits
- Inconsistent rate limiting
- Rate limit bypass

#### Diagnostic Steps
1. Check rate limit configuration:
```go
import "github.com/Gimel-Foundation/gauth/pkg/rate"

// Inspect rate limiter
config := limiter.GetConfig()
log.Printf("Rate limit config: %+v", config)

// Check current limits
quota, err := limiter.GetQuota(ctx, clientID)
if err != nil {
    log.Printf("Quota check failed: %v", err)
}
```

2. Monitor rate limit metrics:
```go
// Get rate limit metrics
metrics, err := limiter.GetMetrics(ctx)
if err != nil {
    log.Printf("Metrics error: %v", err)
}

// Check specific client
clientMetrics := metrics.ForClient(clientID)
log.Printf("Client rate metrics: %+v", clientMetrics)
```

3. Test rate limit behavior:
```go
// Test rate limiting
for i := 0; i < 10; i++ {
    err := limiter.Allow(ctx, clientID)
    log.Printf("Request %d: %v", i, err)
}
```

#### Solutions
1. For too many hits:
   - Adjust rate limits
   - Implement caching
   - Add request queuing

2. For inconsistent limiting:
   - Check storage backend
   - Verify clock sync
   - Monitor system resources

3. For bypass issues:
   - Review client identification
   - Check proxy configuration
   - Update rate limit rules

### 3. Performance Problems

#### Symptoms
- Slow token validation
- High latency
- Resource exhaustion

#### Diagnostic Steps
1. Check token store performance:
```go
import "github.com/Gimel-Foundation/gauth/pkg/monitoring"

// Monitor store operations
metrics := monitoring.NewStoreMetrics()
tokenStore.WithMetrics(metrics)

// Check operation latency
log.Printf("Store latency: %v", metrics.GetLatency())
```

2. Profile memory usage:
```go
// Check token store size
stats := tokenStore.GetStats()
log.Printf("Store stats: %+v", stats)

// Monitor cache hit rate
if cache, ok := tokenStore.(CacheStats); ok {
    log.Printf("Cache hit rate: %v", cache.HitRate())
}
```

3. Analyze resource usage:
```go
// Get resource metrics
resources := monitoring.GetResourceMetrics()
log.Printf("Resource usage: %+v", resources)
```

#### Solutions
1. For slow validation:
   - Enable caching
   - Optimize validation
   - Scale horizontally

2. For high latency:
   - Add connection pooling
   - Implement request batching
   - Use async operations

3. For resource issues:
   - Adjust resource limits
   - Implement cleanup
   - Monitor usage

### 4. Integration Problems

#### Symptoms
- Authentication failures
- Integration timeouts
- Configuration issues

#### Diagnostic Steps
1. Test connectivity:
```go
// Check auth server
status := auth.CheckConnection(ctx)
log.Printf("Auth server status: %v", status)

// Verify integration
if err := auth.VerifyIntegration(ctx); err != nil {
    log.Printf("Integration error: %v", err)
}
```

2. Validate configuration:
```go
// Check configuration
config := auth.GetConfig()
if err := config.Validate(); err != nil {
    log.Printf("Config validation failed: %v", err)
}
```

3. Monitor integration:
```go
// Watch integration events
events := auth.WatchEvents(ctx)
for event := range events {
    log.Printf("Integration event: %+v", event)
}
```

#### Solutions
1. For auth failures:
   - Check credentials
   - Verify endpoints
   - Update configuration

2. For timeouts:
   - Adjust timeouts
   - Add retries
   - Check network

3. For config issues:
   - Validate settings
   - Update integration
   - Check compatibility

## Advanced Diagnostics

### 1. Token Debugging

```go
// Enable debug mode
tokenManager.EnableDebug()

// Get detailed token info
info, err := tokenManager.DebugToken(ctx, token)
if err != nil {
    log.Printf("Debug failed: %v", err)
}
log.Printf("Token debug info: %+v", info)
```

### 2. Performance Profiling

```go
// Start profiling
profile := monitoring.StartProfile()
defer profile.Stop()

// Get profile data
data := profile.GetData()
log.Printf("Profile data: %+v", data)
```

### 3. System Health Check

```go
// Run health check
health := monitoring.CheckHealth(ctx)
log.Printf("System health: %+v", health)

// Get detailed diagnostics
diag := monitoring.GetDiagnostics(ctx)
log.Printf("Diagnostics: %+v", diag)
```

## Best Practices

1. **Logging**
   - Enable appropriate log levels
   - Use structured logging
   - Implement log rotation

2. **Monitoring**
   - Set up metrics collection
   - Configure alerts
   - Monitor system health

3. **Maintenance**
   - Regular cleanup
   - Token rotation
   - Configuration review

## Support Resources

1. [GitHub Issues](https://github.com/Gimel-Foundation/gauth/issues)
2. [Documentation](https://gauth.dev/docs)
3. [Community Forum](https://gauth.dev/forum)

## Contributing

If you find a bug or have a solution:
1. Open an issue
2. Provide reproduction steps
3. Submit a fix if possible

For security issues, please use our [security reporting process](SECURITY.md).

---

## Web UI Troubleshooting (Beta Demonstration Interface)

### 1. Tabs or Interactive Elements Not Responding
**Symptoms**: Clicking tab buttons does nothing; only first panel visible.

**Causes**:
- Older implementation relied on implicit global `event` object in inline `onclick` handlers.
- JavaScript failed to load (blocked by CSP or network).

**Fix**:
1. Ensure using updated handler signature `showTab(event,'token-demo')` in `templates/index.html`.
2. Confirm `static/js/app.js` loads (check Network tab in dev tools).
3. Verify `Content-Security-Policy` allows `script-src 'self'` and your script is served from `/static/js/app.js`.

### 2. Icons or Fonts Missing
**Symptoms**: Font Awesome icons replaced with empty squares.

**Causes**:
- CSP missing `font-src` for CDN domains.
- Network blocked to CDN.

**Fix**:
1. Confirm CSP includes: `font-src 'self' https://cdnjs.cloudflare.com https://cdn.jsdelivr.net data:`.
2. Optionally localize fonts (see `web/static/vendor/` scaffolding) and then restrict CSP to `font-src 'self'`.

### 3. Styling Broken / Tailwind Classes Not Applied
**Symptoms**: Page appears unstyled.

**Causes**:
- Tailwind CDN unreachable or CSP blocking styles.
- Missing `'unsafe-inline'` when some CDN styles use inline attributes.

**Fix**:
1. Check Network panel for the Tailwind CSS request.
2. Ensure CSP `style-src` includes the CDN domains and `'unsafe-inline'` (beta demo default) or remove the need for inline styles by local build.

### 4. Port Conflicts When Starting Server
**Symptoms**: `listen tcp :8080: bind: address already in use`.

**Fix**:
1. Find process: `lsof -nP -iTCP:8080 -sTCP:LISTEN`.
2. Choose alternative: `go run web/server.go 9505` or set `GAUTH_WEB_PORT=9505`.

### 5. CSP Blocking Future Localized Assets
If you localize Tailwind / Font Awesome under `web/static/vendor/`, you can tighten CSP:
```
default-src 'self'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' data:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'
```
Remove external CDN hosts and `'unsafe-inline'` once no longer needed.

### 6. Verifying Web UI via Smoke Test
Run `scripts/web-smoke-test.sh` (added) to automatically start the server on a random free port and verify `/`, `/api/v1/beta/health` (primary), legacy `/api/v1/educational/health` (deprecated fallback), and presence of CSP header.

### 7. Rate Limit Responses During Testing
**Symptoms**: 429 errors when clicking quickly.

**Cause**: Beta demo in-memory limiter (60 req/min) triggered.

**Fix**: Wait 60s or reduce click frequency (adjust limit in `web/server.go` if needed).

### 8. Mixed Content / HTTPS Issues
If served behind an HTTPS proxy but assets referenced with `http://`, update to protocol-relative or `https://` URLs. Current code uses `https://` for CDNs already.

---

**Reminder**: This interface is a beta demonstration and intentionally simplified. Harden CSP and remove unused allowances before any real deployment scenario.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
