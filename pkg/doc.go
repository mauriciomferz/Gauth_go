// Package pkg provides a modular, type-safe authentication, authorization, and event system for Go.
//
// # Overview
//
// GAuth is designed for extensibility, security, and ease of integration. It provides:
//   - Strongly-typed tokens and claims
//   - Pluggable authentication and authorization
//   - Typed event/audit system
//   - Modular, reusable packages
//
// # Getting Started
//
// See the top-level README, GETTING_STARTED.md, and LIBRARY.md for onboarding and usage examples.
//
// # Packages
//
//   - auth   – Authentication primitives
//   - authz  – Authorization logic
//   - events – Typed event system
//   - token  – Token management
//   - audit  – Auditing utilities
//
// # Example
//
//	import "github.com/Gimel-Foundation/gauth/pkg/token"
//	claims := token.NewClaims()
//	// ...
//
// # Contributing
//
// Contributions are welcome! See CONTRIBUTING.md.
package pkg
