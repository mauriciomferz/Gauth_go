package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ClientAuthenticator defines the interface for authenticating clients
type ClientAuthenticator interface {
	Authenticate(clientID string, clientAssertion string, clientAssertionType string) error
}

// KeyProvider defines how to retrieve public keys for a client
type KeyProvider interface {
	GetPublicKey(clientID string, keyID string) (*rsa.PublicKey, error)
}

// PrivateKeyJWTValidator implements RFC 7523 client authentication
type PrivateKeyJWTValidator struct {
	KeyProvider KeyProvider
	TokenURL    string // The Audience (aud) the JWT should be issued for
}

// RFC 7523 Constants
const (
	ClientAssertionTypeJWT = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
)

// Authenticate verifies the client assertion
func (v *PrivateKeyJWTValidator) Authenticate(clientID string, clientAssertion string, clientAssertionType string) error {
	if clientAssertionType != ClientAssertionTypeJWT {
		return fmt.Errorf("unsupported client_assertion_type: %s", clientAssertionType)
	}

	if clientAssertion == "" {
		return errors.New("missing client_assertion")
	}

	// Parse WITHOUT verification first to get Key ID (kid) and Claims
	token, _, err := new(jwt.Parser).ParseUnverified(clientAssertion, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("malformed client_assertion: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("invalid token claims")
	}

	// Verify ISS (Issuer) matches Client ID
	iss, _ := claims.GetIssuer()
	if iss != clientID {
		return fmt.Errorf("invalid issuer: expected %s, got %s", clientID, iss)
	}

	// Verify SUB (Subject) matches Client ID
	sub, _ := claims.GetSubject()
	if sub != clientID {
		return fmt.Errorf("invalid subject: expected %s, got %s", clientID, sub)
	}

	// Verify AUD (Audience) matches Token Endpoint URL
	aud, _ := claims.GetAudience()
	audValid := false
	for _, a := range aud {
		if a == v.TokenURL {
			audValid = true
			break
		}
	}
	if !audValid {
		return fmt.Errorf("invalid audience: expected %s", v.TokenURL)
	}

	// Verify Expiration
	exp, _ := claims.GetExpirationTime()
	if exp == nil || exp.Before(time.Now()) {
		return errors.New("token expired")
	}

	// Get Key ID header
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return errors.New("missing 'kid' in header")
	}

	// Retrieve Public Key
	pubKey, err := v.KeyProvider.GetPublicKey(clientID, kid)
	if err != nil {
		return fmt.Errorf("failed to retrieve public key: %v", err)
	}

	// Re-parse WITH signature verification
	_, err = jwt.Parse(clientAssertion, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})

	if err != nil {
		return fmt.Errorf("signature verification failed: %v", err)
	}

	return nil
}
