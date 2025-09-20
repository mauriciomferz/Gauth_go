// Copyright (c) 2025 Gimel Foundation and the persons identified as the document authors.
// All rights reserved. This file is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents.
// See http://GimelFoundation.com or https://github.com/Gimel-Foundation for details.
// Code Components extracted from GiFo-RfC 0111 must include this license text and are provided without warranty.

// [GAuth] Audit types and interfaces for the GAuth protocol.
package gauth

import (
	"context"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
)

// AuditEventType is a string type for audit event types.
type AuditEventType string

// AuditAction is a string type for audit actions.
type AuditAction string

const (
	// AuditTypeAuthRequest is the event type for authorization requests.
	AuditTypeAuthRequest = "auth_request"
	// AuditActionInitiate is the action for initiating authorization.
	AuditActionInitiate  = "initiate_authorization"
)

// AuthRequestMetadata contains metadata for an authorization request audit event.
type AuthRequestMetadata struct {
	GrantID string
}

// AuditLogger defines the interface for pluggable audit logging.
// Implementations should be thread-safe.
type AuditLogger interface {
	// Log records an audit entry.
	Log(ctx context.Context, entry *audit.Entry)
	// GetRecentEvents returns the most recent audit events.
	GetRecentEvents(limit int) []audit.Event
}
