// Package disclosure provides HTTP handlers for AAP-001 transparency endpoints
package disclosure

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
)

// Handler provides HTTP handlers for disclosure API
type Handler struct {
	disclosureService *agentauth.DisclosureService
}

// NewHandler creates a new disclosure handler
func NewHandler(disclosureService *agentauth.DisclosureService) *Handler {
	return &Handler{
		disclosureService: disclosureService,
	}
}

// ListActiveAuthorizationsHandler handles GET /api/v1/disclosure/authorizations
func (h *Handler) ListActiveAuthorizationsHandler(c *gin.Context) {
	// Parse query parameters
	resourceOwnerID := c.Query("resource_owner_id")
	if resourceOwnerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_owner_id is required"})
		return
	}

	clientID := c.Query("client_id")
	status := c.Query("status")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Build request
	request := &agentauth.ListActiveAuthorizationsRequest{
		ResourceOwnerID: resourceOwnerID,
		ClientID:        clientID,
		Status:          status,
		Limit:           limit,
		Offset:          offset,
	}

	// Call service
	response, err := h.disclosureService.ListActiveAuthorizations(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, response)
}

// GetAuthorizationDetailHandler handles GET /api/v1/disclosure/authorizations/{id}
func (h *Handler) GetAuthorizationDetailHandler(c *gin.Context) {
	authorizationID := c.Param("id")
	resourceOwnerID := c.Query("resource_owner_id")

	if resourceOwnerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_owner_id is required"})
		return
	}

	// Call service
	detail, err := h.disclosureService.GetAuthorizationDetail(c.Request.Context(), authorizationID, resourceOwnerID)
	if err != nil {
		if err.Error() == "unauthorized: not the resource owner" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, detail)
}

// RevokeAuthorizationHandler handles POST /api/v1/disclosure/authorizations/{id}/revoke
func (h *Handler) RevokeAuthorizationHandler(c *gin.Context) {
	authorizationID := c.Param("id")

	// Parse request body
	var req struct {
		ResourceOwnerID string `json:"resource_owner_id" binding:"required"`
		Reason          string `json:"reason"`
		RevokedBy       string `json:"revoked_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Build request
	request := &agentauth.RevokeAuthorizationRequest{
		AuthorizationID: authorizationID,
		ResourceOwnerID: req.ResourceOwnerID,
		Reason:          req.Reason,
		RevokedBy:       req.RevokedBy,
	}

	// Call service
	response, err := h.disclosureService.RevokeAuthorization(c.Request.Context(), request)
	if err != nil {
		if err.Error() == "unauthorized: not the resource owner" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, response)
}

// GetAuditTrailHandler handles GET /api/v1/disclosure/authorizations/{id}/audit
func (h *Handler) GetAuditTrailHandler(c *gin.Context) {
	authorizationID := c.Param("id")
	resourceOwnerID := c.Query("resource_owner_id")

	if resourceOwnerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_owner_id is required"})
		return
	}

	// Parse optional parameters
	fromDateStr := c.Query("from_date")
	var fromDate time.Time
	if fromDateStr != "" {
		var err error
		fromDate, err = time.Parse(time.RFC3339, fromDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from_date format, use RFC3339"})
			return
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	// Call service
	entries, err := h.disclosureService.GetAuditTrail(c.Request.Context(), authorizationID, resourceOwnerID, fromDate, limit)
	if err != nil {
		if err.Error() == "unauthorized: not the resource owner" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"authorization_id": authorizationID,
		"audit_trail":      entries,
		"count":            len(entries),
	})
}
