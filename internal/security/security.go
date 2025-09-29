package security

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/hashicorp/vault/api"
	"golang.org/x/crypto/bcrypt"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	TLSConfig      *tls.Config
	VaultConfig    *api.Config
	TokenLifetime  time.Duration
	MaxFailedLogin int
	IPBlacklist    []net.IPNet
}

// SecurityManager handles security-related operations
type SecurityManager struct {
	config    *SecurityConfig
	vaultAPI  *api.Client
	blacklist map[string]time.Time
	rateLimit map[string][]time.Time
	hashCosts int
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityConfig) (*SecurityManager, error) {
	vaultClient, err := api.NewClient(config.VaultConfig)
	if err != nil {
		return nil, err
	}

	return &SecurityManager{
		config:    config,
		vaultAPI:  vaultClient,
		blacklist: make(map[string]time.Time),
		rateLimit: make(map[string][]time.Time),
		hashCosts: 12, // Adjust based on server capability
	}, nil
}

// HashPassword securely hashes a password using bcrypt
func (sm *SecurityManager) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), sm.hashCosts)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ValidatePassword checks if a password matches its hash
func (sm *SecurityManager) ValidatePassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// IsIPAllowed checks if an IP is allowed based on blacklist and rate limiting
func (sm *SecurityManager) IsIPAllowed(ip string) bool {
	// Check blacklist
	if blockedUntil, exists := sm.blacklist[ip]; exists {
		if time.Now().Before(blockedUntil) {
			return false
		}
		delete(sm.blacklist, ip)
	}

	// Rate limiting
	now := time.Now()
	requests := sm.rateLimit[ip]

	// Remove old requests
	validRequests := []time.Time{}
	for _, req := range requests {
		if now.Sub(req) < time.Minute {
			validRequests = append(validRequests, req)
		}
	}

	// Check rate limit
	if len(validRequests) >= 100 { // 100 requests per minute
		return false
	}

	// Update rate limit
	sm.rateLimit[ip] = append(validRequests, now)
	return true
}

// BlacklistIP temporarily blacklists an IP
func (sm *SecurityManager) BlacklistIP(ip string, duration time.Duration) {
	sm.blacklist[ip] = time.Now().Add(duration)
}

// GetSecret retrieves a secret from Vault
func (sm *SecurityManager) GetSecret(ctx context.Context, path string) (string, error) {
	secret, err := sm.vaultAPI.KVv2("secret").Get(ctx, path)
	if err != nil {
		return "", err
	}
	return secret.Data["value"].(string), nil
}

// RotateKeys rotates encryption keys
func (sm *SecurityManager) RotateKeys(ctx context.Context) error {
	// Implementation for key rotation
	return nil
}

// ValidateToken validates a JWT token
func (sm *SecurityManager) ValidateToken(_ string) (bool, error) {
	// Implementation for token validation
	return false, nil
}

// SecureHeaders returns secure HTTP headers
func (sm *SecurityManager) SecureHeaders() map[string]string {
	return map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Frame-Options":           "DENY",
		"X-Content-Type-Options":    "nosniff",
		"Content-Security-Policy":   "default-src 'self'",
		"X-XSS-Protection":          "1; mode=block",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}
}
