package admin

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mauriciomferz/Gauth_go/pkg/subscribers"
)

// SubscriberHandler handles subscriber management endpoints
type SubscriberHandler struct {
	repo *subscribers.Repository
}

// NewSubscriberHandler creates a new subscriber handler
func NewSubscriberHandler(db *pgxpool.Pool) *SubscriberHandler {
	return &SubscriberHandler{
		repo: subscribers.NewRepository(db),
	}
}

// SubscriberRequest represents subscriber creation/update request
type SubscriberRequest struct {
	// Basic Information
	TenantName   string `json:"tenantName" binding:"required"`
	TenantID     string `json:"tenantId" binding:"required"`
	ContactEmail string `json:"contactEmail" binding:"required,email"`
	ContactPhone string `json:"contactPhone"`

	// OIDC Configuration
	OIDCProvider     string `json:"oidcProvider" binding:"required"`
	OIDCClientID     string `json:"oidcClientId" binding:"required"`
	OIDCClientSecret string `json:"oidcClientSecret" binding:"required"`
	OIDCDiscoveryURL string `json:"oidcDiscoveryUrl" binding:"required"`

	// Key Generation
	KeyAlgorithm        string `json:"keyAlgorithm" binding:"required"`
	KeyRotationInterval string `json:"keyRotationInterval"`
	EnableAutoRotation  bool   `json:"enableAutoRotation"`

	// Policy Templates
	PolicyTemplate string `json:"policyTemplate" binding:"required"`
	CustomPolicies string `json:"customPolicies"`

	// Legal Framework
	Jurisdiction         string   `json:"jurisdiction" binding:"required"`
	ComplianceFrameworks []string `json:"complianceFrameworks"`
	DataRetention        string   `json:"dataRetention"`

	// Security Settings
	MFARequired     bool   `json:"mfaRequired"`
	TokenExpiration string `json:"tokenExpiration"`
	SessionTimeout  string `json:"sessionTimeout"`
	IPWhitelist     string `json:"ipWhitelist"`

	// Notification Preferences
	EmailNotifications bool     `json:"emailNotifications"`
	WebhookURL         string   `json:"webhookUrl"`
	NotificationEvents []string `json:"notificationEvents"`

	// Review & Confirm
	AgreedToTerms bool `json:"agreedToTerms" binding:"required"`
}

// Subscriber represents a registered tenant
type Subscriber struct {
	ID             string    `json:"id"`
	TenantName     string    `json:"tenantName"`
	TenantID       string    `json:"tenantId"`
	ContactEmail   string    `json:"contactEmail"`
	ContactPhone   string    `json:"contactPhone"`
	Status         string    `json:"status"` // active, suspended, pending
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
	TotalTokens    int64     `json:"totalTokens"`
	ActiveUsers    int64     `json:"activeUsers"`
	OIDCProvider   string    `json:"oidcProvider"`
	Jurisdiction   string    `json:"jurisdiction"`
	MFARequired    bool      `json:"mfaRequired"`
}

// SubscriberListResponse represents paginated subscriber list
type SubscriberListResponse struct {
	Subscribers []Subscriber `json:"subscribers"`
	Total       int          `json:"total"`
	Page        int          `json:"page"`
	PageSize    int          `json:"pageSize"`
}

// CreateSubscriber registers a new tenant/subscriber
func (h *SubscriberHandler) CreateSubscriber(c *gin.Context) {
	var req SubscriberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Check if tenant ID already exists
	exists, err := h.repo.CheckTenantIDExists(ctx, req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate tenant ID"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Tenant ID already exists"})
		return
	}

	// Create subscriber record
	subscriber := &subscribers.Subscriber{
		TenantName:       req.TenantName,
		TenantID:         req.TenantID,
		Status:           StatusActive,
		Tier:             "standard",
		OIDCProvider:     strPtr(req.OIDCProvider),
		OIDCClientID:     strPtr(req.OIDCClientID),
		OIDCClientSecret: strPtr(req.OIDCClientSecret),
		OIDCDiscoveryURL: strPtr(req.OIDCDiscoveryURL),
		KeyType:          strPtr(req.KeyAlgorithm),
		PolicyTemplate:   strPtr(req.PolicyTemplate),
		LegalFramework:   strPtr(req.Jurisdiction),
		ContactEmail:     strPtr(req.ContactEmail),
		MaxUsers:         100,
		MaxTokens:        1000,
		Metadata:         make(map[string]interface{}),
	}

	// Set notification preferences
	if req.EmailNotifications {
		subscriber.NotificationChannels = []string{"email"}
		subscriber.NotificationEmail = strPtr(req.ContactEmail)
	}
	if req.WebhookURL != "" {
		subscriber.NotificationChannels = append(subscriber.NotificationChannels, "webhook")
		subscriber.NotificationWebhookURL = strPtr(req.WebhookURL)
	}

	err = h.repo.CreateSubscriber(ctx, subscriber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscriber"})
		return
	}

	response := Subscriber{
		ID:             subscriber.ID.String(),
		TenantName:     subscriber.TenantName,
		TenantID:       subscriber.TenantID,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
		Status:         subscriber.Status,
		CreatedAt:      subscriber.CreatedAt,
		UpdatedAt:      subscriber.UpdatedAt,
		LastActivityAt: time.Now(),
		TotalTokens:    0,
		ActiveUsers:    0,
		OIDCProvider:   req.OIDCProvider,
		Jurisdiction:   req.Jurisdiction,
		MFARequired:    req.MFARequired,
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Subscriber created successfully",
		"subscriber": response,
	})
}

// ListSubscribers returns paginated list of subscribers
func (h *SubscriberHandler) ListSubscribers(c *gin.Context) {
	// Parse query parameters
	status := c.Query("status")
	tier := c.Query("tier")
	search := c.Query("search")

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	pageStr := c.DefaultQuery("page", "1")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	// If page is provided but offset is not, calculate offset from page
	if c.Query("page") != "" && c.Query("offset") == "" {
		offset = (page - 1) * limit
	}

	filters := subscribers.SubscriberFilters{
		Status: status,
		Tier:   tier,
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	dbSubscribers, total, err := h.repo.ListSubscribers(c.Request.Context(), filters)
	if err != nil {
		fmt.Printf("[subscribers] ListSubscribers error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list subscribers", "details": err.Error()})
		return
	}

	// Convert to response format
	responseSubscribers := make([]Subscriber, len(dbSubscribers))
	for i, dbSub := range dbSubscribers {
		lastActivity := time.Time{}
		if dbSub.LastActivityAt != nil {
			lastActivity = *dbSub.LastActivityAt
		}

		contactEmail := ""
		if dbSub.ContactEmail != nil {
			contactEmail = *dbSub.ContactEmail
		}

		oidcProvider := ""
		if dbSub.OIDCProvider != nil {
			oidcProvider = *dbSub.OIDCProvider
		}

		jurisdiction := ""
		if dbSub.LegalFramework != nil {
			jurisdiction = *dbSub.LegalFramework
		}

		responseSubscribers[i] = Subscriber{
			ID:             dbSub.ID.String(),
			TenantName:     dbSub.TenantName,
			TenantID:       dbSub.TenantID,
			ContactEmail:   contactEmail,
			ContactPhone:   "",
			Status:         dbSub.Status,
			CreatedAt:      dbSub.CreatedAt,
			UpdatedAt:      dbSub.UpdatedAt,
			LastActivityAt: lastActivity,
			TotalTokens:    dbSub.TotalTokens,
			ActiveUsers:    dbSub.ActiveUsers,
			OIDCProvider:   oidcProvider,
			Jurisdiction:   jurisdiction,
			MFARequired:    false,
		}
	}

	c.JSON(http.StatusOK, SubscriberListResponse{
		Subscribers: responseSubscribers,
		Total:       total,
		Page:        page,
		PageSize:    limit,
	})
}

// GetSubscriber returns detailed subscriber information
func (h *SubscriberHandler) GetSubscriber(c *gin.Context) {
	subscriberID := c.Param("id")

	dbSub, err := h.repo.GetSubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subscriber"})
		return
	}

	lastActivity := time.Time{}
	if dbSub.LastActivityAt != nil {
		lastActivity = *dbSub.LastActivityAt
	}

	contactEmail := ""
	if dbSub.ContactEmail != nil {
		contactEmail = *dbSub.ContactEmail
	}

	oidcProvider := ""
	if dbSub.OIDCProvider != nil {
		oidcProvider = *dbSub.OIDCProvider
	}

	jurisdiction := ""
	if dbSub.LegalFramework != nil {
		jurisdiction = *dbSub.LegalFramework
	}

	subscriber := Subscriber{
		ID:             dbSub.ID.String(),
		TenantName:     dbSub.TenantName,
		TenantID:       dbSub.TenantID,
		ContactEmail:   contactEmail,
		ContactPhone:   "",
		Status:         dbSub.Status,
		CreatedAt:      dbSub.CreatedAt,
		UpdatedAt:      dbSub.UpdatedAt,
		LastActivityAt: lastActivity,
		TotalTokens:    dbSub.TotalTokens,
		ActiveUsers:    dbSub.ActiveUsers,
		OIDCProvider:   oidcProvider,
		Jurisdiction:   jurisdiction,
		MFARequired:    false,
	}

	c.JSON(http.StatusOK, subscriber)
}

// UpdateSubscriber updates subscriber configuration
func (h *SubscriberHandler) UpdateSubscriber(c *gin.Context) {
	subscriberID := c.Param("id")

	var req SubscriberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing subscriber
	existing, err := h.repo.GetSubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subscriber"})
		return
	}

	// Update fields
	existing.TenantName = req.TenantName
	existing.OIDCProvider = strPtr(req.OIDCProvider)
	existing.OIDCClientID = strPtr(req.OIDCClientID)
	existing.OIDCClientSecret = strPtr(req.OIDCClientSecret)
	existing.OIDCDiscoveryURL = strPtr(req.OIDCDiscoveryURL)
	existing.ContactEmail = strPtr(req.ContactEmail)
	existing.LegalFramework = strPtr(req.Jurisdiction)

	// Update notification settings
	if req.EmailNotifications {
		existing.NotificationChannels = []string{"email"}
		existing.NotificationEmail = strPtr(req.ContactEmail)
	}
	if req.WebhookURL != "" {
		existing.NotificationChannels = append(existing.NotificationChannels, "webhook")
		existing.NotificationWebhookURL = strPtr(req.WebhookURL)
	}

	err = h.repo.UpdateSubscriber(c.Request.Context(), subscriberID, existing)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscriber"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscriber updated successfully",
		"id":      subscriberID,
	})
}

// DeleteSubscriber soft-deletes or suspends a subscriber
func (h *SubscriberHandler) DeleteSubscriber(c *gin.Context) {
	subscriberID := c.Param("id")

	err := h.repo.DeleteSubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subscriber"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscriber suspended successfully",
		"id":      subscriberID,
	})
}

// RotateKeys triggers manual key rotation for a subscriber
func (h *SubscriberHandler) RotateKeys(c *gin.Context) {
	subscriberID := c.Param("id")

	// Get subscriber to get tenant_id
	subscriber, err := h.repo.GetSubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subscriber"})
		return
	}

	// TODO: Generate new keys via crypto service
	// For now, just update metadata to simulate rotation
	newKeyID := fmt.Sprintf("key_%s_%d", subscriber.TenantID, time.Now().Unix())
	expiresAt := time.Now().Add(90 * 24 * time.Hour) // 90 days

	err = h.repo.UpdateKeyMetadata(c.Request.Context(), subscriber.TenantID, "RSA", "[public-key-data]", newKeyID, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update key metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Key rotation initiated",
		"id":         subscriberID,
		"new_key_id": newKeyID,
		"expires_at": expiresAt,
	})
}

// GetSubscriberMetrics returns subscriber-specific metrics
func (h *SubscriberHandler) GetSubscriberMetrics(c *gin.Context) {
	subscriberID := c.Param("id")

	// Get subscriber to get tenant_id
	subscriber, err := h.repo.GetSubscriber(c.Request.Context(), subscriberID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subscriber"})
		return
	}

	metrics, err := h.repo.GetSubscriberMetrics(c.Request.Context(), subscriber.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics"})
		return
	}

	// Add computed fields
	metrics["subscriberId"] = subscriberID
	metrics["avgLatency"] = 85.3   // TODO: Calculate from audit events
	metrics["errorRate"] = 0.0011  // TODO: Calculate from audit events
	metrics["dataTransferred"] = 0 // TODO: Track data transfer

	c.JSON(http.StatusOK, metrics)
}

// RegisterRoutes registers subscriber management routes
func (h *SubscriberHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/subscribers", h.CreateSubscriber)
	router.GET("/subscribers", h.ListSubscribers)
	router.GET("/subscribers/:id", h.GetSubscriber)
	router.PUT("/subscribers/:id", h.UpdateSubscriber)
	router.DELETE("/subscribers/:id", h.DeleteSubscriber)
	router.POST("/subscribers/:id/rotate-keys", h.RotateKeys)
	router.GET("/subscribers/:id/metrics", h.GetSubscriberMetrics)
}
