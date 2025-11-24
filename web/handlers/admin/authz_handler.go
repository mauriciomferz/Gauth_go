package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
)

// AuthorizationHandler manages authorization engine operations for the admin portal
type AuthorizationHandler struct {
	repo *authz.Repository
}

// NewAuthorizationHandler creates a new authorization handler instance
func NewAuthorizationHandler(db *pgxpool.Pool) *AuthorizationHandler {
	return &AuthorizationHandler{
		repo: authz.NewRepository(db),
	}
}

// Policy represents an authorization policy
type Policy struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"` // active, draft, disabled
	Effect      string   `json:"effect"` // allow, deny
	Actions     []string `json:"actions"`
	Resources   []string `json:"resources"`
	Conditions  string   `json:"conditions,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// PolicyRequest represents the request to create/update a policy
type PolicyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Effect      string   `json:"effect" binding:"required,oneof=allow deny"`
	Actions     []string `json:"actions" binding:"required"`
	Resources   []string `json:"resources" binding:"required"`
	Conditions  string   `json:"conditions"`
}

// Attribute represents a policy information point attribute
type Attribute struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	LastUpdated string `json:"lastUpdated"`
}

// Decision represents an authorization decision log entry
type Decision struct {
	ID         string `json:"id"`
	Timestamp  string `json:"timestamp"`
	Subject    string `json:"subject"`
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	Decision   string `json:"decision"` // allow, deny
	PolicyID   string `json:"policyId"`
	PolicyName string `json:"policyName"`
	Duration   int    `json:"duration"` // milliseconds
}

// SimulationRequest represents a decision simulation request
type SimulationRequest struct {
	Subject  string                 `json:"subject" binding:"required"`
	Action   string                 `json:"action" binding:"required"`
	Resource string                 `json:"resource" binding:"required"`
	Context  map[string]interface{} `json:"context"`
}

// SimulationResult represents a decision simulation result
type SimulationResult struct {
	Decision   string `json:"decision"`
	PolicyID   string `json:"policyId,omitempty"`
	PolicyName string `json:"policyName,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Duration   int    `json:"duration"`
}

// ListPolicies returns all authorization policies
// GET /api/admin/authz/policies
func (h *AuthorizationHandler) ListPolicies(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	prs, err := h.repo.ListPolicies(c.Request.Context(), tenantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch policies"})
		return
	}

	// Convert to API format
	policies := make([]Policy, len(prs))
	for i, pr := range prs {
		// Convert conditions JSONB to string representation
		var conditionsStr string
		if len(pr.Conditions) > 0 {
			conditionsStr = fmt.Sprintf("%v", pr.Conditions)
		}

		policies[i] = Policy{
			ID:          pr.ID,
			Name:        pr.PolicyName,
			Description: pr.Description,
			Status:      pr.Status,
			Effect:      pr.Effect,
			Actions:     pr.Actions,
			Resources:   pr.Resources,
			Conditions:  conditionsStr,
			CreatedAt:   pr.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   pr.UpdatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"policies": policies,
		"total":    len(policies),
	})
}

// CreatePolicy creates a new authorization policy
// POST /api/admin/authz/policies
func (h *AuthorizationHandler) CreatePolicy(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	var req PolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse conditions string to JSONB
	var conditions map[string]interface{}
	if req.Conditions != "" {
		// For now, store as simple map - in production, parse JSON string
		conditions = map[string]interface{}{"raw": req.Conditions}
	}

	policyRecord := &authz.PolicyRecord{
		TenantID:    tenantID.(string),
		PolicyName:  req.Name,
		PolicyType:  "abac", // Default to ABAC
		Version:     1,
		Status:      "draft",
		Description: req.Description,
		Rules:       map[string]interface{}{}, // Empty rules for now
		Conditions:  conditions,
		Actions:     req.Actions,
		Resources:   req.Resources,
		Effect:      req.Effect,
		Priority:    100, // Default priority
	}

	if err := h.repo.CreatePolicy(c.Request.Context(), policyRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create policy"})
		return
	}

	policy := Policy{
		ID:          policyRecord.ID,
		Name:        req.Name,
		Description: req.Description,
		Status:      "draft",
		Effect:      req.Effect,
		Actions:     req.Actions,
		Resources:   req.Resources,
		Conditions:  req.Conditions,
		CreatedAt:   policyRecord.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   policyRecord.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, policy)
}

// GetPolicy retrieves a specific policy by ID
// GET /api/admin/authz/policies/:id
func (h *AuthorizationHandler) GetPolicy(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	policyID := c.Param("id")

	pr, err := h.repo.GetPolicy(c.Request.Context(), tenantID.(string), policyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	var conditionsStr string
	if len(pr.Conditions) > 0 {
		conditionsStr = fmt.Sprintf("%v", pr.Conditions)
	}

	policy := Policy{
		ID:          pr.ID,
		Name:        pr.PolicyName,
		Description: pr.Description,
		Status:      pr.Status,
		Effect:      pr.Effect,
		Actions:     pr.Actions,
		Resources:   pr.Resources,
		Conditions:  conditionsStr,
		CreatedAt:   pr.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   pr.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, policy)
}

// UpdatePolicy updates an existing policy
// PUT /api/admin/authz/policies/:id
func (h *AuthorizationHandler) UpdatePolicy(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	policyID := c.Param("id")
	var req PolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing policy to preserve fields
	pr, err := h.repo.GetPolicy(c.Request.Context(), tenantID.(string), policyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	// Parse conditions string to JSONB
	var conditions map[string]interface{}
	if req.Conditions != "" {
		conditions = map[string]interface{}{"raw": req.Conditions}
	}

	// Update fields
	pr.PolicyName = req.Name
	pr.Description = req.Description
	pr.Actions = req.Actions
	pr.Resources = req.Resources
	pr.Effect = req.Effect
	pr.Conditions = conditions
	pr.Status = "active" // Activate on update

	if err := h.repo.UpdatePolicy(c.Request.Context(), pr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update policy"})
		return
	}

	policy := Policy{
		ID:          pr.ID,
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		Effect:      req.Effect,
		Actions:     req.Actions,
		Resources:   req.Resources,
		Conditions:  req.Conditions,
		CreatedAt:   pr.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, policy)
}

// DeletePolicy deletes a policy
// DELETE /api/admin/authz/policies/:id
func (h *AuthorizationHandler) DeletePolicy(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	policyID := c.Param("id")

	if err := h.repo.DeletePolicy(c.Request.Context(), tenantID.(string), policyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete policy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Policy deleted successfully",
		"id":      policyID,
	})
}

// ListAttributes returns all attribute sources (PIP)
// GET /api/admin/authz/attributes
func (h *AuthorizationHandler) ListAttributes(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	ars, err := h.repo.ListAttributes(c.Request.Context(), tenantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attributes"})
		return
	}

	// Convert to API format
	attributes := make([]Attribute, len(ars))
	for i, ar := range ars {
		attributes[i] = Attribute{
			ID:          ar.ID,
			Source:      ar.Source,
			Key:         ar.AttributeName,
			Value:       ar.SampleValue,
			Type:        ar.ValueType,
			LastUpdated: ar.CreatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"attributes": attributes,
		"total":      len(attributes),
	})
}

// SimulateDecision simulates an authorization decision (PDP)
// POST /api/admin/authz/simulate
func (h *AuthorizationHandler) SimulateDecision(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	var req SimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Integrate with actual PDP engine (pkg/authz/authz_core.go)
	// Decision evaluation logic
	startTime := time.Now()

	// Simple evaluation
	decision := "allow"
	policyID := "pol_1a2b3c"
	policyName := "document-read-policy"
	reason := "Subject has required permissions for the resource"

	// Check for deny patterns
	if req.Action == "delete" && req.Resource == "sensitive:*" {
		decision = "deny"
		policyID = "pol_0j1k2l"
		policyName = "deny-external-access"
		reason = "Action is denied by security policy"
	}

	duration := int(time.Since(startTime).Milliseconds())

	// Log simulation decision to database (optional for simulations)
	decisionRecord := &authz.DecisionRecord{
		TenantID:         tenantID.(string),
		Timestamp:        time.Now(),
		UserID:           req.Subject,
		Action:           req.Action,
		Resource:         req.Resource,
		Decision:         decision,
		PolicyID:         &policyID,
		PolicyName:       &policyName,
		Context:          req.Context,
		EvaluationTimeMs: duration,
	}
	_ = h.repo.LogDecision(c.Request.Context(), decisionRecord) // Ignore errors for simulations

	result := SimulationResult{
		Decision:   decision,
		PolicyID:   policyID,
		PolicyName: policyName,
		Reason:     reason,
		Duration:   duration,
	}

	c.JSON(http.StatusOK, result)
}

// ListDecisions returns recent authorization decisions (PEP log)
// GET /api/admin/authz/decisions
func (h *AuthorizationHandler) ListDecisions(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	drs, err := h.repo.ListDecisions(c.Request.Context(), tenantID.(string), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch decisions"})
		return
	}

	// Convert to API format
	decisions := make([]Decision, len(drs))
	for i, dr := range drs {
		var policyID, policyName string
		if dr.PolicyID != nil {
			policyID = *dr.PolicyID
		}
		if dr.PolicyName != nil {
			policyName = *dr.PolicyName
		}

		decisions[i] = Decision{
			ID:         dr.ID,
			Timestamp:  dr.Timestamp.Format(time.RFC3339),
			Subject:    dr.UserID,
			Action:     dr.Action,
			Resource:   dr.Resource,
			Decision:   dr.Decision,
			PolicyID:   policyID,
			PolicyName: policyName,
			Duration:   dr.EvaluationTimeMs,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"decisions": decisions,
		"total":     len(decisions),
	})
}

// RegisterRoutes registers all authorization engine routes
func (h *AuthorizationHandler) RegisterRoutes(router *gin.RouterGroup) {
	authz := router.Group("/authz")
	{
		// PAP - Policy Administration Point
		authz.GET("/policies", h.ListPolicies)
		authz.POST("/policies", h.CreatePolicy)
		authz.GET("/policies/:id", h.GetPolicy)
		authz.PUT("/policies/:id", h.UpdatePolicy)
		authz.DELETE("/policies/:id", h.DeletePolicy)

		// PIP - Policy Information Point
		authz.GET("/attributes", h.ListAttributes)

		// PDP - Policy Decision Point
		authz.POST("/simulate", h.SimulateDecision)

		// PEP - Policy Enforcement Point
		authz.GET("/decisions", h.ListDecisions)
	}
}

// Helper function to generate random IDs
func generateID() string {
	return time.Now().Format("20060102150405")
}
