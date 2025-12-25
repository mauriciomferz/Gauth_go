---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth OPA Integration Example

This directory contains working examples of integrating OPA (Open Policy Agent) with GAuth for advanced authorization policies.

## Contents

- `main.go` - Embedded OPA SDK examples (in-process)
- `http_client.go` - OPA HTTP API client examples (sidecar pattern)
- `policies/scope_validation.rego` - Complete Rego policy for GAuth
- `kubernetes/` - Kubernetes deployment manifests

## Quick Start

### 1. Embedded OPA (Recommended for Development)

The embedded SDK runs OPA in the same process as your application:

```bash
# Install dependencies
go mod download

# Run examples
go run main.go

# Run benchmark
go run main.go benchmark
```

**Expected Output:**
```
=== Example 1: Basic Scope Validation ===
Parent: [users:*]
Child:  [users:read users:write]
✅ Validation successful

Parent: [users:read]
Child:  [users:write]
❌ Validation failed: scope validation failed: child scopes not covered by parent
```

### 2. OPA Sidecar (Recommended for Production)

The sidecar pattern runs OPA as a separate container:

```bash
# Start OPA server
docker run -d \
  -p 8181:8181 \
  -v $(pwd)/policies:/policies \
  openpolicyagent/opa:0.60.0-rootless \
  run --server --addr=:8181 /policies

# Test HTTP client
go run main.go http
```

### 3. Kubernetes Deployment

See `kubernetes/README.md` for production deployment instructions.

## Examples Included

### Example 1: Basic Scope Validation
Demonstrates wildcard pattern matching and scope coverage:
- `users:*` covers `users:read` and `users:write`
- `users:read` does NOT cover `users:write` (prevents escalation)

### Example 2: Multi-Tenant Isolation
Shows how to enforce tenant boundaries:
- `tenant:acme:*` covers `tenant:acme:users:read`
- `tenant:acme:*` does NOT cover `tenant:globex:users:read` (prevents cross-tenant access)

### Example 3: API Versioning
Demonstrates version-scoped delegation:
- `api:v1:*` covers `api:v1:users:read`
- `api:v1:*` does NOT cover `api:v2:users:read` (prevents version escalation)

### Example 4: Detailed Decision Info
Shows how to get audit information:
```json
{
  "allow": true,
  "reason": "All child scopes are covered by parent scopes",
  "timestamp": "2025-11-30T10:15:30Z",
  "matched_scopes": [
    {
      "child": "users:read",
      "parent": "users:*"
    }
  ]
}
```

### Benchmark Results
Performance comparison between OPA and native Go:

| Implementation | Latency | Throughput |
|----------------|---------|------------|
| Native Go      | 60 ns   | 16M ops/sec |
| OPA Embedded   | 500 ns  | 2M ops/sec |
| OPA Sidecar    | 3-5 ms  | 200-300 ops/sec |

**Recommendation:** Use embedded OPA for latency-critical paths (<1ms), sidecar for complex policies with acceptable latency (3-5ms).

## Integration Patterns

### Pattern 1: Embedded SDK (In-Process)
```go
validator, _ := NewOPAScopeValidator()
err := validator.ValidateScope(ctx, parentScopes, childScopes)
```

**Pros:**
- Lowest latency (~500ns)
- No network overhead
- Simple deployment

**Cons:**
- Policies bundled with app
- Requires app restart for policy changes

**Use When:**
- Sub-millisecond latency required
- Simple policies that rarely change
- Development and testing

### Pattern 2: HTTP Sidecar
```go
validator := NewOPAHTTPValidator("http://localhost:8181")
err := validator.ValidateScope(ctx, parentScopes, childScopes)
```

**Pros:**
- Hot-reload policies without app restart
- Centralized policy management
- Better for microservices

**Cons:**
- Higher latency (3-5ms)
- Network dependency
- More complex deployment

**Use When:**
- Policies change frequently
- Multiple services share policies
- Acceptable latency budget
- Production environments

### Pattern 3: Remote Cluster
```go
validator := NewOPAHTTPValidator("https://opa.example.com")
err := validator.ValidateScope(ctx, parentScopes, childScopes)
```

**Pros:**
- Centralized policy across datacenters
- Audit and compliance visibility
- Scalable with load balancing

**Cons:**
- Highest latency (10-50ms)
- Network reliability critical
- Complex infrastructure

**Use When:**
- Enterprise-wide policy enforcement
- Regulatory compliance requirements
- Multi-region deployments

## Testing OPA Policies

### Unit Testing Rego Policies
```bash
# Install OPA CLI
brew install opa  # macOS
# or
curl -L -o opa https://openpolicyagent.org/downloads/latest/opa_linux_amd64

# Test policies
opa test policies/
```

### Integration Testing
```bash
# Run all examples
go test -v

# Run specific test
go test -run TestOPAValidator
```

## Advanced Topics

### Custom Input Data
Add user attributes, time context, or environment info:
```go
input := map[string]interface{}{
    "action":        "validate_scope",
    "parent_scopes": parent,
    "child_scopes":  child,
    "user": map[string]interface{}{
        "department": "engineering",
        "clearance":  "secret",
    },
    "time": map[string]interface{}{
        "hour":     14,
        "timezone": "America/Los_Angeles",
    },
}
```

### Performance Optimization
1. **Prepare queries once:**
   ```go
   query, _ := rego.New(...).PrepareForEval(ctx)
   // Reuse query for multiple evaluations
   ```

2. **Use decision caching:**
   ```go
   // Cache decisions for 5 seconds
   cache := NewDecisionCache(5 * time.Second)
   ```

3. **Batch validations:**
   ```go
   // Validate multiple scope pairs in one call
   ValidateBatch(ctx, pairs)
   ```

### Monitoring
See `docs/OPA_INTEGRATION_GUIDE.md` for Prometheus metrics and alerting.

## Troubleshooting

### Error: "OPA returned no results"
**Cause:** Policy file not loaded correctly.
**Fix:** Check policy path in `embed.FS` directive.

### Error: "OPA request failed: connection refused"
**Cause:** OPA server not running.
**Fix:** Start OPA with `docker run -p 8181:8181 openpolicyagent/opa run --server`

### Error: "unexpected result type"
**Cause:** Policy returns wrong data type.
**Fix:** Ensure Rego policy returns boolean for `allow` rule.

### Performance: High latency
**Cause:** Network overhead or complex policy.
**Fix:** 
- Use embedded SDK instead of HTTP
- Optimize Rego policy (avoid recursive rules)
- Cache decisions

## Further Reading

- [Full Integration Guide](../../docs/OPA_INTEGRATION_GUIDE.md)
- [OPA Documentation](https://www.openpolicyagent.org/docs/latest/)
- [Rego Language Reference](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA Performance Tuning](https://www.openpolicyagent.org/docs/latest/performance/)

## Support

For questions or issues:
1. Check [docs/OPA_INTEGRATION_GUIDE.md](../../docs/OPA_INTEGRATION_GUIDE.md)
2. Review [troubleshooting section](#troubleshooting)
3. Search existing issues in repository
4. Open new issue with reproduction steps
