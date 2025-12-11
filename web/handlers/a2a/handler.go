// Package a2a provides HTTP handlers for Agent-to-Agent Authorization Profile.
package a2a

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/a2a"
)

// Handler manages A2A operations.
type Handler struct {
	// In a real implementation, would have a store for agents and active chains
}

// NewHandler creates a new A2A handler.
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes registers A2A endpoints.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/a2a/token", h.IssueToken)
	r.POST("/a2a/verify", h.VerifyChain)
	r.GET("/a2a/agents", h.ListAgents)
}

// IssueTokenRequest request body for issuing an A2A token.
type IssueTokenRequest struct {
	Audience string              `json:"aud"`
	Subject  a2a.AgentIdentity   `json:"sub"`
	Context  *a2a.A2ACallContext `json:"context,omitempty"` // For continuation/chaining
}

// IssueToken handles POST /a2a/token.
func (h *Handler) IssueToken(c *gin.Context) {
	var req IssueTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "details": err.Error()})
		return
	}

	// Mock logic: Create or extend chain
	var ctx *a2a.A2ACallContext
	if req.Context != nil {
		ctx = req.Context
		// Add hop: current agent calling next agent (aud)
		// We'd need signer identity here, for now using Subject as caller
		caller := req.Subject
		callee := a2a.AgentIdentity{ID: req.Audience, Type: "unknown"}

		// Reconstruct builder to extend
		// Ideally we'd validte existing chain first
		// builder := a2a.NewChainBuilderFromContext(ctx) ...

		// For MVP, just extending manually
		lastHopHash := ""
		if len(ctx.Hops) > 0 {
			lastHopHash = ctx.Hops[len(ctx.Hops)-1].ComputeHash()
		} else {
			lastHopHash = ctx.ChainID
		}

		hop := a2a.CallHop{
			Caller:    caller,
			Callee:    callee,
			Timestamp: time.Now().UTC(),
			Action:    "call",
			PrevHash:  lastHopHash,
		}
		ctx.Hops = append(ctx.Hops, hop)
	} else {
		// Start new chain
		chainID := generateID()
		builder := a2a.NewChainBuilder(chainID, req.Subject)
		ctx = builder.Build()
	}

	token := a2a.A2ATransactionToken{
		Context:   *ctx,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().Add(1 * time.Hour).UTC(),
	}

	c.JSON(http.StatusOK, token)
}

// VerifyChainRequest request body for chain verification.
type VerifyChainRequest struct {
	Context a2a.A2ACallContext `json:"context"`
}

// VerifyChain handles POST /a2a/verify.
func (h *Handler) VerifyChain(c *gin.Context) {
	var req VerifyChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	validator := a2a.ChainValidator{}
	if err := validator.Validate(&req.Context); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// ListAgents handles GET /a2a/agents.
func (h *Handler) ListAgents(c *gin.Context) {
	// Mock registry
	agents := []a2a.AgentIdentity{
		{ID: "agent-1", Type: "orchestrator", Capabilities: []string{"plan", "execute"}},
		{ID: "agent-2", Type: "audit-logger", Capabilities: []string{"log"}},
	}
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
