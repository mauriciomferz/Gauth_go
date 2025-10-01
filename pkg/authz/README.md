# Authorization Package

The `authz` package provides flexible, policy-based authorization for Go applications.

## Features

- Multiple authorization models:
  - Role-Based Access Control (RBAC)
  - Attribute-Based Access Control (ABAC)
  - Policy-Based Access Control (PBAC)
  - Custom authorization logic

- Policy management:
  - Policy creation and validation
  - Policy inheritance
  - Policy versioning
  - Dynamic policies
  - Policy caching

- Integration with:
  - Authentication
  - Audit logging
  - Monitoring
  - Distributed caching

## Usage Examples

### Basic Authorization

```go
import "github.com/Gimel-Foundation/gauth/pkg/authz"

// Create authorizer
az := authz.New(authz.Config{
    PolicyStore: authz.NewMemoryPolicyStore(),
    EnableAudit: true,
})

// Create policy
policy := &authz.Policy{
    Subject:  "user123",
    Resource: "documents/*",
    Actions:  []string{"read", "write"},
    Effect:   authz.Allow,
}

// Add policy
err := az.AddPolicy(ctx, policy)
if err != nil {
    // Handle error
}

// Check authorization
allowed, err := az.IsAllowed(ctx, authz.Request{
    Subject:  "user123",
    Resource: "documents/report.pdf",
    Action:   "read",
})
```

### Role-Based Access Control (RBAC)

```go
// Define roles
admin := authz.Role{
    Name: "admin",
    Permissions: []authz.Permission{
        {Resource: "*", Actions: []string{"*"}},
    },
}

editor := authz.Role{
    Name: "editor",
    Permissions: []authz.Permission{
        {Resource: "articles/*", Actions: []string{"read", "write"}},
        {Resource: "comments/*", Actions: []string{"moderate"}},
    },
}

// Create RBAC authorizer
rbac := authz.NewRBAC(authz.RBACConfig{
    Roles: []authz.Role{admin, editor},
})

// Assign role
err := rbac.AssignRole(ctx, "user123", "editor")
if err != nil {
    // Handle error
}

// Check permission
allowed, err := rbac.CheckPermission(ctx, "user123", "articles/123", "write")
```

### Attribute-Based Access Control (ABAC)

```go
// Define attribute rules
rules := []authz.Rule{
    {
        Name: "WorkingHours",
        Condition: authz.Condition{
            Attribute: "request.time",
            Operator: "between",
            Values: []string{"09:00", "17:00"},
        },
    },
    {
        Name: "IPRestriction",
        Condition: authz.Condition{
            Attribute: "request.ip",
            Operator: "cidr",
            Value: "10.0.0.0/8",
        },
    },
}

// Create ABAC authorizer
abac := authz.NewABAC(authz.ABACConfig{
    Rules: rules,
})

// Check with attributes
allowed, err := abac.Authorize(ctx, authz.Request{
    Subject: "user123",
    Resource: "api/sensitive",
    Action: "access",
    Attributes: map[string]interface{}{
        "request.time": time.Now(),
        "request.ip": "10.0.1.100",
    },
})
```

### Policy-Based Access Control (PBAC)

```go
// Define policy set
policies := authz.PolicySet{
    {
        Name: "DocumentAccess",
        Rules: []authz.Rule{
            {
                Effect: authz.Allow,
                Subjects: []string{"authors", "editors"},
                Resources: []string{"documents/*"},
                Actions: []string{"read", "write"},
                Conditions: map[string]interface{}{
                    "document.status": []string{"draft", "review"},
                },
            },
        },
    },
}

// Create PBAC authorizer
pbac := authz.NewPBAC(authz.PBACConfig{
    Policies: policies,
})

// Check against policies
allowed, err := pbac.Evaluate(ctx, authz.Request{
    Subject: "user123",
    Groups: []string{"authors"},
    Resource: "documents/article.md",
    Action: "write",
    Context: map[string]interface{}{
        "document.status": "draft",
    },
})
```

### Custom Authorization Logic

```go
// Implement custom authorizer
type CustomAuthorizer struct {
    // ...
}

func (ca *CustomAuthorizer) Authorize(ctx context.Context, req authz.Request) (bool, error) {
    // Custom authorization logic
    return true, nil
}

// Use custom authorizer
az := authz.New(authz.Config{
    Authorizer: &CustomAuthorizer{},
})
```

### Policy Inheritance

```go
// Create hierarchical policies
policies := []*authz.Policy{
    {
        Subject: "admin",
        Resource: "/*",
        Actions: []string{"*"},
        Effect: authz.Allow,
    },
    {
        Subject: "user",
        Resource: "/docs/*",
        Actions: []string{"read"},
        Effect: authz.Allow,
    },
}

// Add policies with inheritance
for _, p := range policies {
    err := az.AddPolicy(ctx, p)
    if err != nil {
        // Handle error
    }
}
```

### Caching and Performance

```go
// Configure with caching
az := authz.New(authz.Config{
    Cache: authz.CacheConfig{
        Enabled: true,
        TTL: time.Minute,
        Size: 10000,
    },
})

// Optionally use distributed cache
az := authz.New(authz.Config{
    Cache: authz.CacheConfig{
        Type: authz.RedisCache,
        Redis: authz.RedisConfig{
            Addrs: []string{"localhost:6379"},
        },
    },
})
```

### Integration with Auth

```go
// Create middleware
func AuthzMiddleware(az *authz.Authorizer) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get user from context
            user := auth.UserFromContext(r.Context())

            // Check authorization
            allowed, err := az.IsAllowed(r.Context(), authz.Request{
                Subject:  user.ID,
                Resource: r.URL.Path,
                Action:   r.Method,
            })
            if err != nil || !allowed {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Error Handling

```go
// Handle specific authorization errors
allowed, err := az.IsAllowed(ctx, req)
switch {
case errors.Is(err, authz.ErrPolicyNotFound):
    // Handle missing policy
case errors.Is(err, authz.ErrInvalidPolicy):
    // Handle invalid policy
case errors.Is(err, authz.ErrInvalidRequest):
    // Handle invalid request
default:
    // Handle other errors
}
```

### Monitoring

```go
// Enable monitoring
az := authz.New(authz.Config{
    Monitoring: authz.MonitoringConfig{
        Enabled: true,
        MetricsPrefix: "authz",
    },
})

// Get metrics
metrics := az.GetMetrics()
log.Printf("Total requests: %d", metrics.Requests)
log.Printf("Allow rate: %.2f%%", metrics.AllowRate)
```

## Best Practices

1. Use the most specific authorization model for your needs
2. Implement proper policy management and versioning
3. Enable caching for better performance
4. Monitor authorization decisions
5. Integrate with authentication and audit logging
6. Handle errors appropriately
7. Use middleware for consistent authorization
8. Keep policies simple and maintainable
9. Test authorization logic thoroughly
10. Document policies and authorization rules

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for details on contributing to this package.

## License

This package is part of the GAuth project and is licensed under the same terms.
See [LICENSE](../../LICENSE) for details.