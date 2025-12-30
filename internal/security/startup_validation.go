package security

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// StartupValidator validates critical security configuration at server startup
type StartupValidator struct {
	productionMode bool
	errors         []string
	warnings       []string
}

// NewStartupValidator creates a new startup validator
func NewStartupValidator(productionMode bool) *StartupValidator {
	return &StartupValidator{
		productionMode: productionMode,
		errors:         make([]string, 0),
		warnings:       make([]string, 0),
	}
}

// ValidateAll performs all security validations
func (v *StartupValidator) ValidateAll() error {
	v.validateJWTSigningKey()
	v.validateProductionMode()
	v.validateCORSConfiguration()
	v.validateDatabaseCredentials()
	v.validateReplayStore() // CV-2025-005: Container replay store validation

	if len(v.errors) > 0 {
		return fmt.Errorf("security validation failed:\n%s", strings.Join(v.errors, "\n"))
	}

	if len(v.warnings) > 0 {
		for _, warning := range v.warnings {
			fmt.Fprintf(os.Stderr, "[SECURITY WARNING] %s\n", warning)
		}
	}

	return nil
}

// validateJWTSigningKey ensures JWT signing key is secure
func (v *StartupValidator) validateJWTSigningKey() {
	signingKey := os.Getenv("AGENTAUTH_JWT_SIGNING_KEY")

	// CRITICAL: Must be set
	if signingKey == "" {
		v.errors = append(v.errors, "AGENTAUTH_JWT_SIGNING_KEY is not set - server cannot start without a signing key")
		return
	}

	// CRITICAL: Known weak values
	weakKeys := []string{
		"dev-please-change",
		"dev-signing-key-change-in-production",
		"changeme",
		"secret",
		"test",
		"development",
		"demo",
	}

	for _, weak := range weakKeys {
		if signingKey == weak {
			v.errors = append(v.errors, fmt.Sprintf(
				"AGENTAUTH_JWT_SIGNING_KEY is set to known weak value '%s' - this allows attackers to forge PoA tokens and bypass all PDP checks. Set a strong random key (min 32 bytes).",
				weak,
			))
			return
		}
	}

	// WARNING: Short keys in production
	if v.productionMode && len(signingKey) < 32 {
		v.warnings = append(v.warnings, fmt.Sprintf(
			"AGENTAUTH_JWT_SIGNING_KEY is only %d bytes - recommended minimum is 32 bytes for production use",
			len(signingKey),
		))
	}

	// WARNING: Base patterns that suggest dev keys
	if v.productionMode {
		lowercaseKey := strings.ToLower(signingKey)
		devPatterns := []string{"dev", "test", "demo", "example", "sample"}
		for _, pattern := range devPatterns {
			if strings.Contains(lowercaseKey, pattern) {
				v.warnings = append(v.warnings, fmt.Sprintf(
					"AGENTAUTH_JWT_SIGNING_KEY contains '%s' - ensure this is not a development key",
					pattern,
				))
				break
			}
		}
	}
}

// validateProductionMode ensures production-specific requirements
func (v *StartupValidator) validateProductionMode() {
	if !v.productionMode {
		return
	}

	// In production, these MUST be disabled
	if os.Getenv("AGENTAUTH_DEV_INDEX") == "1" {
		v.errors = append(v.errors, "AGENTAUTH_DEV_INDEX=1 exposes debug UI and development endpoints - MUST be disabled in production (unset or set to 0)")
	}

	if os.Getenv("AGENTAUTH_DEV_MODE") == "true" || os.Getenv("AGENTAUTH_DEV_MODE") == "1" {
		v.errors = append(v.errors, "AGENTAUTH_DEV_MODE enables development shortcuts - MUST be disabled in production")
	}

	// Rate limiting should be enabled in production
	if os.Getenv("AGENTAUTH_RATE_LIMIT_ENABLED") == "false" || os.Getenv("AGENTAUTH_RATE_LIMIT_ENABLED") == "0" {
		v.warnings = append(v.warnings, "AGENTAUTH_RATE_LIMIT_ENABLED is disabled - strongly recommended for production")
	}

	// External identity verification should be configured
	pvpProvider := os.Getenv("AGENTAUTH_PVP_PROVIDER")
	if pvpProvider == "" || pvpProvider == "mock" {
		v.warnings = append(v.warnings, "AGENTAUTH_PVP_PROVIDER not set or set to 'mock' - production should use external identity verification (e.g., 'stripe', 'idemia', 'veriff')")
	}
}

// validateCORSConfiguration checks CORS settings
func (v *StartupValidator) validateCORSConfiguration() {
	corsAllow := os.Getenv("AGENTAUTH_CORS_ALLOW")

	if corsAllow == "*" && v.productionMode {
		v.errors = append(v.errors, "AGENTAUTH_CORS_ALLOW='*' allows any origin - MUST be restricted to specific domains in production")
	}

	if strings.Contains(corsAllow, "localhost") && v.productionMode {
		v.warnings = append(v.warnings, "AGENTAUTH_CORS_ALLOW contains 'localhost' - typically not needed in production")
	}
}

// validateDatabaseCredentials checks database security
func (v *StartupValidator) validateDatabaseCredentials() {
	dbPassword := os.Getenv("AGENTAUTH_DB_PASSWORD")

	weakPasswords := []string{
		"dev-password-please-change",
		"dev-password-change-me",
		"password",
		"changeme",
		"admin",
		"root",
		"postgres",
	}

	for _, weak := range weakPasswords {
		if dbPassword == weak {
			if v.productionMode {
				v.errors = append(v.errors, fmt.Sprintf(
					"AGENTAUTH_DB_PASSWORD is set to known weak value '%s' - database is vulnerable to credential stuffing attacks",
					weak,
				))
			} else {
				v.warnings = append(v.warnings, fmt.Sprintf(
					"AGENTAUTH_DB_PASSWORD is set to development default '%s' - change before deploying to production",
					weak,
				))
			}
			break
		}
	}
}

// validateReplayStore checks replay store configuration for security
// Addresses CV-2025-005: BoltDB ephemeral storage vulnerability
func (v *StartupValidator) validateReplayStore() {
	// Check if we're in a container
	env, inContainer := IsRunningInContainer()

	if inContainer {
		// In containers, BoltDB is extremely risky
		if os.Getenv("AGENTAUTH_REPLAY_STORE") == "bolt" || os.Getenv("AGENTAUTH_REPLAY_STORE_PATH") != "" {
			if os.Getenv("AGENTAUTH_ALLOW_UNSAFE_BOLTDB") == "1" {
				v.warnings = append(v.warnings, fmt.Sprintf(
					"BoltDB replay store enabled in %s with safety bypass - UNSAFE for production (CV-2025-005). "+
						"Replay protection will FAIL after container restart unless using persistent volume. "+
						"Migrate to Redis for production deployments. See REPLAY_STORE_MIGRATION_GUIDE.md",
					env,
				))
			} else {
				// This is good - safety checks are active
				if v.productionMode {
					v.warnings = append(v.warnings, fmt.Sprintf(
						"Running in %s - ensure replay store uses Redis or persistent volume (not BoltDB with ephemeral storage)",
						env,
					))
				}
			}
		}

		// Recommend Redis for production in containers
		if v.productionMode {
			redisHost := os.Getenv("REDIS_HOST")
			redisAddr := os.Getenv("REDIS_ADDR")
			if redisHost == "" && redisAddr == "" {
				v.warnings = append(v.warnings, fmt.Sprintf(
					"Running in %s without Redis configuration - replay store may not persist across restarts. "+
						"Set REDIS_HOST or REDIS_ADDR for production deployments.",
					env,
				))
			}
		}
	}

	// General production recommendations
	if v.productionMode {
		// Check for in-memory replay store in production
		if os.Getenv("AGENTAUTH_REPLAY_STORE") == "memory" {
			v.warnings = append(v.warnings,
				"In-memory replay store detected in production - replay protection will not persist across restarts. "+
					"Use Redis or distributed store for production.")
		}
	}
}

// GetErrors returns all validation errors
func (v *StartupValidator) GetErrors() []string {
	return v.errors
}

// GetWarnings returns all validation warnings
func (v *StartupValidator) GetWarnings() []string {
	return v.warnings
}

// URIValidator validates URIs for SSRF protection
type URIValidator struct {
	allowedSchemes []string
	blockPrivateIP bool
	blockMetadata  bool
}

// NewURIValidator creates a URI validator with safe defaults
func NewURIValidator() *URIValidator {
	return &URIValidator{
		allowedSchemes: []string{"https"},
		blockPrivateIP: true,
		blockMetadata:  true,
	}
}

// NewURIValidatorWithSchemes creates a URI validator with specific allowed schemes
func NewURIValidatorWithSchemes(schemes []string) *URIValidator {
	return &URIValidator{
		allowedSchemes: schemes,
		blockPrivateIP: true,
		blockMetadata:  true,
	}
}

// ValidateURI validates a URI for security (SSRF protection)
func (v *URIValidator) ValidateURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("URI cannot be empty")
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid URI format: %w", err)
	}

	// Check scheme allowlist
	schemeAllowed := false
	for _, allowed := range v.allowedSchemes {
		if parsed.Scheme == allowed {
			schemeAllowed = true
			break
		}
	}

	if !schemeAllowed {
		return fmt.Errorf("URI scheme '%s' not allowed (allowed: %v) - potential SSRF attack vector",
			parsed.Scheme, v.allowedSchemes)
	}

	// Block file:// explicitly (common SSRF vector)
	if parsed.Scheme == "file" {
		return fmt.Errorf("file:// scheme blocked - potential local file disclosure attack")
	}

	// For network schemes, validate hostname
	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		if err := v.validateHostname(parsed.Hostname()); err != nil {
			return err
		}
	}

	return nil
}

// validateHostname checks hostname for SSRF vectors
func (v *URIValidator) validateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}

	// Block localhost and variants
	localhostVariants := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"::1",
		"[::1]",
	}

	for _, variant := range localhostVariants {
		if strings.EqualFold(hostname, variant) {
			return fmt.Errorf("localhost access blocked - potential SSRF to local services")
		}
	}

	// Block cloud metadata endpoints
	if v.blockMetadata {
		metadataHosts := []string{
			"169.254.169.254",          // AWS/Azure/GCP metadata
			"metadata.google.internal", // GCP
			"169.254.169.253",          // AWS ECS
		}

		for _, meta := range metadataHosts {
			if strings.EqualFold(hostname, meta) {
				return fmt.Errorf("cloud metadata endpoint blocked - potential SSRF to steal credentials")
			}
		}
	}

	// Parse and validate IP addresses
	if v.blockPrivateIP {
		ip := net.ParseIP(hostname)
		if ip != nil {
			if isPrivateIP(ip) {
				return fmt.Errorf("private IP address blocked - potential SSRF to internal network")
			}
		}
	}

	return nil
}

// isPrivateIP checks if an IP is in private address space
func isPrivateIP(ip net.IP) bool {
	// RFC 1918 private IPv4 ranges
	privateIPv4Blocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",    // Loopback
		"169.254.0.0/16", // Link-local
	}

	for _, block := range privateIPv4Blocks {
		_, subnet, _ := net.ParseCIDR(block)
		if subnet != nil && subnet.Contains(ip) {
			return true
		}
	}

	// IPv6 private ranges
	if ip.To4() == nil {
		// Check for IPv6 loopback and link-local
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
		// Check for IPv6 ULA (fc00::/7)
		if len(ip) == 16 && ip[0] == 0xfc || ip[0] == 0xfd {
			return true
		}
	}

	return false
}

// ProductionModeDetector determines if server is running in production
func ProductionModeDetector() bool {
	env := strings.ToLower(os.Getenv("AGENTAUTH_ENV"))
	if env == "production" || env == "prod" {
		return true
	}

	mode := strings.ToLower(os.Getenv("AGENTAUTH_MODE"))
	if mode == "production" || mode == "prod" {
		return true
	}

	// Absence of dev indicators suggests production
	if os.Getenv("AGENTAUTH_DEV_MODE") == "" && os.Getenv("AGENTAUTH_DEV_INDEX") == "" {
		// Check for production-like port binding
		port := os.Getenv("AGENTAUTH_PORT")
		if port == "443" || port == "8443" {
			return true
		}
	}

	return false
}
