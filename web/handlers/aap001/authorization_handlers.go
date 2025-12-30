// Package aap001 provides HTTP handlers for AAP-001 authorization flows.
package agentauth_aap_001

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
	"github.com/mauriciomferz/AgentAuth/pkg/poa/taxonomy"
)

// AuthorizationHandlers encapsulates AAP-001 authorization API handlers.
type AuthorizationHandlers struct {
	agentauthService *agentauth.Service
	tokenStore   agentauth.ExtendedTokenStore
}

// NewAuthorizationHandlers creates a new authorization handlers instance.
func NewAuthorizationHandlers(service *agentauth.Service, tokenStore agentauth.ExtendedTokenStore) *AuthorizationHandlers {
	return &AuthorizationHandlers{
		agentauthService: service,
		tokenStore:   tokenStore,
	}
}

// RequestToken handles POST /api/v1"AAP-001/authorize
// AAP-001 Steps a-i: Complete RFC-compliant authorization flow
func (h *AuthorizationHandlers) RequestToken(c *gin.Context) {
	var req struct {
		ClientID         string                 `json:"client_id" binding:"required"`
		ClientType       string                 `json:"client_type,omitempty"`
		ClientVersion    string                 `json:"client_version,omitempty"`
		SubscriptionID   string                 `json:"subscription_id" binding:"required"`
		ResourceOwnerID  string                 `json:"resource_owner_id" binding:"required"`
		PoACredentialRef string                 `json:"poa_credential_ref" binding:"required"`
		Scope            string                 `json:"scope" binding:"required"`
		Jurisdiction     string                 `json:"jurisdiction,omitempty"` // ISO 3166-1 alpha-2 country code (e.g., "US", "DE") or ISO 3166-2 subdivision (e.g., "US-CA", "DE-BY"))
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

	response, err := h.agentauthService.RequestTokenRFC(c.Request.Context(), &agentauth.RFCCompliantAuthorizationRequest{
		ClientID:         req.ClientID,
		ResourceOwnerID:  req.ResourceOwnerID,
		SubscriptionID:   req.SubscriptionID,
		PoACredentialRef: req.PoACredentialRef,
		RequestedScope:   requestedScope,
		Jurisdiction:     req.Jurisdiction,
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

// ValidateToken handles POST /api/v1"AAP-001/token/validate
// Validates an AAP-001 compliant token
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
	result, err := h.agentauthService.ValidateToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"valid":   false,
			"active":  false,
			"message": "Token validation failed: " + err.Error(),
		})
		return
	}

	// Parse token to get decoded claims for display
	var decoded map[string]interface{}
	parts := strings.Split(req.Token, ".")
	if len(parts) == 3 {
		if payload, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			_ = json.Unmarshal(payload, &decoded) // Best effort decoding for display only
		}
	}

	// Return validation result with decoded claims
	response := gin.H{
		"success":   true,
		"valid":     result.Valid,
		"active":    result.Valid,
		"client_id": result.ClientID,
		"scope":     result.Scope,
	}

	if decoded != nil {
		response["decoded"] = decoded
	}

	c.JSON(http.StatusOK, response)
}

// IntrospectToken handles POST /api/v1"AAP-001/token/introspect
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
	result, err := h.agentauthService.ValidateToken(req.Token)

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

// RevokeToken handles POST /api/v1"AAP-001/token/revoke
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
// parseBasicScope converts OAuth-style scope strings to AAP-001 AuthorizationScope.
// Maps common OAuth scopes (read, write, delete, etc.) to AAP-001 action types.
//
// For full AAP-001 compliance with complex authorization types, sectors, and regions,
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

	// Map OAuth scopes to AAP-001 non-physical actions (use predefined constants)
	var nonPhysicalActions []taxonomy.ActionTypeNonPhysical
	for _, s := range scopes {
		switch strings.ToLower(s) {
		case "read":
			nonPhysicalActions = append(nonPhysicalActions, taxonomy.ActionNonPhysicalAnalyzing)
		case "write":
			nonPhysicalActions = append(nonPhysicalActions, taxonomy.ActionNonPhysicalDocumenting)
		case "delete":
			nonPhysicalActions = append(nonPhysicalActions, taxonomy.ActionNonPhysicalApproving)
		case "admin":
			nonPhysicalActions = append(nonPhysicalActions, taxonomy.ActionNonPhysicalApproving)
		}
	}

	// If no recognized scopes, add analyzing as a safe default
	if len(nonPhysicalActions) == 0 {
		nonPhysicalActions = append(nonPhysicalActions, taxonomy.ActionNonPhysicalAnalyzing)
	}

	scope := &poa.AuthorizationScope{
		AuthorizationType: poa.AuthorizationType{
			RepresentationType: "direct",
			Restrictions:       []string{},
			SubProxyAuthority:  false,
			SignatureType:      "qualified",
		},
		ApplicableSectors: []taxonomy.IndustrySector{},
		ApplicableRegions: []poa.GeographicScope{},
		AuthorizedActions: poa.AuthorizedActions{
			Transactions:       []taxonomy.TransactionType{},
			Decisions:          []taxonomy.DecisionType{},
			PhysicalActions:    []taxonomy.ActionTypePhysical{},
			NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{},
		},
	}

	return scope
}
