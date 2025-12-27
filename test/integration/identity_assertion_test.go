package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/Gauth_go/web/handlers/grant_jwt"
)

// MockAuthenticator mimics ClientAuthenticator for testing
type MockAuthenticator struct {
	VerifyFunc func(issuer, token string, assertionType string) error
}

func (m *MockAuthenticator) Authenticate(issuer, token string, assertionType string) error {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(issuer, token, assertionType)
	}
	return nil
}

func (m *MockAuthenticator) ValidatePoASignature(ctx context.Context, poaID string, signature []byte, payload []byte) error {
	return nil
}

func TestIdentityAssertionGrant(t *testing.T) {
	// Setup keys for signing assertion
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create assertion token
	assertionClaims := jwt.MapClaims{
		"iss": "https://idp.example.com",
		"sub": "user@example.com",
		"aud": "https://gauth.example.com",
		"exp": time.Now().Add(time.Hour).Unix(),
		"jti": "test-nonce-123",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, assertionClaims)
	assertion, err := token.SignedString(privKey)
	if err != nil {
		t.Fatalf("Failed to sign assertion: %v", err)
	}

	// Setup Handler with Mock Authenticator
	mockAuth := &MockAuthenticator{}
	handler := grant_jwt.NewHandler(mockAuth)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router)

	t.Run("ValidIdentityAssertion", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:identity-assertion")
		form.Set("assertion", assertion)

		req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if resp["token_type"] != "Bearer" {
			t.Errorf("Expected token_type Bearer, got %v", resp["token_type"])
		}
		if _, ok := resp["access_token"]; !ok {
			t.Error("Response missing access_token")
		}
	})

	t.Run("InvalidGrantType", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "invalid:grant")
		form.Set("assertion", assertion)

		req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("MissingAssertion", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:identity-assertion")

		req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})
}
