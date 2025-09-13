package tokenmanagement

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// CustomValidator demonstrates how to implement custom token validation
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

	// Create token store
	store := token.NewMemoryStore(24 * time.Hour)
	blacklist := token.NewBlacklist()
	defer blacklist.Close()

	// Create custom validator
	customValidator := NewCustomValidator([]string{
		"example-app",
		"partner-app",
	})

	// Create validation chain with custom validator
	validator := token.NewValidationChain(blacklist, customValidator)

	// Create some test tokens
	tokens := []*token.Token{
		{
			ID:        token.NewID(),
			Type:      token.Access,
			Subject:   "user1",
			Issuer:    "example-app",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Scopes:    []string{"read"},
		},
		{
			ID:        token.NewID(),
			Type:      token.Access,
			Subject:   "user1",
			Issuer:    "partner-app",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(2 * time.Hour),
			Scopes:    []string{"write"},
		},
		{
			ID:        token.NewID(),
			Type:      token.Refresh,
			Subject:   "user2",
			Issuer:    "example-app",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Scopes:    []string{"refresh"},
		},
	}

	// Save tokens
	for _, t := range tokens {
		if err := store.Save(ctx, t); err != nil {
			log.Fatalf("Failed to save token: %v", err)
		}
	}

	// Create a querier
	querier := token.NewDefaultQuerier(store)

	// Query tokens by subject
	count, err := querier.CountBySubject(ctx, "user1")
	if err != nil {
		log.Printf("Count query not implemented: %v", err)
	} else {
		fmt.Printf("User1 has %d tokens\n", count)
	}

	// Query expiring tokens
	expiring, err := querier.ListExpiringSoon(ctx, time.Hour)
	if err != nil {
		log.Printf("Expiring query not implemented: %v", err)
	} else {
		fmt.Printf("Found %d tokens expiring within 1 hour\n", len(expiring))
	}

	// Demonstrate validation
	fmt.Println("\nValidating tokens:")
	for _, t := range tokens {
		err := validator.Validate(ctx, t)
		if err != nil {
			fmt.Printf("Token %s validation failed: %v\n", t.ID, err)
		} else {
			fmt.Printf("Token %s is valid (issuer: %s)\n", t.ID, t.Issuer)
		}
	}

	// Try invalid issuer
	invalidToken := &token.Token{
		ID:        token.NewID(),
		Issuer:    "unauthorized-app",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err = validator.Validate(ctx, invalidToken)
	if err != nil {
		fmt.Printf("\nExpected validation failure for unauthorized issuer: %v\n", err)
	}
}
