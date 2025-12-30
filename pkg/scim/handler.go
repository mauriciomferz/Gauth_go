package scim

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler manages SCIM HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new SCIM handler
func NewHandler(db *pgxpool.Pool) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service}
}

// RegisterRoutes registers SCIM endpoints
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	// Group /scim/v2
	scimGroup := router.Group("/scim/v2")
	{
		scimGroup.GET("/Users", h.ListUsers)
		scimGroup.POST("/Users", h.CreateUser)
		scimGroup.GET("/Users/:id", h.GetUser)
		scimGroup.PUT("/Users/:id", h.ReplaceUser)
		scimGroup.PATCH("/Users/:id", h.PatchUser)
		scimGroup.DELETE("/Users/:id", h.DeleteUser)
		scimGroup.GET("/ServiceProviderConfig", h.ServiceProviderConfig)
	}
	// Group /admin/scim (Admin API)
	adminGroup := router.Group("/admin/scim")
	{
		adminGroup.GET("/clients", h.ListClients)
		adminGroup.POST("/clients", h.CreateClient)
		adminGroup.DELETE("/clients/:id", h.DeleteClient)
	}
}

// CreateUser
// POST /api/scim/v2/Users
func (h *Handler) CreateUser(c *gin.Context) {
	// Authentication: In production, check Bearer token against scim_clients
	// For reference, assume middleware or header Tenant-ID
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default" // Fallback for testing
	}

	var req User
	if err := c.ShouldBindJSON(&req); err != nil {
		h.error(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUser
// GET /api/scim/v2/Users/:id
func (h *Handler) GetUser(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	id := c.Param("id")

	user, err := h.service.GetUser(c.Request.Context(), tenantID, id)
	if err != nil {
		h.error(c, http.StatusNotFound, "notFound", "User not found")
		return
	}

	c.JSON(http.StatusOK, user)
}

// ListUsers
// GET /api/scim/v2/Users
func (h *Handler) ListUsers(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	startIndex := 0
	count := 100

	if s := c.Query("startIndex"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 1 {
			startIndex = v - 1
		}
	}
	if s := c.Query("count"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			count = v
		}
	}

	list, err := h.service.ListUsers(c.Request.Context(), tenantID, count, startIndex)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

// Stubs for other methods
func (h *Handler) ReplaceUser(c *gin.Context) { c.Status(http.StatusNotImplemented) }
func (h *Handler) PatchUser(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *Handler) DeleteUser(c *gin.Context)  { c.Status(http.StatusNotImplemented) }

// ServiceProviderConfig
func (h *Handler) ServiceProviderConfig(c *gin.Context) {
	// Minimal config
	c.JSON(http.StatusOK, gin.H{
		"schemas":        []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"patch":          gin.H{"supported": true},
		"bulk":           gin.H{"supported": false},
		"filter":         gin.H{"supported": false},
		"changePassword": gin.H{"supported": false},
		"sort":           gin.H{"supported": false},
		"etag":           gin.H{"supported": true},
		"authenticationSchemes": []gin.H{
			{"name": "OAuth Bearer Token", "description": "Authentication scheme using the OAuth Bearer Token Standard",
				"specUri": "https://www.rfc-editor.org/info/rfc6750", "type": "oauthbearertoken"},
		},
	})
}

// Error helper
func (h *Handler) error(c *gin.Context, status int, scimType, detail string) {
	c.JSON(status, ErrorResponse{
		Schemas:  []string{ErrorSchema},
		Status:   strconv.Itoa(status),
		ScimType: scimType,
		Detail:   detail,
	})
}

// ---------------------------------------------------------------------
// SCIM Admin API Handlers
// ---------------------------------------------------------------------

// getTenantID helper (duplicate of SAML one, ideally should be middleware context)
func (h *Handler) getTenantID(c *gin.Context) string {
	tid := c.GetHeader("X-Tenant-ID")
	if tid == "" {
		return "default"
	}
	return tid
}

// ListClients
func (h *Handler) ListClients(c *gin.Context) {
	clients, err := h.service.ListClients(c.Request.Context(), h.getTenantID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, clients)
}

// CreateClient
func (h *Handler) CreateClient(c *gin.Context) {
	var req struct {
		ClientName string `json:"clientName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.service.CreateClient(c.Request.Context(), h.getTenantID(c), req.ClientName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, client)
}

// DeleteClient
func (h *Handler) DeleteClient(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteClient(c.Request.Context(), h.getTenantID(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
