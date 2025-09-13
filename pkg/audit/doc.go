// Package audit provides a comprehensive audit logging system for tracking security events and user actions in the GAuth framework.
//
// # RFC111 Mapping
//
// This package implements the audit, compliance, and transparency requirements of GiFo-RfC 0111 (September 2025):
//   - All authorization and token actions are auditable and traceable.
//   - Strongly-typed audit events for power-of-attorney, delegation, revocation, and attestation.
//   - Centralized, verifiable audit trail for all protocol steps.
//   - No support for excluded features (Web3, DNA-based identity, AI-controlled audit logic).
//
// # Quick Start
//
//	import "github.com/Gimel-Foundation/gauth/pkg/audit"
//
//	evt := audit.NewEvent(audit.EventTypeGrant, "user123", "token456")
//	err := audit.DefaultLogger.Log(ctx, evt)
//	// ...
//
// See runnable examples in examples/audit/basic/main.go.
//
// # Key Types
//   - Event: Represents an audit event (grant, revoke, attest, etc.)
//   - Logger: Interface for audit event logging backends
//   - Store: Interface for persistent audit storage
//
// # Extension Points
//   - Implement custom Logger or Store for integration with external systems
//   - Add new event types for domain-specific auditing
//
// # See Also
//   - package token: for token lifecycle and revocation events
//   - package authz: for authorization decisions and policy enforcement
//   - package events: for event bus and event-driven integration
//
// For advanced usage and integration, see the examples/ directory and the project README.
package audit
