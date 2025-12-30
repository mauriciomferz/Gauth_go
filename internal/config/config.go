package config

import (
	"os"
)

// Config aggregates all application configuration
type Config struct {
	// Core AgentAuth Settings
	TokenSigMode      string
	JTITTLSeconds     int
	KeyRotationHours  int
	LegacyOAuthMode   bool
	UseJWTLib         bool
	JWTKeyID          string
	JWTAlg            string
	StrictJSONParsing bool

	// Sub-configurations
	Cascade CascadeConfig
	JWE     *JWEConfig
	AAP001 *AAP001Config
}

// JWEConfig holds JWE encryption configuration
type JWEConfig struct {
	Enabled         bool
	Algorithm       string
	Encryption      string
	PublicKeyPath   string
	PrivateKeyPath  string
	KeyID           string
	KeyRotationDays int
}

// AAP001Config holds RFC-0111 compliance configuration
type AAP001Config struct {
	Enabled  bool
	UseMocks bool
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		TokenSigMode:      Get("GAUTH_TOKEN_SIG_MODE", "hmac"),
		JTITTLSeconds:     int(GetInt("GAUTH_JTI_TTL_SEC", 600)), // 10 minutes default
		KeyRotationHours:  int(GetInt("GAUTH_KEY_ROTATION_HOURS", 24)),
		LegacyOAuthMode:   parseBoolEnv("GAUTH_LEGACY_OAUTH_MODE"),
		UseJWTLib:         parseBoolEnv("GAUTH_USE_JWT_LIB"),
		JWTKeyID:          Get("GAUTH_JWT_KID", "demo-key"),
		JWTAlg:            Get("GAUTH_JWT_ALG", "HS256"),
		StrictJSONParsing: parseBoolEnv("GAUTH_STRICT_JSON_PARSING"),
		Cascade:           LoadCascadeConfigFromEnv(),
		JWE:               loadJWEConfig(),
		AAP001:           loadAAP001Config(),
	}

	return cfg, nil
}

func loadJWEConfig() *JWEConfig {
	cfg := &JWEConfig{
		Enabled:         parseBoolEnv("GAUTH_JWE_ENABLED"),
		Algorithm:       Get("GAUTH_JWE_ALGORITHM", "RSA-OAEP-256"),
		Encryption:      Get("GAUTH_JWE_ENCRYPTION", "A256GCM"),
		PublicKeyPath:   os.Getenv("GAUTH_JWE_PUBLIC_KEY"),
		PrivateKeyPath:  os.Getenv("GAUTH_JWE_PRIVATE_KEY"),
		KeyID:           os.Getenv("GAUTH_JWE_KEY_ID"),
		KeyRotationDays: int(GetInt("GAUTH_JWE_ROTATION_DAYS", 365)),
	}
	// Default enabled to true if not set, based on previous logic?
	// Actually previous logic in jwe_env_config.go said "default: true" but code checked if env != "".
	// Let's stick to explicit env vars for now to be safe, or check the original file.
	// Original: if envEnabled != "" { ... } else { config.Enabled = true } (via DefaultJWEConfig)
	// We should probably replicate defaults.
	if os.Getenv("GAUTH_JWE_ENABLED") == "" {
		cfg.Enabled = true
	}
	return cfg
}

func loadAAP001Config() *AAP001Config {
	cfg := &AAP001Config{
		Enabled:  os.Getenv("GAUTH_AAP001_ENABLED") == "1",
		UseMocks: true,
	}
	if val := os.Getenv("GAUTH_AAP001_USE_MOCKS"); val == "0" {
		cfg.UseMocks = false
	}
	return cfg
}

// Helper to parse bool env vars (reused from cascade.go if public, or redefined)
// Since cascade.go has parseBoolEnv but it's not exported, we redefine or export it.
// cascade.go is in the same package `config`, so we can use it if it's there.
// But wait, cascade.go has `func parseBoolEnv(key string) bool`. It is unexported.
// So we can use it directly since we are in package `config`.
