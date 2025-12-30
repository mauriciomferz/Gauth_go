package saml

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler manages SAML HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new SAML handler
func NewHandler(db *pgxpool.Pool) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service}
}

// RegisterRoutes registers SAML endpoints
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	samlGroup := router.Group("/saml")
	{
		samlGroup.GET("/login/:providerId", h.InitiateLogin)
		samlGroup.POST("/acs/:providerId", h.AssertionConsumerService)
		samlGroup.GET("/metadata/:providerId", h.Metadata)

		// CRUD Endpoints
		samlGroup.GET("/providers", h.ListProviders)
		samlGroup.POST("/providers", h.CreateProvider)
		samlGroup.GET("/providers/:id", h.GetProvider)
		samlGroup.PUT("/providers/:id", h.UpdateProvider)
		samlGroup.DELETE("/providers/:id", h.DeleteProvider)
	}
}

// getTenantID helper
func (h *Handler) getTenantID(c *gin.Context) string {
	tid := c.GetHeader("X-Tenant-ID")
	if tid == "" {
		return "default"
	}
	return tid
}

// ListProviders returns all providers
func (h *Handler) ListProviders(c *gin.Context) {
	providers, err := h.service.ListProviders(c.Request.Context(), h.getTenantID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, providers)
}

// GetProvider returns a specific provider
func (h *Handler) GetProvider(c *gin.Context) {
	id := c.Param("id")
	provider, err := h.service.GetProvider(c.Request.Context(), h.getTenantID(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}
	c.JSON(http.StatusOK, provider)
}

// CreateProvider creates a new provider
func (h *Handler) CreateProvider(c *gin.Context) {
	var p SAMLProvider
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.TenantID = h.getTenantID(c)
	// Audit/User tracking could go here
	p.CreatedBy = pointer("admin")

	if err := h.service.CreateProvider(c.Request.Context(), &p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// UpdateProvider updates a provider
func (h *Handler) UpdateProvider(c *gin.Context) {
	id := c.Param("id")
	var p SAMLProvider
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.ID = id
	p.TenantID = h.getTenantID(c)
	p.UpdatedBy = pointer("admin")

	if err := h.service.UpdateProvider(c.Request.Context(), &p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

// DeleteProvider deletes a provider
func (h *Handler) DeleteProvider(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteProvider(c.Request.Context(), h.getTenantID(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func pointer(s string) *string {
	return &s
}

// InitiateLogin redirects to the IdP
// GET /api/saml/login/:providerId?tenant_id=...
func (h *Handler) InitiateLogin(c *gin.Context) {
	providerID := c.Param("providerId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	redirectURL, err := h.service.BuildAuthnRequest(c.Request.Context(), tenantID, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, redirectURL)
}

// AssertionConsumerService processes the IdP response
// POST /api/saml/acs/:providerId
func (h *Handler) AssertionConsumerService(c *gin.Context) {
	providerID := c.Param("providerId")
	// In real ACS, tenant_id might be inferred from RelayState or ProviderID lookup
	// For reference, we'll try to get it from query or header, or assume context
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		// Fallback: look up provider to get tenant (TODO: implement LookupProviderByID without tenant)
		// For now, require it or error
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required (or encoded in RelayState)"})
		return
	}

	samlResponse := c.PostForm("SAMLResponse")
	if samlResponse == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SAMLResponse missing"})
		return
	}

	identity, err := h.service.ParseResponse(c.Request.Context(), tenantID, providerID, samlResponse)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid SAML Response", "details": err.Error()})
		return
	}

	// Login successful - in real App, issue JWT here
	c.JSON(http.StatusOK, gin.H{
		"message": "SAML Login Successful",
		"user":    identity,
	})
}

// Metadata returns the SP metadata XML
// GET /api/saml/metadata/:providerId
func (h *Handler) Metadata(c *gin.Context) {
	providerID := c.Param("providerId")
	// Generate static metadata for this SP
	// Real impl: Sign this XML
	metadata := fmt.Sprintf(`
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://agentauth.example.com/api/saml/acs/%s">
  <md:SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true" `+
		`protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" `+
		`Location="https://agentauth.example.com/api/saml/acs/%s" index="1"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>`, providerID, providerID)

	c.Data(http.StatusOK, "application/xml", []byte(metadata))
}
