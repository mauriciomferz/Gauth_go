//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package token

import (
	"context"
	"testing"
	"time"
)

func TestBlacklist(t *testing.T) {
	bl := NewBlacklist()
	ctx := context.Background()
	defer bl.Close()

	t.Run("Add and Check Token", func(t *testing.T) {
		token := &Token{
			ID:        NewID(),
			ExpiresAt: time.Now().Add(time.Hour),
		}

		// Add to blacklist
		err := bl.Add(ctx, token, "test revocation")
		if err != nil {
			t.Fatalf("Failed to add token to blacklist: %v", err)
		}

		// Check if blacklisted
		if !bl.IsBlacklisted(ctx, token.ID) {
			t.Error("Token should be blacklisted")
		}

		// Get blacklist details
		blToken, exists := bl.GetBlacklistedToken(ctx, token.ID)
		if !exists {
			t.Error("Blacklisted token details not found")
		}
		if blToken.Reason != "test revocation" {
			t.Errorf("Wrong revocation reason: got %v, want test revocation",
				blToken.Reason)
		}
	})

	t.Run("Non-existent Token", func(t *testing.T) {
		if bl.IsBlacklisted(ctx, "nonexistent") {
			t.Error("Non-existent token should not be blacklisted")
		}

		_, exists := bl.GetBlacklistedToken(ctx, "nonexistent")
		if exists {
			t.Error("Should not find non-existent blacklisted token")
		}
	})

	t.Run("Cleanup Expired", func(t *testing.T) {
		// Add expired token
		expiredToken := &Token{
			ID:        NewID(),
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		if err := bl.Add(ctx, expiredToken, "expired"); err != nil {
			t.Fatalf("Failed to add expired token: %v", err)
		}

		// Force cleanup
		bl.cleanup()

		// Should be removed
		if bl.IsBlacklisted(ctx, expiredToken.ID) {
			t.Error("Expired token should be removed after cleanup")
		}
	})
}

func TestRotator(t *testing.T) {
	store := NewMemoryStore(time.Hour)
	bl := NewBlacklist()
	config := Config{
		ValidityPeriod: time.Hour,
	}
	rotator := NewRotator(store, bl, config)
	ctx := context.Background()

	t.Run("Rotate Token", func(t *testing.T) {
		// Create original token
		original := &Token{
			ID:        NewID(),
			Type:      Access,
			Subject:   "user123",
			Issuer:    "test",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Scopes:    []string{"read", "write"},
			Metadata:  &Metadata{AppData: map[string]string{"device": "mobile"}},
		}

		// Store original
		if err := store.Save(ctx, original.ID, original); err != nil {
			t.Fatalf("Failed to save original token: %v", err)
		}

		// Rotate token
		rotated, err := rotator.RotateToken(ctx, original)
		if err != nil {
			t.Fatalf("Failed to rotate token: %v", err)
		}

		// Check new token
		if rotated.ID == original.ID {
			t.Error("Rotated token should have new ID")
		}
		if rotated.Subject != original.Subject {
			t.Error("Subject should be preserved")
		}
		if rotated.Type != original.Type {
			t.Error("Type should be preserved")
		}
		if len(rotated.Scopes) != len(original.Scopes) {
			t.Error("Scopes should be preserved")
		}
		if rotated.Metadata == nil || rotated.Metadata.AppData["device"] != original.Metadata.AppData["device"] {
			t.Error("Metadata should be preserved")
		}

		// Original should be blacklisted
		if !bl.IsBlacklisted(ctx, original.ID) {
			t.Error("Original token should be blacklisted")
		}

		// New token should be stored
		stored, err := store.Get(ctx, rotated.ID)
		if err != nil {
			t.Fatalf("Failed to get rotated token: %v", err)
		}
		if stored.ID != rotated.ID {
			t.Error("Rotated token not properly stored")
		}
	})
}
