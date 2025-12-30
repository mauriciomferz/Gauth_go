package agentauth

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// EdDSAValidator validates tokens using EdDSA (Ed25519) signature
// Implements TokenSignatureValidator interface
type EdDSAValidator struct {
	keyProvider   crypto.KeyProvider
	strictParsing bool
}

// NewEdDSAValidator creates a new EdDSA validator
func NewEdDSAValidator(kp crypto.KeyProvider, strictParsing bool) *EdDSAValidator {
	return &EdDSAValidator{
		keyProvider:   kp,
		strictParsing: strictParsing,
	}
}

// ValidateSignature validates EdDSA signature and extracts claims
func (v *EdDSAValidator) ValidateSignature(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Decode and parse header
	headBytes, hErr := base64.RawURLEncoding.DecodeString(parts[0])
	if hErr != nil {
		return nil, ErrInvalidToken
	}

	var head map[string]any
	if v.strictParsing {
		parser := DefaultSecureParser()
		if uErr := parser.ParseSecure(headBytes, &head); uErr != nil {
			return nil, ErrInvalidToken
		}
	} else {
		if uErr := json.Unmarshal(headBytes, &head); uErr != nil {
			return nil, ErrInvalidToken
		}
	}

	// Validate header fields
	algVal, okAlg := head["alg"].(string)
	kidVal, okKid := head["kid"].(string)
	if !okAlg || !okKid || algVal != edDSAAlgConst || kidVal == "" {
		return nil, ErrInvalidToken
	}

	// Verify signature
	unsigned := parts[0] + "." + parts[1]
	sigBytes, sigErr := base64.RawURLEncoding.DecodeString(parts[2])
	if sigErr != nil {
		return nil, ErrInvalidToken
	}

	if v.keyProvider == nil {
		return nil, ErrInvalidToken
	}

	if vErr := v.keyProvider.VerifyWith([]byte(unsigned), sigBytes, kidVal); vErr != nil {
		return nil, ErrInvalidToken
	}

	// Decode and parse payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

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
func (v *EdDSAValidator) Name() string {
	return "EdDSA"
}
