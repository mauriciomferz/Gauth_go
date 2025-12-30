package main

import (
	"context"
	"fmt"
	"log"
	"time"

	token "github.com/mauriciomferz/AgentAuth/pkg/token"
)

type CustomValidator struct {
	allowedIssuers map[string]bool
}

func NewCustomValidator(issuers []string) *CustomValidator {
	allowed := make(map[string]bool)
	for _, issuer := range issuers {
		allowed[issuer] = true
	}
	return &CustomValidator{allowedIssuers: allowed}
}

func (v *CustomValidator) Validate(ctx context.Context, t *token.Token) error {
	if !v.allowedIssuers[t.Issuer] {
		return fmt.Errorf("issuer %s not allowed", t.Issuer)
	}
	return nil
}

func main() {
	ctx := context.Background()
	store := token.NewMemoryStore(24 * time.Hour)
	// Removed: blacklist := token.NewBlacklist()
	// Removed: customValidator := NewCustomValidator([]string{"example-app", "partner-app"})
	_ = token.NewValidationChain(token.ValidationConfig{
		CheckExpiry:    true,
		CheckBlacklist: true,
		CheckSignature: false,
		MaxAge:         2 * time.Minute,
	})

	tokens := []*token.Token{
		{
			ID:        token.GenerateID(),
			Type:      token.Access,
			Subject:   "user1",
			Issuer:    "example-app",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Scopes:    []string{"read"},
		},
		{
			ID:        token.GenerateID(),
			Type:      token.Access,
			Subject:   "user1",
			Issuer:    "partner-app",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(2 * time.Hour),
			Scopes:    []string{"write"},
		},
		{
			ID:        token.GenerateID(),
			Type:      token.Refresh,
			Subject:   "user2",
			Issuer:    "example-app",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Scopes:    []string{"refresh"},
		},
	}

	for _, t := range tokens {
		if err := store.Save(ctx, t.ID, t); err != nil {
			log.Fatalf("Failed to save token: %v", err)
		}
	}
}
