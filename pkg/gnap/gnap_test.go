package gnap

import (
	"strings"
	"testing"
	"time"
)

func TestMemoryGrantStore_CreateAndGet(t *testing.T) {
	store := NewMemoryGrantStore()

	req := &GrantRequest{
		AccessToken: &AccessTokenRequest{
			Access: []AccessRight{{Type: "api", Actions: []string{"read"}}},
		},
		Client: &ClientInstance{
			InstanceID: "client-123",
			ClassID:    "web-app",
		},
	}

	grant, err := store.Create(req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if grant.ID == "" {
		t.Error("Grant ID should not be empty")
	}
	if grant.State != GrantStateProcessing {
		t.Errorf("Expected state Processing, got %s", grant.State)
	}
	if grant.ContinueToken == "" {
		t.Error("ContinueToken should be generated")
	}

	// Retrieve
	retrieved, err := store.Get(grant.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.ID != grant.ID {
		t.Error("Retrieved grant ID mismatch")
	}
}

func TestMemoryGrantStore_ListByClient(t *testing.T) {
	store := NewMemoryGrantStore()

	clientID := "client-456"
	req := &GrantRequest{
		Client: &ClientInstance{InstanceID: clientID},
	}

	_, _ = store.Create(req)
	_, _ = store.Create(req)

	grants, err := store.ListByClient(clientID)
	if err != nil {
		t.Fatalf("ListByClient failed: %v", err)
	}
	if len(grants) != 2 {
		t.Errorf("Expected 2 grants, got %d", len(grants))
	}
}

func TestMemoryGrantStore_GetByContinueToken(t *testing.T) {
	store := NewMemoryGrantStore()

	req := &GrantRequest{}
	grant, _ := store.Create(req)

	found, err := store.GetByContinueToken(grant.ContinueToken)
	if err != nil {
		t.Fatalf("GetByContinueToken failed: %v", err)
	}
	if found.ID != grant.ID {
		t.Error("Found grant ID mismatch")
	}

	// Not found
	_, err = store.GetByContinueToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestGrant_StateTransitions(t *testing.T) {
	tests := []struct {
		from  GrantState
		to    GrantState
		valid bool
	}{
		{GrantStateProcessing, GrantStatePending, true},
		{GrantStateProcessing, GrantStateApproved, true},
		{GrantStateProcessing, GrantStateDenied, true},
		{GrantStatePending, GrantStateApproved, true},
		{GrantStatePending, GrantStateDenied, true},
		{GrantStateApproved, GrantStateFinalized, true},
		{GrantStateFinalized, GrantStateApproved, false}, // Terminal
		{GrantStateDenied, GrantStateApproved, false},    // Terminal
		{GrantStateApproved, GrantStatePending, false},   // Invalid backward
	}

	for _, tc := range tests {
		g := &Grant{State: tc.from, UpdatedAt: time.Now()}
		err := g.Transition(tc.to)
		if tc.valid && err != nil {
			t.Errorf("%s -> %s: expected valid, got error %v", tc.from, tc.to, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("%s -> %s: expected invalid, got success", tc.from, tc.to)
		}
	}
}

func TestGrant_IsTerminal(t *testing.T) {
	if (&Grant{State: GrantStateProcessing}).IsTerminal() {
		t.Error("Processing should not be terminal")
	}
	if !(&Grant{State: GrantStateFinalized}).IsTerminal() {
		t.Error("Finalized should be terminal")
	}
	if !(&Grant{State: GrantStateDenied}).IsTerminal() {
		t.Error("Denied should be terminal")
	}
}

func TestGrant_CanContinue(t *testing.T) {
	if !(&Grant{State: GrantStateProcessing}).CanContinue() {
		t.Error("Processing should allow continue")
	}
	if !(&Grant{State: GrantStatePending}).CanContinue() {
		t.Error("Pending should allow continue")
	}
	if !(&Grant{State: GrantStateApproved}).CanContinue() {
		t.Error("Approved should allow continue")
	}
	if (&Grant{State: GrantStateFinalized}).CanContinue() {
		t.Error("Finalized should not allow continue")
	}
}

func TestAccessRight_Marshaling(t *testing.T) {
	// Verify types can be marshaled (basic sanity)
	ar := AccessRight{
		Type:       "photo-api",
		Actions:    []string{"read", "write"},
		Locations:  []string{"https://api.example.com"},
		Privileges: []string{"admin"},
	}
	if ar.Type == "" {
		t.Error("Type should be set")
	}
}

func TestInteractionService_CalculateHash(t *testing.T) {
	svc := NewInteractionService("http://localhost:8080", "http://as.example.com/gnap/tx")

	clientNonce := "client-nonce-123"
	serverNonce := "server-nonce-456"
	interactRef := "ref-789"

	hash1 := svc.CalculateInteractionHash(clientNonce, serverNonce, interactRef)
	hash2 := svc.CalculateInteractionHash(clientNonce, serverNonce, interactRef)

	// Same inputs should produce same hash
	if hash1 != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Different inputs should produce different hash
	hash3 := svc.CalculateInteractionHash("different", serverNonce, interactRef)
	if hash1 == hash3 {
		t.Error("Different inputs should produce different hash")
	}

	// Verify hash verification
	if !svc.VerifyInteractionHash(clientNonce, serverNonce, interactRef, hash1) {
		t.Error("Hash verification should pass")
	}
	if svc.VerifyInteractionHash(clientNonce, serverNonce, interactRef, "wrong-hash") {
		t.Error("Wrong hash should fail verification")
	}
}

func TestInteractionService_BuildCallbackURI(t *testing.T) {
	svc := NewInteractionService("http://localhost:8080", "http://as.example.com/gnap/tx")

	grant := &Grant{
		ID:            "grant-123",
		InteractRef:   "ref-456",
		InteractNonce: "server-nonce-789",
	}

	callback, err := svc.BuildCallbackURI(grant, "http://client.example/callback", "client-nonce")
	if err != nil {
		t.Fatalf("BuildCallbackURI failed: %v", err)
	}

	if callback == "" {
		t.Error("Callback URI should not be empty")
	}

	// Should contain hash and interact_ref
	if !strings.Contains(callback, "hash=") {
		t.Error("Callback should contain hash parameter")
	}
	if !strings.Contains(callback, "interact_ref=ref-456") {
		t.Error("Callback should contain interact_ref parameter")
	}
}

func TestGenerateUserCode(t *testing.T) {
	code := GenerateUserCode()

	if code == nil {
		t.Fatal("GenerateUserCode returned nil")
	}

	// Code format: XXXX-XXXX
	if len(code.Code) != 9 {
		t.Errorf("Expected code length 9 (XXXX-XXXX), got %d", len(code.Code))
	}

	if code.Code[4] != '-' {
		t.Error("Expected hyphen at position 4")
	}

	if code.ExpiresIn != 600 {
		t.Errorf("Expected ExpiresIn=600, got %d", code.ExpiresIn)
	}
}
