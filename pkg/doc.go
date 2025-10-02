// Package pkg provides a modular, type-safe authentication, authorization, and event system for Go.
//
// Copyright (c) 2025 Gimel Foundation gGmbH i.G.
// Licensed under Apache 2.0
//
// Gimel Foundation gGmbH i.G., www.GimelFoundation.com
// Operated by Gimel Technologies GmbH
// MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert
// Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com
//
// This implementation follows GiFo-RFC-0115 Power-of-Attorney Credential Definition (PoA-Definition)
// standard as published by the Gimel Foundation.
//
// # Overview
//
// GAuth is designed for extensibility, security, and ease of integration. It provides:
//   - Strongly-typed tokens and claims
//   - Pluggable authentication and authorization
//   - Typed event/audit system
//   - Modular, reusable packages
//   - RFC-0115 compliant PoA-Definition structures
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
