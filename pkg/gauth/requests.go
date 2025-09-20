// Copyright (c) 2025 Gimel Foundation and the persons identified as the document authors.
// All rights reserved. This file is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents.
// See http://GimelFoundation.com or https://github.com/Gimel-Foundation for details.
// Code Components extracted from GiFo-RfC 0111 must include this license text and are provided without warranty.

// [GAuth] Request types for the GAuth protocol.
package gauth

import (
	"context"
)

// AuthorizationRequest represents a request to initiate authorization (delegation).
// Used as input to GAuth.InitiateAuthorization. ClientID is the requesting client, Scopes are the requested permissions.
type AuthorizationRequest struct {
	ClientID string   // Unique client identifier
	Scopes   []string // Requested scopes/permissions
}

// TokenRequest represents a request for a token.
// Used as input to GAuth.RequestToken. GrantID is the authorization grant, Scope is the requested scope, Restrictions are optional constraints, and Context is for cancellation/deadline.
type TokenRequest struct {
	GrantID      string            // Authorization grant ID
	Scope        []string          // Requested scopes
	Restrictions []Restriction     // Optional restrictions (e.g., IP, time window)
	Context      context.Context   // Request context
}
