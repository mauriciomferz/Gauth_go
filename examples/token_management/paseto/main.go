// Moved from paseto.go
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"time"

	token "github.com/Gimel-Foundation/gauth/pkg/token"
	paseto "github.com/o1egl/paseto"
)

type PasetoManager struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	footer     string
}

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

func (m *PasetoManager) SignToken(ctx context.Context, t *token.Token) (string, error) {
	v2 := paseto.NewV2()
	claims := PasetoClaims{
		TokenID:   t.ID,
		Subject:   t.Subject,
		Issuer:    t.Issuer,
		IssuedAt:  t.IssuedAt,
		ExpiresAt: t.ExpiresAt,
		NotBefore: time.Now(),
		Type:      t.Type,
		Scopes:    t.Scopes,
		Metadata:  nil, // simplified for migration
	}

	signed, err := v2.Sign(m.privateKey, claims, m.footer)
	if err != nil {
		return "", fmt.Errorf("failed to sign PASETO token: %w", err)
	}

	return signed, nil
}

func (m *PasetoManager) VerifyToken(ctx context.Context, pasetoToken string) (*token.Token, error) {
	var claims PasetoClaims
	v2 := paseto.NewV2()
	err := v2.Verify(pasetoToken, m.publicKey, &claims, m.footer)
	if err != nil {
		return nil, fmt.Errorf("failed to verify PASETO token: %w", err)
	}

	return &token.Token{
		ID:        claims.TokenID,
		Subject:   claims.Subject,
		Issuer:    claims.Issuer,
		IssuedAt:  claims.IssuedAt,
		ExpiresAt: claims.ExpiresAt,
		Type:      claims.Type,
		Scopes:    claims.Scopes,
		Metadata:  nil, // simplified for migration
	}, nil
}

func main() {
	fmt.Println("PASETO token management example loaded.")
}
