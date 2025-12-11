// Package httpsig implements RFC 9421 HTTP Message Signatures for GNAP.
// This provides cryptographic proof of client identity for GNAP requests.
package httpsig

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Signer creates HTTP Message Signatures.
type Signer struct {
	KeyID      string
	PrivateKey crypto.PrivateKey
	Algorithm  string // ed25519, ecdsa-p256-sha256, rsa-pss-sha256
}

// NewSigner creates a signer with the given key.
func NewSigner(keyID string, privateKey crypto.PrivateKey) (*Signer, error) {
	var alg string
	switch privateKey.(type) {
	case ed25519.PrivateKey:
		alg = "ed25519"
	case *ecdsa.PrivateKey:
		alg = "ecdsa-p256-sha256"
	case *rsa.PrivateKey:
		alg = "rsa-pss-sha256"
	default:
		return nil, errors.New("unsupported key type")
	}

	return &Signer{
		KeyID:      keyID,
		PrivateKey: privateKey,
		Algorithm:  alg,
	}, nil
}

// Sign adds HTTP Message Signature headers to a request.
// Per RFC 9421, this adds Signature-Input and Signature headers.
func (s *Signer) Sign(req *http.Request) error {
	// Create signature base per RFC 9421 §2.5
	components := []string{
		"@method",
		"@target-uri",
		"content-type",
	}

	// Add content-digest for requests with body
	if req.Body != nil && req.ContentLength > 0 {
		components = append(components, "content-digest")
	}

	// Build signature parameters
	created := time.Now().Unix()
	expires := created + 300 // 5 minute validity
	nonce := generateNonce()

	sigParams := fmt.Sprintf(`(%s);created=%d;expires=%d;keyid="%s";alg="%s";nonce="%s"`,
		formatComponents(components), created, expires, s.KeyID, s.Algorithm, nonce)

	// Build signature base
	sigBase := s.buildSignatureBase(req, components, sigParams)

	// Sign
	sig, err := s.signPayload([]byte(sigBase))
	if err != nil {
		return fmt.Errorf("signing failed: %w", err)
	}

	// Set headers
	req.Header.Set("Signature-Input", "sig1="+sigParams)
	req.Header.Set("Signature", "sig1=:"+base64.StdEncoding.EncodeToString(sig)+":")

	return nil
}

// buildSignatureBase constructs the signature base string.
func (s *Signer) buildSignatureBase(req *http.Request, components []string, sigParams string) string {
	var lines []string

	for _, c := range components {
		var value string
		switch c {
		case "@method":
			value = req.Method
		case "@target-uri":
			value = req.URL.String()
		case "@path":
			value = req.URL.Path
		case "@query":
			value = "?" + req.URL.RawQuery
		case "content-type":
			value = req.Header.Get("Content-Type")
		case "content-digest":
			value = req.Header.Get("Content-Digest")
		default:
			value = req.Header.Get(c)
		}
		lines = append(lines, fmt.Sprintf(`"%s": %s`, c, value))
	}

	// Add signature params as final line
	lines = append(lines, fmt.Sprintf(`"@signature-params": %s`, sigParams))

	return strings.Join(lines, "\n")
}

// signPayload signs the payload with the private key.
func (s *Signer) signPayload(payload []byte) ([]byte, error) {
	switch key := s.PrivateKey.(type) {
	case ed25519.PrivateKey:
		return ed25519.Sign(key, payload), nil

	case *ecdsa.PrivateKey:
		hash := sha256.Sum256(payload)
		return ecdsa.SignASN1(rand.Reader, key, hash[:])

	case *rsa.PrivateKey:
		hash := sha256.Sum256(payload)
		return rsa.SignPSS(rand.Reader, key, crypto.SHA256, hash[:], nil)

	default:
		return nil, errors.New("unsupported key type")
	}
}

// Verifier validates HTTP Message Signatures.
type Verifier struct {
	KeyResolver func(keyID string) (any, error)
	MaxAge      time.Duration // Maximum signature age
}

// NewVerifier creates a signature verifier.
func NewVerifier(keyResolver func(keyID string) (any, error)) *Verifier {
	return &Verifier{
		KeyResolver: keyResolver,
		MaxAge:      5 * time.Minute,
	}
}

// Verify checks the signature on an HTTP request.
func (v *Verifier) Verify(req *http.Request) error {
	// Parse Signature-Input header
	sigInput := req.Header.Get("Signature-Input")
	if sigInput == "" {
		return errors.New("missing Signature-Input header")
	}

	// Parse signature header
	sigHeader := req.Header.Get("Signature")
	if sigHeader == "" {
		return errors.New("missing Signature header")
	}

	// Extract signature parameters
	params, err := parseSignatureInput(sigInput)
	if err != nil {
		return fmt.Errorf("invalid Signature-Input: %w", err)
	}

	// Check expiration
	if params.Expires > 0 && time.Now().Unix() > params.Expires {
		return errors.New("signature expired")
	}

	// Check age
	if params.Created > 0 {
		age := time.Since(time.Unix(params.Created, 0))
		if age > v.MaxAge {
			return errors.New("signature too old")
		}
	}

	// Get public key
	pubKey, err := v.KeyResolver(params.KeyID)
	if err != nil {
		return fmt.Errorf("key not found: %w", err)
	}

	// Rebuild signature base
	sigBase := buildVerifySignatureBase(req, params.Components, params.Raw)

	// Extract signature bytes
	sigBytes, err := extractSignature(sigHeader)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	// Verify
	return verifySignature(pubKey, params.Algorithm, []byte(sigBase), sigBytes)
}

// SignatureParams holds parsed signature parameters.
type SignatureParams struct {
	Components []string
	Created    int64
	Expires    int64
	KeyID      string
	Algorithm  string
	Nonce      string
	Raw        string // Original parameter string
}

// parseSignatureInput parses the Signature-Input header value.
func parseSignatureInput(input string) (*SignatureParams, error) {
	// Format: sig1=(@method @target-uri);created=...;keyid=...;alg=...
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid format")
	}

	params := &SignatureParams{Raw: parts[1]}

	// Parse components from parentheses
	if idx := strings.Index(parts[1], "("); idx >= 0 {
		end := strings.Index(parts[1], ")")
		if end > idx {
			compStr := parts[1][idx+1 : end]
			rawComps := strings.Fields(compStr)
			// Strip quotes from each component
			for _, c := range rawComps {
				params.Components = append(params.Components, strings.Trim(c, `"`))
			}
		}
	}

	// Parse key-value parameters
	for _, kv := range strings.Split(parts[1], ";") {
		kv = strings.TrimSpace(kv)
		if strings.HasPrefix(kv, "created=") {
			params.Created, _ = strconv.ParseInt(strings.TrimPrefix(kv, "created="), 10, 64)
		} else if strings.HasPrefix(kv, "expires=") {
			params.Expires, _ = strconv.ParseInt(strings.TrimPrefix(kv, "expires="), 10, 64)
		} else if strings.HasPrefix(kv, "keyid=") {
			params.KeyID = strings.Trim(strings.TrimPrefix(kv, "keyid="), `"`)
		} else if strings.HasPrefix(kv, "alg=") {
			params.Algorithm = strings.Trim(strings.TrimPrefix(kv, "alg="), `"`)
		} else if strings.HasPrefix(kv, "nonce=") {
			params.Nonce = strings.Trim(strings.TrimPrefix(kv, "nonce="), `"`)
		}
	}

	return params, nil
}

// buildVerifySignatureBase rebuilds the signature base for verification.
func buildVerifySignatureBase(req *http.Request, components []string, sigParams string) string {
	var lines []string

	for _, c := range components {
		var value string
		switch c {
		case "@method":
			value = req.Method
		case "@target-uri":
			value = req.URL.String()
		case "@path":
			value = req.URL.Path
		default:
			value = req.Header.Get(c)
		}
		lines = append(lines, fmt.Sprintf(`"%s": %s`, c, value))
	}

	lines = append(lines, fmt.Sprintf(`"@signature-params": %s`, sigParams))
	return strings.Join(lines, "\n")
}

// extractSignature extracts signature bytes from the header value.
func extractSignature(header string) ([]byte, error) {
	// Format: sig1=:base64...:
	parts := strings.SplitN(header, "=", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid format")
	}

	sigStr := strings.Trim(parts[1], ":")
	return base64.StdEncoding.DecodeString(sigStr)
}

// verifySignature verifies the signature with the given public key.
func verifySignature(pubKey crypto.PublicKey, alg string, payload, sig []byte) error {
	switch key := pubKey.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(key, payload, sig) {
			return errors.New("invalid ed25519 signature")
		}
		return nil

	case *ecdsa.PublicKey:
		hash := sha256.Sum256(payload)
		if !ecdsa.VerifyASN1(key, hash[:], sig) {
			return errors.New("invalid ecdsa signature")
		}
		return nil

	case *rsa.PublicKey:
		hash := sha256.Sum256(payload)
		if err := rsa.VerifyPSS(key, crypto.SHA256, hash[:], sig, nil); err != nil {
			return errors.New("invalid rsa signature")
		}
		return nil

	default:
		return errors.New("unsupported key type")
	}
}

// formatComponents formats component list for signature input.
func formatComponents(components []string) string {
	var quoted []string
	for _, c := range components {
		quoted = append(quoted, `"`+c+`"`)
	}
	return strings.Join(quoted, " ")
}

// generateNonce creates a random nonce.
func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
