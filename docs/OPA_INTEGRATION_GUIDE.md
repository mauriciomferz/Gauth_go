# Open Policy Agent (OPA) Integration Guide

**Status:** P1.2 Implementation (HIGH Priority)  
**Date:** November 30, 2025  
**Purpose:** External policy engine integration for complex authorization

---

## Overview

While wildcard patterns (P1.1) handle **90% of authorization scenarios**, complex use cases require a full-featured policy engine. This guide shows how to integrate **Open Policy Agent (OPA)** with AgentAuth.

### Why OPA?

OPA provides:
- **Attribute-based access control (ABAC)** beyond simple pattern matching
- **Conditional policies** with runtime context evaluation
- **Policy versioning** and audit trails
- **Unified authorization** across microservices
- **Industry-standard Rego** policy language

---

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ 1. Authorization Request
       ▼
┌─────────────────────┐
│   AgentAuth Server      │
│  ┌──────────────┐   │
│  │ PEP (Policy  │   │ 2. Policy Query
│  │ Enforcement  ├───┼──────────┐
│  │ Point)       │   │          │
│  └──────────────┘   │          ▼
└─────────────────────┘   ┌──────────────┐
                          │ OPA Server   │
                          │ (PDP)        │
                          │              │
                          │ ┌──────────┐ │
                          │ │  Rego    │ │
                          │ │ Policies │ │
                          │ └──────────┘ │
                          └──────────────┘
```

**Deployment Options:**
1. **OPA Sidecar** (Recommended): OPA container alongside AgentAuth in same Pod
2. **OPA Cluster**: Centralized OPA service for multiple applications
3. **Embedded OPA**: In-process OPA using Go SDK

---

## Quick Start

### 1. Install OPA

```bash
# Docker
docker pull openpolicyagent/opa:latest

# Kubernetes Helm
helm repo add opa https://open-policy-agent.github.io/helm-charts
helm install opa opa/opa

# Binary
curl -L -o opa https://openpolicyagent.org/downloads/latest/opa_linux_amd64
chmod +x opa
```

### 2. Create Policy File

`policies/scope_validation.rego`:
```rego
package gauth.authz

# Default deny
default allow = false

# Allow if all child scopes are covered by parent scopes
allow {
    input.action == "validate_scope"
    scope_subset(input.parent_scopes, input.child_scopes)
}

# Helper: Check if child is subset of parent
scope_subset(parent, child) {
    # Every child scope must match at least one parent pattern
    every_child_covered(parent, child)
}

# Check each child scope
every_child_covered(parent, child) {
    count([c | c := child[_]; not scope_matches_any(parent, c)]) == 0
}

# Check if scope matches any parent pattern
scope_matches_any(patterns, scope) {
    pattern := patterns[_]
    scope_matches(pattern, scope)
}

# Exact match
scope_matches(pattern, scope) {
    pattern == scope
}

# Global wildcard
scope_matches(pattern, scope) {
    pattern == "*"
}

# Prefix wildcard
scope_matches(pattern, scope) {
    endswith(pattern, "*")
    prefix := trim_suffix(pattern, "*")
    startswith(scope, prefix)
}

# Suffix wildcard  
scope_matches(pattern, scope) {
    startswith(pattern, "*")
    suffix := trim_prefix(pattern, "*")
    endswith(scope, suffix)
}
```

### 3. Start OPA Server

```bash
# Start OPA with policy bundle
opa run --server --addr=:8181 policies/

# Test policy
curl -X POST http://localhost:8181/v1/data/gauth/authz/allow \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "action": "validate_scope",
      "parent_scopes": ["users:*"],
      "child_scopes": ["users:read", "users:write"]
    }
  }'

# Response: {"result": true}
```

---

## Go Integration Example

### Option A: OPA Go SDK (Embedded)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/open-policy-agent/opa/rego"
)

// OPAScopeValidator replaces validateInheritedScope with OPA
type OPAScopeValidator struct {
    query rego.PreparedEvalQuery
}

// NewOPAScopeValidator creates validator with Rego policy
func NewOPAScopeValidator(policyPath string) (*OPAScopeValidator, error) {
    // Load policy from file
    policy, err := os.ReadFile(policyPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read policy: %w", err)
    }

    // Prepare query
    query, err := rego.New(
        rego.Query("data.gauth.authz.allow"),
        rego.Module("scope_validation.rego", string(policy)),
    ).PrepareForEval(context.Background())
    
    if err != nil {
        return nil, fmt.Errorf("failed to prepare query: %w", err)
    }

    return &OPAScopeValidator{query: query}, nil
}

// ValidateScope checks if child scopes are covered by parent
func (v *OPAScopeValidator) ValidateScope(ctx context.Context, parent, child []string) error {
    // Build input for OPA
    input := map[string]interface{}{
        "action":        "validate_scope",
        "parent_scopes": parent,
        "child_scopes":  child,
    }

    // Evaluate policy
    results, err := v.query.Eval(ctx, rego.EvalInput(input))
    if err != nil {
        return fmt.Errorf("OPA evaluation failed: %w", err)
    }

    // Check result
    if len(results) == 0 {
        return fmt.Errorf("OPA returned no results")
    }

    allowed, ok := results[0].Expressions[0].Value.(bool)
    if !ok || !allowed {
        return fmt.Errorf("scope validation failed: child scopes not covered by parent")
    }

    return nil
}

// Example usage
func main() {
    validator, err := NewOPAScopeValidator("policies/scope_validation.rego")
    if err != nil {
        log.Fatal(err)
    }

    parent := []string{"users:*", "files:read"}
    child := []string{"users:read", "users:write"}

    err = validator.ValidateScope(context.Background(), parent, child)
    if err != nil {
        log.Printf("Validation failed: %v", err)
    } else {
        log.Println("✅ Validation successful")
    }
}
```

### Option B: OPA HTTP API (Sidecar)

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// OPAHTTPValidator uses OPA sidecar via HTTP
type OPAHTTPValidator struct {
    endpoint string
    client   *http.Client
}

// NewOPAHTTPValidator creates HTTP-based validator
func NewOPAHTTPValidator(opaURL string) *OPAHTTPValidator {
    return &OPAHTTPValidator{
        endpoint: opaURL + "/v1/data/gauth/authz/allow",
        client:   &http.Client{Timeout: 5 * time.Second},
    }
}

// ValidateScope checks scopes via OPA HTTP API
func (v *OPAHTTPValidator) ValidateScope(ctx context.Context, parent, child []string) error {
    // Build request payload
    payload := map[string]interface{}{
        "input": map[string]interface{}{
            "action":        "validate_scope",
            "parent_scopes": parent,
            "child_scopes":  child,
        },
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal request: %w", err)
    }

    // Send request to OPA
    req, err := http.NewRequestWithContext(ctx, "POST", v.endpoint, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := v.client.Do(req)
    if err != nil {
        return fmt.Errorf("OPA request failed: %w", err)
    }
    defer resp.Body.Close()

    // Parse response
    var result struct {
        Result bool `json:"result"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to decode OPA response: %w", err)
    }

    if !result.Result {
        return fmt.Errorf("scope validation failed")
    }

    return nil
}
```

---

## Advanced Rego Policies

### Policy 1: Attribute-Based Access Control (ABAC)

```rego
package gauth.authz

# Allow if user has required attributes
allow {
    input.action == "access_resource"
    input.user.department == input.resource.owner_department
    input.user.clearance_level >= input.resource.classification_level
    time.now_ns() < input.resource.expiry_time
}

# Allow managers to access team resources
allow {
    input.action == "access_resource"
    input.user.role == "manager"
    input.resource.team == input.user.team
}
```

### Policy 2: Time-Based Access Control

```rego
package gauth.authz

import future.keywords.if

# Business hours check
is_business_hours if {
    now := time.now_ns()
    hour := time.clock([now, "America/Los_Angeles"])[0]
    hour >= 9
    hour < 18
}

# Allow sensitive operations only during business hours
allow if {
    input.action == "sensitive_operation"
    is_business_hours
    input.user.requires_mfa == true
}
```

### Policy 3: Multi-Tenant Isolation

```rego
package gauth.authz

# Ensure tenant isolation
allow {
    input.action == "access_data"
    input.user.tenant_id == input.resource.tenant_id
    has_permission(input.user.permissions, input.resource.required_permission)
}

# Helper: Check permissions
has_permission(user_perms, required) {
    user_perms[_] == required
}

has_permission(user_perms, required) {
    # Wildcard permission
    user_perms[_] == "*"
}
```

---

## Kubernetes Deployment

### OPA Sidecar Pattern

`k8s/gauth-with-opa.yaml`:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: opa-policy
  namespace: gauth
data:
  scope_validation.rego: |
    package gauth.authz
    
    default allow = false
    
    allow {
        input.action == "validate_scope"
        scope_subset(input.parent_scopes, input.child_scopes)
    }
    
    scope_subset(parent, child) {
        every_child_covered(parent, child)
    }
    
    every_child_covered(parent, child) {
        count([c | c := child[_]; not scope_matches_any(parent, c)]) == 0
    }
    
    scope_matches_any(patterns, scope) {
        pattern := patterns[_]
        scope_matches(pattern, scope)
    }
    
    scope_matches(pattern, scope) { pattern == scope }
    scope_matches(pattern, scope) { pattern == "*" }
    scope_matches(pattern, scope) {
        endswith(pattern, "*")
        prefix := trim_suffix(pattern, "*")
        startswith(scope, prefix)
    }

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gauth
  namespace: gauth
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gauth
  template:
    metadata:
      labels:
        app: gauth
    spec:
      containers:
      # AgentAuth application
      - name: gauth
        image: ghcr.io/mauriciomferz/gauth:latest
        ports:
        - containerPort: 8080
        env:
        - name: OPA_URL
          value: "http://localhost:8181"
        - name: GAUTH_USE_OPA
          value: "true"
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      
      # OPA sidecar
      - name: opa
        image: openpolicyagent/opa:0.59.0
        ports:
        - containerPort: 8181
        args:
        - "run"
        - "--server"
        - "--addr=:8181"
        - "/policies"
        volumeMounts:
        - name: opa-policy
          mountPath: /policies
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8181
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8181
          initialDelaySeconds: 5
          periodSeconds: 10
      
      volumes:
      - name: opa-policy
        configMap:
          name: opa-policy

---
apiVersion: v1
kind: Service
metadata:
  name: gauth
  namespace: gauth
spec:
  selector:
    app: gauth
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Deploy

```bash
# Create namespace
kubectl create namespace gauth

# Deploy AgentAuth with OPA sidecar
kubectl apply -f k8s/gauth-with-opa.yaml

# Verify deployment
kubectl get pods -n gauth
kubectl logs -n gauth <pod-name> -c opa
kubectl logs -n gauth <pod-name> -c gauth
```

---

## Testing OPA Integration

### Unit Tests

```go
package main

import (
    "context"
    "testing"
)

func TestOPAScopeValidator(t *testing.T) {
    validator, err := NewOPAScopeValidator("../../policies/scope_validation.rego")
    if err != nil {
        t.Fatalf("failed to create validator: %v", err)
    }

    tests := []struct {
        name      string
        parent    []string
        child     []string
        shouldErr bool
    }{
        {
            name:      "exact match",
            parent:    []string{"users:read"},
            child:     []string{"users:read"},
            shouldErr: false,
        },
        {
            name:      "wildcard coverage",
            parent:    []string{"users:*"},
            child:     []string{"users:read", "users:write"},
            shouldErr: false,
        },
        {
            name:      "scope escalation",
            parent:    []string{"users:read"},
            child:     []string{"users:write"},
            shouldErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.ValidateScope(context.Background(), tt.parent, tt.child)
            if (err != nil) != tt.shouldErr {
                t.Errorf("expected error=%v, got %v", tt.shouldErr, err)
            }
        })
    }
}
```

### Integration Tests

```bash
# Start OPA
opa run --server --addr=:8181 policies/ &

# Test policy directly
curl -X POST http://localhost:8181/v1/data/gauth/authz/allow \
  -d '{"input": {"action": "validate_scope", "parent_scopes": ["users:*"], "child_scopes": ["users:read"]}}'

# Expected: {"result": true}

# Test escalation
curl -X POST http://localhost:8181/v1/data/gauth/authz/allow \
  -d '{"input": {"action": "validate_scope", "parent_scopes": ["users:read"], "child_scopes": ["users:write"]}}'

# Expected: {"result": false}
```

---

## Performance Considerations

### Benchmarks

```
Operation                    Latency    Notes
─────────────────────────────────────────────
Native Go (wildcards)        60ns      Fastest
OPA Embedded (Go SDK)        ~500ns    In-process
OPA HTTP (localhost)         ~2ms      Network overhead
OPA HTTP (k8s sidecar)       ~3-5ms    Container network
OPA Cluster (remote)         ~10-20ms  Full network roundtrip
```

### Optimization Tips

1. **Use OPA Sidecar** for lowest latency (3-5ms acceptable for authz)
2. **Cache decisions** in AgentAuth for frequently checked permissions
3. **Batch policy queries** when checking multiple permissions
4. **Pre-compile policies** at startup (embedded OPA)
5. **Monitor OPA metrics** (decision count, latency, errors)

---

## Migration Strategy

### Phase 1: Parallel Run (Week 1-2)
```go
// Run both wildcard and OPA, compare results
wildcardErr := validateInheritedScope(parent, child)
opaErr := opaValidator.ValidateScope(ctx, parent, child)

if wildcardErr != opaErr {
    log.Warn("Decision mismatch between wildcard and OPA")
    metrics.IncPolicyMismatch()
}

// Use wildcard result (existing behavior)
return wildcardErr
```

### Phase 2: OPA Primary (Week 3-4)
```go
// Use OPA, fallback to wildcard on error
err := opaValidator.ValidateScope(ctx, parent, child)
if err != nil && isOPAUnavailable(err) {
    log.Warn("OPA unavailable, falling back to wildcards")
    return validateInheritedScope(parent, child)
}
return err
```

### Phase 3: OPA Only (Week 5+)
```go
// Pure OPA, no fallback
return opaValidator.ValidateScope(ctx, parent, child)
```

---

## Monitoring

### Metrics to Track

```prometheus
# OPA decision count
gauth_opa_decisions_total{decision="allow|deny",policy="scope_validation"}

# OPA latency
gauth_opa_latency_seconds{quantile="0.5|0.9|0.99"}

# OPA errors
gauth_opa_errors_total{type="timeout|network|policy_error"}

# OPA availability
gauth_opa_available{status="up|down"}
```

### Alerts

```yaml
groups:
- name: opa_alerts
  rules:
  - alert: OPAHighLatency
    expr: histogram_quantile(0.99, gauth_opa_latency_seconds) > 0.1
    for: 5m
    annotations:
      summary: "OPA P99 latency > 100ms"
  
  - alert: OPAUnavailable
    expr: gauth_opa_available{status="down"} == 1
    for: 1m
    annotations:
      summary: "OPA sidecar unavailable"
```

---

## When to Use OPA vs Wildcards

| Requirement | Wildcards | OPA |
|------------|-----------|-----|
| Simple pattern matching | ✅ Best | ⚠️ Overkill |
| Delegation chains | ✅ Sufficient | ✅ Also works |
| Attribute-based rules | ❌ Not supported | ✅ Required |
| Conditional logic | ❌ Not supported | ✅ Required |
| Time-based access | ❌ Not supported | ✅ Required |
| Multi-tenant isolation | ✅ With patterns | ✅ More flexible |
| Regulatory compliance | ⚠️ Limited | ✅ Full audit trail |
| Performance | ✅ 60ns | ⚠️ 3-5ms |
| Operational complexity | ✅ None | ⚠️ Additional service |

---

## Troubleshooting

### Problem: OPA returns undefined
```
Error: OPA returned no results

Solution: Check policy package matches query
- Query: data.gauth.authz.allow
- Policy must have: package gauth.authz
```

### Problem: OPA sidecar won't start
```
Error: OPA container CrashLoopBackOff

Solution: Check policy syntax
opa check policies/scope_validation.rego
```

### Problem: High OPA latency
```
P99 latency > 100ms

Solutions:
1. Check OPA resource limits (increase CPU)
2. Optimize Rego policy (avoid expensive operations)
3. Add caching layer in AgentAuth
4. Use OPA bundles for faster policy loading
```

---

## References

- [OPA Official Documentation](https://www.openpolicyagent.org/docs/latest/)
- [Rego Playground](https://play.openpolicyagent.org/)
- [OPA Kubernetes Tutorial](https://www.openpolicyagent.org/docs/latest/kubernetes-tutorial/)
- [OPA Go SDK](https://pkg.go.dev/github.com/open-policy-agent/opa/rego)
- [AgentAuth Wildcard Patterns Guide](./WILDCARD_SCOPE_PATTERNS_GUIDE.md)

---

**Implementation Status:** ✅ **COMPLETE** (P1.2)  
**Integration:** Example code + K8s manifests provided  
**Testing:** Unit and integration test examples  
**Documentation:** Complete with deployment guide  

**Next:** P1.3 - OAuth 2.0 Migration Feasibility Study (ETA: December 2025)
