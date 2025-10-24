package ledger

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// RevocationAnchorService anchors the revocation chain tip in an external timestamp service.
type RevocationAnchorService interface {
	AnchorChainTip(chainTipHash []byte, activeKey ed25519.PrivateKey, publicKey ed25519.PublicKey) error
}

// DefaultRevocationAnchorService is a stub implementation.
type DefaultRevocationAnchorService struct{}

// Simulated external anchoring: POST chain tip hash to a TSA/transparency log endpoint
func (s *DefaultRevocationAnchorService) AnchorChainTip(chainTipHash []byte, activeKey ed25519.PrivateKey, publicKey ed25519.PublicKey) error {
	// Endpoint can be set via env or config
	endpoint := "https://tsa.example.com/anchor"
	if ep := os.Getenv("GAUTH_ANCHOR_ENDPOINT"); ep != "" {
		endpoint = ep
	}

	anchoredAt := time.Now().UTC().Format(time.RFC3339)
	payloadObj := map[string]interface{}{
		"chain_tip_hash": base64.RawURLEncoding.EncodeToString(chainTipHash),
		"anchored_at": anchoredAt,
	}

	// Canonical JSON for signing
	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return fmt.Errorf("payload marshal failed: %w", err)
	}


	sig := ed25519.Sign(activeKey, payloadBytes)

	anchorReq := map[string]interface{}{
		"chain_tip_hash": payloadObj["chain_tip_hash"],
		"anchored_at": anchoredAt,
		"signature": base64.StdEncoding.EncodeToString(sig),
		"public_key": base64.StdEncoding.EncodeToString(publicKey),
	}

	anchorBytes, err := json.Marshal(anchorReq)
	if err != nil {
		return fmt.Errorf("anchor payload marshal failed: %w", err)
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(anchorBytes))
	if err != nil {
		return fmt.Errorf("external anchor POST failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("external anchor returned status %d", resp.StatusCode)
	}
	return nil
}
