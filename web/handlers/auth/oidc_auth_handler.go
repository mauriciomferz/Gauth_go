package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OIDCAuthHandler handles OIDC authentication flow
type OIDCAuthHandler struct {
	db *pgxpool.Pool
}

// NewOIDCAuthHandler creates a new OIDC authentication handler
func NewOIDCAuthHandler(db *pgxpool.Pool) *OIDCAuthHandler {
	return &OIDCAuthHandler{db: db}
}

// AuthorizeRequest represents the authorization request parameters
type AuthorizeRequest struct {
	ProviderID  string `form:"provider_id" binding:"required"`
	TenantID    string `form:"tenant_id" binding:"required"`
	RedirectURI string `form:"redirect_uri"`
	State       string `form:"state"`
	Scope       string `form:"scope"`
}

// CallbackRequest represents the callback parameters
type CallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

// TokenResponse represents the token exchange response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope,omitempty"`
}

// IDTokenClaims represents the parsed ID token claims
type IDTokenClaims struct {
	Issuer         string   `json:"iss"`
	Subject        string   `json:"sub"`
	Audience       string   `json:"aud"`
	Expiration     int64    `json:"exp"`
	IssuedAt       int64    `json:"iat"`
	Nonce          string   `json:"nonce,omitempty"`
	Email          string   `json:"email,omitempty"`
	EmailVerified  bool     `json:"email_verified,omitempty"`
	Name           string   `json:"name,omitempty"`
	GivenName      string   `json:"given_name,omitempty"`
	FamilyName     string   `json:"family_name,omitempty"`
	Picture        string   `json:"picture,omitempty"`
	Groups         []string `json:"groups,omitempty"`
}

// Authorize initiates the OIDC authorization code flow
func (h *OIDCAuthHandler) Authorize(c *gin.Context) {
	var req AuthorizeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters", "details": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Get provider details
	query := `
		SELECT 
			provider_name, provider_type, issuer_url, authorization_endpoint,
			client_id, scopes, redirect_uris, pkce_enabled, response_type, response_mode, status
		FROM oidc_providers 
		WHERE id = $1 AND tenant_id = $2 AND status = 'active'
	`
	var provider struct {
		ProviderName          string
		ProviderType          string
		IssuerURL             string
		AuthorizationEndpoint *string
		ClientID              string
		Scopes                []string
		RedirectURIs          []string
		PKCEEnabled           bool
		ResponseType          string
		ResponseMode          string
		Status                string
	}
	err := h.db.QueryRow(ctx, query, req.ProviderID, req.TenantID).Scan(
		&provider.ProviderName,
		&provider.ProviderType,
		&provider.IssuerURL,
		&provider.AuthorizationEndpoint,
		&provider.ClientID,
		&provider.Scopes,
		&provider.RedirectURIs,
		&provider.PKCEEnabled,
		&provider.ResponseType,
		&provider.ResponseMode,
		&provider.Status,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found or inactive"})
		return
	}

	// Determine authorization endpoint
	authEndpoint := ""
	if provider.AuthorizationEndpoint != nil && *provider.AuthorizationEndpoint != "" {
		authEndpoint = *provider.AuthorizationEndpoint
	} else {
		authEndpoint = h.getAuthorizationEndpoint(provider.IssuerURL)
	}
	if authEndpoint == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not determine authorization endpoint"})
		return
	}

	// Validate redirect URI
	redirectURI := req.RedirectURI
	if redirectURI == "" {
		if len(provider.RedirectURIs) > 0 {
			redirectURI = provider.RedirectURIs[0]
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No redirect URI specified"})
			return
		}
	}
	if !h.isValidRedirectURI(redirectURI, provider.RedirectURIs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid redirect URI"})
		return
	}

	// Generate state parameter if not provided
	state := req.State
	if state == "" {
		state = h.generateRandomString(32)
	}

	// Generate nonce for ID token validation
	nonce := h.generateRandomString(32)

	// PKCE: Generate code verifier and challenge
	var codeVerifier, codeChallenge string
	if provider.PKCEEnabled {
		codeVerifier = h.generateCodeVerifier()
		codeChallenge = h.generateCodeChallenge(codeVerifier)
	}

	// Create auth session
	sessionID := uuid.New().String()
	insertQuery := `
		INSERT INTO oidc_auth_sessions (
			id, provider_id, tenant_id, state, nonce, code_verifier, code_challenge,
			redirect_uri, scope, response_type, status, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	scopes := req.Scope
	if scopes == "" {
		scopes = strings.Join(provider.Scopes, " ")
	}
	// Convert space-separated scopes to array for database
	scopeArray := strings.Split(strings.TrimSpace(scopes), " ")
	expiresAt := time.Now().Add(10 * time.Minute)
	_, err = h.db.Exec(ctx, insertQuery,
		sessionID, req.ProviderID, req.TenantID, state, nonce, codeVerifier, codeChallenge,
		redirectURI, scopeArray, "code", "pending", time.Now(), expiresAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create auth session", "details": err.Error()})
		return
	}

	// Build authorization URL
	authURL, err := url.Parse(authEndpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid authorization endpoint"})
		return
	}

	params := url.Values{}
	params.Set("client_id", provider.ClientID)
	params.Set("response_type", provider.ResponseType)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scopes)
	params.Set("state", state)
	params.Set("nonce", nonce)
	if provider.PKCEEnabled {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}
	authURL.RawQuery = params.Encode()

	// Redirect to provider's authorization endpoint
	c.Redirect(http.StatusFound, authURL.String())
}

// Callback handles the OAuth callback from the identity provider
func (h *OIDCAuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDescription := c.Query("error_description")

	if errorParam != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             errorParam,
			"error_description": errorDescription,
		})
		return
	}

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing code or state parameter"})
		return
	}

	ctx := c.Request.Context()

	// Retrieve auth session
	query := `
		SELECT 
			s.id, s.provider_id, s.tenant_id, s.nonce, s.code_verifier, s.redirect_uri,
			p.provider_name, p.token_endpoint, p.client_id, p.client_secret, p.pkce_enabled
		FROM oidc_auth_sessions s
		JOIN oidc_providers p ON s.provider_id = p.id
		WHERE s.state = $1 AND s.status = 'pending' AND s.expires_at > NOW()
	`
	var session struct {
		ID            string
		ProviderID    string
		TenantID      string
		Nonce         string
		CodeVerifier  *string
		RedirectURI   string
		ProviderName  string
		TokenEndpoint *string
		ClientID      string
		ClientSecret  *string
		PKCEEnabled   bool
	}
	err := h.db.QueryRow(ctx, query, state).Scan(
		&session.ID,
		&session.ProviderID,
		&session.TenantID,
		&session.Nonce,
		&session.CodeVerifier,
		&session.RedirectURI,
		&session.ProviderName,
		&session.TokenEndpoint,
		&session.ClientID,
		&session.ClientSecret,
		&session.PKCEEnabled,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired state parameter"})
		return
	}

	// Determine token endpoint
	tokenEndpoint := ""
	if session.TokenEndpoint != nil && *session.TokenEndpoint != "" {
		tokenEndpoint = *session.TokenEndpoint
	}
	if tokenEndpoint == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not determine token endpoint"})
		return
	}

	// Exchange authorization code for tokens
	tokenResp, err := h.exchangeCodeForToken(
		tokenEndpoint,
		code,
		session.ClientID,
		session.ClientSecret,
		&session.RedirectURI,
		session.CodeVerifier,
		session.PKCEEnabled,
	)
	if err != nil {
		h.db.Exec(ctx, `UPDATE oidc_auth_sessions SET status = 'failed', updated_at = NOW() WHERE id = $1`, session.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed", "details": err.Error()})
		return
	}

	// Validate ID token
	claims, err := h.validateIDToken(tokenResp.IDToken, session.ClientID, session.Nonce)
	if err != nil {
		h.db.Exec(ctx, `UPDATE oidc_auth_sessions SET status = 'failed', updated_at = NOW() WHERE id = $1`, session.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ID token validation failed", "details": err.Error()})
		return
	}

	// Update session with token info
	updateQuery := `
		UPDATE oidc_auth_sessions 
		SET 
			authorization_code = $1,
			access_token = $2,
			id_token = $3,
			refresh_token = $4,
			token_expires_at = $5,
			status = 'completed',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $6
	`
	tokenExpiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	_, err = h.db.Exec(ctx, updateQuery,
		code,
		tokenResp.AccessToken,
		tokenResp.IDToken,
		tokenResp.RefreshToken,
		tokenExpiresAt,
		session.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	// Provision user (auto-create mapping)
	userID, err := h.provisionUser(c, session.ProviderID, session.TenantID, claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User provisioning failed", "details": err.Error()})
		return
	}

	// Update session with user_id
	h.db.Exec(ctx, `UPDATE oidc_auth_sessions SET user_id = $1 WHERE id = $2`, userID, session.ID)

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"sessionId": session.ID,
		"userId":    userID,
		"user": gin.H{
			"email": claims.Email,
			"name":  claims.Name,
			"sub":   claims.Subject,
		},
	})
}

// Helper functions

func (h *OIDCAuthHandler) getAuthorizationEndpoint(issuerURL string) string {
	discoveryURL := issuerURL
	if discoveryURL[len(discoveryURL)-1] != '/' {
		discoveryURL += "/"
	}
	discoveryURL += ".well-known/openid-configuration"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var discovery struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return ""
	}

	return discovery.AuthorizationEndpoint
}

func (h *OIDCAuthHandler) isValidRedirectURI(uri string, allowed []string) bool {
	for _, allowedURI := range allowed {
		if uri == allowedURI {
			return true
		}
	}
	return false
}

func (h *OIDCAuthHandler) generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

func (h *OIDCAuthHandler) generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (h *OIDCAuthHandler) generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func (h *OIDCAuthHandler) exchangeCodeForToken(
	tokenEndpoint, code, clientID string,
	clientSecret, redirectURI *string,
	codeVerifier *string,
	pkceEnabled bool,
) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("redirect_uri", *redirectURI)
	if clientSecret != nil && *clientSecret != "" {
		data.Set("client_secret", *clientSecret)
	}
	if pkceEnabled && codeVerifier != nil && *codeVerifier != "" {
		data.Set("code_verifier", *codeVerifier)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(tokenEndpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

func (h *OIDCAuthHandler) validateIDToken(idToken, clientID, nonce string) (*IDTokenClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims IDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	now := time.Now().Unix()
	if claims.Expiration < now {
		return nil, fmt.Errorf("token expired")
	}
	if claims.Audience != clientID {
		return nil, fmt.Errorf("invalid audience")
	}
	if nonce != "" && claims.Nonce != nonce {
		return nil, fmt.Errorf("invalid nonce")
	}

	return &claims, nil
}

func (h *OIDCAuthHandler) provisionUser(c *gin.Context, providerID, tenantID string, claims *IDTokenClaims) (string, error) {
	ctx := c.Request.Context()
	
	// Check if user mapping already exists
	query := `
		SELECT user_id, last_login_at FROM oidc_user_mappings 
		WHERE provider_id = $1 AND tenant_id = $2 AND provider_user_id = $3 AND status = 'active'
	`
	var userID string
	var lastLogin time.Time
	err := h.db.QueryRow(ctx, query, providerID, tenantID, claims.Subject).Scan(&userID, &lastLogin)
	if err == nil {
		// User exists - update last login
		updateQuery := `
			UPDATE oidc_user_mappings 
			SET last_login_at = $1, updated_at = $2
			WHERE user_id = $3 AND provider_id = $4
		`
		_, err = h.db.Exec(ctx, updateQuery, time.Now(), time.Now(), userID, providerID)
		if err != nil {
			// Log error but don't fail - user can still proceed
			fmt.Fprintf(c.Writer, "[WARN] Failed to update last login: %v\n", err)
		}
		return userID, nil
	}

	// Get provider configuration for auto-provisioning settings
	providerQuery := `
		SELECT 
			auto_provision_users, user_attribute_mapping, default_role, provider_name
		FROM oidc_providers 
		WHERE id = $1 AND tenant_id = $2 AND status = 'active'
	`
	var autoProvision *bool
	var attributeMapping map[string]interface{}
	var defaultRole *string
	var providerName string
	
	err = h.db.QueryRow(ctx, providerQuery, providerID, tenantID).Scan(
		&autoProvision,
		&attributeMapping,
		&defaultRole,
		&providerName,
	)
	if err != nil {
		return "", fmt.Errorf("failed to fetch provider config: %w", err)
	}

	// Check if auto-provisioning is enabled
	if autoProvision == nil || !*autoProvision {
		return "", fmt.Errorf("user not found and auto-provisioning is disabled for provider %s", providerName)
	}

	// Create new user mapping with attribute mapping
	userID = uuid.New().String()
	mappingID := uuid.New().String()
	
	// Apply attribute mapping
	displayName := claims.Name
	email := claims.Email
	givenName := claims.GivenName
	familyName := claims.FamilyName
	
	if attributeMapping != nil {
		// Map email
		if emailMap, ok := attributeMapping["email"].(string); ok && emailMap != "" {
			if mappedEmail := h.extractClaimValue(claims, emailMap); mappedEmail != "" {
				email = mappedEmail
			}
		}
		
		// Map display name
		if nameMap, ok := attributeMapping["name"].(string); ok && nameMap != "" {
			if mappedName := h.extractClaimValue(claims, nameMap); mappedName != "" {
				displayName = mappedName
			}
		}
		
		// Map given name
		if givenMap, ok := attributeMapping["given_name"].(string); ok && givenMap != "" {
			if mappedGiven := h.extractClaimValue(claims, givenMap); mappedGiven != "" {
				givenName = mappedGiven
			}
		}
		
		// Map family name
		if familyMap, ok := attributeMapping["family_name"].(string); ok && familyMap != "" {
			if mappedFamily := h.extractClaimValue(claims, familyMap); mappedFamily != "" {
				familyName = mappedFamily
			}
		}
	}
	
	// Construct full display name if not provided
	if displayName == "" {
		if givenName != "" && familyName != "" {
			displayName = givenName + " " + familyName
		} else if givenName != "" {
			displayName = givenName
		} else if email != "" {
			displayName = email
		} else {
			displayName = claims.Subject
		}
	}
	
	// Validate required fields
	if email == "" {
		return "", fmt.Errorf("email is required for user provisioning but not provided in claims")
	}
	
	insertQuery := `
		INSERT INTO oidc_user_mappings (
			id, tenant_id, provider_id, user_id, provider_user_id,
			email, display_name, status, created_at, updated_at, last_login_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = h.db.Exec(ctx, insertQuery,
		mappingID,
		tenantID,
		providerID,
		userID,
		claims.Subject,
		email,
		displayName,
		"active",
		time.Now(),
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create user mapping: %w", err)
	}

	// Assign default role if specified
	if defaultRole != nil && *defaultRole != "" {
		err = h.assignDefaultRole(ctx, userID, tenantID, *defaultRole)
		if err != nil {
			// Log error but don't fail provisioning
			fmt.Fprintf(c.Writer, "[WARN] Failed to assign default role: %v\n", err)
		}
	}
	
	// Process group memberships if available
	if len(claims.Groups) > 0 {
		err = h.syncGroupMemberships(ctx, userID, tenantID, providerID, claims.Groups)
		if err != nil {
			// Log error but don't fail provisioning
			fmt.Fprintf(c.Writer, "[WARN] Failed to sync group memberships: %v\n", err)
		}
	}

	return userID, nil
}

// extractClaimValue extracts a value from claims using a path (supports nested fields)
func (h *OIDCAuthHandler) extractClaimValue(claims *IDTokenClaims, path string) string {
	switch path {
	case "sub", "subject":
		return claims.Subject
	case "email":
		return claims.Email
	case "name":
		return claims.Name
	case "given_name", "givenName":
		return claims.GivenName
	case "family_name", "familyName":
		return claims.FamilyName
	case "picture":
		return claims.Picture
	default:
		return ""
	}
}

// assignDefaultRole assigns a default role to a newly provisioned user
func (h *OIDCAuthHandler) assignDefaultRole(ctx context.Context, userID, tenantID, role string) error {
	// Check if a user_roles table exists and assign role
	// This is a placeholder - implement based on your RBAC schema
	query := `
		INSERT INTO user_roles (id, user_id, tenant_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, tenant_id, role) DO NOTHING
	`
	_, err := h.db.Exec(ctx, query,
		uuid.New().String(),
		userID,
		tenantID,
		role,
		time.Now(),
	)
	return err
}

// syncGroupMemberships synchronizes OIDC group claims with local group memberships
func (h *OIDCAuthHandler) syncGroupMemberships(ctx context.Context, userID, tenantID, providerID string, groups []string) error {
	// This is a placeholder - implement based on your group management schema
	// For now, we'll store groups in a separate table
	for _, group := range groups {
		query := `
			INSERT INTO oidc_user_groups (id, user_id, tenant_id, provider_id, group_name, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (user_id, provider_id, group_name) DO UPDATE 
			SET updated_at = $7
		`
		_, err := h.db.Exec(ctx, query,
			uuid.New().String(),
			userID,
			tenantID,
			providerID,
			group,
			time.Now(),
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to sync group %s: %w", group, err)
		}
	}
	return nil
}

// RegisterRoutes registers authentication routes
func (h *OIDCAuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/authorize", h.Authorize)
	router.GET("/callback", h.Callback)
}
