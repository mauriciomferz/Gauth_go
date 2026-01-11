// Package ai provides API endpoints for AI capability governance
// This file implements REST API endpoints for managing AI capability enforcement
package ai

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIHandler provides HTTP endpoints for AI capability management
type APIHandler struct {
	integration *ServerIntegration
}

// NewAPIHandler creates a new API handler for AI capabilities
func NewAPIHandler(integration *ServerIntegration) *APIHandler {
	return &APIHandler{
		integration: integration,
	}
}

// RegisterRoutes registers AI capability API routes with the Gin router
func (h *APIHandler) RegisterRoutes(router *gin.Engine) {
	ai := router.Group("/api/v1/ai")
	{
		// Status and configuration endpoints
		ai.GET("/capabilities/status", h.getAICapabilityStatus)
		ai.GET("/capabilities/entity-types", h.getSupportedEntityTypes)
		ai.GET("/capabilities/policies", h.getGovernancePolicies)
		ai.GET("/capabilities/policies/:policy_id", h.getGovernancePolicy)

		// Enforcement control endpoints
		ai.POST("/capabilities/enforcement/enable", h.enableEnforcement)
		ai.POST("/capabilities/enforcement/disable", h.disableEnforcement)
		ai.GET("/capabilities/enforcement/status", h.getEnforcementStatus)

		// AI profile validation and testing
		ai.POST("/capabilities/validate/profile", h.validateAIProfile)
		ai.POST("/capabilities/test/enforcement", h.testAIEnforcement)
		ai.POST("/capabilities/simulate/decision", h.simulateEnforcementDecision)

		// Entity rule management
		ai.GET("/capabilities/rules/:entity_type", h.getEntityRule)
		ai.PUT("/capabilities/rules/:entity_type", h.updateEntityRule)

		// Health check specific to AI capabilities
		ai.GET("/health", h.healthCheck)
	}
}

// getAICapabilityStatus returns overall AI capability enforcement status
func (h *APIHandler) getAICapabilityStatus(c *gin.Context) {
	status := h.integration.GetAICapabilityStatus()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  status,
	})
}

// getSupportedEntityTypes returns all supported AI entity types
func (h *APIHandler) getSupportedEntityTypes(c *gin.Context) {
	entityTypes := h.integration.GetSupportedEntityTypes()
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"entity_types": entityTypes,
	})
}

// getGovernancePolicies returns all loaded governance policies
func (h *APIHandler) getGovernancePolicies(c *gin.Context) {
	policies := h.integration.GetGovernancePolicies()

	// Filter sensitive information for API response
	publicPolicies := make([]map[string]any, len(policies))
	for i, policy := range policies {
		publicPolicies[i] = map[string]any{
			"policy_id":            policy.PolicyID,
			"jurisdiction":         policy.Jurisdiction,
			"industry_context":     policy.IndustryContext,
			"compliance_framework": policy.ComplianceFramework,
			"effective_date":       policy.EffectiveDate,
			"expiration_date":      policy.ExpirationDate,
			"last_updated":         policy.LastUpdated,
			"prohibited_actions":   policy.ProhibitedActions,
			"mandatory_claims":     policy.MandatoryClaims,
			"audit_requirements":   policy.AuditRequirements,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"policies": publicPolicies,
		"count":    len(publicPolicies),
	})
}

// getGovernancePolicy returns a specific governance policy
func (h *APIHandler) getGovernancePolicy(c *gin.Context) {
	policyID := c.Param("policy_id")
	if policyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "policy_id parameter is required",
		})
		return
	}

	policies := h.integration.GetGovernancePolicies()
	for _, policy := range policies {
		if policy.PolicyID == policyID {
			// Return detailed policy information
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"policy": map[string]any{
					"policy_id":            policy.PolicyID,
					"jurisdiction":         policy.Jurisdiction,
					"industry_context":     policy.IndustryContext,
					"compliance_framework": policy.ComplianceFramework,
					"entity_restrictions":  policy.EntityRestrictions,
					"prohibited_actions":   policy.ProhibitedActions,
					"mandatory_claims":     policy.MandatoryClaims,
					"audit_requirements":   policy.AuditRequirements,
					"effective_date":       policy.EffectiveDate,
					"expiration_date":      policy.ExpirationDate,
					"last_updated":         policy.LastUpdated,
				},
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error":   "policy not found",
	})
}

// enableEnforcement enables AI capability enforcement
func (h *APIHandler) enableEnforcement(c *gin.Context) {
	h.integration.EnableEnforcement(true)
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"enforcement_active": true,
		"message":            "AI capability enforcement enabled",
	})
}

// disableEnforcement disables AI capability enforcement
func (h *APIHandler) disableEnforcement(c *gin.Context) {
	h.integration.EnableEnforcement(false)
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"enforcement_active": false,
		"message":            "AI capability enforcement disabled",
	})
}

// getEnforcementStatus returns current enforcement status
func (h *APIHandler) getEnforcementStatus(c *gin.Context) {
	active := h.integration.IsEnforcementEnabled()
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"enforcement_active": active,
		"last_check":         time.Now().Format(time.RFC3339),
	})
}

// validateAIProfile validates an AI system profile
func (h *APIHandler) validateAIProfile(c *gin.Context) {
	var profile AISystemProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid AI profile JSON",
			"details": err.Error(),
		})
		return
	}

	errors := h.integration.ValidateAIProfile(profile)
	valid := len(errors) == 0

	response := gin.H{
		"success": true,
		"valid":   valid,
		"profile": profile,
	}

	if !valid {
		response["validation_errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}

// testAIEnforcement tests AI enforcement for a given profile and action
func (h *APIHandler) testAIEnforcement(c *gin.Context) {
	var request struct {
		Profile AISystemProfile `json:"profile"`
		Action  string          `json:"action"`
		Claims  map[string]any  `json:"claims"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request JSON",
			"details": err.Error(),
		})
		return
	}

	if request.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "action field is required",
		})
		return
	}

	// Add profile information to claims for testing
	if request.Claims == nil {
		request.Claims = make(map[string]any)
	}
	request.Claims["ai_entity_type"] = string(request.Profile.EntityType)
	request.Claims["system_id"] = request.Profile.SystemID
	request.Claims["jurisdiction"] = request.Profile.Jurisdiction
	request.Claims["risk_level"] = request.Profile.RiskLevel
	request.Claims["industry_context"] = request.Profile.IndustryContext

	// Perform enforcement test
	allowed, missing, metadata := h.integration.EnforceAICapabilities(request.Action, request.Claims)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"test_result": map[string]any{
			"allowed":              allowed,
			"missing_capabilities": missing,
			"metadata":             metadata,
			"profile":              request.Profile,
			"action":               request.Action,
			"test_timestamp":       time.Now().Format(time.RFC3339),
		},
	})
}

// simulateEnforcementDecision simulates an enforcement decision without side effects
func (h *APIHandler) simulateEnforcementDecision(c *gin.Context) {
	var request struct {
		EntityType      string         `json:"entity_type"`
		Action          string         `json:"action"`
		Jurisdiction    string         `json:"jurisdiction"`
		IndustryContext string         `json:"industry_context,omitempty"`
		RiskLevel       string         `json:"risk_level,omitempty"`
		Claims          map[string]any `json:"claims"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request JSON",
			"details": err.Error(),
		})
		return
	}

	// Create test profile
	profile := h.integration.CreateTestProfile(request.EntityType, request.Jurisdiction)
	if request.IndustryContext != "" {
		profile.IndustryContext = request.IndustryContext
	}
	if request.RiskLevel != "" {
		profile.RiskLevel = request.RiskLevel
	}

	// Get underlying matrix for direct access
	matrix := h.integration.GetAICapabilityMatrix()

	// Simulate enforcement decision
	decision := matrix.EnforceAICapabilities(profile, request.Action, request.Claims)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"simulation": map[string]any{
			"decision":             decision.Decision,
			"reason":               decision.Reason,
			"system_profile":       decision.SystemProfile,
			"requested_action":     decision.RequestedAction,
			"applied_policies":     decision.AppliedPolicies,
			"violated_rules":       decision.ViolatedRules,
			"missing_capabilities": decision.MissingCapabilities,
			"required_human_auth":  decision.RequiredHumanAuth,
			"audit_level":          decision.AuditLevel,
			"decision_id":          decision.DecisionID,
			"simulation_timestamp": decision.Timestamp.Format(time.RFC3339),
		},
	})
}

// getEntityRule returns the rule for a specific entity type
func (h *APIHandler) getEntityRule(c *gin.Context) {
	entityTypeStr := c.Param("entity_type")
	if entityTypeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "entity_type parameter is required",
		})
		return
	}

	entityType := AIEntityType(entityTypeStr)
	matrix := h.integration.GetAICapabilityMatrix()
	rule, exists := matrix.GetEntityRule(entityType)

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "entity type not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"entity_type": entityTypeStr,
		"rule":        rule,
	})
}

// updateEntityRule updates the rule for a specific entity type
func (h *APIHandler) updateEntityRule(c *gin.Context) {
	entityTypeStr := c.Param("entity_type")
	if entityTypeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "entity_type parameter is required",
		})
		return
	}

	var rule AICapabilityRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid rule JSON",
			"details": err.Error(),
		})
		return
	}

	entityType := AIEntityType(entityTypeStr)
	matrix := h.integration.GetAICapabilityMatrix()
	matrix.UpdateEntityRule(entityType, rule)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Entity rule updated successfully",
		"entity_type": entityTypeStr,
		"updated_at":  time.Now().Format(time.RFC3339),
	})
}

// healthCheck returns health status of AI capability system
func (h *APIHandler) healthCheck(c *gin.Context) {
	status := h.integration.GetAICapabilityStatus()

	// Check if enforcement is working
	healthy := true
	issues := []string{}

	// Test with a simple profile
	testProfile := h.integration.CreateTestProfile("assistant", "US")
	testClaims := map[string]any{
		"ai_entity_type": "assistant",
		"system_id":      testProfile.SystemID,
	}

	// Test enforcement
	allowed, _, metadata := h.integration.EnforceAICapabilities("info:read", testClaims)
	if metadata == nil {
		healthy = false
		issues = append(issues, "AI enforcement metadata not generated")
	}

	// Check policy loading
	if loadedPolicies, ok := status["loaded_policies"].(int); ok && loadedPolicies == 0 {
		healthy = false
		issues = append(issues, "No governance policies loaded")
	}

	statusCode := http.StatusOK
	if !healthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"success":            healthy,
		"healthy":            healthy,
		"status":             "AI capability enforcement system",
		"enforcement_active": h.integration.IsEnforcementEnabled(),
		"test_result": map[string]any{
			"action":   "info:read",
			"allowed":  allowed,
			"metadata": metadata != nil,
		},
		"issues":      issues,
		"last_check":  time.Now().Format(time.RFC3339),
		"system_info": status,
	})
}

// GetAPIDocumentation returns OpenAPI documentation for AI capability endpoints
func (h *APIHandler) GetAPIDocumentation() map[string]any {
	return map[string]any{
		"openapi": "3.0.0",
		"info": map[string]any{
			"title":       "AgentAuth AI Capability Governance API",
			"version":     "1.0.0",
			"description": "API endpoints for AI capability matrix enforcement and governance",
		},
		"paths": map[string]any{
			"/api/v1/ai/capabilities/status": map[string]any{
				"get": map[string]any{
					"summary":     "Get AI capability enforcement status",
					"description": "Returns overall status of AI capability enforcement system",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Status information",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"success": map[string]any{"type": "boolean"},
											"status": map[string]any{
												"type": "object",
												"properties": map[string]any{
													"enforcement_active":     map[string]any{"type": "boolean"},
													"supported_entity_types": map[string]any{"type": "array"},
													"loaded_policies":        map[string]any{"type": "integer"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/ai/capabilities/test/enforcement": map[string]any{
				"post": map[string]any{
					"summary":     "Test AI enforcement decision",
					"description": "Test AI capability enforcement for a given profile and action",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"profile": map[string]any{"$ref": "#/components/schemas/AISystemProfile"},
										"action":  map[string]any{"type": "string"},
										"claims":  map[string]any{"type": "object"},
									},
									"required": []string{"profile", "action"},
								},
							},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Test result",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"success":     map[string]any{"type": "boolean"},
											"test_result": map[string]any{"type": "object"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"AISystemProfile": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"entity_type": map[string]any{
							"type": "string",
							"enum": []string{"human", "assistant", "agent", "model", "system", "robot", "analytics", "automation"},
						},
						"system_id":     map[string]any{"type": "string"},
						"model_name":    map[string]any{"type": "string"},
						"model_version": map[string]any{"type": "string"},
						"risk_level": map[string]any{
							"type": "string",
							"enum": []string{"low", "medium", "high", "critical"},
						},
						"industry_context": map[string]any{"type": "string"},
						"jurisdiction":     map[string]any{"type": "string"},
						"certified_by":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"compliance_flags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
					"required": []string{"entity_type", "system_id", "jurisdiction"},
				},
			},
		},
	}
}
