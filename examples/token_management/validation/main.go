package main

import (
	"context"
	"fmt"
	"log"
	"time"

	token "github.com/Gimel-Foundation/gauth/pkg/token"
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
	blacklist := token.NewBlacklist()
	defer blacklist.Close()

	customValidator := NewCustomValidator([]string{"example-app", "partner-app"})
	_ = token.NewValidationChain(token.ValidationConfig{
		AllowedIssuers: []string{"example-app", "partner-app"},
		ClockSkew:      2 * time.Minute,
	}, blacklist, customValidator)

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

	querier := token.NewDefaultQuerier(store)
	count, err := querier.CountBySubject(ctx, "user1")
	if err == nil {
		fmt.Printf("User1 has %d tokens\n", count)
	}

	expiring, err := querier.ListExpiringSoon(ctx, time.Hour)
	if err == nil {
		fmt.Printf("Tokens expiring within 1 hour: %d\n", len(expiring))
	}
}
