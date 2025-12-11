package gnap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSignatureMiddleware_NoResolver(t *testing.T) {
	r := gin.New()
	r.Use(SignatureMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestSignatureMiddleware_NoHeaders(t *testing.T) {
	resolver := func(keyID string) (any, error) {
		return nil, nil
	}

	r := gin.New()
	r.Use(SignatureMiddleware(resolver))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should still pass (gradual rollout mode)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

type mockTokenStore struct {
	tokens map[string]*IssuedToken
}

func (m *mockTokenStore) Get(value string) (*IssuedToken, error) {
	if token, ok := m.tokens[value]; ok {
		return token, nil
	}
	return nil, nil
}

func TestRequireGNAPToken_Valid(t *testing.T) {
	store := &mockTokenStore{
		tokens: map[string]*IssuedToken{
			"valid-token": {Value: "valid-token", GrantID: "grant-123"},
		},
	}

	r := gin.New()
	r.Use(RequireGNAPToken(store))
	r.GET("/protected", func(c *gin.Context) {
		grantID := c.GetString("gnap_grant_id")
		c.String(200, grantID)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "GNAP valid-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "grant-123" {
		t.Errorf("Expected grant-123, got %s", w.Body.String())
	}
}

func TestRequireGNAPToken_Missing(t *testing.T) {
	store := &mockTokenStore{tokens: map[string]*IssuedToken{}}

	r := gin.New()
	r.Use(RequireGNAPToken(store))
	r.GET("/protected", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestRequireGNAPToken_InvalidFormat(t *testing.T) {
	store := &mockTokenStore{tokens: map[string]*IssuedToken{}}

	r := gin.New()
	r.Use(RequireGNAPToken(store))
	r.GET("/protected", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-format")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestRequireGNAPToken_Revoked(t *testing.T) {
	revokedAt := "2024-01-01T00:00:00Z"
	store := &mockTokenStore{
		tokens: map[string]*IssuedToken{
			"revoked-token": {Value: "revoked-token", GrantID: "grant-456", RevokedAt: &revokedAt},
		},
	}

	r := gin.New()
	r.Use(RequireGNAPToken(store))
	r.GET("/protected", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "GNAP revoked-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}
