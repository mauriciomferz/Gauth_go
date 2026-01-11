package auth

import (
	"crypto/ecdsa"
	"crypto/ed25519"
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
	GetPublicKey(clientID string, keyID string) (any, error)
}

// ReplayStore defines the interface for checking and recording token JTI for replay protection
type ReplayStore interface {
	CheckAndStore(jti string) error
}

// PrivateKeyJWTValidator implements RFC 7523 client authentication
type PrivateKeyJWTValidator struct {
	KeyProvider    KeyProvider
	ValidAudiences []string    // The allowed Audiences (aud) the JWT should be issued for
	Replay         ReplayStore // Optional replay protection
}

// RFC 7523 Constants
const (
	ClientAssertionTypeJWT = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
)

// Authenticate verifies the client assertion
func (v *PrivateKeyJWTValidator) Authenticate(
	clientID string,
	clientAssertion string,
	clientAssertionType string,
) error {
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

	// Verify AUD (Audience) matches one of the allowed audiences
	aud, _ := claims.GetAudience()
	if !v.isAudienceAllowed(aud) {
		return fmt.Errorf("invalid audience: expected one of %v", v.ValidAudiences)
	}

	// Verify JTI (Replay Protection)
	if v.Replay != nil {
		jti, _ := claims["jti"].(string)
		if jti == "" {
			return errors.New("missing 'jti' claim for replay protection")
		}
		if replayErr := v.Replay.CheckAndStore(jti); replayErr != nil {
			return fmt.Errorf("replay check failed: %v", replayErr)
		}
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
		if keyErr := validateSigningMethodAndKey(token, pubKey); keyErr != nil {
			return nil, keyErr
		}
		return pubKey, nil
	})

	if err != nil {
		return fmt.Errorf("signature verification failed: %v", err)
	}

	return nil
}

func (v *PrivateKeyJWTValidator) isAudienceAllowed(aud []string) bool {
	for _, a := range aud {
		for _, validAud := range v.ValidAudiences {
			if a == validAud {
				return true
			}
		}
	}
	return false
}

func validateSigningMethodAndKey(token *jwt.Token, pubKey any) error {
	switch token.Method.(type) {
	case *jwt.SigningMethodRSA:
		if _, ok := pubKey.(*rsa.PublicKey); !ok {
			return errors.New("key type mismatch: expected RSA public key")
		}
	case *jwt.SigningMethodECDSA:
		if _, ok := pubKey.(*ecdsa.PublicKey); !ok {
			return errors.New("key type mismatch: expected ECDSA public key")
		}
	case *jwt.SigningMethodEd25519:
		if _, ok := pubKey.(ed25519.PublicKey); !ok {
			return errors.New("key type mismatch: expected Ed25519 public key")
		}
	default:
		return fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	return nil
}
