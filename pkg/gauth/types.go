// Copyright (c) 2025 Gimel Foundation and the persons identified as the document authors.
// All rights reserved. This file is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents.
// See http://GimelFoundation.com or https://github.com/Gimel-Foundation for details.
// Code Components extracted from GiFo-RfC 0111 must include this license text and are provided without warranty.
//
// GAuth Protocol Compliance: This file implements the GAuth protocol (GiFo-RfC 0111).
//
// Protocol Usage Declaration:
//   - GAuth protocol: IMPLEMENTED throughout this file (see [GAuth] comments below)
//   - OAuth 2.0:      NOT USED anywhere in this file
//   - PKCE:           NOT USED anywhere in this file
//   - OpenID:         NOT USED anywhere in this file
//
// [GAuth] = GAuth protocol logic (GiFo-RfC 0111)
// [Other] = Placeholder for OAuth2, OpenID, PKCE, or other protocols (none present in this file)
//
// [GAuth] Package gauth provides GAuth protocol types and requests.
package gauth

import (
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/common"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

// Config represents the configuration for GAuth

type Config struct {
	AuthServerURL     string                 // URL of the authorization server
	ClientID          string                 // Client identifier
	ClientSecret      string                 // Client secret
	Scopes            []string               // Default scopes
	RateLimit         common.RateLimitConfig // Rate limiting configuration
	AccessTokenExpiry time.Duration          // Token expiry duration
	TokenConfig       *token.Config          // Embedded token config (pointer)
}
