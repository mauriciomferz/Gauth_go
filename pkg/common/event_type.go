// Package common/event_type.go: RFC111 Compliance Mapping
//
// This file implements the shared event type enum as required by RFC111:
//   - Type-safe event type enum for audit, protocol, and compliance events
//   - Used by audit, events, and service layers for all protocol steps
//
// Relevant RFC111 Sections:
//   - Section 6: How GAuth works (event, audit, compliance)
//   - Section 7: Benefits (verifiability, auditability)
//
// Compliance:
//   - All event types are enums/constants (no ambiguous types)
//   - Enum is used throughout the codebase for type safety and auditability
//   - No exclusions (Web3, DNA, decentralized auth) are present
//   - See README and docs/ for full protocol mapping
//
// License: Apache 2.0 (see LICENSE file)

// Package common provides shared types for the GAuth project.
package common

// EventType represents the type of audit event (moved from gauth to break import cycles).
type EventType int

const (
	EventAuthRequest EventType = iota
	EventAuthGrant
	EventTokenIssue
	EventTokenRevoke
	EventTransactionStart
	EventTransactionComplete
	EventTransactionFailed
	EventRateLimited
)

func (e EventType) String() string {
	switch e {
	case EventAuthRequest:
		return "auth_request"
	case EventAuthGrant:
		return "auth_grant"
	case EventTokenIssue:
		return "token_issue"
	case EventTokenRevoke:
		return "token_revoke"
	case EventTransactionStart:
		return "transaction_start"
	case EventTransactionComplete:
		return "transaction_complete"
	case EventTransactionFailed:
		return "transaction_failed"
	case EventRateLimited:
		return "rate_limited"
	default:
		return "unknown"
	}
}
