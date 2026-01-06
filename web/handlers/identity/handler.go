package identity

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/identity"
)

type Handler struct {
	provisioner identity.IdentityProvisioner
}

func NewHandler(provisioner identity.IdentityProvisioner) *Handler {
	return &Handler{
		provisioner: provisioner,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/provision", h.HandleProvisioningRequest)
}

type ProvisioningRequest struct {
	AgentID        string `json:"agent_id" binding:"required"`
	TargetAudience string `json:"target_audience" binding:"required"`
	// Proof is optional for this phase, but would be required in production
	// Proof          string `json:"proof"`
}

type ProvisioningResponse struct {
	Assertion string `json:"assertion"`
	ExpiresIn int    `json:"expires_in"`
}

func (h *Handler) HandleProvisioningRequest(c *gin.Context) {
	var req ProvisioningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	// In a real scenario, we would validate the caller here.
	// For now, we assume the API gateway or middleware handles authentication.

	assertion, err := h.provisioner.MintAssertion(c.Request.Context(), req.AgentID, req.TargetAudience)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provisioning_failed", "message": err.Error()})
		return
	}

	resp := ProvisioningResponse{
		Assertion: assertion,
		ExpiresIn: 3600,
	}

	c.JSON(http.StatusOK, resp)
}
