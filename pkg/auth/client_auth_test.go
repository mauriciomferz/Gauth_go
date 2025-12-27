package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MockKeyProvider for testing
type MockKeyProvider struct {
	Keys map[string]any
}

func (m *MockKeyProvider) GetPublicKey(clientID string, keyID string) (any, error) {
	if key, ok := m.Keys[clientID+":"+keyID]; ok {
		return key, nil
	}
	return nil, errors.New("key not found")
}

// MockReplayStore for testing
type MockReplayStore struct {
	SeenJTIs map[string]bool
}

func (m *MockReplayStore) CheckAndStore(jti string) error {
	if m.SeenJTIs[jti] {
		return errors.New("replay detected")
	}
	m.SeenJTIs[jti] = true
	return nil
}

func TestPrivateKeyJWTValidator_Authenticate(t *testing.T) {
	// Generate RSA Key Pair
	privateKeyRSA, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKeyRSA := &privateKeyRSA.PublicKey

	// Generate Ed25519 Key Pair
	pubEd, privEd, _ := ed25519.GenerateKey(rand.Reader)

	// Setup Validator
	mockProvider := &MockKeyProvider{
		Keys: map[string]any{
			"test-client:key-rsa": publicKeyRSA,
			"test-client:key-ed":  pubEd,
		},
	}

	mockReplay := &MockReplayStore{SeenJTIs: make(map[string]bool)}

	validator := &PrivateKeyJWTValidator{
		KeyProvider:    mockProvider,
		ValidAudiences: []string{"https://auth.example.com/token", "https://api.example.com/token"},
		Replay:         mockReplay,
	}

	// Helper to create token
	createToken := func(method jwt.SigningMethod, key any, iss, sub string, aud []string, exp time.Time, kid string, jti string) string {
		claims := jwt.MapClaims{
			"iss": iss,
			"sub": sub,
			"aud": aud,
			"exp": jwt.NewNumericDate(exp),
			"jti": jti,
		}
		token := jwt.NewWithClaims(method, claims)
		token.Header["kid"] = kid

		signed, err := token.SignedString(key)
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
			name:     "Valid RSA Assertion",
			clientID: "test-client",
			assertion: createToken(
				jwt.SigningMethodRS256,
				privateKeyRSA,
				"test-client",
				"test-client",
				[]string{"https://auth.example.com/token"},
				time.Now().Add(time.Hour),
				"key-rsa",
				"unique-1",
			),
			assertionType: ClientAssertionTypeJWT,
			wantErr:       false,
		},
		{
			name:     "Valid EdDSA Assertion",
			clientID: "test-client",
			assertion: createToken(
				jwt.SigningMethodEdDSA,
				privEd,
				"test-client",
				"test-client",
				[]string{"https://api.example.com/token"},
				time.Now().Add(time.Hour),
				"key-ed",
				"unique-2",
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
				jwt.SigningMethodRS256,
				privateKeyRSA,
				"test-client",
				"test-client",
				[]string{"https://auth.example.com/token"},
				time.Now().Add(-time.Hour),
				"key-rsa",
				"unique-3",
			),
			assertionType: ClientAssertionTypeJWT,
			wantErr:       true,
		},
		{
			name:     "Replay Attack",
			clientID: "test-client",
			assertion: createToken(
				jwt.SigningMethodRS256,
				privateKeyRSA,
				"test-client",
				"test-client",
				[]string{"https://auth.example.com/token"},
				time.Now().Add(time.Hour),
				"key-rsa",
				"unique-1", // Already used in first test case
			),
			assertionType: ClientAssertionTypeJWT,
			wantErr:       true,
		},
		{
			name:     "Wrong Audience",
			clientID: "test-client",
			assertion: createToken(
				jwt.SigningMethodRS256,
				privateKeyRSA,
				"test-client",
				"test-client",
				[]string{"https://wrong.com"},
				time.Now().Add(time.Hour),
				"key-rsa",
				"unique-4",
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
