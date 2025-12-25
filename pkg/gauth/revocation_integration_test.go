package gauth

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
)

// Mock objects for testing (lightweight versions) Since actual mocks are in a different package (gauth/mocks)
// we will just implement minimal interfaces inline or usage logic if possible, or skip strict mode dependencies.
// To make it easier, we will construct a validator with minimal dependencies.

func TestRevocationIntegration_DelegationRevoked(t *testing.T) {
	// 1. Setup Revocation Chain
	revChain := delegation.NewRevocationChain()
	delegationID := "del-123"

	// Revoke the delegation
	_, err := revChain.Append(delegation.RevocationEvent{
		ID:           "rev-1",
		DelegationID: delegationID,
		Reason:       "compromise",
	})
	if err != nil {
		t.Fatalf("failed to append revocation: %v", err)
	}

	// 2. Setup Validator with Adapter
	adapter := NewDelegationRevocationAdapter(revChain)

	validator := NewAuthorizationChainValidator(nil, nil, adapter)
	validator.strictMode = false // disable strict checks for identity/register to focus on revocation

	// 3. Construct Chain with Revoked Delegation
	now := time.Now()
	chain := &AuthorizationChain{
		ChainDepth: 3,
		OwnersAuthorizer: &AuthorizationLink{
			EntityID:          "auth-1",
			Role:              "authorizer",
			Status:            "active",
			IdentityVerified:  true,
			ValidFrom:         now.Add(-1 * time.Hour),
			ValidUntil:        now.Add(1 * time.Hour),
			AuthorizationType: "delegated",
			DelegationID:      delegationID, // THE LINKED DELEGATION
		},
		ClientOwner: &AuthorizationLink{
			EntityID:          "owner-1",
			Role:              "owner",
			Status:            "active",
			IdentityVerified:  true,
			ValidFrom:         now.Add(-1 * time.Hour),
			ValidUntil:        now.Add(1 * time.Hour),
			AuthorizedBy:      "auth-1",
			AuthorizationType: "statutory",
		},
		Client: &AuthorizationLink{
			EntityID:         "client-1",
			Role:             "client",
			Status:           "active",
			EntityType:       "ai_system",
			IdentityVerified: true,
			ValidFrom:        now.Add(-1 * time.Hour),
			ValidUntil:       now.Add(1 * time.Hour),
			AuthorizedBy:     "owner-1",
		},
	}

	// 4. Validate
	ctx := context.Background()
	result, err := validator.ValidateAuthorizationChain(ctx, chain)

	// 5. Assertions
	if err != nil {
		// It might fail with error or just mark valid=false.
		// Our validator returns result AND error sometimes.
	}

	if result.Valid {
		t.Error("expected chain to be INVALID due to revocation")
	}

	if !result.RevocationStatus.Revoked {
		t.Error("expected revocation status to be true")
	}

	if result.RevocationStatus.RevokedEntity != "owners_authorizer_delegation" {
		t.Errorf("expected revoked entity 'owners_authorizer_delegation', got '%s'", result.RevocationStatus.RevokedEntity)
	}

	if !result.RevocationStatus.LinkRevocations["owners_authorizer_delegation"] {
		t.Error("expected specific link revocation flag set")
	}
}

func TestRevocationIntegration_NotRevoked(t *testing.T) {
	// 1. Setup Empty Revocation Chain
	revChain := delegation.NewRevocationChain()

	// 2. Setup Validator with Adapter
	adapter := NewDelegationRevocationAdapter(revChain)
	validator := NewAuthorizationChainValidator(nil, nil, adapter)
	validator.strictMode = false

	// 3. Construct Valid Chain
	now := time.Now()
	chain := &AuthorizationChain{
		ChainDepth: 3,
		OwnersAuthorizer: &AuthorizationLink{
			EntityID:          "auth-1",
			Role:              "authorizer",
			Status:            "active",
			IdentityVerified:  true,
			ValidFrom:         now.Add(-1 * time.Hour),
			ValidUntil:        now.Add(1 * time.Hour),
			AuthorizationType: "delegated",
			DelegationID:      "del-999", // Not revoked
		},
		ClientOwner: &AuthorizationLink{
			EntityID:          "owner-1",
			Role:              "owner",
			Status:            "active",
			IdentityVerified:  true,
			ValidFrom:         now.Add(-1 * time.Hour),
			ValidUntil:        now.Add(1 * time.Hour),
			AuthorizedBy:      "auth-1",
			AuthorizationType: "statutory",
		},
		Client: &AuthorizationLink{
			EntityID:         "client-1",
			Role:             "client",
			Status:           "active",
			EntityType:       "ai_system",
			IdentityVerified: true,
			ValidFrom:        now.Add(-1 * time.Hour),
			ValidUntil:       now.Add(1 * time.Hour),
			AuthorizedBy:     "owner-1",
		},
	}

	// 4. Validate
	ctx := context.Background()
	result, err := validator.ValidateAuthorizationChain(ctx, chain)

	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid chain, failed: %s", result.FailureReason)
	}

	if result.RevocationStatus.Revoked {
		t.Error("expected revocation status false")
	}
}
