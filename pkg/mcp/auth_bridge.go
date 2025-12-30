// Package mcp - Authorization Bridge for AgentAuth + MCP Integration
// Maps AgentAuth Extended Tokens to MCP resource/tool permissions via PDP
package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/pdp"
)

// AuthorizationBridge maps AgentAuth Extended Tokens to MCP permissions
// This is the core integration point between AgentAuth authorization and MCP operations
type AuthorizationBridge struct {
	pdpEngine pdp.Engine
}

// NewAuthorizationBridge creates a new authorization bridge
func NewAuthorizationBridge(pdpEngine pdp.Engine) *AuthorizationBridge {
	return &AuthorizationBridge{
		pdpEngine: pdpEngine,
	}
}

// AuthorizeResourceRead checks if token authorizes reading an MCP resource
func (b *AuthorizationBridge) AuthorizeResourceRead(
	ctx context.Context,
	token *agentauth.ExtendedToken,
	resourceURI string,
) (bool, error) {
	// Validate token first
	if err := token.Validate(); err != nil {
		return false, fmt.Errorf("token validation failed: %w", err)
	}

	// Check token expiration
	if time.Now().After(token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)) {
		return false, fmt.Errorf("token expired")
	}

	// Check for required MCP scope
	if !b.hasScope(token, "mcp:resource:read") {
		// Check for resource-specific scope
		specificScope := fmt.Sprintf("mcp:resource:read:%s", extractResourceType(resourceURI))
		if !b.hasScope(token, specificScope) {
			return false, fmt.Errorf("token missing required scope: mcp:resource:read or %s", specificScope)
		}
	}

	// Build PDP request for policy evaluation
	pdpRequest := pdp.Request{
		Subject:  token.AuthorizationChain.Client.EntityID,
		Action:   "mcp:read_resource",
		Resource: resourceURI,
		Attributes: map[string]string{
			"token_id":         token.AccessToken,
			"client_id":        token.AuthorizationChain.Client.EntityID,
			"client_name":      token.AuthorizationChain.Client.EntityName,
			"client_type":      token.AuthorizationChain.Client.EntityType,
			"client_owner_id":  token.ClientOwner.OwnerID,
			"authorizer_id":    token.OwnersAuthorizer.AuthorizerID,
			"resource_uri":     resourceURI,
			"resource_type":    extractResourceType(resourceURI),
			"jurisdiction":     token.JurisdictionContext.PrimaryJurisdiction,
			"compliance_level": token.ComplianceLevel,
			"agent_id":         token.AuthorizationChain.Client.EntityID,
			"agent_name":       token.AuthorizationChain.Client.EntityName,
			"agent_assurance":  b.extractAgentAssurance(token),
		},
		Time: time.Now(),
	}

	// Evaluate policy through PDP
	decision, err := b.pdpEngine.Evaluate(ctx, pdpRequest)
	if err != nil {
		return false, fmt.Errorf("PDP evaluation failed: %w", err)
	}

	if !decision.Allow {
		return false, fmt.Errorf("access denied by policy: %s", decision.Reason)
	}

	return true, nil
}

// AuthorizeToolCall checks if token authorizes calling an MCP tool
func (b *AuthorizationBridge) AuthorizeToolCall(
	ctx context.Context,
	token *agentauth.ExtendedToken,
	toolName string,
	arguments map[string]interface{},
) (bool, error) {
	// Validate token first
	if err := token.Validate(); err != nil {
		return false, fmt.Errorf("token validation failed: %w", err)
	}

	// Check token expiration
	if time.Now().After(token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)) {
		return false, fmt.Errorf("token expired")
	}

	// Check for required MCP scope
	if !b.hasScope(token, "mcp:tool:call") {
		// Check for tool-specific scope
		specificScope := fmt.Sprintf("mcp:tool:call:%s", toolName)
		if !b.hasScope(token, specificScope) {
			return false, fmt.Errorf("token missing required scope: mcp:tool:call or %s", specificScope)
		}
	}

	// Check for restrictions on tool usage
	if err := b.checkToolRestrictions(token, toolName, arguments); err != nil {
		return false, fmt.Errorf("tool restriction violation: %w", err)
	}

	// Update PDP request with argument-specific context if applicable
	monetaryValue, hasValue := extractMonetaryValue(arguments)

	// Build PDP request for policy evaluation
	pdpRequest := pdp.Request{
		Subject:  token.AuthorizationChain.Client.EntityID,
		Action:   "mcp:call_tool",
		Resource: fmt.Sprintf("mcp:tool:%s", toolName),
		Attributes: map[string]string{
			"token_id":         token.AccessToken,
			"client_id":        token.AuthorizationChain.Client.EntityID,
			"client_name":      token.AuthorizationChain.Client.EntityName,
			"client_type":      token.AuthorizationChain.Client.EntityType,
			"client_owner_id":  token.ClientOwner.OwnerID,
			"authorizer_id":    token.OwnersAuthorizer.AuthorizerID,
			"tool_name":        toolName,
			"jurisdiction":     token.JurisdictionContext.PrimaryJurisdiction,
			"compliance_level": token.ComplianceLevel,
			"agent_id":         token.AuthorizationChain.Client.EntityID,
			"agent_name":       token.AuthorizationChain.Client.EntityName,
			"agent_assurance":  b.extractAgentAssurance(token),
			"argument_count":   fmt.Sprintf("%d", len(arguments)),
		},
		Time: time.Now(),
	}

	if hasValue {
		pdpRequest.Attributes["monetary_value"] = fmt.Sprintf("%.2f", monetaryValue)
	}

	// Evaluate policy through PDP
	decision, err := b.pdpEngine.Evaluate(ctx, pdpRequest)
	if err != nil {
		return false, fmt.Errorf("PDP evaluation failed: %w", err)
	}

	if !decision.Allow {
		return false, fmt.Errorf("access denied by policy: %s", decision.Reason)
	}

	return true, nil
}

// AuthorizePromptGet checks if token authorizes accessing an MCP prompt template
func (b *AuthorizationBridge) AuthorizePromptGet(
	ctx context.Context,
	token *agentauth.ExtendedToken,
	promptName string,
) (bool, error) {
	// Validate token first
	if err := token.Validate(); err != nil {
		return false, fmt.Errorf("token validation failed: %w", err)
	}

	// Check token expiration
	if time.Now().After(token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)) {
		return false, fmt.Errorf("token expired")
	}

	// Check for required MCP scope
	if !b.hasScope(token, "mcp:prompt:get") {
		// Check for prompt-specific scope
		specificScope := fmt.Sprintf("mcp:prompt:get:%s", promptName)
		if !b.hasScope(token, specificScope) {
			return false, fmt.Errorf("token missing required scope: mcp:prompt:get or %s", specificScope)
		}
	}

	// Build PDP request for policy evaluation
	pdpRequest := pdp.Request{
		Subject:  token.AuthorizationChain.Client.EntityID,
		Action:   "mcp:get_prompt",
		Resource: fmt.Sprintf("mcp:prompt:%s", promptName),
		Attributes: map[string]string{
			"token_id":         token.AccessToken,
			"client_id":        token.AuthorizationChain.Client.EntityID,
			"client_name":      token.AuthorizationChain.Client.EntityName,
			"client_type":      token.AuthorizationChain.Client.EntityType,
			"client_owner_id":  token.ClientOwner.OwnerID,
			"authorizer_id":    token.OwnersAuthorizer.AuthorizerID,
			"prompt_name":      promptName,
			"jurisdiction":     token.JurisdictionContext.PrimaryJurisdiction,
			"compliance_level": token.ComplianceLevel,
			"agent_id":         token.AuthorizationChain.Client.EntityID,
			"agent_name":       token.AuthorizationChain.Client.EntityName,
			"agent_assurance":  b.extractAgentAssurance(token),
		},
		Time: time.Now(),
	}

	// Evaluate policy through PDP
	decision, err := b.pdpEngine.Evaluate(ctx, pdpRequest)
	if err != nil {
		return false, fmt.Errorf("PDP evaluation failed: %w", err)
	}

	if !decision.Allow {
		return false, fmt.Errorf("access denied by policy: %s", decision.Reason)
	}

	return true, nil
}

// hasScope checks if token contains required scope (with wildcard support)
func (b *AuthorizationBridge) hasScope(token *agentauth.ExtendedToken, requiredScope string) bool {
	for _, scope := range token.Scope {
		if scope == requiredScope {
			return true
		}
		// Check wildcard scopes (e.g., "mcp:*" allows all MCP operations)
		if strings.HasSuffix(scope, ":*") {
			prefix := strings.TrimSuffix(scope, "*")
			if strings.HasPrefix(requiredScope, prefix) {
				return true
			}
		}
	}
	return false
}

// ExtractMCPScopes extracts all MCP-related scopes from token
func (b *AuthorizationBridge) ExtractMCPScopes(token *agentauth.ExtendedToken) []string {
	mcpScopes := make([]string, 0)
	for _, scope := range token.Scope {
		if strings.HasPrefix(scope, "mcp:") {
			mcpScopes = append(mcpScopes, scope)
		}
	}
	return mcpScopes
}

// ValidateMCPScopes checks if token has valid MCP scopes
func (b *AuthorizationBridge) ValidateMCPScopes(token *agentauth.ExtendedToken) error {
	mcpScopes := b.ExtractMCPScopes(token)
	if len(mcpScopes) == 0 {
		return fmt.Errorf("token contains no MCP scopes")
	}

	// Validate scope format
	validPrefixes := []string{
		"mcp:resource:",
		"mcp:tool:",
		"mcp:prompt:",
		"mcp:sampling:",
		"mcp:*", // Wildcard
	}

	for _, scope := range mcpScopes {
		valid := false
		for _, prefix := range validPrefixes {
			if strings.HasPrefix(scope, prefix) || scope == "mcp:*" {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid MCP scope format: %s", scope)
		}
	}

	return nil
}

// checkToolRestrictions validates tool call against token restrictions
func (b *AuthorizationBridge) checkToolRestrictions(
	token *agentauth.ExtendedToken,
	toolName string,
	arguments map[string]interface{},
) error {
	// Check value restrictions if tool involves monetary transactions
	if isMonetaryTool(toolName) {
		if err := b.checkValueRestrictions(token, arguments); err != nil {
			return err
		}
	}

	// Check scope restrictions
	for _, restriction := range token.Restrictions {
		if restriction.RestrictionType == "scope_limit" && restriction.EnforcementLevel == "mandatory" {
			// Check if tool is in allowed scope
			if len(restriction.Scope) > 0 {
				allowed := false
				for _, allowedScope := range restriction.Scope {
					if strings.Contains(toolName, allowedScope) {
						allowed = true
						break
					}
				}
				if !allowed {
					return fmt.Errorf("tool %s not in allowed scope", toolName)
				}
			}
		}

		// Check time restrictions
		if restriction.RestrictionType == "time_limit" && restriction.EnforcementLevel == "mandatory" {
			now := time.Now()
			// Validate based on restriction value structure
			// (implementation depends on time restriction format)
			_ = now // Placeholder for time validation logic
		}
	}

	return nil
}

// checkValueRestrictions validates monetary value against token restrictions
func (b *AuthorizationBridge) checkValueRestrictions(
	token *agentauth.ExtendedToken,
	arguments map[string]interface{},
) error {
	// Extract value from arguments
	value, hasValue := extractMonetaryValue(arguments)
	if !hasValue {
		return nil // No value restriction applicable
	}

	// Check against token value restrictions
	for _, restriction := range token.Restrictions {
		if restriction.RestrictionType == "value_limit" && restriction.EnforcementLevel == "mandatory" {
			if limitValue, ok := restriction.Value.(float64); ok {
				if value > limitValue {
					return fmt.Errorf("value %.2f exceeds limit %.2f", value, limitValue)
				}
			}
		}
	}

	return nil
}

// extractResourceType extracts resource type from URI (e.g., "file", "db", "api")
func extractResourceType(uri string) string {
	// Parse URI scheme
	if strings.HasPrefix(uri, "file://") {
		return "file"
	} else if strings.HasPrefix(uri, "db://") || strings.HasPrefix(uri, "postgres://") || strings.HasPrefix(uri, "mysql://") {
		return "database"
	} else if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return "http"
	} else if strings.HasPrefix(uri, "mcp://") {
		return "mcp"
	}
	return "unknown"
}

// isMonetaryTool checks if tool involves monetary transactions
func isMonetaryTool(toolName string) bool {
	monetaryKeywords := []string{"payment", "transfer", "transaction", "invoice", "purchase"}
	toolLower := strings.ToLower(toolName)
	for _, keyword := range monetaryKeywords {
		if strings.Contains(toolLower, keyword) {
			return true
		}
	}
	return false
}

// extractMonetaryValue extracts monetary value from tool arguments
func extractMonetaryValue(arguments map[string]interface{}) (float64, bool) {
	// Common field names for monetary values
	valueFields := []string{"amount", "value", "price", "cost", "total"}

	for _, field := range valueFields {
		if val, ok := arguments[field]; ok {
			switch v := val.(type) {
			case float64:
				return v, true
			case float32:
				return float64(v), true
			case int:
				return float64(v), true
			case int64:
				return float64(v), true
			}
		}
	}
	return 0, false
}

// AuthorizationResult represents the result of an authorization check
type AuthorizationResult struct {
	Allowed      bool
	Reason       string
	Restrictions []string
	Obligations  []string
	Timestamp    time.Time
	TokenID      string
	ClientID     string
	Decision     string
}

// AuthorizeWithDetails performs authorization and returns detailed result
func (b *AuthorizationBridge) AuthorizeWithDetails(
	ctx context.Context,
	token *agentauth.ExtendedToken,
	operation string,
	resourceOrTool string,
	arguments map[string]interface{},
) (*AuthorizationResult, error) {
	result := &AuthorizationResult{
		Timestamp: time.Now(),
		TokenID:   token.AccessToken,
		ClientID:  token.AuthorizationChain.Client.EntityID,
	}

	// Validate token
	if err := token.Validate(); err != nil {
		result.Allowed = false
		result.Reason = fmt.Sprintf("token validation failed: %v", err)
		return result, nil
	}

	// Check expiration
	if time.Now().After(token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)) {
		result.Allowed = false
		result.Reason = "token expired"
		return result, nil
	}

	// Build PDP request based on operation
	pdpRequest := pdp.Request{
		Subject:  token.AuthorizationChain.Client.EntityID,
		Action:   operation,
		Resource: resourceOrTool,
		Attributes: map[string]string{
			"token_id":         token.AccessToken,
			"client_id":        token.AuthorizationChain.Client.EntityID,
			"client_owner_id":  token.ClientOwner.OwnerID,
			"authorizer_id":    token.OwnersAuthorizer.AuthorizerID,
			"jurisdiction":     token.JurisdictionContext.PrimaryJurisdiction,
			"compliance_level": token.ComplianceLevel,
			"agent_id":         token.AuthorizationChain.Client.EntityID,
			"agent_name":       token.AuthorizationChain.Client.EntityName,
			"agent_assurance":  b.extractAgentAssurance(token),
		},
		Time: time.Now(),
	}

	// Evaluate policy
	decision, err := b.pdpEngine.Evaluate(ctx, pdpRequest)
	if err != nil {
		return nil, fmt.Errorf("PDP evaluation failed: %w", err)
	}

	result.Allowed = decision.Allow
	result.Reason = decision.Reason
	if decision.Allow {
		result.Decision = "permit"
	} else {
		result.Decision = "deny"
	}

	if len(decision.Obligations) > 0 {
		// Convert obligations to string array
		result.Obligations = make([]string, len(decision.Obligations))
		for i, obligation := range decision.Obligations {
			result.Obligations[i] = obligation.ID
		}
	}

	return result, nil
}

// extractAgentAssurance retrieves the identity assurance level of the agent (client)
func (b *AuthorizationBridge) extractAgentAssurance(token *agentauth.ExtendedToken) string {
	if token.VerificationProof == nil {
		return "none"
	}
	agentID := token.AuthorizationChain.Client.EntityID
	for _, level := range token.VerificationProof.VerificationLevels {
		if level.EntityID == agentID && level.AssuranceLevel != "" {
			return level.AssuranceLevel
		}
	}
	return "none"
}
