package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/ai"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("🤖 AI Capability Matrix Enforcement Demo")
	fmt.Println("========================================")

	// Create AI capability integration
	integration := ai.NewServerIntegration()
	integration.EnableEnforcement(true)

	// Set up audit and metrics callbacks
	integration.SetAuditCallback(func(action string, metadata map[string]any) {
		fmt.Printf("📋 AUDIT: %s - %v\n", action, metadata["decision"])
	})

	integration.SetMetricsCallback(func(metric string) {
		fmt.Printf("📊 METRIC: %s incremented\n", metric)
	})

	// Demo scenarios
	fmt.Println("\n🎭 Demo Scenarios:")
	fmt.Println("================")

	// Scenario 1: Human User (should be allowed for everything)
	fmt.Println("\n1. 👤 Human User Access:")
	testHumanAccess(integration)

	// Scenario 2: AI Assistant (restricted access)
	fmt.Println("\n2. 🤖 AI Assistant Access:")
	testAIAssistantAccess(integration)

	// Scenario 3: AI Agent with proper compliance (should be allowed with restrictions)
	fmt.Println("\n3. 🤖 AI Agent with Compliance:")
	testAIAgentAccess(integration)

	// Scenario 4: EU AI Agent (stricter compliance)
	fmt.Println("\n4. 🇪🇺 EU AI Agent (EU AI Act):")
	testEUAIAgentAccess(integration)

	// Scenario 5: Healthcare AI (HIPAA compliance)
	fmt.Println("\n5. 🏥 Healthcare AI (HIPAA):")
	testHealthcareAIAccess(integration)

	// Scenario 6: Financial AI (SOX compliance)
	fmt.Println("\n6. 💰 Financial AI (SOX):")
	testFinancialAIAccess(integration)

	// Show governance policies
	fmt.Println("\n📋 Loaded Governance Policies:")
	fmt.Println("=============================")
	policies := integration.GetGovernancePolicies()
	for _, policy := range policies {
		fmt.Printf("- %s (%s, %s)\n", policy.PolicyID, policy.Jurisdiction, policy.ComplianceFramework)
	}

	// Start API server
	fmt.Println("\n🌐 Starting AI Capability API Server...")
	fmt.Println("======================================")
	startAPIServer(integration)
}

func testHumanAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"user_id": "human-123",
		"name":    "John Doe",
	}

	allowed, missing, metadata := integration.EnforceAICapabilities("admin:delete", claims)
	fmt.Printf("   Action: admin:delete | Allowed: %v | Missing: %v | Entity: %v\n",
		allowed, missing, metadata["entity_type"])
}

func testAIAssistantAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":             "assistant",
		"system_id":                  "chatgpt-4",
		"jurisdiction":               "US",
		"ai_entity_verified":         true,
		"algorithmic_accountability": true,
	}

	// Test allowed action
	allowed, missing, metadata := integration.EnforceAICapabilities("transaction:read", claims)
	fmt.Printf("   Action: transaction:read | Allowed: %v | Missing: %v | Human Auth: %v\n",
		allowed, missing, metadata["required_human_auth"])

	// Test forbidden action
	allowed, missing, metadata = integration.EnforceAICapabilities("transaction:execute", claims)
	fmt.Printf("   Action: transaction:execute | Allowed: %v | Missing: %v | Reason: %v\n",
		allowed, missing, metadata["reason"])
}

func testAIAgentAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":             "agent",
		"system_id":                  "autonomous-agent-1",
		"jurisdiction":               "US",
		"risk_level":                 "medium",
		"ai_entity_verified":         true,
		"ai_agent_registered":        true,
		"algorithmic_accountability": true,
	}

	allowed, missing, metadata := integration.EnforceAICapabilities("transaction:execute", claims)
	fmt.Printf("   Action: transaction:execute | Allowed: %v | Missing: %v | Human Auth: %v | Audit: %v\n",
		allowed, missing, metadata["required_human_auth"], metadata["audit_level"])
}

func testEUAIAgentAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":       "agent",
		"system_id":            "eu-agent-1",
		"jurisdiction":         "EU",
		"risk_level":           "high",
		"ai_entity_verified":   true,
		"ai_agent_registered":  true,
		// Missing EU compliance claims
	}

	allowed, missing, metadata := integration.EnforceAICapabilities("transaction:execute", claims)
	fmt.Printf("   Action: transaction:execute | Allowed: %v | Missing: %v | Policies: %v\n",
		allowed, missing, metadata["applied_policies"])

	// Now with EU compliance
	claims["eu_ai_conformity"] = true
	claims["human_oversight"] = true
	claims["ai_risk_assessment"] = true

	allowed, _, metadata = integration.EnforceAICapabilities("transaction:read", claims)
	fmt.Printf("   Action: transaction:read (with compliance) | Allowed: %v | Human Auth: %v\n",
		allowed, metadata["required_human_auth"])
}

func testHealthcareAIAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":             "analytics",
		"system_id":                  "healthcare-ai-1",
		"jurisdiction":               "US",
		"industry_context":           "healthcare",
		"risk_level":                 "critical",
		"ai_analytics_approved":      true,
		"ai_entity_verified":         true,
		"hipaa_compliance":           true,
		"phi_protection":             true,
		"healthcare_cert":            true,
		"de_identification":          true,
		"algorithmic_accountability": true,
	}

	allowed, _, metadata := integration.EnforceAICapabilities("transaction:read", claims)
	fmt.Printf("   Action: transaction:read | Allowed: %v | Human Auth: %v | Audit: %v | Policies: %v\n",
		allowed, metadata["required_human_auth"], metadata["audit_level"], metadata["applied_policies"])
}

func testFinancialAIAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":             "automation",
		"system_id":                  "finance-ai-1",
		"jurisdiction":               "US",
		"industry_context":           "finance",
		"risk_level":                 "high",
		"ai_automation_certified":    true,
		"ai_entity_verified":         true,
		"sox_compliance":             true,
		"financial_cert":             true,
		"model_validation":           true,
		"algorithmic_accountability": true,
	}

	allowed, _, metadata := integration.EnforceAICapabilities("transaction:read", claims)
	fmt.Printf("   Action: transaction:read | Allowed: %v | Human Auth: %v | Policies: %v\n",
		allowed, metadata["required_human_auth"], metadata["applied_policies"])

	// Test forbidden action
	allowed, _, metadata = integration.EnforceAICapabilities("transaction:pay", claims)
	fmt.Printf("   Action: transaction:pay | Allowed: %v | Reason: %v\n",
		allowed, metadata["reason"])
}

func startAPIServer(integration *ai.ServerIntegration) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Add AI capability API routes
	apiHandler := ai.NewAPIHandler(integration)
	apiHandler.RegisterRoutes(router)

	// Add demo routes
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "GAuth AI Capability Matrix Demo",
			"version": "1.0.0",
			"endpoints": []string{
				"GET /api/v1/ai/capabilities/status",
				"GET /api/v1/ai/capabilities/entity-types",
				"GET /api/v1/ai/capabilities/policies",
				"POST /api/v1/ai/capabilities/test/enforcement",
				"POST /api/v1/ai/capabilities/simulate/decision",
				"GET /api/v1/ai/health",
			},
		})
	})

	// Demo enforcement endpoint
	router.POST("/demo/enforce", func(c *gin.Context) {
		var request struct {
			Action string         `json:"action"`
			Claims map[string]any `json:"claims"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		allowed, missing, metadata := integration.EnforceAICapabilities(request.Action, request.Claims)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"result": map[string]any{
				"action":               request.Action,
				"allowed":              allowed,
				"missing_capabilities": missing,
				"metadata":             metadata,
				"timestamp":            time.Now().Format(time.RFC3339),
			},
		})
	})

	// API documentation
	router.GET("/api/docs", func(c *gin.Context) {
		docs := apiHandler.GetAPIDocumentation()
		c.JSON(http.StatusOK, docs)
	})

	fmt.Println("🚀 Server starting on http://localhost:8080")
	fmt.Println("\nExample API calls:")
	fmt.Println("==================")
	fmt.Println("curl http://localhost:8080/api/v1/ai/capabilities/status")
	fmt.Println("curl http://localhost:8080/api/v1/ai/capabilities/entity-types")
	fmt.Println("curl http://localhost:8080/api/v1/ai/health")
	fmt.Println("")
	fmt.Println("Demo enforcement:")
	fmt.Println("curl -X POST http://localhost:8080/demo/enforce \\")
	fmt.Println("  -H 'Content-Type: application/json' \\")
	fmt.Println("  -d '{\"action\":\"transaction:read\",\"claims\":{\"ai_entity_type\":\"assistant\",\"ai_entity_verified\":true,\"algorithmic_accountability\":true}}'")
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop the server")

	log.Fatal(http.ListenAndServe(":8080", router))
}