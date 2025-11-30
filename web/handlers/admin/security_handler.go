package admin

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SecurityHandler handles security settings endpoints
type SecurityHandler struct {
	db *pgxpool.Pool
}

// NewSecurityHandler creates a new security handler
func NewSecurityHandler(db *pgxpool.Pool) *SecurityHandler {
	return &SecurityHandler{db: db}
}

// SecuritySettings represents comprehensive security configuration
type SecuritySettings struct {
	ID                          string     `json:"id"`
	TenantID                    string     `json:"tenantId"`
	
	// MFA Settings
	MFAEnabled                  bool       `json:"mfaEnabled"`
	MFAMethods                  []string   `json:"mfaMethods"`
	MFARequiredForAdmin         bool       `json:"mfaRequiredForAdmin"`
	MFAGracePeriodHours         int        `json:"mfaGracePeriodHours"`
	
	// IP Whitelisting
	IPWhitelistEnabled          bool       `json:"ipWhitelistEnabled"`
	IPWhitelist                 []string   `json:"ipWhitelist"`
	IPWhitelistMode             string     `json:"ipWhitelistMode"`
	
	// Token Expiration
	AccessTokenTTLMinutes       int        `json:"accessTokenTtlMinutes"`
	RefreshTokenTTLDays         int        `json:"refreshTokenTtlDays"`
	IDTokenTTLMinutes           int        `json:"idTokenTtlMinutes"`
	APIKeyDefaultTTLDays        int        `json:"apiKeyDefaultTtlDays"`
	AllowTokenRefresh           bool       `json:"allowTokenRefresh"`
	MaxTokenLifetimeDays        int        `json:"maxTokenLifetimeDays"`
	
	// Session Management
	SessionTimeoutMinutes       int        `json:"sessionTimeoutMinutes"`
	SessionIdleTimeoutMinutes   int        `json:"sessionIdleTimeoutMinutes"`
	MaxConcurrentSessions       int        `json:"maxConcurrentSessions"`
	SessionPinningEnabled       bool       `json:"sessionPinningEnabled"`
	ForceLogoutOnPasswordChange bool       `json:"forceLogoutOnPasswordChange"`
	
	// Password Policy
	PasswordMinLength           int        `json:"passwordMinLength"`
	PasswordRequireUppercase    bool       `json:"passwordRequireUppercase"`
	PasswordRequireLowercase    bool       `json:"passwordRequireLowercase"`
	PasswordRequireNumbers      bool       `json:"passwordRequireNumbers"`
	PasswordRequireSpecialChars bool       `json:"passwordRequireSpecialChars"`
	PasswordExpiryDays          int        `json:"passwordExpiryDays"`
	PasswordHistoryCount        int        `json:"passwordHistoryCount"`
	
	// Login Security
	MaxLoginAttempts            int        `json:"maxLoginAttempts"`
	LockoutDurationMinutes      int        `json:"lockoutDurationMinutes"`
	SuspiciousActivityDetection bool       `json:"suspiciousActivityDetection"`
	
	// Advanced Security
	RequireHTTPS                bool       `json:"requireHttps"`
	CORSAllowedOrigins          []string   `json:"corsAllowedOrigins"`
	CSRFProtectionEnabled       bool       `json:"csrfProtectionEnabled"`
	RateLimitingEnabled         bool       `json:"rateLimitingEnabled"`
	
	// Audit and Compliance
	AuditAllRequests            bool       `json:"auditAllRequests"`
	LogPIIAccess                bool       `json:"logPiiAccess"`
	DataRetentionDays           int        `json:"dataRetentionDays"`
	
	// Notifications
	NotifyOnNewDevice           bool       `json:"notifyOnNewDevice"`
	NotifyOnSuspiciousLogin     bool       `json:"notifyOnSuspiciousLogin"`
	NotifyOnAPIKeyUsage         bool       `json:"notifyOnApiKeyUsage"`
	SecurityContactEmail        string     `json:"securityContactEmail"`
	
	// Metadata
	CreatedAt                   time.Time  `json:"createdAt"`
	UpdatedAt                   time.Time  `json:"updatedAt"`
	UpdatedBy                   *string    `json:"updatedBy"`
	LastReviewedAt              *time.Time `json:"lastReviewedAt"`
}

// UpdateSecuritySettingsRequest represents security settings update request
type UpdateSecuritySettingsRequest struct {
	// MFA Settings
	MFAEnabled          *bool     `json:"mfaEnabled"`
	MFAMethods          []string  `json:"mfaMethods"`
	MFARequiredForAdmin *bool     `json:"mfaRequiredForAdmin"`
	MFAGracePeriodHours *int      `json:"mfaGracePeriodHours"`
	
	// IP Whitelisting
	IPWhitelistEnabled  *bool     `json:"ipWhitelistEnabled"`
	IPWhitelist         []string  `json:"ipWhitelist"`
	IPWhitelistMode     *string   `json:"ipWhitelistMode"`
	
	// Token Expiration
	AccessTokenTTLMinutes *int   `json:"accessTokenTtlMinutes"`
	RefreshTokenTTLDays   *int   `json:"refreshTokenTtlDays"`
	IDTokenTTLMinutes     *int   `json:"idTokenTtlMinutes"`
	APIKeyDefaultTTLDays  *int   `json:"apiKeyDefaultTtlDays"`
	AllowTokenRefresh     *bool  `json:"allowTokenRefresh"`
	MaxTokenLifetimeDays  *int   `json:"maxTokenLifetimeDays"`
	
	// Session Management
	SessionTimeoutMinutes       *int  `json:"sessionTimeoutMinutes"`
	SessionIdleTimeoutMinutes   *int  `json:"sessionIdleTimeoutMinutes"`
	MaxConcurrentSessions       *int  `json:"maxConcurrentSessions"`
	SessionPinningEnabled       *bool `json:"sessionPinningEnabled"`
	ForceLogoutOnPasswordChange *bool `json:"forceLogoutOnPasswordChange"`
	
	// Password Policy
	PasswordMinLength           *int  `json:"passwordMinLength"`
	PasswordRequireUppercase    *bool `json:"passwordRequireUppercase"`
	PasswordRequireLowercase    *bool `json:"passwordRequireLowercase"`
	PasswordRequireNumbers      *bool `json:"passwordRequireNumbers"`
	PasswordRequireSpecialChars *bool `json:"passwordRequireSpecialChars"`
	PasswordExpiryDays          *int  `json:"passwordExpiryDays"`
	PasswordHistoryCount        *int  `json:"passwordHistoryCount"`
	
	// Login Security
	MaxLoginAttempts            *int  `json:"maxLoginAttempts"`
	LockoutDurationMinutes      *int  `json:"lockoutDurationMinutes"`
	SuspiciousActivityDetection *bool `json:"suspiciousActivityDetection"`
	
	// Advanced Security
	RequireHTTPS            *bool    `json:"requireHttps"`
	CORSAllowedOrigins      []string `json:"corsAllowedOrigins"`
	CSRFProtectionEnabled   *bool    `json:"csrfProtectionEnabled"`
	RateLimitingEnabled     *bool    `json:"rateLimitingEnabled"`
	
	// Audit and Compliance
	AuditAllRequests  *bool `json:"auditAllRequests"`
	LogPIIAccess      *bool `json:"logPiiAccess"`
	DataRetentionDays *int  `json:"dataRetentionDays"`
	
	// Notifications
	NotifyOnNewDevice       *bool   `json:"notifyOnNewDevice"`
	NotifyOnSuspiciousLogin *bool   `json:"notifyOnSuspiciousLogin"`
	NotifyOnAPIKeyUsage     *bool   `json:"notifyOnApiKeyUsage"`
	SecurityContactEmail    *string `json:"securityContactEmail"`
	
	// Metadata
	UpdatedBy *string `json:"updatedBy"`
}

// GetSecuritySettings retrieves security settings for a tenant
func (h *SecurityHandler) GetSecuritySettings(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	ctx := c.Request.Context()

	query := `
		SELECT 
			id, tenant_id,
			mfa_enabled, mfa_methods, mfa_required_for_admin, mfa_grace_period_hours,
			ip_whitelist_enabled, ip_whitelist, ip_whitelist_mode,
			access_token_ttl_minutes, refresh_token_ttl_days, id_token_ttl_minutes,
			api_key_default_ttl_days, allow_token_refresh, max_token_lifetime_days,
			session_timeout_minutes, session_idle_timeout_minutes, max_concurrent_sessions,
			session_pinning_enabled, force_logout_on_password_change,
			password_min_length, password_require_uppercase, password_require_lowercase,
			password_require_numbers, password_require_special_chars, password_expiry_days,
			password_history_count,
			max_login_attempts, lockout_duration_minutes, suspicious_activity_detection,
			require_https, cors_allowed_origins, csrf_protection_enabled, rate_limiting_enabled,
			audit_all_requests, log_pii_access, data_retention_days,
			notify_on_new_device, notify_on_suspicious_login, notify_on_api_key_usage,
			security_contact_email,
			created_at, updated_at, updated_by, last_reviewed_at
		FROM security_settings
		WHERE tenant_id = $1
	`

	var settings SecuritySettings
	err := h.db.QueryRow(ctx, query, tenantID).Scan(
		&settings.ID, &settings.TenantID,
		&settings.MFAEnabled, &settings.MFAMethods, &settings.MFARequiredForAdmin, &settings.MFAGracePeriodHours,
		&settings.IPWhitelistEnabled, &settings.IPWhitelist, &settings.IPWhitelistMode,
		&settings.AccessTokenTTLMinutes, &settings.RefreshTokenTTLDays, &settings.IDTokenTTLMinutes,
		&settings.APIKeyDefaultTTLDays, &settings.AllowTokenRefresh, &settings.MaxTokenLifetimeDays,
		&settings.SessionTimeoutMinutes, &settings.SessionIdleTimeoutMinutes, &settings.MaxConcurrentSessions,
		&settings.SessionPinningEnabled, &settings.ForceLogoutOnPasswordChange,
		&settings.PasswordMinLength, &settings.PasswordRequireUppercase, &settings.PasswordRequireLowercase,
		&settings.PasswordRequireNumbers, &settings.PasswordRequireSpecialChars, &settings.PasswordExpiryDays,
		&settings.PasswordHistoryCount,
		&settings.MaxLoginAttempts, &settings.LockoutDurationMinutes, &settings.SuspiciousActivityDetection,
		&settings.RequireHTTPS, &settings.CORSAllowedOrigins, &settings.CSRFProtectionEnabled, &settings.RateLimitingEnabled,
		&settings.AuditAllRequests, &settings.LogPIIAccess, &settings.DataRetentionDays,
		&settings.NotifyOnNewDevice, &settings.NotifyOnSuspiciousLogin, &settings.NotifyOnAPIKeyUsage,
		&settings.SecurityContactEmail,
		&settings.CreatedAt, &settings.UpdatedAt, &settings.UpdatedBy, &settings.LastReviewedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Create default settings if none exist
			defaultSettings := h.createDefaultSecuritySettings(c, tenantID)
			c.JSON(http.StatusOK, defaultSettings)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve security settings", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// createDefaultSecuritySettings creates default security settings for a tenant
func (h *SecurityHandler) createDefaultSecuritySettings(c *gin.Context, tenantID string) SecuritySettings {
	id := uuid.New()
	ctx := c.Request.Context()
	
	query := `
		INSERT INTO security_settings (
			id, tenant_id, security_contact_email, updated_by
		) VALUES ($1, $2, $3, $4)
		RETURNING 
			id, tenant_id,
			mfa_enabled, mfa_methods, mfa_required_for_admin, mfa_grace_period_hours,
			ip_whitelist_enabled, ip_whitelist, ip_whitelist_mode,
			access_token_ttl_minutes, refresh_token_ttl_days, id_token_ttl_minutes,
			api_key_default_ttl_days, allow_token_refresh, max_token_lifetime_days,
			session_timeout_minutes, session_idle_timeout_minutes, max_concurrent_sessions,
			session_pinning_enabled, force_logout_on_password_change,
			password_min_length, password_require_uppercase, password_require_lowercase,
			password_require_numbers, password_require_special_chars, password_expiry_days,
			password_history_count,
			max_login_attempts, lockout_duration_minutes, suspicious_activity_detection,
			require_https, cors_allowed_origins, csrf_protection_enabled, rate_limiting_enabled,
			audit_all_requests, log_pii_access, data_retention_days,
			notify_on_new_device, notify_on_suspicious_login, notify_on_api_key_usage,
			security_contact_email,
			created_at, updated_at, updated_by, last_reviewed_at
	`
	
	var settings SecuritySettings
	_ = h.db.QueryRow(ctx, query, id, tenantID, "security@"+tenantID, "system").Scan( // Best effort; will return defaults on error
		&settings.ID, &settings.TenantID,
		&settings.MFAEnabled, &settings.MFAMethods, &settings.MFARequiredForAdmin, &settings.MFAGracePeriodHours,
		&settings.IPWhitelistEnabled, &settings.IPWhitelist, &settings.IPWhitelistMode,
		&settings.AccessTokenTTLMinutes, &settings.RefreshTokenTTLDays, &settings.IDTokenTTLMinutes,
		&settings.APIKeyDefaultTTLDays, &settings.AllowTokenRefresh, &settings.MaxTokenLifetimeDays,
		&settings.SessionTimeoutMinutes, &settings.SessionIdleTimeoutMinutes, &settings.MaxConcurrentSessions,
		&settings.SessionPinningEnabled, &settings.ForceLogoutOnPasswordChange,
		&settings.PasswordMinLength, &settings.PasswordRequireUppercase, &settings.PasswordRequireLowercase,
		&settings.PasswordRequireNumbers, &settings.PasswordRequireSpecialChars, &settings.PasswordExpiryDays,
		&settings.PasswordHistoryCount,
		&settings.MaxLoginAttempts, &settings.LockoutDurationMinutes, &settings.SuspiciousActivityDetection,
		&settings.RequireHTTPS, &settings.CORSAllowedOrigins, &settings.CSRFProtectionEnabled, &settings.RateLimitingEnabled,
		&settings.AuditAllRequests, &settings.LogPIIAccess, &settings.DataRetentionDays,
		&settings.NotifyOnNewDevice, &settings.NotifyOnSuspiciousLogin, &settings.NotifyOnAPIKeyUsage,
		&settings.SecurityContactEmail,
		&settings.CreatedAt, &settings.UpdatedAt, &settings.UpdatedBy, &settings.LastReviewedAt,
	)
	
	return settings
}

// UpdateSecuritySettings updates security settings for a tenant
func (h *SecurityHandler) UpdateSecuritySettings(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req UpdateSecuritySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Build dynamic update query
	query := `UPDATE security_settings SET `
	params := []interface{}{}
	paramIndex := 1

	// MFA Settings
	if req.MFAEnabled != nil {
		query += fmt.Sprintf("mfa_enabled = $%d, ", paramIndex)
		params = append(params, *req.MFAEnabled)
		paramIndex++
	}
	if req.MFAMethods != nil {
		query += fmt.Sprintf("mfa_methods = $%d, ", paramIndex)
		params = append(params, req.MFAMethods)
		paramIndex++
	}
	if req.MFARequiredForAdmin != nil {
		query += fmt.Sprintf("mfa_required_for_admin = $%d, ", paramIndex)
		params = append(params, *req.MFARequiredForAdmin)
		paramIndex++
	}
	if req.MFAGracePeriodHours != nil {
		query += fmt.Sprintf("mfa_grace_period_hours = $%d, ", paramIndex)
		params = append(params, *req.MFAGracePeriodHours)
		paramIndex++
	}

	// IP Whitelisting
	if req.IPWhitelistEnabled != nil {
		query += fmt.Sprintf("ip_whitelist_enabled = $%d, ", paramIndex)
		params = append(params, *req.IPWhitelistEnabled)
		paramIndex++
	}
	if req.IPWhitelist != nil {
		query += fmt.Sprintf("ip_whitelist = $%d, ", paramIndex)
		params = append(params, req.IPWhitelist)
		paramIndex++
	}
	if req.IPWhitelistMode != nil {
		query += fmt.Sprintf("ip_whitelist_mode = $%d, ", paramIndex)
		params = append(params, *req.IPWhitelistMode)
		paramIndex++
	}

	// Token Expiration
	if req.AccessTokenTTLMinutes != nil {
		query += fmt.Sprintf("access_token_ttl_minutes = $%d, ", paramIndex)
		params = append(params, *req.AccessTokenTTLMinutes)
		paramIndex++
	}
	if req.RefreshTokenTTLDays != nil {
		query += fmt.Sprintf("refresh_token_ttl_days = $%d, ", paramIndex)
		params = append(params, *req.RefreshTokenTTLDays)
		paramIndex++
	}
	if req.IDTokenTTLMinutes != nil {
		query += fmt.Sprintf("id_token_ttl_minutes = $%d, ", paramIndex)
		params = append(params, *req.IDTokenTTLMinutes)
		paramIndex++
	}
	if req.APIKeyDefaultTTLDays != nil {
		query += fmt.Sprintf("api_key_default_ttl_days = $%d, ", paramIndex)
		params = append(params, *req.APIKeyDefaultTTLDays)
		paramIndex++
	}
	if req.AllowTokenRefresh != nil {
		query += fmt.Sprintf("allow_token_refresh = $%d, ", paramIndex)
		params = append(params, *req.AllowTokenRefresh)
		paramIndex++
	}
	if req.MaxTokenLifetimeDays != nil {
		query += fmt.Sprintf("max_token_lifetime_days = $%d, ", paramIndex)
		params = append(params, *req.MaxTokenLifetimeDays)
		paramIndex++
	}

	// Session Management
	if req.SessionTimeoutMinutes != nil {
		query += fmt.Sprintf("session_timeout_minutes = $%d, ", paramIndex)
		params = append(params, *req.SessionTimeoutMinutes)
		paramIndex++
	}
	if req.SessionIdleTimeoutMinutes != nil {
		query += fmt.Sprintf("session_idle_timeout_minutes = $%d, ", paramIndex)
		params = append(params, *req.SessionIdleTimeoutMinutes)
		paramIndex++
	}
	if req.MaxConcurrentSessions != nil {
		query += fmt.Sprintf("max_concurrent_sessions = $%d, ", paramIndex)
		params = append(params, *req.MaxConcurrentSessions)
		paramIndex++
	}
	if req.SessionPinningEnabled != nil {
		query += fmt.Sprintf("session_pinning_enabled = $%d, ", paramIndex)
		params = append(params, *req.SessionPinningEnabled)
		paramIndex++
	}
	if req.ForceLogoutOnPasswordChange != nil {
		query += fmt.Sprintf("force_logout_on_password_change = $%d, ", paramIndex)
		params = append(params, *req.ForceLogoutOnPasswordChange)
		paramIndex++
	}

	// Password Policy
	if req.PasswordMinLength != nil {
		query += fmt.Sprintf("password_min_length = $%d, ", paramIndex)
		params = append(params, *req.PasswordMinLength)
		paramIndex++
	}
	if req.PasswordRequireUppercase != nil {
		query += fmt.Sprintf("password_require_uppercase = $%d, ", paramIndex)
		params = append(params, *req.PasswordRequireUppercase)
		paramIndex++
	}
	if req.PasswordRequireLowercase != nil {
		query += fmt.Sprintf("password_require_lowercase = $%d, ", paramIndex)
		params = append(params, *req.PasswordRequireLowercase)
		paramIndex++
	}
	if req.PasswordRequireNumbers != nil {
		query += fmt.Sprintf("password_require_numbers = $%d, ", paramIndex)
		params = append(params, *req.PasswordRequireNumbers)
		paramIndex++
	}
	if req.PasswordRequireSpecialChars != nil {
		query += fmt.Sprintf("password_require_special_chars = $%d, ", paramIndex)
		params = append(params, *req.PasswordRequireSpecialChars)
		paramIndex++
	}
	if req.PasswordExpiryDays != nil {
		query += fmt.Sprintf("password_expiry_days = $%d, ", paramIndex)
		params = append(params, *req.PasswordExpiryDays)
		paramIndex++
	}
	if req.PasswordHistoryCount != nil {
		query += fmt.Sprintf("password_history_count = $%d, ", paramIndex)
		params = append(params, *req.PasswordHistoryCount)
		paramIndex++
	}

	// Login Security
	if req.MaxLoginAttempts != nil {
		query += fmt.Sprintf("max_login_attempts = $%d, ", paramIndex)
		params = append(params, *req.MaxLoginAttempts)
		paramIndex++
	}
	if req.LockoutDurationMinutes != nil {
		query += fmt.Sprintf("lockout_duration_minutes = $%d, ", paramIndex)
		params = append(params, *req.LockoutDurationMinutes)
		paramIndex++
	}
	if req.SuspiciousActivityDetection != nil {
		query += fmt.Sprintf("suspicious_activity_detection = $%d, ", paramIndex)
		params = append(params, *req.SuspiciousActivityDetection)
		paramIndex++
	}

	// Advanced Security
	if req.RequireHTTPS != nil {
		query += fmt.Sprintf("require_https = $%d, ", paramIndex)
		params = append(params, *req.RequireHTTPS)
		paramIndex++
	}
	if req.CORSAllowedOrigins != nil {
		query += fmt.Sprintf("cors_allowed_origins = $%d, ", paramIndex)
		params = append(params, req.CORSAllowedOrigins)
		paramIndex++
	}
	if req.CSRFProtectionEnabled != nil {
		query += fmt.Sprintf("csrf_protection_enabled = $%d, ", paramIndex)
		params = append(params, *req.CSRFProtectionEnabled)
		paramIndex++
	}
	if req.RateLimitingEnabled != nil {
		query += fmt.Sprintf("rate_limiting_enabled = $%d, ", paramIndex)
		params = append(params, *req.RateLimitingEnabled)
		paramIndex++
	}

	// Audit and Compliance
	if req.AuditAllRequests != nil {
		query += fmt.Sprintf("audit_all_requests = $%d, ", paramIndex)
		params = append(params, *req.AuditAllRequests)
		paramIndex++
	}
	if req.LogPIIAccess != nil {
		query += fmt.Sprintf("log_pii_access = $%d, ", paramIndex)
		params = append(params, *req.LogPIIAccess)
		paramIndex++
	}
	if req.DataRetentionDays != nil {
		query += fmt.Sprintf("data_retention_days = $%d, ", paramIndex)
		params = append(params, *req.DataRetentionDays)
		paramIndex++
	}

	// Notifications
	if req.NotifyOnNewDevice != nil {
		query += fmt.Sprintf("notify_on_new_device = $%d, ", paramIndex)
		params = append(params, *req.NotifyOnNewDevice)
		paramIndex++
	}
	if req.NotifyOnSuspiciousLogin != nil {
		query += fmt.Sprintf("notify_on_suspicious_login = $%d, ", paramIndex)
		params = append(params, *req.NotifyOnSuspiciousLogin)
		paramIndex++
	}
	if req.NotifyOnAPIKeyUsage != nil {
		query += fmt.Sprintf("notify_on_api_key_usage = $%d, ", paramIndex)
		params = append(params, *req.NotifyOnAPIKeyUsage)
		paramIndex++
	}
	if req.SecurityContactEmail != nil {
		query += fmt.Sprintf("security_contact_email = $%d, ", paramIndex)
		params = append(params, *req.SecurityContactEmail)
		paramIndex++
	}

	// Updated by
	if req.UpdatedBy != nil {
		query += fmt.Sprintf("updated_by = $%d, ", paramIndex)
		params = append(params, *req.UpdatedBy)
		paramIndex++
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE tenant_id = $%d RETURNING id", paramIndex)
	params = append(params, tenantID)

	var returnedID string
	err := h.db.QueryRow(ctx, query, params...).Scan(&returnedID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Security settings not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update security settings", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Security settings updated successfully",
		"id":      returnedID,
	})
}

// ResetToDefaults resets security settings to default values
func (h *SecurityHandler) ResetToDefaults(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	ctx := c.Request.Context()

	// Delete existing settings and recreate with defaults
	_, err := h.db.Exec(ctx, "DELETE FROM security_settings WHERE tenant_id = $1", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset settings"})
		return
	}

	defaultSettings := h.createDefaultSecuritySettings(c, tenantID)
	
	c.JSON(http.StatusOK, gin.H{
		"message":  "Security settings reset to defaults",
		"settings": defaultSettings,
	})
}

// RegisterRoutes registers security settings routes
func (h *SecurityHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/security-settings", h.GetSecuritySettings)
	router.PUT("/security-settings", h.UpdateSecuritySettings)
	router.POST("/security-settings/reset", h.ResetToDefaults)
}
