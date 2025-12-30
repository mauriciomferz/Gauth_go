package agentauth

import (
	"os"
	"testing"
)

func TestPVPFactory_CreatePVP(t *testing.T) {
	// Save original env vars
	origProvider := os.Getenv("AGENTAUTH_PVP_PROVIDER")
	origStripeKey := os.Getenv("STRIPE_API_KEY")
	origVeriffKey := os.Getenv("VERIFF_API_KEY")
	origVeriffSecret := os.Getenv("VERIFF_API_SECRET")
	origIdemiaKey := os.Getenv("IDEMIA_API_KEY")

	defer func() {
		os.Setenv("AGENTAUTH_PVP_PROVIDER", origProvider)
		os.Setenv("STRIPE_API_KEY", origStripeKey)
		os.Setenv("VERIFF_API_KEY", origVeriffKey)
		os.Setenv("VERIFF_API_SECRET", origVeriffSecret)
		os.Setenv("IDEMIA_API_KEY", origIdemiaKey)
	}()

	t.Run("Stripe Provider", func(t *testing.T) {
		os.Setenv("AGENTAUTH_PVP_PROVIDER", "stripe")
		os.Setenv("STRIPE_API_KEY", "sk_test_123")

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
		os.Setenv("AGENTAUTH_PVP_PROVIDER", "veriff")
		os.Setenv("VERIFF_API_KEY", "test_key")
		os.Setenv("VERIFF_API_SECRET", "test_secret")

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
		os.Setenv("AGENTAUTH_PVP_PROVIDER", "idemia")
		os.Setenv("IDEMIA_API_KEY", "test_key")

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
		os.Setenv("AGENTAUTH_PVP_PROVIDER", "stripe")
		os.Unsetenv("STRIPE_API_KEY")

		f := NewPVPFactory(true)
		_, err := f.CreatePVP()
		if err == nil {
			t.Error("expected error for missing config, got nil")
		}
	})
}
