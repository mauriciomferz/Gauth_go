package config

import (
	"os"
	"strconv"
	"strings"
)

// CascadeMode represents the cascade revocation processing mode
type CascadeMode string

const (
	CascadeModeRevoke  CascadeMode = "revoke"  // Immediately revoke descendants
	CascadeModeSuspend CascadeMode = "suspend" // Suspend descendants pending review
	CascadeModeNotify  CascadeMode = "notify"  // Only emit metrics, no status changes
)

// CascadeConfig holds cascade revocation configuration
type CascadeConfig struct {
	// Enabled controls whether cascade revocation processing is active
	Enabled bool
	
	// Mode determines how descendants are processed when parent is revoked
	Mode CascadeMode
	
	// MaxDepth limits cascade depth to prevent runaway processing (0 = unlimited)
	MaxDepth int
	
	// BatchSize controls how many descendants are processed in each batch
	BatchSize int
}

// DefaultCascadeConfig returns sensible defaults for cascade configuration
func DefaultCascadeConfig() CascadeConfig {
	return CascadeConfig{
		Enabled:   false,
		Mode:      CascadeModeSuspend,
		MaxDepth:  10,
		BatchSize: 100,
	}
}

// LoadCascadeConfigFromEnv loads cascade configuration from environment variables
func LoadCascadeConfigFromEnv() CascadeConfig {
	config := DefaultCascadeConfig()
	
	// GAUTH_CASCADE_PARENT_REVOCATION enables cascade processing
	if enabled := parseBoolEnv("GAUTH_CASCADE_PARENT_REVOCATION"); enabled {
		config.Enabled = enabled
	}
	
	// GAUTH_CASCADE_MODE sets processing mode (revoke|suspend|notify)
	if mode := strings.ToLower(strings.TrimSpace(os.Getenv("GAUTH_CASCADE_MODE"))); mode != "" {
		switch mode {
		case "revoke":
			config.Mode = CascadeModeRevoke
		case "suspend":
			config.Mode = CascadeModeSuspend  
		case "notify":
			config.Mode = CascadeModeNotify
		}
	}
	
	// GAUTH_CASCADE_MAX_DEPTH limits cascade depth
	if depth := getIntEnv("GAUTH_CASCADE_MAX_DEPTH", int64(config.MaxDepth)); depth >= 0 {
		config.MaxDepth = int(depth)
	}
	
	// GAUTH_CASCADE_BATCH_SIZE controls batch processing size
	if batchSize := getIntEnv("GAUTH_CASCADE_BATCH_SIZE", int64(config.BatchSize)); batchSize > 0 {
		config.BatchSize = int(batchSize)
	}
	
	return config
}

// parseBoolEnv parses a boolean environment variable
func parseBoolEnv(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// getIntEnv parses an integer environment variable with fallback
func getIntEnv(key string, defaultVal int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return defaultVal
	}
	return n
}

// IsValidMode checks if the cascade mode is valid
func (c CascadeConfig) IsValidMode() bool {
	switch c.Mode {
	case CascadeModeRevoke, CascadeModeSuspend, CascadeModeNotify:
		return true
	default:
		return false
	}
}

// ShouldProcessCascade returns true if cascade processing should occur
func (c CascadeConfig) ShouldProcessCascade() bool {
	return c.Enabled && c.IsValidMode()
}

// IsUnlimitedDepth returns true if cascade depth is unlimited
func (c CascadeConfig) IsUnlimitedDepth() bool {
	return c.MaxDepth == 0
}