// Copyright (c) 2025 Gimel Foundation and the persons identified as the document authors.
// All rights reserved. This file is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents.
// See http://GimelFoundation.com or https://github.com/Gimel-Foundation for details.
// Code Components extracted from GiFo-RfC 0111 must include this license text and are provided without warranty.

// [GAuth] Request types for the GAuth protocol.
package gauth

import (
	"context"
)

// AuthorizationRequest represents a request to initiate authorization (delegation)
type AuthorizationRequest struct {
	ClientID string
	Scopes   []string
}

// TokenRequest represents a request for a token
type TokenRequest struct {
	GrantID      string
	Scope        []string
	Restrictions []Restriction
	Context      context.Context
}
