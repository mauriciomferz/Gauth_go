package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OIDCHandler handles OIDC provider configuration endpoints
type OIDCHandler struct {
	db *pgxpool.Pool
}

// NewOIDCHandler creates a new OIDC handler
func NewOIDCHandler(db *pgxpool.Pool) *OIDCHandler {
	return &OIDCHandler{db: db}
}

// OIDCProvider represents an OpenID Connect provider configuration
type OIDCProvider struct {
	ID                       string    `json:"id"`
	TenantID                 string    `json:"tenantId"`
	ProviderName             string    `json:"providerName"`
	ProviderType             string    `json:"providerType"`
	DisplayName              string    `json:"displayName"`
	IssuerURL                string    `json:"issuerUrl"`
	AuthorizationEndpoint    *string   `json:"authorizationEndpoint"`
	TokenEndpoint            *string   `json:"tokenEndpoint"`
	UserinfoEndpoint         *string   `json:"userinfoEndpoint"`
	JwksURI                  *string   `json:"jwksUri"`
	EndSessionEndpoint       *string   `json:"endSessionEndpoint"`
	ClientID                 string    `json:"clientId"`
	ClientSecret             *string   `json:"clientSecret,omitempty"`
	Scopes                   []string  `json:"scopes"`
	ClaimsMapping            map[string]interface{} `json:"claimsMapping"`
	RedirectURIs             []string  `json:"redirectUris"`
	PostLogoutRedirectURIs   []string  `json:"postLogoutRedirectUris"`
	ValidateIssuer           bool      `json:"validateIssuer"`
	ValidateAudience         bool      `json:"validateAudience"`
	ValidateSignature        bool      `json:"validateSignature"`
	ClockSkewSeconds         int       `json:"clockSkewSeconds"`
	AutoProvisionUsers       bool      `json:"autoProvisionUsers"`
	UserAttributeMapping     map[string]interface{} `json:"userAttributeMapping"`
	DefaultRole              *string   `json:"defaultRole"`
	PKCEEnabled              bool      `json:"pkceEnabled"`
	ResponseType             string    `json:"responseType"`
	ResponseMode             string    `json:"responseMode"`
	Prompt                   *string   `json:"prompt"`
	MaxAge                   *int      `json:"maxAge"`
	AzureTenantID            *string   `json:"azureTenantId"`
	AzureResource            *string   `json:"azureResource"`
	AdditionalParams         map[string]interface{} `json:"additionalParams"`
	Status                   string    `json:"status"`
	IsDefault                bool      `json:"isDefault"`
	Priority                 int       `json:"priority"`
	CreatedBy                string    `json:"createdBy"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
	UpdatedBy                *string   `json:"updatedBy"`
	LastSyncAt               *time.Time `json:"lastSyncAt"`
	ConfigValid              bool      `json:"configValid"`
	ValidationErrors         []string  `json:"validationErrors"`
	LastValidatedAt          *time.Time `json:"lastValidatedAt"`
}

// CreateOIDCProviderRequest represents a request to create an OIDC provider
type CreateOIDCProviderRequest struct {
	ProviderName           string                 `json:"providerName" binding:"required"`
	ProviderType           string                 `json:"providerType" binding:"required"`
	DisplayName            string                 `json:"displayName" binding:"required"`
	IssuerURL              string                 `json:"issuerUrl" binding:"required"`
	AuthorizationEndpoint  *string                `json:"authorizationEndpoint"`
	TokenEndpoint          *string                `json:"tokenEndpoint"`
	UserinfoEndpoint       *string                `json:"userinfoEndpoint"`
	JwksURI                *string                `json:"jwksUri"`
	EndSessionEndpoint     *string                `json:"endSessionEndpoint"`
	ClientID               string                 `json:"clientId" binding:"required"`
	ClientSecret           string                 `json:"clientSecret" binding:"required"`
	Scopes                 []string               `json:"scopes"`
	ClaimsMapping          map[string]interface{} `json:"claimsMapping"`
	RedirectURIs           []string               `json:"redirectUris" binding:"required"`
	PostLogoutRedirectURIs []string               `json:"postLogoutRedirectUris"`
	ValidateIssuer         *bool                  `json:"validateIssuer"`
	ValidateAudience       *bool                  `json:"validateAudience"`
	ValidateSignature      *bool                  `json:"validateSignature"`
	ClockSkewSeconds       *int                   `json:"clockSkewSeconds"`
	AutoProvisionUsers     *bool                  `json:"autoProvisionUsers"`
	UserAttributeMapping   map[string]interface{} `json:"userAttributeMapping"`
	DefaultRole            *string                `json:"defaultRole"`
	PKCEEnabled            *bool                  `json:"pkceEnabled"`
	ResponseType           *string                `json:"responseType"`
	ResponseMode           *string                `json:"responseMode"`
	Prompt                 *string                `json:"prompt"`
	MaxAge                 *int                   `json:"maxAge"`
	AzureTenantID          *string                `json:"azureTenantId"`
	AzureResource          *string                `json:"azureResource"`
	AdditionalParams       map[string]interface{} `json:"additionalParams"`
	IsDefault              *bool                  `json:"isDefault"`
	Priority               *int                   `json:"priority"`
}

// UpdateOIDCProviderRequest represents a request to update an OIDC provider
type UpdateOIDCProviderRequest struct {
	DisplayName            *string                `json:"displayName"`
	IssuerURL              *string                `json:"issuerUrl"`
	AuthorizationEndpoint  *string                `json:"authorizationEndpoint"`
	TokenEndpoint          *string                `json:"tokenEndpoint"`
	UserinfoEndpoint       *string                `json:"userinfoEndpoint"`
	JwksURI                *string                `json:"jwksUri"`
	EndSessionEndpoint     *string                `json:"endSessionEndpoint"`
	ClientID               *string                `json:"clientId"`
	ClientSecret           *string                `json:"clientSecret"`
	Scopes                 []string               `json:"scopes"`
	ClaimsMapping          map[string]interface{} `json:"claimsMapping"`
	RedirectURIs           []string               `json:"redirectUris"`
	PostLogoutRedirectURIs []string               `json:"postLogoutRedirectUris"`
	ValidateIssuer         *bool                  `json:"validateIssuer"`
	ValidateAudience       *bool                  `json:"validateAudience"`
	ValidateSignature      *bool                  `json:"validateSignature"`
	ClockSkewSeconds       *int                   `json:"clockSkewSeconds"`
	AutoProvisionUsers     *bool                  `json:"autoProvisionUsers"`
	UserAttributeMapping   map[string]interface{} `json:"userAttributeMapping"`
	DefaultRole            *string                `json:"defaultRole"`
	PKCEEnabled            *bool                  `json:"pkceEnabled"`
	ResponseType           *string                `json:"responseType"`
	ResponseMode           *string                `json:"responseMode"`
	Prompt                 *string                `json:"prompt"`
	MaxAge                 *int                   `json:"maxAge"`
	AzureTenantID          *string                `json:"azureTenantId"`
	AzureResource          *string                `json:"azureResource"`
	AdditionalParams       map[string]interface{} `json:"additionalParams"`
	Status                 *string                `json:"status"`
	IsDefault              *bool                  `json:"isDefault"`
	Priority               *int                   `json:"priority"`
}

// ListOIDCProviders returns all OIDC providers for a tenant
func (h *OIDCHandler) ListOIDCProviders(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		if val, exists := c.Get("tenant_id"); exists {
			tenantID, _ = val.(string)
		}
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	ctx := c.Request.Context()
	query := `
		SELECT 
			id, tenant_id, provider_name, provider_type, display_name,
			issuer_url, authorization_endpoint, token_endpoint, userinfo_endpoint,
			jwks_uri, end_session_endpoint, client_id,
			scopes, claims_mapping, redirect_uris, post_logout_redirect_uris,
			validate_issuer, validate_audience, validate_signature, clock_skew_seconds,
			auto_provision_users, user_attribute_mapping, default_role,
			pkce_enabled, response_type, response_mode, prompt, max_age,
			azure_tenant_id, azure_resource, additional_params,
			status, is_default, priority,
			created_by, created_at, updated_at, updated_by,
			last_sync_at, config_valid, validation_errors, last_validated_at
		FROM oidc_providers
		WHERE tenant_id = $1
		ORDER BY priority ASC, created_at DESC
	`

	rows, err := h.db.Query(ctx, query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch providers"})
		return
	}
	defer rows.Close()

	providers := []OIDCProvider{}
	for rows.Next() {
		var p OIDCProvider
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.ProviderName, &p.ProviderType, &p.DisplayName,
			&p.IssuerURL, &p.AuthorizationEndpoint, &p.TokenEndpoint, &p.UserinfoEndpoint,
			&p.JwksURI, &p.EndSessionEndpoint, &p.ClientID,
			&p.Scopes, &p.ClaimsMapping, &p.RedirectURIs, &p.PostLogoutRedirectURIs,
			&p.ValidateIssuer, &p.ValidateAudience, &p.ValidateSignature, &p.ClockSkewSeconds,
			&p.AutoProvisionUsers, &p.UserAttributeMapping, &p.DefaultRole,
			&p.PKCEEnabled, &p.ResponseType, &p.ResponseMode, &p.Prompt, &p.MaxAge,
			&p.AzureTenantID, &p.AzureResource, &p.AdditionalParams,
			&p.Status, &p.IsDefault, &p.Priority,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.UpdatedBy,
			&p.LastSyncAt, &p.ConfigValid, &p.ValidationErrors, &p.LastValidatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse provider data"})
			return
		}
		// Don't send client secret in list response
		p.ClientSecret = nil
		providers = append(providers, p)
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"total":     len(providers),
	})
}

// GetOIDCProvider returns a specific OIDC provider
func (h *OIDCHandler) GetOIDCProvider(c *gin.Context) {
	providerID := c.Param("id")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		if val, exists := c.Get("tenant_id"); exists {
			tenantID, _ = val.(string)
		}
	}

	ctx := c.Request.Context()
	query := `
		SELECT 
			id, tenant_id, provider_name, provider_type, display_name,
			issuer_url, authorization_endpoint, token_endpoint, userinfo_endpoint,
			jwks_uri, end_session_endpoint, client_id,
			scopes, claims_mapping, redirect_uris, post_logout_redirect_uris,
			validate_issuer, validate_audience, validate_signature, clock_skew_seconds,
			auto_provision_users, user_attribute_mapping, default_role,
			pkce_enabled, response_type, response_mode, prompt, max_age,
			azure_tenant_id, azure_resource, additional_params,
			status, is_default, priority,
			created_by, created_at, updated_at, updated_by,
			last_sync_at, config_valid, validation_errors, last_validated_at
		FROM oidc_providers
		WHERE id = $1 AND tenant_id = $2
	`

	var p OIDCProvider
	err := h.db.QueryRow(ctx, query, providerID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.ProviderName, &p.ProviderType, &p.DisplayName,
		&p.IssuerURL, &p.AuthorizationEndpoint, &p.TokenEndpoint, &p.UserinfoEndpoint,
		&p.JwksURI, &p.EndSessionEndpoint, &p.ClientID,
		&p.Scopes, &p.ClaimsMapping, &p.RedirectURIs, &p.PostLogoutRedirectURIs,
		&p.ValidateIssuer, &p.ValidateAudience, &p.ValidateSignature, &p.ClockSkewSeconds,
		&p.AutoProvisionUsers, &p.UserAttributeMapping, &p.DefaultRole,
		&p.PKCEEnabled, &p.ResponseType, &p.ResponseMode, &p.Prompt, &p.MaxAge,
		&p.AzureTenantID, &p.AzureResource, &p.AdditionalParams,
		&p.Status, &p.IsDefault, &p.Priority,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.UpdatedBy,
		&p.LastSyncAt, &p.ConfigValid, &p.ValidationErrors, &p.LastValidatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch provider"})
		return
	}

	// Don't send client secret in response
	p.ClientSecret = nil

	c.JSON(http.StatusOK, p)
}

// CreateOIDCProvider creates a new OIDC provider
func (h *OIDCHandler) CreateOIDCProvider(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		if val, exists := c.Get("tenant_id"); exists {
			tenantID, _ = val.(string)
		}
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req CreateOIDCProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Scopes == nil || len(req.Scopes) == 0 {
		req.Scopes = []string{"openid", "profile", "email"}
	}
	if req.ClaimsMapping == nil {
		req.ClaimsMapping = make(map[string]interface{})
	}
	if req.UserAttributeMapping == nil {
		req.UserAttributeMapping = make(map[string]interface{})
	}
	if req.AdditionalParams == nil {
		req.AdditionalParams = make(map[string]interface{})
	}
	if req.PostLogoutRedirectURIs == nil {
		req.PostLogoutRedirectURIs = []string{}
	}

	validateIssuer := true
	if req.ValidateIssuer != nil {
		validateIssuer = *req.ValidateIssuer
	}
	validateAudience := true
	if req.ValidateAudience != nil {
		validateAudience = *req.ValidateAudience
	}
	validateSignature := true
	if req.ValidateSignature != nil {
		validateSignature = *req.ValidateSignature
	}
	clockSkewSeconds := 300
	if req.ClockSkewSeconds != nil {
		clockSkewSeconds = *req.ClockSkewSeconds
	}
	autoProvisionUsers := true
	if req.AutoProvisionUsers != nil {
		autoProvisionUsers = *req.AutoProvisionUsers
	}
	pkceEnabled := true
	if req.PKCEEnabled != nil {
		pkceEnabled = *req.PKCEEnabled
	}
	responseType := "code"
	if req.ResponseType != nil {
		responseType = *req.ResponseType
	}
	responseMode := "query"
	if req.ResponseMode != nil {
		responseMode = *req.ResponseMode
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	id := uuid.New()
	ctx := c.Request.Context()

	query := `
		INSERT INTO oidc_providers (
			id, tenant_id, provider_name, provider_type, display_name,
			issuer_url, authorization_endpoint, token_endpoint, userinfo_endpoint,
			jwks_uri, end_session_endpoint, client_id, client_secret,
			scopes, claims_mapping, redirect_uris, post_logout_redirect_uris,
			validate_issuer, validate_audience, validate_signature, clock_skew_seconds,
			auto_provision_users, user_attribute_mapping, default_role,
			pkce_enabled, response_type, response_mode, prompt, max_age,
			azure_tenant_id, azure_resource, additional_params,
			is_default, priority, status, created_by
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $19, $20, $21,
			$22, $23, $24,
			$25, $26, $27, $28, $29,
			$30, $31, $32,
			$33, $34, $35, $36
		) RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := h.db.QueryRow(ctx, query,
		id, tenantID, req.ProviderName, req.ProviderType, req.DisplayName,
		req.IssuerURL, req.AuthorizationEndpoint, req.TokenEndpoint, req.UserinfoEndpoint,
		req.JwksURI, req.EndSessionEndpoint, req.ClientID, req.ClientSecret,
		req.Scopes, req.ClaimsMapping, req.RedirectURIs, req.PostLogoutRedirectURIs,
		validateIssuer, validateAudience, validateSignature, clockSkewSeconds,
		autoProvisionUsers, req.UserAttributeMapping, req.DefaultRole,
		pkceEnabled, responseType, responseMode, req.Prompt, req.MaxAge,
		req.AzureTenantID, req.AzureResource, req.AdditionalParams,
		isDefault, priority, "active", "admin",
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Provider created successfully",
		"providerId": id.String(),
		"createdAt":  createdAt,
	})
}

// UpdateOIDCProvider updates an existing OIDC provider
func (h *OIDCHandler) UpdateOIDCProvider(c *gin.Context) {
	providerID := c.Param("id")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		if val, exists := c.Get("tenant_id"); exists {
			tenantID, _ = val.(string)
		}
	}

	var req UpdateOIDCProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Build dynamic update query
	query := `UPDATE oidc_providers SET `
	params := []interface{}{}
	paramIndex := 1

	if req.DisplayName != nil {
		query += fmt.Sprintf("display_name = $%d, ", paramIndex)
		params = append(params, *req.DisplayName)
		paramIndex++
	}
	if req.IssuerURL != nil {
		query += fmt.Sprintf("issuer_url = $%d, ", paramIndex)
		params = append(params, *req.IssuerURL)
		paramIndex++
	}
	if req.AuthorizationEndpoint != nil {
		query += fmt.Sprintf("authorization_endpoint = $%d, ", paramIndex)
		params = append(params, *req.AuthorizationEndpoint)
		paramIndex++
	}
	if req.TokenEndpoint != nil {
		query += fmt.Sprintf("token_endpoint = $%d, ", paramIndex)
		params = append(params, *req.TokenEndpoint)
		paramIndex++
	}
	if req.UserinfoEndpoint != nil {
		query += fmt.Sprintf("userinfo_endpoint = $%d, ", paramIndex)
		params = append(params, *req.UserinfoEndpoint)
		paramIndex++
	}
	if req.JwksURI != nil {
		query += fmt.Sprintf("jwks_uri = $%d, ", paramIndex)
		params = append(params, *req.JwksURI)
		paramIndex++
	}
	if req.EndSessionEndpoint != nil {
		query += fmt.Sprintf("end_session_endpoint = $%d, ", paramIndex)
		params = append(params, *req.EndSessionEndpoint)
		paramIndex++
	}
	if req.ClientID != nil {
		query += fmt.Sprintf("client_id = $%d, ", paramIndex)
		params = append(params, *req.ClientID)
		paramIndex++
	}
	if req.ClientSecret != nil {
		query += fmt.Sprintf("client_secret = $%d, ", paramIndex)
		params = append(params, *req.ClientSecret)
		paramIndex++
	}
	if req.Scopes != nil {
		query += fmt.Sprintf("scopes = $%d, ", paramIndex)
		params = append(params, req.Scopes)
		paramIndex++
	}
	if req.ClaimsMapping != nil {
		query += fmt.Sprintf("claims_mapping = $%d, ", paramIndex)
		params = append(params, req.ClaimsMapping)
		paramIndex++
	}
	if req.RedirectURIs != nil {
		query += fmt.Sprintf("redirect_uris = $%d, ", paramIndex)
		params = append(params, req.RedirectURIs)
		paramIndex++
	}
	if req.PostLogoutRedirectURIs != nil {
		query += fmt.Sprintf("post_logout_redirect_uris = $%d, ", paramIndex)
		params = append(params, req.PostLogoutRedirectURIs)
		paramIndex++
	}
	if req.ValidateIssuer != nil {
		query += fmt.Sprintf("validate_issuer = $%d, ", paramIndex)
		params = append(params, *req.ValidateIssuer)
		paramIndex++
	}
	if req.ValidateAudience != nil {
		query += fmt.Sprintf("validate_audience = $%d, ", paramIndex)
		params = append(params, *req.ValidateAudience)
		paramIndex++
	}
	if req.ValidateSignature != nil {
		query += fmt.Sprintf("validate_signature = $%d, ", paramIndex)
		params = append(params, *req.ValidateSignature)
		paramIndex++
	}
	if req.ClockSkewSeconds != nil {
		query += fmt.Sprintf("clock_skew_seconds = $%d, ", paramIndex)
		params = append(params, *req.ClockSkewSeconds)
		paramIndex++
	}
	if req.AutoProvisionUsers != nil {
		query += fmt.Sprintf("auto_provision_users = $%d, ", paramIndex)
		params = append(params, *req.AutoProvisionUsers)
		paramIndex++
	}
	if req.UserAttributeMapping != nil {
		query += fmt.Sprintf("user_attribute_mapping = $%d, ", paramIndex)
		params = append(params, req.UserAttributeMapping)
		paramIndex++
	}
	if req.DefaultRole != nil {
		query += fmt.Sprintf("default_role = $%d, ", paramIndex)
		params = append(params, *req.DefaultRole)
		paramIndex++
	}
	if req.PKCEEnabled != nil {
		query += fmt.Sprintf("pkce_enabled = $%d, ", paramIndex)
		params = append(params, *req.PKCEEnabled)
		paramIndex++
	}
	if req.ResponseType != nil {
		query += fmt.Sprintf("response_type = $%d, ", paramIndex)
		params = append(params, *req.ResponseType)
		paramIndex++
	}
	if req.ResponseMode != nil {
		query += fmt.Sprintf("response_mode = $%d, ", paramIndex)
		params = append(params, *req.ResponseMode)
		paramIndex++
	}
	if req.Prompt != nil {
		query += fmt.Sprintf("prompt = $%d, ", paramIndex)
		params = append(params, *req.Prompt)
		paramIndex++
	}
	if req.MaxAge != nil {
		query += fmt.Sprintf("max_age = $%d, ", paramIndex)
		params = append(params, *req.MaxAge)
		paramIndex++
	}
	if req.AzureTenantID != nil {
		query += fmt.Sprintf("azure_tenant_id = $%d, ", paramIndex)
		params = append(params, *req.AzureTenantID)
		paramIndex++
	}
	if req.AzureResource != nil {
		query += fmt.Sprintf("azure_resource = $%d, ", paramIndex)
		params = append(params, *req.AzureResource)
		paramIndex++
	}
	if req.AdditionalParams != nil {
		query += fmt.Sprintf("additional_params = $%d, ", paramIndex)
		params = append(params, req.AdditionalParams)
		paramIndex++
	}
	if req.Status != nil {
		query += fmt.Sprintf("status = $%d, ", paramIndex)
		params = append(params, *req.Status)
		paramIndex++
	}
	if req.IsDefault != nil {
		query += fmt.Sprintf("is_default = $%d, ", paramIndex)
		params = append(params, *req.IsDefault)
		paramIndex++
	}
	if req.Priority != nil {
		query += fmt.Sprintf("priority = $%d, ", paramIndex)
		params = append(params, *req.Priority)
		paramIndex++
	}

	if len(params) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add updated_by and WHERE clause
	query += fmt.Sprintf("updated_by = $%d, updated_at = CURRENT_TIMESTAMP WHERE id = $%d AND tenant_id = $%d", paramIndex, paramIndex+1, paramIndex+2)
	params = append(params, "admin", providerID, tenantID)

	result, err := h.db.Exec(ctx, query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider", "details": err.Error()})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider updated successfully"})
}

// DeleteOIDCProvider deletes an OIDC provider
func (h *OIDCHandler) DeleteOIDCProvider(c *gin.Context) {
	providerID := c.Param("id")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		if val, exists := c.Get("tenant_id"); exists {
			tenantID, _ = val.(string)
		}
	}

	ctx := c.Request.Context()
	query := `DELETE FROM oidc_providers WHERE id = $1 AND tenant_id = $2`

	result, err := h.db.Exec(ctx, query, providerID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted successfully"})
}

// TestOIDCProvider tests OIDC provider connectivity
func (h *OIDCHandler) TestOIDCProvider(c *gin.Context) {
	providerID := c.Param("id")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		if val, exists := c.Get("tenant_id"); exists {
			tenantID, _ = val.(string)
		}
	}

	// Get provider details
	ctx := c.Request.Context()
	query := `SELECT issuer_url, client_id FROM oidc_providers WHERE id = $1 AND tenant_id = $2`
	
	var issuerURL, clientID string
	err := h.db.QueryRow(ctx, query, providerID, tenantID).Scan(&issuerURL, &clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Perform OIDC discovery
	discoveryResult, err := h.discoverOIDCProvider(issuerURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"message":    "Provider validation failed",
			"providerId": providerID,
			"tenantId":   tenantID,
			"error":      err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Provider configuration is valid",
		"providerId": providerID,
		"tenantId":   tenantID,
		"details":    discoveryResult,
	})
}

// OIDCDiscoveryDocument represents the OIDC discovery response
type OIDCDiscoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	EndSessionEndpoint                string   `json:"end_session_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

// discoverOIDCProvider performs OIDC discovery for a provider
func (h *OIDCHandler) discoverOIDCProvider(issuerURL string) (map[string]interface{}, error) {
	// Build discovery URL
	discoveryURL := issuerURL
	if discoveryURL[len(discoveryURL)-1] != '/' {
		discoveryURL += "/"
	}
	discoveryURL += ".well-known/openid-configuration"

	// Fetch discovery document
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned status %d", resp.StatusCode)
	}

	// Parse discovery document
	var discovery OIDCDiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	// Validate required fields
	if discovery.Issuer == "" {
		return nil, fmt.Errorf("missing required field: issuer")
	}
	if discovery.AuthorizationEndpoint == "" {
		return nil, fmt.Errorf("missing required field: authorization_endpoint")
	}
	if discovery.TokenEndpoint == "" {
		return nil, fmt.Errorf("missing required field: token_endpoint")
	}
	if discovery.JwksURI == "" {
		return nil, fmt.Errorf("missing required field: jwks_uri")
	}

	// Test JWKS endpoint
	jwksResp, err := client.Get(discovery.JwksURI)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer jwksResp.Body.Close()

	if jwksResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", jwksResp.StatusCode)
	}

	// Return validation results
	return map[string]interface{}{
		"discovery": "success",
		"endpoints": map[string]string{
			"issuer":        discovery.Issuer,
			"authorization": discovery.AuthorizationEndpoint,
			"token":         discovery.TokenEndpoint,
			"userinfo":      discovery.UserinfoEndpoint,
			"jwks":          discovery.JwksURI,
			"endSession":    discovery.EndSessionEndpoint,
		},
		"capabilities": map[string]interface{}{
			"responseTypes":        discovery.ResponseTypesSupported,
			"scopes":               discovery.ScopesSupported,
			"pkceSupported":        len(discovery.CodeChallengeMethodsSupported) > 0,
			"pkceMethod":           discovery.CodeChallengeMethodsSupported,
		},
		"jwks": "valid",
	}, nil
}

// RegisterRoutes registers OIDC provider routes
func (h *OIDCHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/oidc-providers", h.ListOIDCProviders)
	router.GET("/oidc-providers/:id", h.GetOIDCProvider)
	router.POST("/oidc-providers", h.CreateOIDCProvider)
	router.PUT("/oidc-providers/:id", h.UpdateOIDCProvider)
	router.DELETE("/oidc-providers/:id", h.DeleteOIDCProvider)
	router.POST("/oidc-providers/:id/test", h.TestOIDCProvider)
}
