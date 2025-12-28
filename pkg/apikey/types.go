package apikey

import (
	"time"
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          string `json:"id" db:"id"`
	KeyID       string `json:"key_id" db:"key_id"`
	KeyHash     string `json:"-" db:"key_hash"` // Never expose in JSON
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
	UserID      string `json:"user_id" db:"user_id"`

	// Rate limiting
	RateLimitPerMinute int `json:"rate_limit_per_minute" db:"rate_limit_per_minute"`
	RateLimitPerHour   int `json:"rate_limit_per_hour" db:"rate_limit_per_hour"`
	RateLimitPerDay    int `json:"rate_limit_per_day" db:"rate_limit_per_day"`

	// Quotas
	QuotaRequestsTotal *int `json:"quota_requests_total,omitempty" db:"quota_requests_total"` // nil = unlimited
	QuotaRequestsUsed  int  `json:"quota_requests_used" db:"quota_requests_used"`

	// Status
	Enabled    bool       `json:"enabled" db:"enabled"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`

	// Security
	IPWhitelist      []string `json:"ip_whitelist,omitempty" db:"ip_whitelist"`
	AllowedEndpoints []string `json:"allowed_endpoints,omitempty" db:"allowed_endpoints"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// APIKeyWithSecret includes the secret key (only used during creation)
type APIKeyWithSecret struct {
	*APIKey
	SecretKey string `json:"secret_key"`
}

// CreateAPIKeyRequest represents a request to create a new API key
type CreateAPIKeyRequest struct {
	Name               string                 `json:"name" binding:"required,min=3,max=255"`
	Description        string                 `json:"description,omitempty"`
	RateLimitPerMinute *int                   `json:"rate_limit_per_minute,omitempty"`
	RateLimitPerHour   *int                   `json:"rate_limit_per_hour,omitempty"`
	RateLimitPerDay    *int                   `json:"rate_limit_per_day,omitempty"`
	QuotaRequestsTotal *int                   `json:"quota_requests_total,omitempty"`
	ExpiresAt          *time.Time             `json:"expires_at,omitempty"`
	IPWhitelist        []string               `json:"ip_whitelist,omitempty"`
	AllowedEndpoints   []string               `json:"allowed_endpoints,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateAPIKeyRequest represents a request to update an API key
type UpdateAPIKeyRequest struct {
	Name               *string                `json:"name,omitempty"`
	Description        *string                `json:"description,omitempty"`
	Enabled            *bool                  `json:"enabled,omitempty"`
	RateLimitPerMinute *int                   `json:"rate_limit_per_minute,omitempty"`
	RateLimitPerHour   *int                   `json:"rate_limit_per_hour,omitempty"`
	RateLimitPerDay    *int                   `json:"rate_limit_per_day,omitempty"`
	QuotaRequestsTotal *int                   `json:"quota_requests_total,omitempty"`
	IPWhitelist        []string               `json:"ip_whitelist,omitempty"`
	AllowedEndpoints   []string               `json:"allowed_endpoints,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// ListAPIKeysQuery represents query parameters for listing API keys
type ListAPIKeysQuery struct {
	UserID  string `form:"user_id"`
	Enabled *bool  `form:"enabled"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}

// APIKeyUsage represents a single API key usage record
type APIKeyUsage struct {
	ID             int64     `json:"id" db:"id"`
	KeyID          string    `json:"key_id" db:"key_id"`
	Endpoint       string    `json:"endpoint" db:"endpoint"`
	Method         string    `json:"method" db:"method"`
	StatusCode     int       `json:"status_code" db:"status_code"`
	ResponseTimeMs int       `json:"response_time_ms" db:"response_time_ms"`
	RequestIP      string    `json:"request_ip" db:"request_ip"`
	UserAgent      string    `json:"user_agent" db:"user_agent"`
	ErrorMessage   string    `json:"error_message,omitempty" db:"error_message"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
}

// APIKeyStats represents aggregated statistics for an API key
type APIKeyStats struct {
	KeyID              string     `json:"key_id" db:"key_id"`
	Name               string     `json:"name" db:"name"`
	Enabled            bool       `json:"enabled" db:"enabled"`
	QuotaRequestsTotal *int       `json:"quota_requests_total,omitempty" db:"quota_requests_total"`
	QuotaRequestsUsed  int        `json:"quota_requests_used" db:"quota_requests_used"`
	RequestsToday      int        `json:"requests_today" db:"requests_today"`
	RequestsThisWeek   int        `json:"requests_this_week" db:"requests_this_week"`
	RequestsThisMonth  int        `json:"requests_this_month" db:"requests_this_month"`
	AvgResponseTime24h float64    `json:"avg_response_time_24h" db:"avg_response_time_24h"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// RegenerateSecretResponse represents the response when regenerating a secret
type RegenerateSecretResponse struct {
	KeyID         string    `json:"key_id"`
	SecretKey     string    `json:"secret_key"`
	RegeneratedAt time.Time `json:"regenerated_at"`
}
