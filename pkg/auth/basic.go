package auth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
)

var (
	// ErrInvalidCredentials indicates the provided credentials are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")
)

const (
	basicAuthScheme = "Basic"
)

// basicCredentials represents username/password credentials
type basicCredentials struct {
	Username string
	Password string
}

// basicAuthenticator implements the Authenticator interface using basic auth
type basicAuthenticator struct {
	config  Config
	clients sync.Map // map[string]string - username to password hash
}

// newBasicAuthenticator creates a new basic authenticator
func newBasicAuthenticator(config Config) (Authenticator, error) {
	return &basicAuthenticator{
		config: config,
	}, nil
}

func (a *basicAuthenticator) Initialize(ctx context.Context) error {
	// No initialization needed for now
	return nil
}

func (a *basicAuthenticator) Close() error {
	// No cleanup needed for now
	return nil
}

func (a *basicAuthenticator) ValidateCredentials(ctx context.Context, creds interface{}) error {
	var username, password string

	switch c := creds.(type) {
	case basicCredentials:
		username = c.Username
		password = c.Password
	case struct{ Username, Password string }:
		username = c.Username
		password = c.Password
	case map[string]string:
		var ok bool
		username, ok = c["username"]
		if !ok {
			return fmt.Errorf("missing username in credentials")
		}
		password, ok = c["password"]
		if !ok {
			return fmt.Errorf("missing password in credentials")
		}
	default:
		return fmt.Errorf("expected basicCredentials or struct with Username/Password, got %T", creds)
	}

	// In a real implementation, this would validate against a user store
	storedPassword, exists := a.clients.Load(username)
	if !exists {
		return ErrInvalidCredentials
	}

	if subtle.ConstantTimeCompare([]byte(password), []byte(storedPassword.(string))) != 1 {
		return ErrInvalidCredentials
	}

	if a.config.AuditLogger != nil {
		a.config.AuditLogger.Log(ctx, &audit.Entry{
			Type:    audit.TypeAuth,
			Action:  audit.ActionLogin,
			ActorID: username,
			Result:  audit.ResultSuccess,
		})
	}

	return nil
}

func (a *basicAuthenticator) GenerateToken(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	// For basic auth, we generate a simple base64 token
	// In a real implementation, you'd want to use JWT or another token format
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%d", req.Subject, time.Now().Unix())))

	if a.config.AuditLogger != nil {
		a.config.AuditLogger.Log(ctx, &audit.Entry{
			Type:    audit.TypeToken,
			Action:  "generate",
			ActorID: req.Subject,
			Result:  audit.ResultSuccess,
			Metadata: map[string]string{
				"scope": fmt.Sprintf("%v", req.Scopes),
			},
		})
	}

	return &TokenResponse{
		Token:     token,
		TokenType: basicAuthScheme,
		ExpiresIn: int64(a.config.AccessTokenExpiry.Seconds()),
		Scope:     req.Scopes,
		Claims:    nil, // No claims for basic auth
	}, nil
}

func (a *basicAuthenticator) ValidateToken(ctx context.Context, tokenStr string) (*TokenData, error) {
	parts := strings.SplitN(tokenStr, " ", 2)
	if len(parts) != 2 || parts[0] != basicAuthScheme {
		return nil, ErrInvalidToken
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return nil, ErrInvalidToken
	}

	// Extract username and timestamp
	username := credentials[0]
	timestamp, err := parseTimestamp(credentials[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Check if token has expired
	if time.Since(timestamp) > a.config.AccessTokenExpiry {
		return nil, ErrTokenExpired
	}

	if a.config.AuditLogger != nil {
		a.config.AuditLogger.Log(ctx, &audit.Entry{
			Type:    audit.TypeToken,
			Action:  "validate", // No constant defined, use string
			ActorID: username,
			Result:  audit.ResultSuccess,
		})
	}

	return &TokenData{
		Valid:     true,
		Subject:   username,
		IssuedAt:  timestamp,
		ExpiresAt: timestamp.Add(a.config.AccessTokenExpiry),
	}, nil
}

func (a *basicAuthenticator) RevokeToken(ctx context.Context, tokenStr string) error {
	// Basic auth doesn't support token revocation
	return errors.New("token revocation not supported by basic authenticator")
}

// Helper functions
func parseTimestamp(s string) (time.Time, error) {
	unix, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(unix, 0), nil
}

// AddClient adds a new client to the authenticator
func (a *basicAuthenticator) AddClient(username, password string) {
	a.clients.Store(username, password)
}
