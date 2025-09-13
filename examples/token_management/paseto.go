package tokenmanagement

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
	"github.com/o1egl/paseto"
)

// PasetoManager handles PASETO token operations
type PasetoManager struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	footer     string
}

// NewPasetoManager creates a new PASETO token manager
func NewPasetoManager() (*PasetoManager, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &PasetoManager{
		publicKey:  public,
		privateKey: private,
		footer:     "gauth-paseto-v2",
	}, nil
}

// PasetoClaims represents PASETO token claims
type PasetoClaims struct {
	TokenID   string            `json:"jti"`
	Subject   string            `json:"sub"`
	Issuer    string            `json:"iss"`
	IssuedAt  time.Time         `json:"iat"`
	ExpiresAt time.Time         `json:"exp"`
	NotBefore time.Time         `json:"nbf"`
	Type      token.Type        `json:"typ"`
	Scopes    []string          `json:"scp"`
	Metadata  map[string]string `json:"meta,omitempty"`
}

// SignToken creates a PASETO token
func (m *PasetoManager) SignToken(ctx context.Context, t *token.Token) (string, error) {
	// Create PASETO v2 instance
	v2 := paseto.NewV2()

	// Prepare claims
	claims := PasetoClaims{
		TokenID:   t.ID,
		Subject:   t.Subject,
		Issuer:    t.Issuer,
		IssuedAt:  t.IssuedAt,
		ExpiresAt: t.ExpiresAt,
		NotBefore: time.Now(),
		Type:      t.Type,
		Scopes:    t.Scopes,
		Metadata:  t.Metadata,
	}

	// Sign token
	signed, err := v2.Sign(m.privateKey, claims, m.footer)
	if err != nil {
		return "", fmt.Errorf("failed to sign PASETO token: %w", err)
	}

	return signed, nil
}

// VerifyToken validates a PASETO token and extracts the claims
func (m *PasetoManager) VerifyToken(ctx context.Context, pasetoToken string) (*token.Token, error) {
	var claims PasetoClaims
	v2 := paseto.NewV2()

	// Verify and parse token
	err := v2.Verify(pasetoToken, m.publicKey, &claims, m.footer)
	if err != nil {
		return nil, fmt.Errorf("failed to verify PASETO token: %w", err)
	}

	// Convert to Token
	return &token.Token{
		ID:        claims.TokenID,
		Subject:   claims.Subject,
		Issuer:    claims.Issuer,
		IssuedAt:  claims.IssuedAt,
		ExpiresAt: claims.ExpiresAt,
		Type:      claims.Type,
		Scopes:    claims.Scopes,
		Metadata:  claims.Metadata,
	}, nil
}

func main() {
	ctx := context.Background()

	// Create PASETO manager
	pasetoMgr, err := NewPasetoManager()
	if err != nil {
		log.Fatalf("Failed to create PASETO manager: %v", err)
	}

	// Create a token store
	store := token.NewMemoryStore(24 * time.Hour)
	blacklist := token.NewBlacklist()
	validator := token.NewValidationChain(blacklist)

	// Create a test token
	t := &token.Token{
		ID:        token.NewID(),
		Type:      token.Access,
		Subject:   "user123",
		Issuer:    "paseto-example",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"read", "write"},
		Metadata: map[string]string{
			"device": "mobile",
			"os":     "ios",
		},
	}

	// Store token
	if err := store.Save(ctx, t); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}

	// Sign token with PASETO
	signed, err := pasetoMgr.SignToken(ctx, t)
	if err != nil {
		log.Fatalf("Failed to sign PASETO token: %v", err)
	}
	fmt.Printf("Signed PASETO token:\n%s\n\n", signed)

	// Verify PASETO token
	verified, err := pasetoMgr.VerifyToken(ctx, signed)
	if err != nil {
		log.Fatalf("Failed to verify PASETO token: %v", err)
	}

	// Validate token
	if err := validator.Validate(ctx, verified); err != nil {
		log.Fatalf("Token validation failed: %v", err)
	}

	fmt.Printf("Verified token details:\n")
	fmt.Printf("ID: %s\n", verified.ID)
	fmt.Printf("Subject: %s\n", verified.Subject)
	fmt.Printf("Type: %s\n", verified.Type)
	fmt.Printf("Scopes: %v\n", verified.Scopes)
	fmt.Printf("Metadata: %v\n", verified.Metadata)

	// Compare security properties
	fmt.Printf("\nSecurity Properties:\n")
	fmt.Printf("- PASETO provides stronger security guarantees than JWT\n")
	fmt.Printf("- Uses modern crypto (Ed25519 for signatures)\n")
	fmt.Printf("- No algorithm confusion attacks\n")
	fmt.Printf("- Simpler to use correctly\n")
}
