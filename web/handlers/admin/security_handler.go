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
	ID       string `json:"id"`
	TenantID string `json:"tenantId"`

	// MFA Settings
	MFAEnabled          bool     `json:"mfaEnabled"`
	MFAMethods          []string `json:"mfaMethods"`
	MFARequiredForAdmin bool     `json:"mfaRequiredForAdmin"`
	MFAGracePeriodHours int      `json:"mfaGracePeriodHours"`

	// IP Whitelisting
	IPWhitelistEnabled bool     `json:"ipWhitelistEnabled"`
	IPWhitelist        []string `json:"ipWhitelist"`
	IPWhitelistMode    string   `json:"ipWhitelistMode"`

	// Token Expiration
	AccessTokenTTLMinutes int  `json:"accessTokenTtlMinutes"`
	RefreshTokenTTLDays   int  `json:"refreshTokenTtlDays"`
	IDTokenTTLMinutes     int  `json:"idTokenTtlMinutes"`
	APIKeyDefaultTTLDays  int  `json:"apiKeyDefaultTtlDays"`
	AllowTokenRefresh     bool `json:"allowTokenRefresh"`
	MaxTokenLifetimeDays  int  `json:"maxTokenLifetimeDays"`

	// Session Management
	SessionTimeoutMinutes       int  `json:"sessionTimeoutMinutes"`
	SessionIdleTimeoutMinutes   int  `json:"sessionIdleTimeoutMinutes"`
	MaxConcurrentSessions       int  `json:"maxConcurrentSessions"`
	SessionPinningEnabled       bool `json:"sessionPinningEnabled"`
	ForceLogoutOnPasswordChange bool `json:"forceLogoutOnPasswordChange"`

	// Password Policy
	PasswordMinLength           int  `json:"passwordMinLength"`
	PasswordRequireUppercase    bool `json:"passwordRequireUppercase"`
	PasswordRequireLowercase    bool `json:"passwordRequireLowercase"`
	PasswordRequireNumbers      bool `json:"passwordRequireNumbers"`
	PasswordRequireSpecialChars bool `json:"passwordRequireSpecialChars"`
	PasswordExpiryDays          int  `json:"passwordExpiryDays"`
	PasswordHistoryCount        int  `json:"passwordHistoryCount"`

	// Login Security
	MaxLoginAttempts            int  `json:"maxLoginAttempts"`
	LockoutDurationMinutes      int  `json:"lockoutDurationMinutes"`
	SuspiciousActivityDetection bool `json:"suspiciousActivityDetection"`

	// Advanced Security
	RequireHTTPS          bool     `json:"requireHttps"`
	CORSAllowedOrigins    []string `json:"corsAllowedOrigins"`
	CSRFProtectionEnabled bool     `json:"csrfProtectionEnabled"`
	RateLimitingEnabled   bool     `json:"rateLimitingEnabled"`

	// Audit and Compliance
	AuditAllRequests  bool `json:"auditAllRequests"`
	LogPIIAccess      bool `json:"logPiiAccess"`
	DataRetentionDays int  `json:"dataRetentionDays"`

	// Notifications
	NotifyOnNewDevice       bool   `json:"notifyOnNewDevice"`
	NotifyOnSuspiciousLogin bool   `json:"notifyOnSuspiciousLogin"`
	NotifyOnAPIKeyUsage     bool   `json:"notifyOnApiKeyUsage"`
	SecurityContactEmail    string `json:"securityContactEmail"`

	// Metadata
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	UpdatedBy      *string    `json:"updatedBy"`
	LastReviewedAt *time.Time `json:"lastReviewedAt"`
}

// UpdateSecuritySettingsRequest represents security settings update request
type UpdateSecuritySettingsRequest struct {
	// MFA Settings
	MFAEnabled          *bool    `json:"mfaEnabled"`
	MFAMethods          []string `json:"mfaMethods"`
	MFARequiredForAdmin *bool    `json:"mfaRequiredForAdmin"`
	MFAGracePeriodHours *int     `json:"mfaGracePeriodHours"`

	// IP Whitelisting
	IPWhitelistEnabled *bool    `json:"ipWhitelistEnabled"`
	IPWhitelist        []string `json:"ipWhitelist"`
	IPWhitelistMode    *string  `json:"ipWhitelistMode"`

	// Token Expiration
	AccessTokenTTLMinutes *int  `json:"accessTokenTtlMinutes"`
	RefreshTokenTTLDays   *int  `json:"refreshTokenTtlDays"`
	IDTokenTTLMinutes     *int  `json:"idTokenTtlMinutes"`
	APIKeyDefaultTTLDays  *int  `json:"apiKeyDefaultTtlDays"`
	AllowTokenRefresh     *bool `json:"allowTokenRefresh"`
	MaxTokenLifetimeDays  *int  `json:"maxTokenLifetimeDays"`

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
	RequireHTTPS          *bool    `json:"requireHttps"`
	CORSAllowedOrigins    []string `json:"corsAllowedOrigins"`
	CSRFProtectionEnabled *bool    `json:"csrfProtectionEnabled"`
	RateLimitingEnabled   *bool    `json:"rateLimitingEnabled"`

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
	if h.db == nil {
		// In degraded mode, return default settings
		c.JSON(http.StatusOK, h.createDefaultSecuritySettings(c, tenantID))
		return
	}
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

	// In degraded mode, just return the defaults without inserting
	if h.db == nil {
		settings.ID = id.String()
		settings.TenantID = tenantID
		settings.SecurityContactEmail = "security@" + tenantID
		settings.UpdatedAt = time.Now()
		updatedBy := "system"
		settings.UpdatedBy = &updatedBy
		// Populate other defaults manually or let them be zero-valued/handled by frontend
		return settings
	}

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

	query, params := h.buildSecuritySettingsUpdateQuery(&req, tenantID)

	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

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

func (h *SecurityHandler) buildSecuritySettingsUpdateQuery(req *UpdateSecuritySettingsRequest, tenantID string) (string, []interface{}) {
	query := `UPDATE security_settings SET `
	params := []interface{}{}
	paramIndex := 1

	query, params, paramIndex = h.appendMFAUpdates(query, params, paramIndex, req)
	query, params, paramIndex = h.appendIPUpdates(query, params, paramIndex, req)
	query, params, paramIndex = h.appendTokenUpdates(query, params, paramIndex, req)
	query, params, paramIndex = h.appendSessionUpdates(query, params, paramIndex, req)
	query, params, paramIndex = h.appendPasswordUpdates(query, params, paramIndex, req)
	query, params, paramIndex = h.appendLoginSecurityUpdates(query, params, paramIndex, req)
	query, params, paramIndex = h.appendAdvancedSecurityUpdates(query, params, paramIndex, req)
	query, params, paramIndex = h.appendAuditUpdates(query, params, paramIndex, req)
	query, params, paramIndex = h.appendNotificationUpdates(query, params, paramIndex, req)

	// Updated by
	if req.UpdatedBy != nil {
		query += fmt.Sprintf("updated_by = $%d, ", paramIndex)
		params = append(params, *req.UpdatedBy)
		paramIndex++
	}

	// Remove trailing comma and add WHERE clause
	if len(params) > 0 { // Only strip if we added something
		query = query[:len(query)-2]
	}

	query += fmt.Sprintf(" WHERE tenant_id = $%d RETURNING id", paramIndex)
	params = append(params, tenantID)

	return query, params
}

func (h *SecurityHandler) appendMFAUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.MFAEnabled != nil {
		query += fmt.Sprintf("mfa_enabled = $%d, ", idx)
		params = append(params, *req.MFAEnabled)
		idx++
	}
	if req.MFAMethods != nil {
		query += fmt.Sprintf("mfa_methods = $%d, ", idx)
		params = append(params, req.MFAMethods)
		idx++
	}
	if req.MFARequiredForAdmin != nil {
		query += fmt.Sprintf("mfa_required_for_admin = $%d, ", idx)
		params = append(params, *req.MFARequiredForAdmin)
		idx++
	}
	if req.MFAGracePeriodHours != nil {
		query += fmt.Sprintf("mfa_grace_period_hours = $%d, ", idx)
		params = append(params, *req.MFAGracePeriodHours)
		idx++
	}
	return query, params, idx
}

func (h *SecurityHandler) appendIPUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.IPWhitelistEnabled != nil {
		query += fmt.Sprintf("ip_whitelist_enabled = $%d, ", idx)
		params = append(params, *req.IPWhitelistEnabled)
		idx++
	}
	if req.IPWhitelist != nil {
		query += fmt.Sprintf("ip_whitelist = $%d, ", idx)
		params = append(params, req.IPWhitelist)
		idx++
	}
	if req.IPWhitelistMode != nil {
		query += fmt.Sprintf("ip_whitelist_mode = $%d, ", idx)
		params = append(params, *req.IPWhitelistMode)
		idx++
	}
	return query, params, idx
}

func (h *SecurityHandler) appendTokenUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.AccessTokenTTLMinutes != nil {
		query += fmt.Sprintf("access_token_ttl_minutes = $%d, ", idx)
		params = append(params, *req.AccessTokenTTLMinutes)
		idx++
	}
	if req.RefreshTokenTTLDays != nil {
		query += fmt.Sprintf("refresh_token_ttl_days = $%d, ", idx)
		params = append(params, *req.RefreshTokenTTLDays)
		idx++
	}
	if req.IDTokenTTLMinutes != nil {
		query += fmt.Sprintf("id_token_ttl_minutes = $%d, ", idx)
		params = append(params, *req.IDTokenTTLMinutes)
		idx++
	}
	if req.APIKeyDefaultTTLDays != nil {
		query += fmt.Sprintf("api_key_default_ttl_days = $%d, ", idx)
		params = append(params, *req.APIKeyDefaultTTLDays)
		idx++
	}
	if req.AllowTokenRefresh != nil {
		query += fmt.Sprintf("allow_token_refresh = $%d, ", idx)
		params = append(params, *req.AllowTokenRefresh)
		idx++
	}
	if req.MaxTokenLifetimeDays != nil {
		query += fmt.Sprintf("max_token_lifetime_days = $%d, ", idx)
		params = append(params, *req.MaxTokenLifetimeDays)
		idx++
	}
	return query, params, idx
}

func (h *SecurityHandler) appendSessionUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.SessionTimeoutMinutes != nil {
		query += fmt.Sprintf("session_timeout_minutes = $%d, ", idx)
		params = append(params, *req.SessionTimeoutMinutes)
		idx++
	}
	if req.SessionIdleTimeoutMinutes != nil {
		query += fmt.Sprintf("session_idle_timeout_minutes = $%d, ", idx)
		params = append(params, *req.SessionIdleTimeoutMinutes)
		idx++
	}
	if req.MaxConcurrentSessions != nil {
		query += fmt.Sprintf("max_concurrent_sessions = $%d, ", idx)
		params = append(params, *req.MaxConcurrentSessions)
		idx++
	}
	if req.SessionPinningEnabled != nil {
		query += fmt.Sprintf("session_pinning_enabled = $%d, ", idx)
		params = append(params, *req.SessionPinningEnabled)
		idx++
	}
	if req.ForceLogoutOnPasswordChange != nil {
		query += fmt.Sprintf("force_logout_on_password_change = $%d, ", idx)
		params = append(params, *req.ForceLogoutOnPasswordChange)
		idx++
	}
	return query, params, idx
}

func (h *SecurityHandler) appendPasswordUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.PasswordMinLength != nil {
		query += fmt.Sprintf("password_min_length = $%d, ", idx)
		params = append(params, *req.PasswordMinLength)
		idx++
	}
	if req.PasswordRequireUppercase != nil {
		query += fmt.Sprintf("password_require_uppercase = $%d, ", idx)
		params = append(params, *req.PasswordRequireUppercase)
		idx++
	}
	if req.PasswordRequireLowercase != nil {
		query += fmt.Sprintf("password_require_lowercase = $%d, ", idx)
		params = append(params, *req.PasswordRequireLowercase)
		idx++
	}
	if req.PasswordRequireNumbers != nil {
		query += fmt.Sprintf("password_require_numbers = $%d, ", idx)
		params = append(params, *req.PasswordRequireNumbers)
		idx++
	}
	if req.PasswordRequireSpecialChars != nil {
		query += fmt.Sprintf("password_require_special_chars = $%d, ", idx)
		params = append(params, *req.PasswordRequireSpecialChars)
		idx++
	}
	if req.PasswordExpiryDays != nil {
		query += fmt.Sprintf("password_expiry_days = $%d, ", idx)
		params = append(params, *req.PasswordExpiryDays)
		idx++
	}
	if req.PasswordHistoryCount != nil {
		query += fmt.Sprintf("password_history_count = $%d, ", idx)
		params = append(params, *req.PasswordHistoryCount)
		idx++
	}
	return query, params, idx
}

func (h *SecurityHandler) appendLoginSecurityUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.MaxLoginAttempts != nil {
		query += fmt.Sprintf("max_login_attempts = $%d, ", idx)
		params = append(params, *req.MaxLoginAttempts)
		idx++
	}
	if req.LockoutDurationMinutes != nil {
		query += fmt.Sprintf("lockout_duration_minutes = $%d, ", idx)
		params = append(params, *req.LockoutDurationMinutes)
		idx++
	}
	if req.SuspiciousActivityDetection != nil {
		query += fmt.Sprintf("suspicious_activity_detection = $%d, ", idx)
		params = append(params, *req.SuspiciousActivityDetection)
		idx++
	}
	return query, params, idx
}

func (h *SecurityHandler) appendAdvancedSecurityUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.RequireHTTPS != nil {
		query += fmt.Sprintf("require_https = $%d, ", idx)
		params = append(params, *req.RequireHTTPS)
		idx++
	}
	if req.CORSAllowedOrigins != nil {
		query += fmt.Sprintf("cors_allowed_origins = $%d, ", idx)
		params = append(params, req.CORSAllowedOrigins)
		idx++
	}
	if req.CSRFProtectionEnabled != nil {
		query += fmt.Sprintf("csrf_protection_enabled = $%d, ", idx)
		params = append(params, *req.CSRFProtectionEnabled)
		idx++
	}
	if req.RateLimitingEnabled != nil {
		query += fmt.Sprintf("rate_limiting_enabled = $%d, ", idx)
		params = append(params, *req.RateLimitingEnabled)
		idx++
	}
	return query, params, idx
}

func (h *SecurityHandler) appendAuditUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.AuditAllRequests != nil {
		query += fmt.Sprintf("audit_all_requests = $%d, ", idx)
		params = append(params, *req.AuditAllRequests)
		idx++
	}
	if req.LogPIIAccess != nil {
		query += fmt.Sprintf("log_pii_access = $%d, ", idx)
		params = append(params, *req.LogPIIAccess)
		idx++
	}
	if req.DataRetentionDays != nil {
		query += fmt.Sprintf("data_retention_days = $%d, ", idx)
		params = append(params, *req.DataRetentionDays)
		idx++
	}
	return query, params, idx
}

func (h *SecurityHandler) appendNotificationUpdates(query string, params []interface{}, idx int, req *UpdateSecuritySettingsRequest) (string, []interface{}, int) {
	if req.NotifyOnNewDevice != nil {
		query += fmt.Sprintf("notify_on_new_device = $%d, ", idx)
		params = append(params, *req.NotifyOnNewDevice)
		idx++
	}
	if req.NotifyOnSuspiciousLogin != nil {
		query += fmt.Sprintf("notify_on_suspicious_login = $%d, ", idx)
		params = append(params, *req.NotifyOnSuspiciousLogin)
		idx++
	}
	if req.NotifyOnAPIKeyUsage != nil {
		query += fmt.Sprintf("notify_on_api_key_usage = $%d, ", idx)
		params = append(params, *req.NotifyOnAPIKeyUsage)
		idx++
	}
	if req.SecurityContactEmail != nil {
		query += fmt.Sprintf("security_contact_email = $%d, ", idx)
		params = append(params, *req.SecurityContactEmail)
		idx++
	}
	return query, params, idx
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
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

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
