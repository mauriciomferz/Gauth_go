package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MockKeyProvider for testing
type MockKeyProvider struct {
	Keys map[string]*rsa.PublicKey
}

func (m *MockKeyProvider) GetPublicKey(clientID string, keyID string) (*rsa.PublicKey, error) {
	if key, ok := m.Keys[clientID+":"+keyID]; ok {
		return key, nil
	}
	return nil, errors.New("key not found")
}

func TestPrivateKeyJWTValidator_Authenticate(t *testing.T) {
	// Generate RSA Key Pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := &privateKey.PublicKey

	// Setup Validator
	mockProvider := &MockKeyProvider{
		Keys: map[string]*rsa.PublicKey{
			"test-client:key-1": publicKey,
		},
	}

	validator := &PrivateKeyJWTValidator{
		KeyProvider: mockProvider,
		TokenURL:    "https://auth.example.com/token",
	}

	// Helper to create token
	createToken := func(iss, sub string, aud []string, exp time.Time, kid string) string {
		claims := jwt.MapClaims{
			"iss": iss,
			"sub": sub,
			"aud": aud,
			"exp": jwt.NewNumericDate(exp),
			"jti": "unique-id",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = kid

		signed, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatal(err)
		}
		return signed
	}

	tests := []struct {
		name          string
		clientID      string
		assertion     string
		assertionType string
		wantErr       bool
	}{
		{
			name:     "Valid Assertion",
			clientID: "test-client",
			assertion: createToken(
				"test-client",
				"test-client",
				[]string{"https://auth.example.com/token"},
				time.Now().Add(time.Hour),
				"key-1",
			),
			assertionType: ClientAssertionTypeJWT,
			wantErr:       false,
		},
		{
			name:          "Invalid Assertion Type",
			clientID:      "test-client",
			assertion:     "foo",
			assertionType: "bar",
			wantErr:       true,
		},
		{
			name:     "Expired Token",
			clientID: "test-client",
			assertion: createToken(
				"test-client",
				"test-client",
				[]string{"https://auth.example.com/token"},
				time.Now().Add(-time.Hour),
				"key-1",
			),
			assertionType: ClientAssertionTypeJWT,
			wantErr:       true,
		},
		{
			name:     "Wrong Issuer",
			clientID: "test-client",
			assertion: createToken(
				"wrong-client",
				"test-client",
				[]string{"https://auth.example.com/token"},
				time.Now().Add(time.Hour),
				"key-1",
			),
			assertionType: ClientAssertionTypeJWT,
			wantErr:       true,
		},
		{
			name:     "Wrong Audience",
			clientID: "test-client",
			assertion: createToken(
				"test-client",
				"test-client",
				[]string{"https://wrong.com"},
				time.Now().Add(time.Hour),
				"key-1",
			),
			assertionType: ClientAssertionTypeJWT,
			wantErr:       true,
		},
		{
			name:     "Unknown Key",
			clientID: "test-client",
			assertion: createToken(
				"test-client",
				"test-client",
				[]string{"https://auth.example.com/token"},
				time.Now().Add(time.Hour),
				"unknown-key",
			),
			assertionType: ClientAssertionTypeJWT,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Authenticate(tt.clientID, tt.assertion, tt.assertionType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
