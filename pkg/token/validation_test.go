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
	"errors"
	"testing"
	"time"
)

type mockValidator struct {
	shouldError bool
}

func (m *mockValidator) Validate(_ context.Context, _ *Token) error {
	if m.shouldError {
		return errors.New("mock validation error")
	}
	return nil
}

func TestValidationChain(t *testing.T) {
	ctx := context.Background()
	bl := NewBlacklist()
	defer func() {
		if err := bl.Close(); err != nil {
			t.Errorf("bl.Close() error: %v", err)
		}
	}()
	config := ValidationConfig{}

	t.Run("Valid Token", func(t *testing.T) {
		validator1 := &mockValidator{shouldError: false}
		validator2 := &mockValidator{shouldError: false}
		chain := NewValidationChain(config, bl, validator1, validator2)

		token := &Token{
			ID:        NewID(),
			IssuedAt:  time.Now(),
			NotBefore: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}

		err := chain.Validate(ctx, token)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestDefaultQuerier(t *testing.T) {
	querier := NewDefaultQuerier(nil)
	ctx := context.Background()

	t.Run("Query Interface", func(t *testing.T) {
		query := Query{
			Subject: "user123",
			Type:    Access,
			Scopes:  []string{"read"},
			ValidAt: time.Now(),
		}
		// Query should return "not implemented" for default querier
		_, err := querier.Query(ctx, query, 0, 10)
		if err == nil {
			t.Error("Expected not implemented error")
		}
	})

	t.Run("Count By Subject", func(t *testing.T) {
		// Should return "not implemented" for default querier
		_, err := querier.CountBySubject(ctx, "user123")
		if err == nil {
			t.Error("Expected not implemented error")
		}
	})

	t.Run("List Expiring Soon", func(t *testing.T) {
		// Should return "not implemented" for default querier
		_, err := querier.ListExpiringSoon(ctx, time.Hour)
		if err == nil {
			t.Error("Expected not implemented error")
		}
	})
}
