# GAuth Library Usage Guide

This document provides a concise overview for developers who want to use GAuth as a library in their own Go projects.

## Quick Start

1. **Install the package:**
   ```sh
   go get github.com/Gimel-Foundation/gauth
   ```
2. **Import and initialize:**
   ```go
   import "github.com/Gimel-Foundation/gauth/pkg/gauth"

   svc, err := gauth.New(gauth.Config{ /* ... */ })
   if err != nil {
       // handle error
   }
   ```
3. **Use the service:**
   - Authorize, issue tokens, audit, etc.

## Package Structure
- `pkg/gauth` — Main entry point for service usage
- `pkg/token` — Token management
- `pkg/auth` / `pkg/authz` — Authentication/authorization
- `pkg/audit` — Audit logging
- `pkg/events` — Event system

## Example
```go
// ...see examples/basic/main.go for a full example
```

## Extending & Customizing
- Implement your own token store, audit backend, or event handler by following the interfaces in each package.

---
For more, see the main `README.md` and package-level `doc.go` files.
