package gnap

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
)

// InteractionService handles user interaction flows for GNAP grants.
type InteractionService struct {
	BaseURL string
	ASURL   string // Authorization Server URL for hash calculation
}

// NewInteractionService creates an interaction service.
func NewInteractionService(baseURL, asURL string) *InteractionService {
	return &InteractionService{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		ASURL:   asURL,
	}
}

// BuildRedirectURI constructs the redirect URI for interaction.
func (s *InteractionService) BuildRedirectURI(grant *Grant) string {
	return s.BaseURL + "/gnap/interact/" + grant.ID
}

// BuildCallbackURI constructs the callback URI after interaction completes.
// Per RFC 9635 §4.2.1, the callback includes:
// - hash: interaction hash
// - interact_ref: reference for continuation
func (s *InteractionService) BuildCallbackURI(grant *Grant, finishURI string, clientNonce string) (string, error) {
	if grant.InteractRef == "" || grant.InteractNonce == "" {
		return "", errors.New("grant missing interaction reference or nonce")
	}

	// Calculate interaction hash per RFC 9635 §4.2.3
	hash := s.CalculateInteractionHash(clientNonce, grant.InteractNonce, grant.InteractRef)

	// Build callback URL
	u, err := url.Parse(finishURI)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("hash", hash)
	q.Set("interact_ref", grant.InteractRef)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// CalculateInteractionHash computes the hash per RFC 9635 §4.2.3.
// hash = BASE64URL(SHA-256(client_nonce + "\n" + server_nonce + "\n" + interact_ref + "\n" + grant_endpoint))
func (s *InteractionService) CalculateInteractionHash(clientNonce, serverNonce, interactRef string) string {
	data := clientNonce + "\n" + serverNonce + "\n" + interactRef + "\n" + s.ASURL
	h := sha256.Sum256([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// VerifyInteractionHash validates the hash from a callback.
func (s *InteractionService) VerifyInteractionHash(clientNonce, serverNonce, interactRef, providedHash string) bool {
	expected := s.CalculateInteractionHash(clientNonce, serverNonce, interactRef)
	return expected == providedHash
}

// UserCodeInfo holds information for user_code interaction mode.
type UserCodeInfo struct {
	Code      string // Short code for user to enter
	URI       string // URI where user enters code
	ExpiresIn int    // Seconds until code expires
}

// GenerateUserCode creates a user-friendly code for device-style flow.
func GenerateUserCode() *UserCodeInfo {
	// Generate 8 character code (excluding ambiguous chars)
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 8)
	rand.Read(code)
	for i := range code {
		code[i] = chars[code[i]%byte(len(chars))]
	}

	return &UserCodeInfo{
		Code:      string(code[:4]) + "-" + string(code[4:]),
		ExpiresIn: 600, // 10 minutes
	}
}
