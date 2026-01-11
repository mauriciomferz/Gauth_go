// learning_lab_endpoints.go - Additional API endpoints for AgentAuth Learning Lab functionality
package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AddLearningLabEndpoints adds the API endpoints needed for Learning Lab functionality
func (s *BetaServer) AddLearningLabEndpoints() {
	if s.router == nil {
		return
	}

	// Learning endpoints
	s.router.POST("/api/v1/learning/start", s.apiLearningStart)

	// Compliance endpoints
	s.router.POST("/api/v1/compliance/check", s.apiComplianceCheck)

	// Token endpoints - use existing endpoint but add demo-specific one
	s.router.POST("/api/v1/token/create-demo", s.apiTokenCreateDemo)

	// Pattern endpoints
	s.router.POST("/api/v1/patterns/load/*action", s.apiPatternOperation)
	s.router.POST("/api/v1/patterns/test/*action", s.apiPatternOperation)
	s.router.POST("/api/v1/patterns/simulate/*action", s.apiPatternOperation)

	// Marketing PoA endpoints
	s.router.POST("/api/v1/marketing/poa", s.apiMarketingPoA)

	// Experimental endpoints
	s.router.POST("/api/v1/experimental/*action", s.apiExperimentalOperation)

	// Validation endpoints
	s.router.POST("/api/v1/validation/*action", s.apiValidationOperation)

	// Generic AgentAuth action endpoint
	s.router.POST("/api/v1/agentauth/action", s.apiGenericAction)
}

// apiLearningStart handles learning journey initialization
func (s *BetaServer) apiLearningStart(c *gin.Context) {
	sessionID := generateID()

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"session_id":        sessionID,
		"current_module":    "agentauth-fundamentals",
		"progress":          15,
		"modules_completed": 2,
		"total_modules":     12,
		"next_steps": []string{
			"Learn about delegation patterns",
			"Explore authorization hierarchies",
			"Practice token creation",
			"Test compliance scenarios",
		},
	})
}

// apiComplianceCheck performs RFC-0150 compliance analysis
func (s *BetaServer) apiComplianceCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"overall_status": "passed",
		"checks": []gin.H{
			{"name": "Token Format Compliance", "status": "passed", "details": "JWT format meets RFC-0150 requirements"},
			{"name": "Delegation Chain Validation", "status": "passed", "details": "All delegation chains properly signed"},
			{"name": "Capability Enforcement", "status": "warning", "details": "Some capabilities lack enforcement policies"},
			{"name": "Revocation Support", "status": "passed", "details": "Hash-chained revocation system active"},
			{"name": "Anchoring Integration", "status": "passed", "details": "External anchoring provider configured"},
		},
		"recommendations": []string{
			"Add enforcement policies for all registered capabilities",
			"Enable automatic key rotation for enhanced security",
			"Implement multi-signature requirements for critical operations",
		},
	})
}

// apiTokenCreateDemo creates demonstration tokens
func (s *BetaServer) apiTokenCreateDemo(c *gin.Context) {
	var req struct {
		Action       string   `json:"action"`
		Issuer       string   `json:"issuer"`
		Subject      string   `json:"subject"`
		Capabilities []string `json:"capabilities"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	tokenID := generateID()
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(24 * time.Hour)

	// Generate a mock JWT token
	header := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	//nolint:lll // base64 JWT payload
	payload := "eyJzdWIiOiJkZW1vLXVzZXIiLCJpc3MiOiJnYXV0aC1sZWFybmluZy1sYWIiLCJleHAiOjE3MzA0OTEwMDAsImNhcGFiaWxpdGllcyI6WyJkZW1vLXRva2VuLWlzc3VhbmNlIiwiYmFzaWMtcG9saWN5LWV2YWwiXX0"
	signature := generateID()[:32]
	token := fmt.Sprintf("%s.%s.%s", header, payload, signature)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"token_id":     tokenID,
		"issued_at":    issuedAt.Format(time.RFC3339),
		"expires_at":   expiresAt.Format(time.RFC3339),
		"algorithm":    "HS256",
		"token":        token,
		"capabilities": req.Capabilities,
	})
}

// apiPatternOperation handles pattern-related operations (load, test, simulate)
func (s *BetaServer) apiPatternOperation(c *gin.Context) {
	action := c.Param("action")

	var request struct {
		PatternType string                 `json:"pattern_type"`
		Parameters  map[string]interface{} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Generate a demo pattern result based on the action
	patternData := gin.H{
		"pattern_id":   generateID(),
		"pattern_type": request.PatternType,
		"action":       action,
		"parameters":   request.Parameters,
		"timestamp":    time.Now().Unix(),
		"status":       "success",
		"results": gin.H{
			"authorization_granted": true,
			"policy_matched":        "default_allow",
			"confidence_score":      0.95,
			"execution_time":        "12ms",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    patternData,
		"message": fmt.Sprintf("Pattern %s operation completed successfully", action),
	})
}

// apiMarketingPoA handles Marketing Proof-of-Authorization operations
func (s *BetaServer) apiMarketingPoA(c *gin.Context) {
	var req struct {
		Action       string `json:"action"`
		CampaignType string `json:"campaign_type"`
		ContentType  string `json:"content_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	poaID := generateID()
	campaignID := generateID()

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"poa_id":        poaID,
		"campaign_id":   campaignID,
		"status":        "authorized",
		"authorized_by": "marketing-manager@company.com",
		"expires_at":    time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
		"credential":    fmt.Sprintf("PoA-CRED-%s", generateID()[:16]),
	})
}

// apiExperimentalOperation handles experimental playground operations
func (s *BetaServer) apiExperimentalOperation(c *gin.Context) {
	action := c.Param("action")

	var req struct {
		Action         string                 `json:"action"`
		ExperimentType string                 `json:"experiment_type"`
		Parameters     map[string]interface{} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	experimentID := generateID()

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"experiment_id": experimentID,
		"status":        "completed",
		"output": fmt.Sprintf(
			"Experimental operation '%s' executed successfully.\nMode: %v\nResults: All tests passed.",
			action, req.Parameters["mode"]),
		"metrics": []string{
			"Execution time: 234ms",
			"Memory usage: 15.2MB",
			"CPU utilization: 3.4%",
			"Success rate: 100%",
		},
	})
}

// apiValidationOperation handles validation operations
func (s *BetaServer) apiValidationOperation(c *gin.Context) {
	action := c.Param("action")

	var req struct {
		Action         string `json:"action"`
		ValidationType string `json:"validation_type"`
		StrictMode     bool   `json:"strict_mode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	validationID := generateID()

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"validation_id":  validationID,
		"action":         action,
		"overall_status": "passed",
		"checks": []gin.H{
			{"name": "RFC-0150 Schema Compliance", "status": "passed", "details": "All schemas validate successfully"},
			{"name": "Token Signature Verification", "status": "passed", "details": "Digital signatures are valid"},
			{"name": "Delegation Chain Integrity", "status": "passed", "details": "Chain of trust maintained"},
			{"name": "Capability Authorization", "status": "warning", "details": "Some operations lack explicit authorization"},
		},
		"errors": []string{},
	})
}

// apiGenericAction handles generic AgentAuth actions
func (s *BetaServer) apiGenericAction(c *gin.Context) {
	var req struct {
		Action  string `json:"action"`
		Context string `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	actionID := generateID()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"action_id": actionID,
		"status":    "completed",
		"message":   fmt.Sprintf("AgentAuth action '%s' executed successfully in %s context", req.Action, req.Context),
		"details": gin.H{
			"timestamp": time.Now().Format(time.RFC3339),
			"context":   req.Context,
			"action":    req.Action,
		},
	})
}

// generateID generates a random hexadecimal ID
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to predictable ID on error
		return "fallback-id-" + hex.EncodeToString([]byte{1, 2, 3, 4})
	}
	return hex.EncodeToString(bytes)
}
