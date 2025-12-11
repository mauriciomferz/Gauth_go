package grant_jwt_test

import (
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
	"github.com/mauriciomferz/Gauth_go/pkg/auth"
	"github.com/mauriciomferz/Gauth_go/web/handlers/grant_jwt"
)

func TestJWTBearerGrant_Success(t *testing.T) {
	// Setup Keys
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	clientID := "service-account-1"
	keyID := "key-sa-1"

	// Setup Auth
	keyStore := auth.NewMemoryKeyStore()
	keyStore.RegisterKey(clientID, keyID, publicKey)

	authenticator := &auth.PrivateKeyJWTValidator{
		KeyProvider: keyStore,
		TokenURL:    "http://test-server/oauth/token",
	}

	// Setup Handler
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := grant_jwt.NewHandler(authenticator)
	h.RegisterRoutes(r)

	// Create JWT Assertion (Self-Signed)
	claims := jwt.MapClaims{
		"iss": clientID,
		"sub": clientID, // Self-assertion
		"aud": []string{"http://test-server/oauth/token"},
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "nonce-sa-1",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	signedAssertion, _ := token.SignedString(privateKey)

	// Request Token
	form := url.Values{}
	form.Add("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Add("assertion", signedAssertion)

	req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["token_type"] != "Bearer" {
		t.Errorf("Expected token_type Bearer, got %v", resp["token_type"])
	}
	if resp["access_token"] == "" {
		t.Error("Expected access_token")
	}
}

func TestJWTBearerGrant_InvalidGrantType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := grant_jwt.NewHandler(nil) // auth not needed for this check
	h.RegisterRoutes(r)

	form := url.Values{}
	form.Add("grant_type", "password") // Wrong type
	form.Add("assertion", "foo")

	req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}
}
