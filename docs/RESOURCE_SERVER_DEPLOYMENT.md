# Resource Server Deployment Guide
## AgentAuth 1.0 Protocol - Production Resource Server Implementation

**Date:** November 15, 2025  
**RFC Reference:** AAP-0111 Corrected Protocol Flow  
**Component:** Resource Server (OAuth/OIDC Foundation + AgentAuth Extensions)

---

## Overview

The **Resource Server (RS)** is part of the OAuth 2.0 / OpenID Connect foundation that AgentAuth builds upon. This guide documents how to deploy a AgentAuth-compliant Resource Server that implements **AgentAuth extensions** while maintaining OAuth/OIDC compatibility.

### What the Resource Server Does

**OAuth/OIDC Foundation:**
- Serves protected resources (APIs, data, services)
- Validates access tokens (Bearer tokens)
- Enforces OAuth scopes
- Returns standard HTTP error codes (401, 403)

**AgentAuth Extensions:**
- Validates **Extended Tokens** with PoA claims
- Enforces **Power of Attorney (PoA)** restrictions
- Implements **PEP** (Policy Enforcement Point) for authorization decisions
- Reports **compliance events** to Authorization Server
- Validates **authorization chains**

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Resource Server (RS)                     │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │         HTTP API Endpoints (Your Business Logic)      │ │
│  └─────────────────────┬─────────────────────────────────┘ │
│                        │                                    │
│  ┌─────────────────────▼─────────────────────────────────┐ │
│  │     AgentAuth PEP Middleware (Demand-Side Enforcement)    │ │
│  │  - Token validation                                   │ │
│  │  - PoA enforcement                                    │ │
│  │  - Scope checking                                     │ │
│  │  - Restriction validation                             │ │
│  └─────────────────────┬─────────────────────────────────┘ │
│                        │                                    │
│  ┌─────────────────────▼─────────────────────────────────┐ │
│  │         Extended Token Validator                      │ │
│  │  - JWT signature verification                         │ │
│  │  - Token introspection (optional)                     │ │
│  │  - Authorization chain validation                     │ │
│  └─────────────────────┬─────────────────────────────────┘ │
│                        │                                    │
│  ┌─────────────────────▼─────────────────────────────────┐ │
│  │         Compliance Tracker                            │ │
│  │  - Event logging                                      │ │
│  │  - Audit trail                                        │ │
│  │  - Reporting to AS                                    │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                         │
                         │ OAuth/OIDC Protocol
                         │ AgentAuth Extensions
                         ▼
          ┌──────────────────────────────┐
          │  Authorization Server (AS)   │
          │  - Token issuance            │
          │  - Token introspection       │
          │  - Compliance tracking       │
          └──────────────────────────────┘
```

---

## Implementation Patterns

### Pattern 1: Embedded PEP (Recommended for Go)

**Use AgentAuth_go library components directly in your application.**

#### Step 1: Import AgentAuth Components

```go
import (
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/agentauth"
)
```

#### Step 2: Initialize PEP

```go
// Initialize Extended Token Service for validation
tokenService := agentauth.NewExtendedTokenService(
    privateKey,           // Your RS private key
    authChainValidator,   // Authorization chain validator
    complianceValidator,  // Compliance validator
    pipClient,           // Policy Information Point
    issuerID,            // "https://your-as.example.com"
    issuerURL,           // Your AS URL
    tokenExpiry,         // e.g., 1 hour
)

// Initialize PEP (Policy Enforcement Point)
tokenValidator := &SimpleTokenValidator{
    extTokenService: tokenService,
}

pdp := agentauth.NewSimplePDP()  // Policy Decision Point

pep := agentauth.NewPowerEnforcementPoint(
    tokenValidator,
    pdp,
    auditLogger,          // Audit logger
    complianceTracker,    // Compliance tracker
    "strict",             // Enforcement mode
)
```

#### Step 3: Create Middleware

```go
func AgentAuthMiddleware(pep *agentauth.PowerEnforcementPoint) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract Bearer token
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, `{"error":"missing_token"}`, http.StatusUnauthorized)
                return
            }
            
            token := strings.TrimPrefix(authHeader, "Bearer ")
            
            // Create enforcement request
            enforcementReq := &agentauth.EnforcementRequest{
                ExtendedToken: token,
                Action:        r.Method + " " + r.URL.Path,
                Resource:      r.URL.Path,
                ClientID:      extractClientID(token),
                Context: map[string]interface{}{
                    "method": r.Method,
                    "path":   r.URL.Path,
                    "ip":     r.RemoteAddr,
                },
            }
            
            // Enforce authorization (Demand-side PEP)
            result, err := pep.ValidateDemandSide(r.Context(), enforcementReq)
            if err != nil {
                http.Error(w, `{"error":"enforcement_error"}`, http.StatusInternalServerError)
                return
            }
            
            if !result.Allowed {
                errorResponse := map[string]interface{}{
                    "error":             "access_denied",
                    "error_description": result.DenyReason,
                    "violations":        result.Violations,
                }
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusForbidden)
                json.NewEncoder(w).Encode(errorResponse)
                return
            }
            
            // Store extended token in context for business logic
            ctx := context.WithValue(r.Context(), "extended_token", result.ExtendedToken)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

#### Step 4: Apply Middleware to Routes

```go
func main() {
    // Initialize PEP (see Step 2)
    pep := initializePEP()
    
    // Create router
    router := http.NewServeMux()
    
    // Protected endpoints
    router.HandleFunc("/api/v1/transaction", handleTransaction)
    router.HandleFunc("/api/v1/decision", handleDecision)
    router.HandleFunc("/api/v1/action", handleAction)
    
    // Apply AgentAuth middleware
    protected := AgentAuthMiddleware(pep)(router)
    
    // Start server
    log.Fatal(http.ListenAndServe(":8443", protected))
}
```

#### Step 5: Implement Business Logic

```go
func handleTransaction(w http.ResponseWriter, r *http.Request) {
    // Get extended token from context
    extToken := r.Context().Value("extended_token").(*agentauth.ExtendedToken)
    
    // Extract request body
    var req TransactionRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Execute transaction with PoA context
    result, err := executeBusinessTransaction(r.Context(), req, extToken)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return success response
    response := TransactionResponse{
        TransactionID: result.ID,
        Status:        "completed",
        Timestamp:     time.Now(),
        Result:        result.Data,
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(response)
}
```

---

### Pattern 2: Reverse Proxy with PEP Sidecar

**Use a separate PEP service as a sidecar/proxy in front of your existing application.**

#### Architecture

```
┌─────────────────┐      ┌─────────────────┐      ┌──────────────────┐
│                 │      │                 │      │                  │
│  Client with    │─────▶│  PEP Sidecar    │─────▶│  Your Existing   │
│  Extended Token │      │  (Port 8443)    │      │  Application     │
│                 │      │                 │      │  (Port 8080)     │
└─────────────────┘      └─────────────────┘      └──────────────────┘
```

#### PEP Sidecar Implementation

```go
// pep-sidecar/main.go
package main

import (
    "net/http"
    "net/http/httputil"
    "net/url"
    
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/agentauth"
)

func main() {
    // Initialize PEP
    pep := initializePEP()
    
    // Upstream application URL
    upstream, _ := url.Parse("http://localhost:8080")
    proxy := httputil.NewSingleHostReverseProxy(upstream)
    
    // Create handler
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Enforce AgentAuth authorization
        if !enforceAgentAuthAuthorization(pep, w, r) {
            return // enforcement failed, response already sent
        }
        
        // Forward to upstream
        proxy.ServeHTTP(w, r)
    })
    
    log.Fatal(http.ListenAndServe(":8443", handler))
}

func enforceAgentAuthAuthorization(pep *agentauth.PowerEnforcementPoint, w http.ResponseWriter, r *http.Request) bool {
    // Extract and validate token (same as Pattern 1)
    // ... enforcement logic ...
    return true // if authorized
}
```

#### Kubernetes Deployment

```yaml
# k8s-resource-server-pep.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: resource-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: resource-server
  template:
    metadata:
      labels:
        app: resource-server
    spec:
      containers:
      # Main application container
      - name: app
        image: your-app:latest
        ports:
        - containerPort: 8080
        
      # PEP sidecar container
      - name: pep-sidecar
        image: agentauth-pep-sidecar:latest
        ports:
        - containerPort: 8443
        env:
        - name: UPSTREAM_URL
          value: "http://localhost:8080"
        - name: AS_URL
          value: "https://auth.example.com"
        - name: ENFORCEMENT_MODE
          value: "strict"
---
apiVersion: v1
kind: Service
metadata:
  name: resource-server
spec:
  selector:
    app: resource-server
  ports:
  - port: 443
    targetPort: 8443  # PEP sidecar port
```

---

## Token Validation

### Local JWT Validation (Recommended for Performance)

```go
func (rs *ResourceServer) ValidateTokenLocally(ctx context.Context, tokenString string) (*agentauth.ExtendedToken, error) {
    // Parse JWT
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        // Get AS public key from JWKS endpoint
        return rs.getASPublicKey(token.Header["kid"].(string))
    })
    
    if err != nil || !token.Valid {
        return nil, fmt.Errorf("invalid token signature")
    }
    
    // Verify standard claims
    claims := token.Claims.(jwt.MapClaims)
    
    // Check expiration
    if exp, ok := claims["exp"].(float64); ok {
        if time.Now().Unix() > int64(exp) {
            return nil, fmt.Errorf("token expired")
        }
    }
    
    // Verify issuer
    if iss, ok := claims["iss"].(string); ok && iss != rs.trustedIssuer {
        return nil, fmt.Errorf("untrusted issuer")
    }
    
    // Verify audience
    if aud, ok := claims["aud"].(string); ok && aud != rs.resourceServerID {
        return nil, fmt.Errorf("invalid audience")
    }
    
    // Check revocation (local cache or Redis)
    if jti, ok := claims["jti"].(string); ok {
        if rs.isRevoked(ctx, jti) {
            return nil, fmt.Errorf("token revoked")
        }
    }
    
    // Deserialize AgentAuth extended claims
    extToken := rs.deserializeExtendedToken(claims)
    
    return extToken, nil
}
```

### Token Introspection (Optional, for Real-time Status)

```go
func (rs *ResourceServer) IntrospectToken(ctx context.Context, tokenString string) (*agentauth.ExtendedToken, error) {
    // Prepare introspection request
    data := url.Values{
        "token": {tokenString},
        "token_type_hint": {"access_token"},
    }
    
    // Call AS introspection endpoint
    req, _ := http.NewRequestWithContext(ctx, "POST", rs.asIntrospectURL, strings.NewReader(data.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.SetBasicAuth(rs.clientID, rs.clientSecret)
    
    resp, err := rs.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    // Parse introspection response
    var introspection struct {
        Active bool                   `json:"active"`
        Scope  string                 `json:"scope"`
        Claims map[string]interface{} `json:",inline"`
    }
    json.NewDecoder(resp.Body).Decode(&introspection)
    
    if !introspection.Active {
        return nil, fmt.Errorf("token not active")
    }
    
    // Deserialize extended token from introspection response
    extToken := rs.deserializeExtendedToken(introspection.Claims)
    
    return extToken, nil
}
```

---

## PoA Enforcement

### Validate PoA Restrictions

```go
func (rs *ResourceServer) EnforcePoA(ctx context.Context, extToken *agentauth.ExtendedToken, action string, resource string) error {
    if extToken.PowerOfAttorney == nil {
        return fmt.Errorf("no PoA present in token")
    }
    
    poa := extToken.PowerOfAttorney
    
    // Check geographic restrictions
    if len(poa.Scope.GeographicScope.Countries) > 0 {
        userCountry := rs.getUserCountry(ctx)
        if !contains(poa.Scope.GeographicScope.Countries, userCountry) {
            return fmt.Errorf("action not allowed in country %s", userCountry)
        }
    }
    
    // Check sector restrictions
    resourceSector := rs.getResourceSector(resource)
    if len(poa.Scope.SectorScope.AllowedSectors) > 0 {
        if !contains(poa.Scope.SectorScope.AllowedSectors, resourceSector) {
            return fmt.Errorf("action not allowed for sector %s", resourceSector)
        }
    }
    
    // Check action types
    actionType := rs.mapActionToType(action)
    if !contains(poa.Scope.ActionTypes, actionType) {
        return fmt.Errorf("action type %s not authorized", actionType)
    }
    
    // Check restrictions
    for _, restriction := range extToken.Restrictions {
        if err := rs.validateRestriction(restriction, action, resource); err != nil {
            return err
        }
    }
    
    return nil
}
```

---

## Compliance Event Reporting

### Report to Authorization Server

```go
func (rs *ResourceServer) ReportComplianceEvent(ctx context.Context, event *agentauth.ComplianceEvent) error {
    // Prepare event
    eventJSON, _ := json.Marshal(event)
    
    // Send to AS compliance endpoint
    req, _ := http.NewRequestWithContext(
        ctx,
        "POST",
        rs.asComplianceURL,
        bytes.NewReader(eventJSON),
    )
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+rs.rsAccessToken)
    
    resp, err := rs.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 && resp.StatusCode != 202 {
        return fmt.Errorf("compliance report failed: %d", resp.StatusCode)
    }
    
    return nil
}

// Usage in middleware
func (rs *ResourceServer) logComplianceEvent(
    ctx context.Context,
    extToken *agentauth.ExtendedToken,
    action string,
    resource string,
    allowed bool,
    reason string,
) {
    event := &agentauth.ComplianceEvent{
        EventID:   uuid.New().String(),
        Timestamp: time.Now(),
        ClientID:  extToken.AuthorizationChain.Client.EntityID,
        Action:    action,
        Resource:  resource,
        Allowed:   allowed,
        Reason:    reason,
        TokenJTI:  extToken.AccessToken, // JTI claim
    }
    
    // Report asynchronously
    go rs.ReportComplianceEvent(context.Background(), event)
}
```

---

## Configuration

### Environment Variables

```bash
# Authorization Server
export AGENTAUTH_AS_URL="https://auth.example.com"
export AGENTAUTH_AS_ISSUER="https://auth.example.com"
export AGENTAUTH_AS_JWKS_URL="https://auth.example.com/.well-known/jwks.json"
export AGENTAUTH_AS_INTROSPECT_URL="https://auth.example.com/introspect"
export AGENTAUTH_AS_COMPLIANCE_URL="https://auth.example.com/compliance/events"

# Resource Server Identity
export AGENTAUTH_RS_ID="resource-server-001"
export AGENTAUTH_RS_CLIENT_ID="rs-client-001"
export AGENTAUTH_RS_CLIENT_SECRET="secret-key-here"

# Enforcement
export AGENTAUTH_ENFORCEMENT_MODE="strict"  # or "advisory"
export AGENTAUTH_TOKEN_VALIDATION_MODE="local"  # or "introspection"

# Performance
export AGENTAUTH_JWKS_CACHE_TTL="3600"  # 1 hour
export AGENTAUTH_TOKEN_CACHE_TTL="300"  # 5 minutes
export AGENTAUTH_REVOCATION_CHECK_TTL="60"  # 1 minute
```

### Configuration File

```yaml
# config/resource-server.yaml
agentauth:
  authorization_server:
    url: "https://auth.example.com"
    issuer: "https://auth.example.com"
    jwks_url: "https://auth.example.com/.well-known/jwks.json"
    introspect_url: "https://auth.example.com/introspect"
    compliance_url: "https://auth.example.com/compliance/events"
  
  resource_server:
    id: "resource-server-001"
    client_id: "rs-client-001"
    client_secret: "${RS_CLIENT_SECRET}"
    
  enforcement:
    mode: "strict"  # strict, advisory
    token_validation: "local"  # local, introspection, hybrid
    
  cache:
    jwks_ttl: 3600
    token_ttl: 300
    revocation_ttl: 60
    
  endpoints:
    # AgentAuth-specific endpoints
    transaction: "/api/v1/transaction"
    decision: "/api/v1/decision"
    action: "/api/v1/action"
```

---

## Error Responses

### OAuth 2.0 Standard Errors (RFC 6750)

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer error="invalid_token", error_description="Token signature verification failed"
Content-Type: application/json

{
  "error": "invalid_token",
  "error_description": "Token signature verification failed"
}
```

### AgentAuth Extension Errors

```http
HTTP/1.1 403 Forbidden
Content-Type: application/json

{
  "error": "insufficient_authorization",
  "error_description": "PoA does not permit this action",
  "agentauth_violations": [
    {
      "type": "poa_scope",
      "description": "Action type 'contract_signature' not in authorized actions",
      "severity": "critical"
    },
    {
      "type": "geographic_restriction",
      "description": "Operation not permitted in country 'US'",
      "severity": "high"
    }
  ],
  "poa_reference": "PoA-2025-001"
}
```

---

## Monitoring & Observability

### Metrics to Track

```go
// Prometheus metrics
var (
    tokenValidationsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agentauth_rs_token_validations_total",
            Help: "Total number of token validations",
        },
        []string{"result"}, // success, invalid, expired, revoked
    )
    
    poaEnforcementsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agentauth_rs_poa_enforcements_total",
            Help: "Total number of PoA enforcements",
        },
        []string{"result"}, // allowed, denied
    )
    
    authorizationLatency = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "agentauth_rs_authorization_duration_seconds",
            Help: "Authorization check duration",
            Buckets: prometheus.DefBuckets,
        },
    )
)
```

### Logging

```go
log.WithFields(log.Fields{
    "event": "authorization_check",
    "client_id": extToken.ClientOwner,
    "action": action,
    "resource": resource,
    "result": "allowed",
    "poa_id": extToken.PowerOfAttorney.PoAID,
    "duration_ms": duration.Milliseconds(),
}).Info("Authorization check completed")
```

---

## Security Best Practices

1. **HTTPS Only** - All RS endpoints MUST use TLS 1.2+
2. **Token Validation** - ALWAYS validate JWT signatures
3. **Revocation Checking** - Check revocation status (cache for performance)
4. **Audience Validation** - Verify `aud` claim matches your RS ID
5. **Rate Limiting** - Implement rate limiting per client
6. **Audit Logging** - Log all authorization decisions
7. **Key Rotation** - Support multiple AS public keys for rotation
8. **Timeout Configuration** - Set reasonable timeouts for AS calls
9. **Circuit Breaker** - Implement circuit breaker for AS introspection
10. **Fail Secure** - Deny access if enforcement check fails

---

## Testing

### Unit Tests

```go
func TestPEPEnforcement(t *testing.T) {
    // Create test token with PoA
    token := createTestExtendedToken()
    
    // Create enforcement request
    req := &agentauth.EnforcementRequest{
        ExtendedToken: token,
        Action:        "POST /api/v1/transaction",
        Resource:      "/api/v1/transaction",
    }
    
    // Test enforcement
    result, err := pep.ValidateDemandSide(context.Background(), req)
    
    assert.NoError(t, err)
    assert.True(t, result.Allowed)
}
```

### Integration Tests

```bash
# Test token validation
curl -H "Authorization: Bearer $TOKEN" \
     https://rs.example.com/api/v1/transaction \
     -d '{"type":"test"}' \
     -v

# Expected: 200 OK (if authorized) or 403 Forbidden (if denied)
```

---

## Production Checklist

- [ ] TLS certificates configured
- [ ] AS public keys cached (JWKS)
- [ ] Token revocation checking enabled
- [ ] PEP middleware applied to all protected endpoints
- [ ] Compliance event reporting configured
- [ ] Metrics and monitoring enabled
- [ ] Error handling and logging configured
- [ ] Rate limiting implemented
- [ ] Health check endpoint exposed
- [ ] Configuration externalized (environment variables)
- [ ] Security headers configured (CSP, HSTS, etc.)
- [ ] Load testing completed
- [ ] Failover and high availability tested

---

## Conclusion

This guide provides comprehensive patterns for deploying a AgentAuth-compliant Resource Server. The implementation properly separates OAuth/OIDC foundation (token validation, standard errors) from AgentAuth extensions (PoA enforcement, compliance reporting).

**Key Takeaways:**
- RS is OAuth/OIDC foundation, extended by AgentAuth
- PEP middleware handles AgentAuth-specific enforcement
- Local token validation recommended for performance
- Compliance event reporting enables AI governance
- Multiple deployment patterns support different architectures

**Next Steps:**
1. Choose deployment pattern (embedded vs sidecar)
2. Implement PEP middleware
3. Configure token validation
4. Enable compliance reporting
5. Deploy and monitor

For questions or issues, refer to:
- AgentAuth_go codebase: `pkg/agentauth/pep.go`
- RFC corrections: `docs/Gifo_0111_CORRECTED_FLOW.md`
- Implementation coverage: `docs/RFC_IMPLEMENTATION_COVERAGE.md`
