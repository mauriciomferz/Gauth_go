// Package rfc0111 provides HTTP handlers for RFC-0111 authorization flows.
package rfc0111

import (
	"net/http"
	"strings"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
	"github.com/gin-gonic/gin"
)

// AuthorizationHandlers encapsulates RFC-0111 authorization API handlers.
type AuthorizationHandlers struct {
	gauthService *gauth.Service
	tokenStore   gauth.ExtendedTokenStore
}

// NewAuthorizationHandlers creates a new authorization handlers instance.
func NewAuthorizationHandlers(service *gauth.Service, tokenStore gauth.ExtendedTokenStore) *AuthorizationHandlers {
	return &AuthorizationHandlers{
		gauthService: service,
		tokenStore:   tokenStore,
	}
}

// RequestToken handles POST /api/v1/rfc0111/authorize
// RFC-0111 Steps a-i: Complete RFC-compliant authorization flow
func (h *AuthorizationHandlers) RequestToken(c *gin.Context) {
	var req struct {
		ClientID         string                 `json:"client_id" binding:"required"`
		ClientType       string                 `json:"client_type,omitempty"`
		ClientVersion    string                 `json:"client_version,omitempty"`
		SubscriptionID   string                 `json:"subscription_id" binding:"required"`
		ResourceOwnerID  string                 `json:"resource_owner_id" binding:"required"`
		PoACredentialRef string                 `json:"poa_credential_ref" binding:"required"`
		Scope            string                 `json:"scope" binding:"required"`
		Context          map[string]interface{} `json:"context,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": err.Error(),
		})
		return
	}

	// Execute RFC-compliant authorization flow
	// Parse scope string into basic AuthorizationScope
	// For simple "read write" scopes, create a basic scope with standard actions
	requestedScope := parseBasicScope(req.Scope)
	
	response, err := h.gauthService.RequestTokenRFC(c.Request.Context(), &gauth.RFCCompliantAuthorizationRequest{
		ClientID:         req.ClientID,
		ResourceOwnerID:  req.ResourceOwnerID,
		SubscriptionID:   req.SubscriptionID,
		PoACredentialRef: req.PoACredentialRef,
		RequestedScope:   requestedScope,
		Context:          req.Context,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "authorization_failed",
			"error_description": err.Error(),
		})
		return
	}

	// Store the extended token if token store is configured
	if h.tokenStore != nil && response.ExtendedToken != nil {
		if err := h.tokenStore.SaveToken(c.Request.Context(), response.ExtendedToken); err != nil {
			// Log error but don't fail the request
			// Token is still returned to client even if storage fails
			c.Header("X-Token-Storage-Warning", "Token storage failed: "+err.Error())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"token_type":        response.TokenType,
		"expires_in":        response.ExpiresIn,
		"scope":             response.Scope,
		"extended_token":    response.ExtendedToken,
		"compliance_status": response.ComplianceStatus,
	})
}

// ValidateToken handles POST /api/v1/rfc0111/token/validate
// Validates an RFC-0111 compliant token
func (h *AuthorizationHandlers) ValidateToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Validate token using existing service
	result, err := h.gauthService.ValidateToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"valid":   false,
			"active":  false,
			"message": "Token validation failed: " + err.Error(),
		})
		return
	}

	// Return validation result
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"valid":     result.Valid,
		"active":    result.Valid,
		"client_id": result.ClientID,
		"scope":     result.Scope,
	})
}

// IntrospectToken handles POST /api/v1/rfc0111/token/introspect
// RFC 7662 compatible introspection endpoint
func (h *AuthorizationHandlers) IntrospectToken(c *gin.Context) {
	var req struct {
		Token         string `json:"token" binding:"required"`
		TokenTypeHint string `json:"token_type_hint,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// RFC 7662 compliant token introspection
	result, err := h.gauthService.ValidateToken(req.Token)
	
	// Per RFC 7662, return active: false for invalid tokens rather than error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"active": false,
		})
		return
	}

	// Return full introspection response for valid token
	c.JSON(http.StatusOK, gin.H{
		"active":     true,
		"scope":      result.Scope,
		"client_id":  result.ClientID,
		"token_type": "Bearer",
		"sub":        result.ClientID,
	})
}

// RevokeToken handles POST /api/v1/rfc0111/token/revoke
// RFC 7009 compatible revocation endpoint
func (h *AuthorizationHandlers) RevokeToken(c *gin.Context) {
	var req struct {
		Token         string `json:"token" binding:"required"`
		TokenTypeHint string `json:"token_type_hint,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// RFC 7009: Revocation endpoint MUST return 200 OK whether or not token existed
	// Token revocation should be idempotent
	
	// TODO: Implement actual revocation in token store
	// For now, acknowledge the revocation request per RFC 7009 spec
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token revocation requested",
	})
}

// parseBasicScope converts a simple scope string (e.g., "read write") into an AuthorizationScope
// This is a basic implementation that handles common OAuth-style scopes.
// parseBasicScope converts OAuth-style scope strings to RFC-0111 AuthorizationScope.
// Maps common OAuth scopes (read, write, delete, etc.) to RFC-0111 action types.
//
// For full RFC-0111 compliance with complex authorization types, sectors, and regions,
// clients should use the full PoA credential specification with proper action types.
func parseBasicScope(scopeString string) *poa.AuthorizationScope {
	if scopeString == "" {
		return nil
	}
	
	// Parse space-separated OAuth scopes
	scopes := strings.Fields(scopeString)
	if len(scopes) == 0 {
		return nil
	}
	
	// Map OAuth scopes to RFC-0111 non-physical actions (use predefined constants)
	var nonPhysicalActions []poa.ActionTypeNonPhysical
	for _, s := range scopes {
		switch strings.ToLower(s) {
		case "read":
			nonPhysicalActions = append(nonPhysicalActions, poa.ActionNonPhysicalAnalyzing)
		case "write":
			nonPhysicalActions = append(nonPhysicalActions, poa.ActionNonPhysicalDocumenting)
		case "delete":
			nonPhysicalActions = append(nonPhysicalActions, poa.ActionNonPhysicalApproving)
		case "admin":
			nonPhysicalActions = append(nonPhysicalActions, poa.ActionNonPhysicalApproving)
		}
	}
	
	// If no recognized scopes, add analyzing as a safe default
	if len(nonPhysicalActions) == 0 {
		nonPhysicalActions = append(nonPhysicalActions, poa.ActionNonPhysicalAnalyzing)
	}
	
	scope := &poa.AuthorizationScope{
		AuthorizationType: poa.AuthorizationType{
			RepresentationType: "direct",
			Restrictions:       []string{},
			SubProxyAuthority:  false,
			SignatureType:      "qualified",
		},
		ApplicableSectors: []poa.IndustrySector{},
		ApplicableRegions: []poa.GeographicScope{},
		AuthorizedActions: poa.AuthorizedActions{
			Transactions:       []poa.TransactionType{},
			Decisions:          []poa.DecisionType{},
			PhysicalActions:    []poa.ActionTypePhysical{},
			NonPhysicalActions: nonPhysicalActions,
		},
	}
	
	return scope
}
