package token

import (
	"testing"
	"time"
)

func TestStore_CreateValidate(t *testing.T) {
	ts := NewStore(10)
	tok := ts.Create(60, map[string]string{"foo": "bar"})
	if tok.Status != "active" {
		t.Fatalf("expected active, got %s", tok.Status)
	}

	status, fetched := ts.Validate(tok.ID)
	if status != TokenStatusValid {
		t.Errorf("validate by ID: expected %s, got %s", TokenStatusValid, status)
	}
	if fetched.ID != tok.ID {
		t.Errorf("fetched ID mismatch")
	}

	status2, _ := ts.Validate(tok.Value)
	if status2 != TokenStatusValid {
		t.Errorf("validate by Value: expected %s, got %s", TokenStatusValid, status2)
	}
}

func TestStore_Revoke(t *testing.T) {
	ts := NewStore(10)
	tok := ts.Create(60, nil)

	if status := ts.Revoke(tok.ID); status != TokenStatusRevoked {
		t.Errorf("expected revoked, got %s", status)
	}

	status, _ := ts.Validate(tok.ID)
	if status != TokenStatusRevoked {
		t.Errorf("validate revoked: expected %s, got %s", TokenStatusRevoked, status)
	}

	// Double revoke
	if status := ts.Revoke(tok.ID); status != TokenStatusAlreadyRevoked {
		t.Errorf("expected already_revoked, got %s", status)
	}
}

func TestStore_Expiry(t *testing.T) {
	ts := NewStore(10)
	// 1 second expiry (min effective might be higher in some implementations, but here direct time check)
	tok := ts.Create(1, nil)
	// Artificial time travel (Store uses time.Now(), so we can't easily mock time without injection.
	// We'll sleep 1.1s)
	time.Sleep(1100 * time.Millisecond)

	status, _ := ts.Validate(tok.ID)
	if status != "expired" {
		t.Errorf("expected expired, got %s", status)
	}
}
