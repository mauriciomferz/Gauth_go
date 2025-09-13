// Package gauth provides the main entry point for the GAuth authentication and authorization framework.
//
// # Overview
//
// GAuth is a modular, type-safe authentication and authorization library for Go, designed for extensibility and compliance (RFC111).
//
// ## Features
//   - Token issuance and validation
//   - Authorization grants
//   - Audit logging
//   - Rate limiting
//   - Event-driven architecture
//
// ## Usage
//
//	import "github.com/Gimel-Foundation/gauth/pkg/gauth"
//	svc, err := gauth.New(gauth.Config{ /* ... */ })
//
// See LIBRARY.md and examples/ for more.
//
// # Extending
//
// Implement your own token store, audit backend, or event handler by following the interfaces in each package.
//
// # Compliance
//
// GAuth is designed to meet GiFo-RfC 0111 requirements for security, auditability, and modularity.
package gauth
