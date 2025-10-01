# GAuth Library Usage Guide

This document provides a concise overview for developers who want to use GAuth as a library in their own Go projects.

---

**Type Safety & Legacy Helpers**
- All public APIs are type-safe and use explicit, strongly-typed structures.
- Legacy helpers using `map[string]interface{}` are marked as deprecated and should not be used in new code.

---


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
   - See runnable, up-to-date examples in `examples/` and `examples/<category>/cmd/`.


## Package Structure
- `pkg/gauth` — Main entry point for service usage
- `pkg/token` — Token management
- `pkg/tokenstore` — Token storage interfaces and implementations
- `pkg/auth` / `pkg/authz` — Authentication/authorization
- `pkg/audit` — Audit logging
- `pkg/events` — Event system
- `examples/` — All runnable examples, now isolated and up-to-date with the latest API


## Example
See `examples/basic/main.go` or any `examples/<category>/cmd/main.go` for a full, runnable example using the latest API.


## Extending & Customizing
- Implement your own token store by following the `tokenstore.Store` interface.
- Add custom event handlers for audit or integration.
- See package-level `doc.go` files for each package for more details.


---


## Migration & Breaking Changes (2025)

- All example logic with a `main` function is now in its own `main.go` under `examples/` or `examples/<category>/cmd/`.
- All obsolete and duplicate example files have been removed.
- All examples are refactored for the new API and type-safe signatures.
- All public APIs are type-safe (no public map[string]interface{}).
- Rate limiting is now per-user (OwnerID) and per-client, using the OwnerID field of the token as the subject for rate limiting.
- Legacy helpers and types are deprecated or removed; use new type-safe alternatives.
- See `docs/IMPROVEMENTS.md` for a summary of codebase modernization.

For more, see the main `README.md`, [MANUAL_TESTING.md](./MANUAL_TESTING.md), and package-level `doc.go` files.
