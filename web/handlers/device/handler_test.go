package device_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/device"
	deviceHandler "github.com/mauriciomferz/Gauth_go/web/handlers/device"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.LoadHTMLGlob("../../templates/*")

	store := device.NewMemoryDeviceCodeStore()
	h := deviceHandler.NewHandler(store)
	h.RegisterRoutes(r)

	return r
}

func TestDeviceAuthorizationFlow(t *testing.T) {
	r := setupRouter()

	// 1. Device Authorization Request
	authReq := device.DeviceAuthRequest{
		ClientID: "test-client",
		Scope:    "read write",
	}
	body, _ := json.Marshal(authReq)
	req, _ := http.NewRequest("POST", "/device/authorize", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var authResp device.DeviceAuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &authResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if authResp.DeviceCode == "" || authResp.UserCode == "" {
		t.Fatal("Expected device_code and user_code")
	}

	// 2. Verify Page Load
	req, _ = http.NewRequest("GET", "/device/verify?code="+authResp.UserCode, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected verify page 200 OK, got %d", w.Code)
	}

	// 3. User Authorization (Submit Code)
	form := url.Values{}
	form.Add("user_code", authResp.UserCode)
	form.Add("action", "authorize")
	form.Add("user_id", "user123")

	req, _ = http.NewRequest("POST", "/device/verify", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for verification, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 4. Device Polling for Token
	tokenReq := device.TokenRequest{
		GrantType:  device.DeviceFlowGrantType,
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}
	body, _ = json.Marshal(tokenReq)
	req, _ = http.NewRequest("POST", "/device/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for token, got %d. Body: %s", w.Code, w.Body.String())
	}

	var tokenResp device.TokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("Failed to parse token response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		t.Fatal("Expected access_token")
	}
}

func TestDevicePolling_Pending(t *testing.T) {
	r := setupRouter()

	// 1. Request Code
	authReq := device.DeviceAuthRequest{ClientID: "client"}
	body, _ := json.Marshal(authReq)
	req, _ := http.NewRequest("POST", "/device/authorize", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var authResp device.DeviceAuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &authResp)

	// 2. Poll immediately (Success -> Polling Wait -> Pending)
	// Note: In real world, we should respect interval.
	// Our mock implementation checks interval strictly.
	// But first poll is usually allowed or pending.

	// Let's mimic waiting for user...

	tokenReq := device.TokenRequest{
		GrantType:  device.DeviceFlowGrantType,
		DeviceCode: authResp.DeviceCode,
		ClientID:   "client",
	}
	body, _ = json.Marshal(tokenReq)
	req, _ = http.NewRequest("POST", "/device/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request (pending), got %d", w.Code)
	}

	var errResp device.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "authorization_pending" {
		t.Errorf("Expected authorization_pending, got %s", errResp.Error)
	}
}

func TestDeviceCode_Expiration(t *testing.T) {
	// Unit test for store expiration logic
	store := device.NewMemoryDeviceCodeStore()
	req := &device.DeviceAuthRequest{ClientID: "test"}
	// Create with 0 expiration (instant expire) or negative
	// Actually we can't easily inject negative time via public API unless we modify Create sig or use mock time
	// For now, let's create with 1 second and sleep 2

	dc, _ := store.Create(req, 1, 5) // 1 second expiry

	time.Sleep(1100 * time.Millisecond)

	if !dc.IsExpired() {
		t.Error("Expected device code to be expired")
	}

	// Test Authorization on expired code
	err := store.Authorize(dc.UserCode, "user")
	if err == nil || err.Error() != "device code expired" {
		t.Errorf("Expected 'device code expired' error, got %v", err)
	}
}
