package tokenmanagement

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// Configuration for the token service
type Config struct {
	// Token settings
	TokenExpiry          time.Duration `json:"token_expiry"`
	RefreshTokenExpiry   time.Duration `json:"refresh_token_expiry"`
	TokenCleanupInterval time.Duration `json:"token_cleanup_interval"`

	// Security settings
	KeyRotationInterval time.Duration    `json:"key_rotation_interval"`
	MinKeySize          int              `json:"min_key_size"`
	AllowedIssuers      []string         `json:"allowed_issuers"`
	TrustedProxies      []string         `json:"trusted_proxies"`
	CorsSettings        *CorsConfig      `json:"cors"`
	RateLimits          *RateLimitConfig `json:"rate_limits"`

	// Storage settings
	StoreType      string          `json:"store_type"`
	RedisConfig    *RedisConfig    `json:"redis,omitempty"`
	PostgresConfig *PostgresConfig `json:"postgres,omitempty"`

	// TLS settings
	TLSConfig *TLSConfig `json:"tls"`
}

type CorsConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	BurstSize         int `json:"burst_size"`
	TokensPerUser     int `json:"tokens_per_user"`
}

type RedisConfig struct {
	Addresses   []string `json:"addresses"`
	Password    string   `json:"password"`
	DB          int      `json:"db"`
	ClusterMode bool     `json:"cluster_mode"`
}

type PostgresConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"`
}

type TLSConfig struct {
	CertFile     string   `json:"cert_file"`
	KeyFile      string   `json:"key_file"`
	MinVersion   string   `json:"min_version"`
	CipherSuites []string `json:"cipher_suites"`
}

// TokenService manages token operations with configuration
type TokenService struct {
	config     *Config
	store      token.Store
	blacklist  *token.Blacklist
	jwtMgr     *token.JWTManager
	validators []token.Validator
	tlsConfig  *tls.Config
	metrics    *ServiceMetrics
	mu         sync.RWMutex
}

// ServiceMetrics tracks service operations
type ServiceMetrics struct {
	TokensIssued   int64
	TokensRevoked  int64
	ActiveTokens   int64
	FailedRequests int64
	LastError      time.Time
	StartTime      time.Time
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func NewTokenService(config *Config) (*TokenService, error) {
	// Initialize store based on config
	var store token.Store
	switch config.StoreType {
	case "memory":
		store = token.NewMemoryStore(config.TokenExpiry)
	case "redis":
		// Initialize Redis store (implementation omitted)
	case "postgres":
		// Initialize Postgres store (implementation omitted)
	default:
		return nil, fmt.Errorf("unsupported store type: %s", config.StoreType)
	}

	// Generate or load TLS certificate
	tlsConfig, err := configureTLS(config.TLSConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}

	// Create service
	svc := &TokenService{
		config:    config,
		store:     store,
		blacklist: token.NewBlacklist(),
		metrics: &ServiceMetrics{
			StartTime: time.Now(),
		},
		tlsConfig: tlsConfig,
	}

	// Initialize validators
	svc.initValidators()

	return svc, nil
}

func configureTLS(config *TLSConfig) (*tls.Config, error) {
	// If cert/key files are provided, load them
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS cert/key: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}, nil
	}

	// Generate self-signed certificate
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"GAuth Token Service"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func (svc *TokenService) initValidators() {
	// Add issuer validator
	svc.validators = append(svc.validators, &IssuerValidator{
		allowedIssuers: svc.config.AllowedIssuers,
	})

	// Add rate limit validator
	if svc.config.RateLimits != nil {
		svc.validators = append(svc.validators, &RateLimitValidator{
			config: svc.config.RateLimits,
		})
	}
}

// IssuerValidator validates token issuers
type IssuerValidator struct {
	allowedIssuers []string
}

func (v *IssuerValidator) Validate(ctx context.Context, t *token.Token) error {
	for _, issuer := range v.allowedIssuers {
		if t.Issuer == issuer {
			return nil
		}
	}
	return fmt.Errorf("invalid issuer: %s", t.Issuer)
}

// RateLimitValidator enforces rate limits
type RateLimitValidator struct {
	config *RateLimitConfig
	mu     sync.RWMutex
	counts map[string]int
}

func (v *RateLimitValidator) Validate(ctx context.Context, t *token.Token) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.counts == nil {
		v.counts = make(map[string]int)
	}

	count := v.counts[t.Subject]
	if count >= v.config.TokensPerUser {
		return fmt.Errorf("rate limit exceeded for user: %s", t.Subject)
	}

	v.counts[t.Subject]++
	return nil
}

func main() {
	// Example configuration
	config := &Config{
		TokenExpiry:         time.Hour,
		RefreshTokenExpiry:  24 * time.Hour,
		KeyRotationInterval: 12 * time.Hour,
		MinKeySize:          2048,
		AllowedIssuers:      []string{"auth-service", "admin-service"},
		StoreType:           "memory",
		RateLimits: &RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
			TokensPerUser:     5,
		},
		CorsSettings: &CorsConfig{
			AllowedOrigins: []string{"https://example.com"},
			AllowedMethods: []string{"GET", "POST"},
			MaxAge:         3600,
		},
	}

	// Create token service
	service, err := NewTokenService(config)
	if err != nil {
		log.Fatalf("Failed to create token service: %v", err)
	}

	// Print service configuration
	fmt.Println("Token Service Configuration:")
	fmt.Printf("Store Type: %s\n", config.StoreType)
	fmt.Printf("Token Expiry: %v\n", config.TokenExpiry)
	fmt.Printf("Refresh Token Expiry: %v\n", config.RefreshTokenExpiry)
	fmt.Printf("Key Rotation Interval: %v\n", config.KeyRotationInterval)
	fmt.Printf("Allowed Issuers: %v\n", config.AllowedIssuers)

	fmt.Printf("\nRate Limits:\n")
	fmt.Printf("Requests/Minute: %d\n", config.RateLimits.RequestsPerMinute)
	fmt.Printf("Tokens/User: %d\n", config.RateLimits.TokensPerUser)

	fmt.Printf("\nCORS Settings:\n")
	fmt.Printf("Allowed Origins: %v\n", config.CorsSettings.AllowedOrigins)
	fmt.Printf("Max Age: %d\n", config.CorsSettings.MaxAge)

	fmt.Printf("\nService Metrics:\n")
	fmt.Printf("Start Time: %v\n", service.metrics.StartTime)
}
