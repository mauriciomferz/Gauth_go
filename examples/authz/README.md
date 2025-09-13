# Authorization Examples

This directory contains examples demonstrating authorization patterns.

## Role-Based Access Control (RBAC)

```go
package main

import (
    "context"
    "log"

    "github.com/Gimel-Foundation/gauth/pkg/authz"
)

func main() {
    // Initialize authorizer
    authz := authz.New(authz.Config{
        Store: authz.NewMemoryStore(),
    })

    // Define roles
    adminRole := authz.NewRole("admin").
        AddPermission("users:*").
        AddPermission("systems:*")

    userRole := authz.NewRole("user").
        AddPermission("users:read").
        AddPermission("users:update:self")

    // Create roles
    ctx := context.Background()
    if err := authz.CreateRole(ctx, adminRole); err != nil {
        log.Fatal(err)
    }
    if err := authz.CreateRole(ctx, userRole); err != nil {
        log.Fatal(err)
    }

    // Assign roles
    if err := authz.AssignRole(ctx, "user123", "user"); err != nil {
        log.Fatal(err)
    }

    // Check permissions
    allowed, err := authz.Check(ctx, authz.Request{
        Subject:  "user123",
        Resource: "users",
        Action:   "read",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Access allowed: %v", allowed)
}
```

## Policy-Based Authorization

```go
package main

import (
    "context"
    "log"

    "github.com/Gimel-Foundation/gauth/pkg/authz"
)

func main() {
    // Create policy engine
    engine := authz.NewPolicyEngine(authz.PolicyConfig{
        Store: authz.NewRedisStore("localhost:6379"),
    })

    // Define policy
    policy := authz.Policy{
        Name: "resource-access",
        Rules: []authz.Rule{
            {
                Subjects:  []string{"user:*"},
                Resources: []string{"documents:*"},
                Actions:   []string{"read", "write"},
                Effect:    authz.Allow,
                Conditions: []authz.Condition{
                    {
                        Type: "TimeOfDay",
                        Options: map[string]interface{}{
                            "start": "09:00",
                            "end":   "17:00",
                        },
                    },
                },
            },
        },
    }

    // Create policy
    ctx := context.Background()
    if err := engine.CreatePolicy(ctx, policy); err != nil {
        log.Fatal(err)
    }

    // Evaluate policy
    result, err := engine.Evaluate(ctx, authz.Request{
        Subject:  "user:123",
        Resource: "documents:456",
        Action:   "read",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Policy evaluation result: %v", result.Allowed)
}
```

## Attribute-Based Access Control (ABAC)

```go
package main

import (
    "context"
    "log"

    "github.com/Gimel-Foundation/gauth/pkg/authz"
)

func main() {
    // Create ABAC engine
    abac := authz.NewABAC(authz.ABACConfig{
        Store: authz.NewMemoryStore(),
    })

    // Define attribute rules
    rule := authz.AttributeRule{
        Name: "document-access",
        Attributes: map[string]interface{}{
            "user.department": "engineering",
            "resource.type":  "technical-doc",
            "action":         "read",
        },
        Effect: authz.Allow,
    }

    // Add rule
    ctx := context.Background()
    if err := abac.AddRule(ctx, rule); err != nil {
        log.Fatal(err)
    }

    // Check access with attributes
    allowed, err := abac.CheckAccess(ctx, authz.AttributeRequest{
        Subject: map[string]interface{}{
            "id":         "user123",
            "department": "engineering",
            "role":      "engineer",
        },
        Resource: map[string]interface{}{
            "id":   "doc123",
            "type": "technical-doc",
            "owner": "team1",
        },
        Action: "read",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Access allowed: %v", allowed)
}
```

## Distributed Authorization

```go
package main

import (
    "context"
    "log"

    "github.com/Gimel-Foundation/gauth/pkg/authz"
)

func main() {
    // Create distributed authorizer
    auth := authz.NewDistributed(authz.DistributedConfig{
        Store: authz.NewRedisStore("localhost:6379"),
        Cache: authz.NewCache(1000),
    })

    // Add nodes
    nodes := []string{
        "http://auth1:8080",
        "http://auth2:8080",
        "http://auth3:8080",
    }
    for _, node := range nodes {
        if err := auth.AddNode(node); err != nil {
            log.Printf("Failed to add node %s: %v", node, err)
        }
    }

    // Check permission (automatically distributed)
    ctx := context.Background()
    allowed, err := auth.Check(ctx, authz.Request{
        Subject:  "user123",
        Resource: "document123",
        Action:   "read",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Access allowed: %v", allowed)
}
```

## Best Practices

1. **Use Context for Deadlines**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
allowed, err := authz.Check(ctx, request)
```

2. **Cache Common Decisions**
```go
cache := authz.NewCache(1000)
authorizer := authz.New(authz.Config{
    Store: store,
    Cache: cache,
})
```

3. **Implement Proper Error Handling**
```go
if err != nil {
    switch {
    case errors.Is(err, authz.ErrUnauthorized):
        // Handle unauthorized
    case errors.Is(err, authz.ErrPolicyNotFound):
        // Handle missing policy
    default:
        // Handle other errors
    }
}
```

4. **Use Fine-Grained Permissions**
```go
// Good
"users:read:self"
"documents:write:department"

// Bad
"read"
"write-all"
```

5. **Regular Policy Reviews**
```go
policies, err := authz.ListPolicies(ctx)
for _, policy := range policies {
    if time.Since(policy.LastReviewed) > 90*24*time.Hour {
        log.Printf("Policy %s needs review", policy.Name)
    }
}
```