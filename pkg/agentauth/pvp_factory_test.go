package agentauth

import (
	"os"
	"testing"
)

func TestPVPFactory_CreatePVP(t *testing.T) {
	t.Run("Stripe Provider", func(t *testing.T) {
		t.Setenv("AGENTAUTH_PVP_PROVIDER", "stripe")
		t.Setenv("STRIPE_API_KEY", "sk_test_123")

		f := NewPVPFactory(true)
		pvp, err := f.CreatePVP()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := pvp.(*StripePVPClient); !ok {
			t.Errorf("expected *StripePVPClient, got %T", pvp)
		}
	})

	t.Run("Veriff Provider", func(t *testing.T) {
		t.Setenv("AGENTAUTH_PVP_PROVIDER", "veriff")
		t.Setenv("VERIFF_API_KEY", "test_key")
		t.Setenv("VERIFF_API_SECRET", "test_secret")

		f := NewPVPFactory(true)
		pvp, err := f.CreatePVP()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := pvp.(*VeriffPVPClient); !ok {
			t.Errorf("expected *VeriffPVPClient, got %T", pvp)
		}
	})

	t.Run("Idemia Provider (Generic)", func(t *testing.T) {
		t.Setenv("AGENTAUTH_PVP_PROVIDER", "idemia")
		t.Setenv("IDEMIA_API_KEY", "test_key")

		f := NewPVPFactory(true)
		pvp, err := f.CreatePVP()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stub, ok := pvp.(*GenericPVPStub)
		if !ok {
			t.Errorf("expected *GenericPVPStub, got %T", pvp)
		} else if stub.providerName != "Idemia" {
			t.Errorf("expected provider name Idemia, got %s", stub.providerName)
		}
	})

	t.Run("Missing Config", func(t *testing.T) {
		t.Setenv("AGENTAUTH_PVP_PROVIDER", "stripe")
		_ = os.Unsetenv("STRIPE_API_KEY")

		f := NewPVPFactory(true)
		_, err := f.CreatePVP()
		if err == nil {
			t.Error("expected error for missing config, got nil")
		}
	})
}
