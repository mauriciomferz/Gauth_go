package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/web/handlers/auth"
	"github.com/stretchr/testify/assert"
)

func TestBackchannelAuthorize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup handler (nil DB for now as implementation supports degradation)
	h := auth.NewOAuth2Handler(nil, "secret")
	router := gin.New()
	router.POST("/bc-authorize", h.BackchannelAuthorize)

	data := url.Values{}
	data.Set("scope", "openid")
	data.Set("login_hint", "user@example.com")
	data.Set("binding_message", "1234")

	req, _ := http.NewRequest("POST", "/bc-authorize", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["auth_req_id"])
	assert.Equal(t, float64(600), resp["expires_in"])
}

func TestTokenExchange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := auth.NewOAuth2Handler(nil, "secret")
	router := gin.New()
	router.POST("/token", h.Token)

	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	data.Set("subject_token", "valid_token")
	data.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")

	req, _ := http.NewRequest("POST", "/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["access_token"])
	assert.Equal(t, "urn:ietf:params:oauth:token-type:access_token", resp["issued_token_type"])
}
