// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// ProperConfig provides structured configuration management
// This replaces hardcoded values throughout the codebase
type ProperConfig struct {
	// Server configuration
	Server ServerConfig `json:"server"`
	
	// Security configuration
	Security SecurityConfig `json:"security"`
	
	// Database configuration
	Database DatabaseConfig `json:"database"`
	
	// Redis configuration
	Redis RedisConfig `json:"redis"`
	
	// Rate limiting configuration
	RateLimit RateLimitConfig `json:"rate_limit"`
	
	// Logging configuration
	Logging LoggingConfig `json:"logging"`
	
	// Compliance configuration
	Compliance ComplianceConfig `json:"compliance"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
	Environment  string        `json:"environment"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// JWT configuration
	JWTIssuer          string        `json:"jwt_issuer"`
	JWTAudience        string        `json:"jwt_audience"`
	JWTExpiration      time.Duration `json:"jwt_expiration"`
	JWTRefreshWindow   time.Duration `json:"jwt_refresh_window"`
	
	// Cryptographic keys (base64 encoded)
	HMACKey        string `json:"hmac_key"`
	EncryptionKey  string `json:"encryption_key"`
	SigningKey     string `json:"signing_key"`
	
	// Security policies
	MaxTokenAge           time.Duration `json:"max_token_age"`
	RequireHTTPS          bool          `json:"require_https"`
	EnableCSRFProtection  bool          `json:"enable_csrf_protection"`
	EnableCORSProtection  bool          `json:"enable_cors_protection"`
	MaxSessionAge         time.Duration `json:"max_session_age"`
	
	// Password policies
	MinPasswordLength     int           `json:"min_password_length"`
	RequireStrongPassword bool          `json:"require_strong_password"`
	PasswordHashCost      int           `json:"password_hash_cost"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Database     string        `json:"database"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	SSLMode      string        `json:"ssl_mode"`
	MaxConns     int           `json:"max_conns"`
	MaxIdleConns int           `json:"max_idle_conns"`
	ConnTimeout  time.Duration `json:"conn_timeout"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Password        string        `json:"password"`
	Database        int           `json:"database"`
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	PoolSize        int           `json:"pool_size"`
	MinIdleConns    int           `json:"min_idle_conns"`
	MaxConnAge      time.Duration `json:"max_conn_age"`
	PoolTimeout     time.Duration `json:"pool_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	IdleCheckFreq   time.Duration `json:"idle_check_freq"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Global rate limits
	GlobalRequestsPerSecond rate.Limit   `json:"global_requests_per_second"`
	GlobalBurst             int          `json:"global_burst"`
	
	// Per-user rate limits
	UserRequestsPerSecond   rate.Limit   `json:"user_requests_per_second"`
	UserBurst               int          `json:"user_burst"`
	
	// Per-IP rate limits
	IPRequestsPerSecond     rate.Limit   `json:"ip_requests_per_second"`
	IPBurst                 int          `json:"ip_burst"`
	
	// Cleanup settings
	CleanupInterval         time.Duration `json:"cleanup_interval"`
	LimiterExpiry           time.Duration `json:"limiter_expiry"`
	
	// Circuit breaker settings
	MaxFailures             int64         `json:"max_failures"`
	ResetTimeout            time.Duration `json:"reset_timeout"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level       string `json:"level"`
	Format      string `json:"format"`
	Output      string `json:"output"`
	EnableJSON  bool   `json:"enable_json"`
	EnableTrace bool   `json:"enable_trace"`
}

// ComplianceConfig holds compliance-related configuration
type ComplianceConfig struct {
	EnableGDPR          bool          `json:"enable_gdpr"`
	DataRetentionPeriod time.Duration `json:"data_retention_period"`
	AuditLogRetention   time.Duration `json:"audit_log_retention"`
	EnableDataResidency bool          `json:"enable_data_residency"`
	AllowedRegions      []string      `json:"allowed_regions"`
	EnableEncryption    bool          `json:"enable_encryption"`
}

// LoadConfig loads configuration from environment variables with secure defaults
func LoadConfig() (*ProperConfig, error) {
	config := &ProperConfig{
		Server: ServerConfig{
			Host:         getEnvString("SERVER_HOST", "localhost"),
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
			Environment:  getEnvString("ENVIRONMENT", "development"),
		},
		Security: SecurityConfig{
			JWTIssuer:             getEnvString("JWT_ISSUER", "gauth-service"),
			JWTAudience:           getEnvString("JWT_AUDIENCE", "gauth-api"),
			JWTExpiration:         getEnvDuration("JWT_EXPIRATION", 1*time.Hour),
			JWTRefreshWindow:      getEnvDuration("JWT_REFRESH_WINDOW", 15*time.Minute),
			MaxTokenAge:           getEnvDuration("MAX_TOKEN_AGE", 24*time.Hour),
			RequireHTTPS:          getEnvBool("REQUIRE_HTTPS", true),
			EnableCSRFProtection:  getEnvBool("ENABLE_CSRF", true),
			EnableCORSProtection:  getEnvBool("ENABLE_CORS", true),
			MaxSessionAge:         getEnvDuration("MAX_SESSION_AGE", 8*time.Hour),
			MinPasswordLength:     getEnvInt("MIN_PASSWORD_LENGTH", 12),
			RequireStrongPassword: getEnvBool("REQUIRE_STRONG_PASSWORD", true),
			PasswordHashCost:      getEnvInt("PASSWORD_HASH_COST", 12),
		},
		Database: DatabaseConfig{
			Host:         getEnvString("DB_HOST", "localhost"),
			Port:         getEnvInt("DB_PORT", 5432),
			Database:     getEnvString("DB_NAME", "gauth"),
			Username:     getEnvString("DB_USER", "gauth"),
			Password:     getEnvString("DB_PASSWORD", ""),
			SSLMode:      getEnvString("DB_SSL_MODE", "require"),
			MaxConns:     getEnvInt("DB_MAX_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnTimeout:  getEnvDuration("DB_CONN_TIMEOUT", 30*time.Second),
		},
		Redis: RedisConfig{
			Host:          getEnvString("REDIS_HOST", "localhost"),
			Port:          getEnvInt("REDIS_PORT", 6379),
			Password:      getEnvString("REDIS_PASSWORD", ""),
			Database:      getEnvInt("REDIS_DB", 0),
			MaxRetries:    getEnvInt("REDIS_MAX_RETRIES", 3),
			RetryDelay:    getEnvDuration("REDIS_RETRY_DELAY", 1*time.Second),
			PoolSize:      getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns:  getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
			MaxConnAge:    getEnvDuration("REDIS_MAX_CONN_AGE", 1*time.Hour),
			PoolTimeout:   getEnvDuration("REDIS_POOL_TIMEOUT", 5*time.Second),
			IdleTimeout:   getEnvDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
			IdleCheckFreq: getEnvDuration("REDIS_IDLE_CHECK_FREQ", 1*time.Minute),
		},
		RateLimit: RateLimitConfig{
			GlobalRequestsPerSecond: rate.Limit(getEnvFloat("GLOBAL_RATE_LIMIT", 1000.0)),
			GlobalBurst:             getEnvInt("GLOBAL_BURST", 100),
			UserRequestsPerSecond:   rate.Limit(getEnvFloat("USER_RATE_LIMIT", 10.0)),
			UserBurst:               getEnvInt("USER_BURST", 5),
			IPRequestsPerSecond:     rate.Limit(getEnvFloat("IP_RATE_LIMIT", 100.0)),
			IPBurst:                 getEnvInt("IP_BURST", 20),
			CleanupInterval:         getEnvDuration("RATE_LIMIT_CLEANUP", 1*time.Hour),
			LimiterExpiry:           getEnvDuration("RATE_LIMIT_EXPIRY", 24*time.Hour),
			MaxFailures:             int64(getEnvInt("CIRCUIT_BREAKER_MAX_FAILURES", 5)),
			ResetTimeout:            getEnvDuration("CIRCUIT_BREAKER_RESET", 1*time.Minute),
		},
		Logging: LoggingConfig{
			Level:       getEnvString("LOG_LEVEL", "info"),
			Format:      getEnvString("LOG_FORMAT", "json"),
			Output:      getEnvString("LOG_OUTPUT", "stdout"),
			EnableJSON:  getEnvBool("LOG_JSON", true),
			EnableTrace: getEnvBool("LOG_TRACE", false),
		},
		Compliance: ComplianceConfig{
			EnableGDPR:          getEnvBool("ENABLE_GDPR", true),
			DataRetentionPeriod: getEnvDuration("DATA_RETENTION_PERIOD", 2*365*24*time.Hour), // 2 years
			AuditLogRetention:   getEnvDuration("AUDIT_LOG_RETENTION", 7*365*24*time.Hour),   // 7 years
			EnableDataResidency: getEnvBool("ENABLE_DATA_RESIDENCY", true),
			AllowedRegions:      getEnvStringSlice("ALLOWED_REGIONS", []string{"EU", "US"}),
			EnableEncryption:    getEnvBool("ENABLE_ENCRYPTION", true),
		},
	}
	
	// Generate or load cryptographic keys
	if err := config.loadOrGenerateKeys(); err != nil {
		return nil, fmt.Errorf("failed to load cryptographic keys: %w", err)
	}
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// loadOrGenerateKeys loads existing keys or generates new ones
func (c *ProperConfig) loadOrGenerateKeys() error {
	// HMAC Key
	if c.Security.HMACKey = os.Getenv("HMAC_KEY"); c.Security.HMACKey == "" {
		key, err := generateSecureKey(32)
		if err != nil {
			return fmt.Errorf("failed to generate HMAC key: %w", err)
		}
		c.Security.HMACKey = key
	}
	
	// Encryption Key
	if c.Security.EncryptionKey = os.Getenv("ENCRYPTION_KEY"); c.Security.EncryptionKey == "" {
		key, err := generateSecureKey(32)
		if err != nil {
			return fmt.Errorf("failed to generate encryption key: %w", err)
		}
		c.Security.EncryptionKey = key
	}
	
	// Signing Key
	if c.Security.SigningKey = os.Getenv("SIGNING_KEY"); c.Security.SigningKey == "" {
		key, err := generateSecureKey(64)
		if err != nil {
			return fmt.Errorf("failed to generate signing key: %w", err)
		}
		c.Security.SigningKey = key
	}
	
	return nil
}

// generateSecureKey generates a cryptographically secure key
func generateSecureKey(length int) (string, error) {
	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Validate validates the configuration
func (c *ProperConfig) Validate() error {
	// Validate server configuration
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	
	// Validate security configuration
	if c.Security.JWTExpiration < time.Minute {
		return fmt.Errorf("JWT expiration too short: %v", c.Security.JWTExpiration)
	}
	
	if c.Security.MinPasswordLength < 8 {
		return fmt.Errorf("minimum password length too short: %d", c.Security.MinPasswordLength)
	}
	
	// Validate cryptographic keys
	if err := c.validateKeys(); err != nil {
		return fmt.Errorf("key validation failed: %w", err)
	}
	
	// Validate rate limiting
	if c.RateLimit.GlobalRequestsPerSecond <= 0 {
		return fmt.Errorf("global rate limit must be positive")
	}
	
	return nil
}

// validateKeys validates that cryptographic keys are properly configured
func (c *ProperConfig) validateKeys() error {
	// Validate HMAC key
	hmacKey, err := base64.StdEncoding.DecodeString(c.Security.HMACKey)
	if err != nil {
		return fmt.Errorf("invalid HMAC key encoding: %w", err)
	}
	if len(hmacKey) < 32 {
		return fmt.Errorf("HMAC key too short: %d bytes (minimum 32)", len(hmacKey))
	}
	
	// Validate encryption key
	encKey, err := base64.StdEncoding.DecodeString(c.Security.EncryptionKey)
	if err != nil {
		return fmt.Errorf("invalid encryption key encoding: %w", err)
	}
	if len(encKey) != 32 {
		return fmt.Errorf("encryption key must be exactly 32 bytes, got %d", len(encKey))
	}
	
	// Validate signing key
	sigKey, err := base64.StdEncoding.DecodeString(c.Security.SigningKey)
	if err != nil {
		return fmt.Errorf("invalid signing key encoding: %w", err)
	}
	if len(sigKey) < 32 {
		return fmt.Errorf("signing key too short: %d bytes (minimum 32)", len(sigKey))
	}
	
	return nil
}

// GetHMACKey returns the decoded HMAC key
func (c *ProperConfig) GetHMACKey() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Security.HMACKey)
}

// GetEncryptionKey returns the decoded encryption key
func (c *ProperConfig) GetEncryptionKey() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Security.EncryptionKey)
}

// GetSigningKey returns the decoded signing key
func (c *ProperConfig) GetSigningKey() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Security.SigningKey)
}

// IsProduction returns true if running in production environment
func (c *ProperConfig) IsProduction() bool {
	return c.Server.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *ProperConfig) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing (could be enhanced)
		return []string{value} // Simplified for now
	}
	return defaultValue
}