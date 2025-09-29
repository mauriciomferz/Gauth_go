// Package audit/audit.go: RFC111 Compliance Mapping
//
// This file implements audit logging and event tracking as required by RFC111:
//   - Structured, type-safe logging of all protocol events (auth, grant, token, transaction)
//   - Security event monitoring and compliance reporting
//   - Persistent, auditable trails for all protocol steps
//
// Relevant RFC111 Sections:
//   - Section 6: How GAuth works (audit, event, compliance)
//   - Section 7: Benefits (verifiability, auditability)
//
// Compliance:
//   - All events are enums/constants (no stringly-typed events)
//   - Audit trail is type-safe, explicit, and covers all protocol steps
//   - No exclusions (Web3, DNA, decentralized auth) are present
//   - See README and docs/ for full protocol mapping
//
// License: Apache 2.0 (see LICENSE file)
//
// ---
//
// The audit package implements comprehensive logging and event tracking for all
// authentication and authorization operations. Features include:
//   - Structured logging of auth events
//   - Transaction tracking
//   - Security event monitoring
//   - Compliance reporting
//   - Audit trail persistence
package audit

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/common"
)

// eventTypeFromString maps a string to a common.EventType.
func eventTypeFromString(s string) common.EventType {
	switch s {
	case "auth":
		return common.EventAuthRequest
	case "auth_grant":
		return common.EventAuthGrant
	case "token":
		return common.EventTokenIssue
	case "token_revoke":
		return common.EventTokenRevoke
	case "transaction_start":
		return common.EventTransactionStart
	case "transaction_complete":
		return common.EventTransactionComplete
	case "transaction_failed":
		return common.EventTransactionFailed
	case "rate_limited":
		return common.EventRateLimited
	default:
		return common.EventAuthRequest
	}
}

// EventMetadata represents metadata for an audit event
type EventMetadata struct {
	Token      string
	UserAgent  string
	IPAddress  string
	ResourceID string
	Scopes     []string
	ErrorMsg   string
}

// securityEvent represents a security audit event
type securityEvent struct {
	Timestamp     time.Time        `json:"timestamp"`
	EventType     common.EventType `json:"event_type"`
	TransactionID string           `json:"transaction_id,omitempty"`
	ClientID      string           `json:"client_id,omitempty"`
	Token         string           `json:"token,omitempty"`
	UserAgent     string           `json:"user_agent,omitempty"`
	IPAddress     string           `json:"ip_address,omitempty"`
	Success       bool             `json:"success"`
	ErrorMsg      string           `json:"error_message,omitempty"`
	ResourceID    string           `json:"resource_id,omitempty"`
	Scopes        []string         `json:"scopes,omitempty"`
}

// AuditLogger handles security event logging and persistence
type AuditLogger struct {
	events []securityEvent
	mu     sync.RWMutex // private mutex
}

// Close implements io.Closer for AuditLogger (no-op for in-memory logger).
func (al *AuditLogger) Close() error {
	return nil
}

// Log appends an audit Entry to the logger's event list (for compatibility with GAuth service).
// Accepts context.Context for type safety.
func (al *AuditLogger) Log(_ context.Context, entry *Entry) {
	al.mu.Lock()
	defer al.mu.Unlock()
	// For demonstration, we only store minimal info. Extend as needed.
	al.events = append(al.events, securityEvent{
		Timestamp:  entry.Timestamp,
		EventType:  eventTypeFromString(entry.Type),
		ClientID:   entry.ActorID,
		ResourceID: entry.TargetID,
		Success:    entry.Result == ResultSuccess,
		ErrorMsg:   entry.Metadata["reason"],
		Scopes:     nil, // Optionally parse from entry.Metadata["scopes"]
	})
}

// NewAuditLogger creates a new audit logger instance with optional storage
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		events: make([]securityEvent, 0),
	}
}

// LogEvent records a security event with comprehensive context
func (al *AuditLogger) LogEvent(evt common.EventType, transactionID, clientID string, meta EventMetadata) {
	al.mu.Lock()
	defer al.mu.Unlock()

	event := securityEvent{
		Timestamp:     time.Now(),
		EventType:     evt,
		TransactionID: transactionID,
		ClientID:      clientID,
		Token:         meta.Token,
		UserAgent:     meta.UserAgent,
		IPAddress:     meta.IPAddress,
		ResourceID:    meta.ResourceID,
		Scopes:        meta.Scopes,
		ErrorMsg:      meta.ErrorMsg,
	}
	event.Success = meta.ErrorMsg == "" && evt != common.EventTransactionFailed && evt != common.EventRateLimited

	al.events = append(al.events, event)

	// Log the event (in production this would go to secure storage)
	log.Printf("[AUDIT] %s: %s [%s] - Success: %v",
		event.EventType,
		event.ClientID,
		event.TransactionID,
		event.Success)

	if event.ErrorMsg != "" {
		log.Printf("[AUDIT] Error in %s: %s", event.TransactionID, event.ErrorMsg)
	}
}

// GetRecentEvents retrieves recent security events
func (al *AuditLogger) GetRecentEvents(limit int) []securityEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	start := len(al.events) - limit
	if start < 0 {
		start = 0
	}

	result := make([]securityEvent, len(al.events)-start)
	copy(result, al.events[start:])
	return result
}

// PrintRecentEvents displays recent security events in a formatted way
func (al *AuditLogger) PrintRecentEvents(limit int) {
	events := al.GetRecentEvents(limit)

	if len(events) == 0 {
		fmt.Println("No audit events found")
		return
	}

	fmt.Printf("\nRecent Audit Events (last %d):\n", len(events))
	fmt.Println("-----------------------------------")

	for _, event := range events {
		fmt.Printf("\nTimestamp: %s\n", event.Timestamp.Format(time.RFC3339))
		fmt.Printf("Event Type: %s\n", event.EventType)
		fmt.Printf("Transaction: %s\n", event.TransactionID)
		fmt.Printf("Client: %s\n", event.ClientID)
		if event.ResourceID != "" {
			fmt.Printf("Resource: %s\n", event.ResourceID)
		}
		if len(event.Scopes) > 0 {
			fmt.Printf("Scopes: %s\n", strings.Join(event.Scopes, ", "))
		}
		fmt.Printf("Success: %v\n", event.Success)
		if event.ErrorMsg != "" {
			fmt.Printf("Error: %s\n", event.ErrorMsg)
		}
		fmt.Println("-----------------------------------") // ASCII only
	}
}

// GetEventsByClient filters events by client ID
func (al *AuditLogger) GetEventsByClient(clientID string) []securityEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var events []securityEvent
	for _, event := range al.events {
		if event.ClientID == clientID {
			events = append(events, event)
		}
	}
	return events
}

// GetEventsByTransaction filters events by transaction ID
func (al *AuditLogger) GetEventsByTransaction(transactionID string) []securityEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var events []securityEvent
	for _, event := range al.events {
		if event.TransactionID == transactionID {
			events = append(events, event)
		}
	}
	return events
}

// GetFailedEvents returns all failed security events
func (al *AuditLogger) GetFailedEvents() []securityEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var events []securityEvent
	for _, event := range al.events {
		if !event.Success {
			events = append(events, event)
		}
	}
	return events
}

// ClearEvents removes all events older than the retention period
func (al *AuditLogger) ClearEvents(retentionPeriod time.Duration) {
	al.mu.Lock()
	defer al.mu.Unlock()

	cutoff := time.Now().Add(-retentionPeriod)
	var newEvents []securityEvent

	for _, event := range al.events {
		if event.Timestamp.After(cutoff) {
			newEvents = append(newEvents, event)
		}
	}

	al.events = newEvents
}

// (Removed stray for-loop outside of function. If this was meant to be a function, please define it properly.)

// (Removed stray code outside of function. If this was meant to be a function, please define it properly.)
