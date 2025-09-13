package gauth
/*
Package gauth provides a modular authentication and authorization framework.

# Core Concepts

GAuth is built around several key abstractions:

  1. Authentication (auth package)
     Authentication verifies identities using various methods:
     - Basic auth (username/password)
     - JWT tokens
     - PASETO tokens
     - OAuth2

  2. Authorization (authz package)
     Policy-based access control:
     - Role-based access control (RBAC)
     - Attribute-based access control (ABAC)
     - Custom policy engines

  3. Token Management (token package)
     Secure token handling:
     - Generation and validation
     - Storage and retrieval
     - Automatic rotation
     - Different token types (access, refresh)

  4. Rate Limiting (rate package)
     Traffic control:
     - Multiple algorithms (token bucket, sliding window)
     - Distributed rate limiting
     - Custom rate limit policies

  5. Audit Logging (audit package)
     Security event tracking:
     - Structured logging
     - Multiple storage backends
     - Event filtering and aggregation

# Quick Start

Basic usage example:

    import "github.com/Gimel-Foundation/gauth"

    // Create auth service
    auth := gauth.New(gauth.Config{
        AuthType: gauth.TypeJWT,
        SigningKey: []byte("your-secret"),
    })

    // Generate token
    token, err := auth.GenerateToken(ctx, gauth.TokenRequest{
        Subject: "user123",
        Scopes: []string{"read", "write"},
    })

    // Validate token
    claims, err := auth.ValidateToken(ctx, token)

# Architecture

GAuth uses a modular architecture where each component is independent:

    ┌──────────────┐     ┌──────────────┐
    │     Auth     │────▶│    Token     │
    └──────────────┘     └──────────────┘
           │                    │
           ▼                    ▼
    ┌──────────────┐     ┌──────────────┐
    │    Authz     │     │    Store     │
    └──────────────┘     └──────────────┘
           │                    │
           ▼                    ▼
    ┌──────────────┐     ┌──────────────┐
    │    Audit     │────▶│   Events     │
    └──────────────┘     └──────────────┘

Each component can be used independently or together:

    // Use just token management
    tokenMgr := token.NewManager(token.Config{...})

    // Use just rate limiting
    limiter := rate.NewSlidingWindow(rate.Config{...})

    // Use just authorization
    authz := authz.New(authz.Config{...})

# Customization

Each package provides interfaces that can be implemented:

    // Custom token store
    type TokenStore interface {
        Save(context.Context, string, *Token) error
        Get(context.Context, string) (*Token, error)
        Delete(context.Context, string) error
    }

    // Custom auth provider
    type AuthProvider interface {
        Authenticate(context.Context, Credentials) (*Token, error)
        Validate(context.Context, string) (*Claims, error)
    }

# Best Practices

1. Token Management:
   - Use short-lived access tokens
   - Implement token rotation
   - Store tokens securely

2. Rate Limiting:
   - Set appropriate limits
   - Use distributed rate limiting in clusters
   - Monitor rate limit metrics

3. Authentication:
   - Use secure password hashing
   - Implement MFA where needed
   - Handle session management

4. Authorization:
   - Define granular policies
   - Use principle of least privilege
   - Regular policy reviews

For detailed examples, see the examples/ directory.
For implementation details of each component, see their respective package documentation.
*/
package gauth