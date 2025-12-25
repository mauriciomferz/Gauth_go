package token_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/web/handlers/token"
)

// MockGAuthService mocks gauth.GAuth interface
type MockGAuthService struct {
	mock.Mock
}

func (m *MockGAuthService) RequestToken(req gauth.TokenRequest) (*gauth.TokenResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gauth.TokenResponse), args.Error(1)
}

func (m *MockGAuthService) InitiateAuthorization(req gauth.AuthorizationRequest) (*gauth.AuthorizationGrant, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gauth.AuthorizationGrant), args.Error(1)
}

func (m *MockGAuthService) ValidateToken(token string) (*gauth.TokenValidationResult, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gauth.TokenValidationResult), args.Error(1)
}

func (m *MockGAuthService) Close() error {
	return nil
}

func TestRFC9396_CreateToken_RAR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockGAuthService)

	// Setup Handler with Mock Service
	h := &token.Handler{} // Initialize empty, then set service
	h.SetGAuthService(mockSvc)

	router := gin.New()
	router.POST("/token", h.Create)

	rarDetail := gauth.AuthorizationDetail{
		Type:       "payment_initiation",
		Locations:  []string{"https://api.example.com/accounts"},
		Actions:    []string{"transfer"},
		Identifier: "tx_123",
	}

	payload := map[string]interface{}{
		"grant_id":              "grant_abc",
		"authorization_details": []gauth.AuthorizationDetail{rarDetail},
		"nonce":                 "nonce_123",
	}
	body, _ := json.Marshal(payload)

	// Expectation
	// We use MatchedBy because Context map comparison can be tricky with nil/interface{}
	expectedResp := &gauth.TokenResponse{
		Token:      "mock_rfc_token",
		Scope:      []string{"payment"},
		ValidUntil: time.Now().Add(1 * time.Hour),
	}

	mockSvc.On("RequestToken", mock.MatchedBy(func(req gauth.TokenRequest) bool {
		if req.GrantID != "grant_abc" {
			return false
		}
		if len(req.AuthorizationDetails) != 1 {
			return false
		}
		if req.AuthorizationDetails[0].Type != "payment_initiation" {
			return false
		}
		return true
	})).Return(expectedResp, nil)

	req, _ := http.NewRequest("POST", "/token", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var respBody map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	assert.Equal(t, "mock_rfc_token", respBody["access_token"])
	assert.Equal(t, true, respBody["success"])
	assert.Equal(t, "Bearer", respBody["token_type"])

	mockSvc.AssertExpectations(t)
}
