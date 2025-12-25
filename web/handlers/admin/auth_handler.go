package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler handles admin authentication
type AuthHandler struct {
	jwtSecret string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtSecret string) *AuthHandler {
	return &AuthHandler{
		jwtSecret: jwtSecret,
	}
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response with session info
type LoginResponse struct {
	Success          bool   `json:"success"`
	RequiresMFA      bool   `json:"requiresMFA"`
	SessionChallenge string `json:"sessionChallenge,omitempty"`
	Token            string `json:"token,omitempty"`
	Error            string `json:"error,omitempty"`
}

// MFARequest represents MFA verification request
type MFARequest struct {
	ChallengeID string `json:"challengeId" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Method      string `json:"method" binding:"required"`
}

// MFAResponse represents successful MFA response with JWT
type MFAResponse struct {
	Success   bool   `json:"success"`
	Token     string `json:"token,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Login authenticates admin credentials (step 1 of 2)
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// For development, accept test credentials
	if req.Email == "admin@example.com" && req.Password == "password" {
		c.JSON(http.StatusOK, LoginResponse{
			Success:          true,
			RequiresMFA:      true,
			SessionChallenge: "challenge-" + req.Email,
		})
		return
	}

	c.JSON(http.StatusUnauthorized, LoginResponse{
		Success: false,
		Error:   "Invalid email or password",
	})
}

// VerifyMFA verifies MFA code (step 2 of 2)
func (h *AuthHandler) VerifyMFA(c *gin.Context) {
	var req MFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MFAResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// For development, accept test code
	if req.Code == "123456" {
		// Generate JWT token
		token, expiresAt, err := h.generateToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, MFAResponse{
				Success: false,
				Error:   "Failed to generate token",
			})
			return
		}

		c.JSON(http.StatusOK, MFAResponse{
			Success:   true,
			Token:     token,
			ExpiresAt: expiresAt,
		})
		return
	}

	c.JSON(http.StatusUnauthorized, MFAResponse{
		Success: false,
		Error:   "Invalid MFA code",
	})
}

// generateToken creates a JWT token
func (h *AuthHandler) generateToken() (string, string, error) {
	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id": 1,
		"email":   "admin@example.com",
		"role":    "admin",
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return "", "", err
	}

	return tokenString, expiresAt.Format(time.RFC3339), nil
}

// RegisterRoutes registers admin auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/login", h.Login)
	router.POST("/auth/verify-mfa", h.VerifyMFA)
}
