package gagent

import (
	"net/http"
	"sync"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/enforcement"
	"github.com/gin-gonic/gin"
)

// APIHandler provides HTTP endpoints for G-Agent management
type APIHandler struct {
	agents map[string]*Agent
	mu     sync.RWMutex
}

// NewAPIHandler creates a new API handler
func NewAPIHandler() *APIHandler {
	return &APIHandler{
		agents: make(map[string]*Agent),
	}
}

// RegisterAgent registers an agent
func (h *APIHandler) RegisterAgent(agent *Agent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agents[agent.GetAgentID()] = agent
}

// UnregisterAgent unregisters an agent
func (h *APIHandler) UnregisterAgent(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.agents, agentID)
}

// GetAgent retrieves an agent by ID
func (h *APIHandler) GetAgent(agentID string) (*Agent, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	agent, ok := h.agents[agentID]
	return agent, ok
}

// RegisterRoutes registers G-Agent API routes
func (h *APIHandler) RegisterRoutes(router *gin.Engine) {
	gagent := router.Group("/api/v1/g-agent")
	{
		// Agent management endpoints
		gagent.GET("/agents", h.listAgents)
		gagent.GET("/agents/:agent_id", h.getAgentInfo)
		gagent.POST("/agents/:agent_id/enable", h.enableAgent)
		gagent.POST("/agents/:agent_id/disable", h.disableAgent)
		gagent.GET("/agents/:agent_id/metrics", h.getAgentMetrics)

		// Enforcement evaluation endpoint
		gagent.POST("/evaluate", h.evaluateEnforcement)

		// Batch evaluation endpoint
		gagent.POST("/evaluate/batch", h.evaluateBatch)

		// Health check
		gagent.GET("/health", h.healthCheck)

		// Capabilities endpoint
		gagent.GET("/capabilities", h.getCapabilities)
	}
}

// listAgents returns all registered agents
func (h *APIHandler) listAgents(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]AgentInfo, 0, len(h.agents))
	for _, agent := range h.agents {
		agents = append(agents, agent.GetInfo())
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(agents),
		"agents":  agents,
	})
}

// getAgentInfo returns agent information
func (h *APIHandler) getAgentInfo(c *gin.Context) {
	agentID := c.Param("agent_id")

	agent, ok := h.GetAgent(agentID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "agent_not_found",
			"message": "Agent not found",
		})
		return
	}

	info := agent.GetInfo()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"agent":   info,
	})
}

// enableAgent enables an agent
func (h *APIHandler) enableAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	agent, ok := h.GetAgent(agentID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "agent_not_found",
			"message": "Agent not found",
		})
		return
	}

	agent.Enable()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Agent enabled",
		"agent":   agent.GetInfo(),
	})
}

// disableAgent disables an agent
func (h *APIHandler) disableAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	agent, ok := h.GetAgent(agentID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "agent_not_found",
			"message": "Agent not found",
		})
		return
	}

	agent.Disable()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Agent disabled",
		"agent":   agent.GetInfo(),
	})
}

// getAgentMetrics returns agent metrics
func (h *APIHandler) getAgentMetrics(c *gin.Context) {
	agentID := c.Param("agent_id")

	agent, ok := h.GetAgent(agentID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "agent_not_found",
			"message": "Agent not found",
		})
		return
	}

	metrics := agent.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"agent_id": agentID,
		"metrics":  metrics,
	})
}

// evaluateEnforcement evaluates an enforcement request
func (h *APIHandler) evaluateEnforcement(c *gin.Context) {
	var request struct {
		AgentID     string                              `json:"agent_id"`
		Subject     string                              `json:"subject" binding:"required"`
		Resource    string                              `json:"resource" binding:"required"`
		Action      string                              `json:"action" binding:"required"`
		Context     map[string]interface{}              `json:"context"`
		Disclosures []enforcement.DisclosureRequirement `json:"disclosures"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// If no agent specified, use first available enabled agent
	var agent *Agent
	if request.AgentID != "" {
		var ok bool
		agent, ok = h.GetAgent(request.AgentID)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "agent_not_found",
				"message": "Specified agent not found",
			})
			return
		}
	} else {
		// Use first enabled agent
		h.mu.RLock()
		for _, a := range h.agents {
			if a.IsEnabled() {
				agent = a
				break
			}
		}
		h.mu.RUnlock()

		if agent == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "no_agent_available",
				"message": "No enabled agent available",
			})
			return
		}
	}

	// Create enforcement request
	enforcementReq := &enforcement.EnforcementRequest{
		Subject:     request.Subject,
		Resource:    request.Resource,
		Action:      request.Action,
		Context:     request.Context,
		Disclosures: request.Disclosures,
	}

	// Evaluate
	recommendation, err := agent.EvaluateEnforcement(c.Request.Context(), enforcementReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "evaluation_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"agent_id":       agent.GetAgentID(),
		"recommendation": recommendation,
		"request": gin.H{
			"subject":  request.Subject,
			"resource": request.Resource,
			"action":   request.Action,
		},
	})
}

// evaluateBatch evaluates multiple enforcement requests
func (h *APIHandler) evaluateBatch(c *gin.Context) {
	var request struct {
		AgentID  string `json:"agent_id"`
		Requests []struct {
			Subject     string                              `json:"subject" binding:"required"`
			Resource    string                              `json:"resource" binding:"required"`
			Action      string                              `json:"action" binding:"required"`
			Context     map[string]interface{}              `json:"context"`
			Disclosures []enforcement.DisclosureRequirement `json:"disclosures"`
		} `json:"requests" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Get agent
	var agent *Agent
	if request.AgentID != "" {
		var ok bool
		agent, ok = h.GetAgent(request.AgentID)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "agent_not_found",
				"message": "Specified agent not found",
			})
			return
		}
	} else {
		// Use first enabled agent
		h.mu.RLock()
		for _, a := range h.agents {
			if a.IsEnabled() {
				agent = a
				break
			}
		}
		h.mu.RUnlock()

		if agent == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "no_agent_available",
				"message": "No enabled agent available",
			})
			return
		}
	}

	// Evaluate all requests
	results := make([]gin.H, 0, len(request.Requests))
	for i, req := range request.Requests {
		enforcementReq := &enforcement.EnforcementRequest{
			Subject:     req.Subject,
			Resource:    req.Resource,
			Action:      req.Action,
			Context:     req.Context,
			Disclosures: req.Disclosures,
		}

		recommendation, err := agent.EvaluateEnforcement(c.Request.Context(), enforcementReq)
		if err != nil {
			results = append(results, gin.H{
				"index":   i,
				"success": false,
				"error":   err.Error(),
			})
		} else {
			results = append(results, gin.H{
				"index":          i,
				"success":        true,
				"recommendation": recommendation,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"agent_id": agent.GetAgentID(),
		"total":    len(request.Requests),
		"results":  results,
	})
}

// healthCheck returns G-Agent API health status
func (h *APIHandler) healthCheck(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalAgents := len(h.agents)
	enabledAgents := 0
	for _, agent := range h.agents {
		if agent.IsEnabled() {
			enabledAgents++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"status":         "healthy",
		"total_agents":   totalAgents,
		"enabled_agents": enabledAgents,
		"timestamp":      gin.H{"utc": c.GetHeader("Date")},
	})
}

// getCapabilities returns G-Agent API capabilities
func (h *APIHandler) getCapabilities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"capabilities": gin.H{
			"enforcement_evaluation": gin.H{
				"description": "AI-assisted enforcement decision making",
				"endpoint":    "/api/v1/g-agent/evaluate",
				"methods":     []string{"POST"},
			},
			"batch_evaluation": gin.H{
				"description": "Batch enforcement evaluation (up to 100 requests)",
				"endpoint":    "/api/v1/g-agent/evaluate/batch",
				"methods":     []string{"POST"},
			},
			"agent_management": gin.H{
				"description": "Agent lifecycle and configuration management",
				"endpoints": []string{
					"/api/v1/g-agent/agents",
					"/api/v1/g-agent/agents/:agent_id",
					"/api/v1/g-agent/agents/:agent_id/enable",
					"/api/v1/g-agent/agents/:agent_id/disable",
				},
			},
			"metrics": gin.H{
				"description": "Agent performance metrics and analytics",
				"endpoint":    "/api/v1/g-agent/agents/:agent_id/metrics",
				"methods":     []string{"GET"},
			},
		},
		"features": []string{
			"policy-based-enforcement",
			"context-aware-analysis",
			"risk-scoring",
			"multi-agent-support",
			"batch-processing",
			"real-time-evaluation",
		},
	})
}
