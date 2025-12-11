package device_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/Gauth_go/pkg/auth"
	"github.com/mauriciomferz/Gauth_go/pkg/device"
	deviceHandler "github.com/mauriciomferz/Gauth_go/web/handlers/device"
)

func TestDeviceToken_WithClientAssertion(t *testing.T) {
	// Setup Keys
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	clientID := "secure-client"
	keyID := "key-1"

	// Setup Auth
	keyStore := auth.NewMemoryKeyStore()
	keyStore.RegisterKey(clientID, keyID, publicKey)

	authenticator := &auth.PrivateKeyJWTValidator{
		KeyProvider: keyStore,
		TokenURL:    "http://test-server/device/token",
	}

	// Setup Router
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := device.NewMemoryDeviceCodeStore()
	h := deviceHandler.NewHandler(store)
	h.SetAuthenticator(authenticator)
	h.RegisterRoutes(r)

	// 1. Authorize
	authReq := device.DeviceAuthRequest{ClientID: clientID}
	body, _ := json.Marshal(authReq)
	req, _ := http.NewRequest("POST", "/device/authorize", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var authResp device.DeviceAuthResponse
	json.Unmarshal(w.Body.Bytes(), &authResp)

	// 2. User Verify (Helper)
	form := url.Values{}
	form.Add("user_code", authResp.UserCode)
	form.Add("action", "authorize")
	form.Add("user_id", "user1")
	req, _ = http.NewRequest("POST", "/device/verify", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Verification failed: %d", w.Code)
	}

	// 3. Create Assertion
	claims := jwt.MapClaims{
		"iss": clientID,
		"sub": clientID,
		"aud": []string{"http://test-server/device/token"},
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "nonce-123",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	signed, _ := token.SignedString(privateKey)

	// 4. Token Request with Assertion
	tokenReq := device.TokenRequest{
		GrantType:           device.DeviceFlowGrantType,
		DeviceCode:          authResp.DeviceCode,
		ClientID:            clientID,
		ClientAssertion:     signed,
		ClientAssertionType: auth.ClientAssertionTypeJWT,
	}
	body, _ = json.Marshal(tokenReq)
	req, _ = http.NewRequest("POST", "/device/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK with assertion, got %d. Body: %s", w.Code, w.Body.String())
	}
}
