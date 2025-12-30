package agentauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
)

// HMACValidator validates tokens using HMAC-SHA256 signature
// Implements TokenSignatureValidator interface
type HMACValidator struct {
	signingKey    []byte
	strictParsing bool
}

// NewHMACValidator creates a new HMAC validator
func NewHMACValidator(signingKey []byte, strictParsing bool) *HMACValidator {
	return &HMACValidator{
		signingKey:    signingKey,
		strictParsing: strictParsing,
	}
}

// ValidateSignature validates HMAC signature and extracts claims
func (v *HMACValidator) ValidateSignature(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Verify HMAC signature
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, v.signingKey)
	mac.Write([]byte(unsigned))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	actualSig := parts[2]
	if !hmac.Equal([]byte(expectedSig), []byte(actualSig)) {
		return nil, ErrInvalidToken
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Parse claims
	var claims map[string]any
	if v.strictParsing {
		parser := DefaultSecureParser()
		if err := parser.ParseSecure(payloadBytes, &claims); err != nil {
			return nil, ErrInvalidToken
		}
	} else {
		if err := json.Unmarshal(payloadBytes, &claims); err != nil {
			return nil, ErrInvalidToken
		}
	}

	return claims, nil
}

// Name returns the validator name
func (v *HMACValidator) Name() string {
	return "HMAC"
}
