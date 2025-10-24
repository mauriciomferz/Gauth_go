package ledger

import (
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDefaultRevocationAnchorService_AnchorChainTip(t *testing.T) {
	service := &DefaultRevocationAnchorService{}
	hash := []byte("dummyhash")
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}

	// Mock external endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()
	os.Setenv("GAUTH_ANCHOR_ENDPOINT", ts.URL)

	if err := service.AnchorChainTip(hash, priv, pub); err != nil {
		t.Fatalf("expected nil error anchoring chain tip, got %v", err)
	}
}
