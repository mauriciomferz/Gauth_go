package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func TestSaveValidateRevokeToken(t *testing.T) {
	ctx := context.Background()
	store := token.NewMemoryStore(24 * time.Hour)

	// Create a token
	tok := &token.Token{
		ID:        "token123",
		Value:     "secret-token-value",
		Type:      token.Access,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Subject:   "user123",
		Scopes:    []string{"read", "write"},
	}

	// Save the token
	err := store.Save(ctx, tok.ID, tok)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Validate the token
	err = store.Validate(ctx, tok)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Revoke the token
	err = store.Revoke(ctx, tok)
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	// Validate again (should fail)
	err = store.Validate(ctx, tok)
	if err == nil {
		t.Error("Token should not be valid after revocation")
	}
}
