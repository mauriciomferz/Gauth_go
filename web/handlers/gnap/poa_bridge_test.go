package gnap

import (
	"testing"

	gnappkg "github.com/mauriciomferz/Gauth_go/pkg/gnap"
)

func TestPoABridge_LinkGrantToPoA(t *testing.T) {
	store := gnappkg.NewMemoryGrantStore()
	bridge := NewPoABridge(store)

	grant, err := store.Create(&gnappkg.GrantRequest{
		AccessToken: &gnappkg.AccessTokenRequest{},
	})
	if err != nil {
		t.Fatalf("failed to create grant: %v", err)
	}

	result, err := bridge.LinkGrantToPoA(grant.ID, "poa-12345", "chain-link-abc")
	if err != nil {
		t.Fatalf("failed to link: %v", err)
	}

	if result.PoAID != "poa-12345" {
		t.Errorf("expected poa-12345, got %s", result.PoAID)
	}
}

func TestPoABridge_GetPoAForGrant(t *testing.T) {
	store := gnappkg.NewMemoryGrantStore()
	bridge := NewPoABridge(store)

	grant, _ := store.Create(&gnappkg.GrantRequest{
		AccessToken: &gnappkg.AccessTokenRequest{},
	})

	_, _ = bridge.LinkGrantToPoA(grant.ID, "poa-xyz", "chain-987")

	poaID, err := bridge.GetPoAForGrant(grant.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if poaID != "poa-xyz" {
		t.Errorf("expected poa-xyz, got %s", poaID)
	}
}

func TestPoABridge_GetPoAForGrant_NotFound(t *testing.T) {
	store := gnappkg.NewMemoryGrantStore()
	bridge := NewPoABridge(store)

	_, err := bridge.GetPoAForGrant("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent grant")
	}
}

func TestPoABridge_ValidatePoAAuthority(t *testing.T) {
	store := gnappkg.NewMemoryGrantStore()
	bridge := NewPoABridge(store)

	grant, _ := store.Create(&gnappkg.GrantRequest{
		AccessToken: &gnappkg.AccessTokenRequest{},
	})

	valid, reason, err := bridge.ValidatePoAAuthority(grant.ID)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if !valid {
		t.Error("expected valid without PoA")
	}
	if reason != "no_poa_required" {
		t.Errorf("expected no_poa_required, got %s", reason)
	}

	_, _ = bridge.LinkGrantToPoA(grant.ID, "poa-valid", "chain-valid")
	valid, reason, err = bridge.ValidatePoAAuthority(grant.ID)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if !valid {
		t.Error("expected valid with PoA")
	}
	if reason != "poa_valid" {
		t.Errorf("expected poa_valid, got %s", reason)
	}
}
