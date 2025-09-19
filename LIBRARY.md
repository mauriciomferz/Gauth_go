# GAuth Go Library: Usage & API Guide

Welcome to the GAuth Go library! This document is your entry point for understanding, using, and extending the core GAuth protocol implementation. It is designed for developers of all backgrounds, including those new to Go.

## What is GAuth?
GAuth is a centralized, auditable authorization protocol (GiFo-RfC 0111) designed for secure, type-safe, and extensible access control. This library provides a reference implementation in Go.

## Quick Start
- See `examples/` for runnable demos and integration patterns.
- Library code is in `pkg/gauth/` and related subpackages.
- All protocol boundaries are clearly annotated with `[GAuth]` comments.

## Key Packages
- `pkg/gauth/` — Core protocol logic, types, and flows ([GAuth])
- `pkg/token/`, `pkg/audit/`, `pkg/ratelimit/` — Supporting components ([GAuth])
- `examples/` — Usage demos, not for production

## How to Use
1. Import the library:
   ```go
   import "github.com/mauriciomferz/Gauth_go/pkg/gauth"
   ```
2. Initialize and configure your GAuth service or client.
3. Use strongly-typed requests and responses (see `types.go`).
4. See `README.md` and this file for more details.

## Extending GAuth
- Add new grant types, token types, or audit event types by extending the relevant structs and interfaces.
- Follow the `[GAuth]` annotation pattern for protocol logic.

## Manual Testing & Demos
- See `MANUAL_TESTING.md` for runtime usage suggestions.
- Run `go run examples/resilient/cascading.go` for a protocol-compliant simulation.

## Contributing
- Please see `CONTRIBUTING.md` for guidelines.
- All contributions should maintain clear protocol boundaries and strong typing.

## Support
- Issues and questions: open a GitHub issue or discussion.

---

For more, see the package-level `doc.go` files and inline code comments throughout the library.
