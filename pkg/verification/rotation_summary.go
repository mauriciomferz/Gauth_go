package verification

// Client-side helper for fetching and verifying the signed rotation summary
// exposed by /api/v1/beta/rotations/summary. This parallels the revocation
// transparency helpers in this package, providing a simple one-call
// verification workflow plus granular building blocks.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// RotationSummaryAPI mirrors the public JSON fields returned by the endpoint.
type RotationSummaryAPI struct {
	Success    bool             `json:"success"`
	Configured bool             `json:"configured"`
	Summary    *RotationSummary `json:"summary"`
	Anchored   bool             `json:"anchored"`
	AnchorHash string           `json:"anchor_hash"`
	AnchorAt   string           `json:"anchor_at"`
	Error      string           `json:"error,omitempty"`
}

// RotationSummary is the signed artifact subset.
type RotationSummary struct {
	ChainLength     int    `json:"chain_length"`
	HeadHash        string `json:"head_hash"`
	AggregateHash   string `json:"aggregate_hash"`
	GeneratedAt     string `json:"generated_at"`
	Kid             string `json:"kid,omitempty"`
	Signature       string `json:"signature,omitempty"`
	Mode            string `json:"mode,omitempty"`
	Threshold       int    `json:"threshold,omitempty"`
	SatisfiedWeight int    `json:"satisfied_weight,omitempty"`
	Signatures      []struct {
		Kid       string `json:"kid"`
		Mode      string `json:"mode"`
		Signature string `json:"signature"`
	} `json:"signatures,omitempty"`
}

// FetchRotationSummary retrieves the summary document.
func FetchRotationSummary(client HTTPClient, base string) (*RotationSummaryAPI, error) {
	url := fmt.Sprintf("%s/api/v1/beta/rotations/summary", base)
	b, err := httpRead(client, url)
	if err != nil {
		return nil, err
	}
	var rs RotationSummaryAPI
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, err
	}
	if !rs.Success {
		return &rs, errors.New("rotation_summary_endpoint: success=false")
	}
	return &rs, nil
}

// VerifyRotationSummarySignature verifies EdDSA signature using any matching public key loaded via JWKS.
// Returns nil if valid; otherwise error with reason.
func VerifyRotationSummarySignature(sum *RotationSummary, kp interface {
	PublicKey(string) ([]byte, string, error)
	VerifyWith([]byte, []byte, string) error
}) error {
	if sum == nil {
		return errors.New("summary_nil")
	}
	if kp == nil {
		return errors.New("key_provider_nil")
	}
	// Prefer multi-signature verification if slice populated; require at least one signature.
	if len(sum.Signatures) > 0 {
		// Validate all signatures whose kids resolve; treat any invalid as failure.
		for _, sigEntry := range sum.Signatures {
			if sigEntry.Signature == "" {
				return errors.New("missing_signature")
			}
			// fallback to single legacy fields for message payload
			enc, err := canonicalRotationSummaryPayload(sum)
			if err != nil {
				return fmt.Errorf("serialization_error:%w", err)
			}
			msg := append([]byte("AGENTAUTH_ROTATION_SUMMARY:"), enc...)
			sigBytes, err := base64.RawURLEncoding.DecodeString(sigEntry.Signature)
			if err != nil {
				return errors.New("signature_decode_error")
			}

			if err := kp.VerifyWith(msg, sigBytes, sigEntry.Kid); err != nil {
				return errors.New("signature_invalid")
			}
		}
		// Optionally enforce threshold satisfaction
		if sum.Threshold > 0 && sum.SatisfiedWeight < sum.Threshold {
			return errors.New("threshold_not_satisfied")
		}
		return nil
	}
	if sum.Signature == "" { // legacy single-signature path
		return errors.New("missing_signature")
	}
	if sum.Mode != "" && sum.Mode != "EdDSA" {
		return fmt.Errorf("mode_unsupported:%s", sum.Mode)
	}

	enc, err := canonicalRotationSummaryPayload(sum)
	if err != nil {
		return fmt.Errorf("serialization_error:%w", err)
	}
	msg := append([]byte("AGENTAUTH_ROTATION_SUMMARY:"), enc...)
	sig, err := base64.RawURLEncoding.DecodeString(sum.Signature)
	if err != nil {
		return errors.New("signature_decode_error")
	}

	if err := kp.VerifyWith(msg, sig, sum.Kid); err != nil {
		return errors.New("signature_invalid")
	}
	return nil
}

// RotationSummaryVerifyResult aggregates verification checks for convenience.
type RotationSummaryVerifyResult struct {
	Summary        *RotationSummary
	SignatureValid bool
	SignatureError string
	AgeSeconds     float64
}

// VerifyRotationSummary end-to-end: fetch summary, ensure JWKS loaded (caller should invoke LoadJWKS first), verify signature.
func VerifyRotationSummary(client HTTPClient, base string, kp interface {
	PublicKey(string) ([]byte, string, error)
	VerifyWith([]byte, []byte, string) error
}) (*RotationSummaryVerifyResult, error) {
	apiSum, err := FetchRotationSummary(client, base)
	if err != nil {
		return nil, err
	}
	res := &RotationSummaryVerifyResult{Summary: apiSum.Summary}
	if apiSum.Summary != nil {
		if t, perr := time.Parse(time.RFC3339Nano, apiSum.Summary.GeneratedAt); perr == nil {
			res.AgeSeconds = time.Since(t).Seconds()
		}
		if err := VerifyRotationSummarySignature(apiSum.Summary, kp); err != nil {
			res.SignatureValid = false
			res.SignatureError = err.Error()
		} else {
			res.SignatureValid = true
		}
	}
	return res, nil
}

// canonicalRotationSummaryPayload mirrors internal/notary helper to avoid map ordering variance.
func canonicalRotationSummaryPayload(sum *RotationSummary) ([]byte, error) {
	payload := struct {
		ChainLength   int    `json:"chain_length"`
		HeadHash      string `json:"head_hash"`
		AggregateHash string `json:"aggregate_hash"`
		GeneratedAt   string `json:"generated_at"`
	}{sum.ChainLength, sum.HeadHash, sum.AggregateHash, sum.GeneratedAt}
	return json.Marshal(payload)
}
