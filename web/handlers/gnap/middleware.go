// Package gnap provides HTTP handlers for RFC 9635 GNAP endpoints.
package gnap

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/gnap/httpsig"
)

// KeyResolver resolves a key ID to a public key for signature verification.
type KeyResolver func(keyID string) (any, error)

// SignatureMiddleware creates middleware that verifies HTTP Message Signatures (RFC 9421).
// If keyResolver is nil, signature verification is skipped.
func SignatureMiddleware(keyResolver KeyResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if no key resolver (development mode)
		if keyResolver == nil {
			c.Next()
			return
		}

		// Check for signature headers
		sigInput := c.GetHeader("Signature-Input")
		sig := c.GetHeader("Signature")

		if sigInput == "" || sig == "" {
			// No signature - allow for now (gradual rollout)
			// In strict mode, reject here
			c.Next()
			return
		}

		// Verify signature
		verifier := httpsig.NewVerifier(keyResolver)
		if err := verifier.Verify(c.Request); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_client",
				"error_description": "HTTP signature verification failed: " + err.Error(),
			})
			return
		}

		c.Next()
	}
}

// RequireGNAPToken creates middleware that requires a valid GNAP token.
func RequireGNAPToken(tokenStore TokenStoreInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Missing Authorization header",
			})
			return
		}

		// Parse "GNAP <token>" format
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "GNAP") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Invalid Authorization header format, expected: GNAP <token>",
			})
			return
		}

		tokenValue := parts[1]

		// Lookup token
		token, err := tokenStore.Get(tokenValue)
		if err != nil || token == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Token not found or revoked",
			})
			return
		}

		// Check if revoked
		if token.RevokedAt != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Token has been revoked",
			})
			return
		}

		// Store token info in context for handlers
		c.Set("gnap_token", token)
		c.Set("gnap_grant_id", token.GrantID)

		c.Next()
	}
}

// TokenStoreInterface defines what middleware needs from token store.
type TokenStoreInterface interface {
	Get(value string) (*IssuedToken, error)
}

// IssuedToken represents a GNAP access token (copied for interface).
type IssuedToken struct {
	Value     string
	GrantID   string
	RevokedAt *string
}
