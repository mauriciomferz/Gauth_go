// Package rfc0111 provides HTTP handlers for RFC-0111 subscription and authorization flows.
//
// NOTE: These handlers are basic stubs that demonstrate the REST API structure.
// Full implementation requires:
// 1. Proper request/response mapping to SubscriptionFlowManager methods
// 2. Mock implementations of external services (PVP, PIP, Commercial Register)
// 3. Error handling and validation
// 4. Authentication and authorization middleware
//
// The subscription flow follows RFC-0111 Steps I-VIII, and authorization follows Steps a-i.
package gauth_rfc_001

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/poa"
	"github.com/mauriciomferz/Gauth_go/pkg/poa/taxonomy"
)

// SubscriptionHandlers encapsulates RFC-0111 subscription API handlers.
type SubscriptionHandlers struct {
	subscriptionManager *gauth.SubscriptionFlowManager
	subscriptionStore   gauth.SubscriptionStore
}

// NewSubscriptionHandlers creates a new subscription handlers instance.
func NewSubscriptionHandlers(manager *gauth.SubscriptionFlowManager, store gauth.SubscriptionStore) *SubscriptionHandlers {
	return &SubscriptionHandlers{
		subscriptionManager: manager,
		subscriptionStore:   store,
	}
}

// CreateSubscription handles POST /api/v1/rfc0111/subscriptions
// RFC-0111 Step I: Owner's Authorizer Identity Proof
func (h *SubscriptionHandlers) CreateSubscription(c *gin.Context) {
	var req struct {
		OwnersAuthorizerID   string `json:"owners_authorizer_id" binding:"required"`
		IdentityProofRequest struct {
			SubjectID     string                 `json:"subject_id" binding:"required"`
			IdentityType  string                 `json:"identity_type" binding:"required"`
			ProofMethod   string                 `json:"proof_method" binding:"required"`
			ProofData     map[string]interface{} `json:"proof_data" binding:"required"`
			RequiredLevel string                 `json:"required_level" binding:"required"`
		} `json:"identity_proof_request" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Step I.a: Initiate subscription
	subscription, err := h.subscriptionManager.InitiateSubscription(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "subscription_failed",
			"message": err.Error(),
		})
		return
	}

	// Step I.b: Execute identity proof
	identityProofRequest := &gauth.IdentityProofRequest{
		SubjectID:     req.IdentityProofRequest.SubjectID,
		IdentityType:  req.IdentityProofRequest.IdentityType,
		ProofMethod:   req.IdentityProofRequest.ProofMethod,
		ProofData:     req.IdentityProofRequest.ProofData,
		RequiredLevel: req.IdentityProofRequest.RequiredLevel,
	}

	err = h.subscriptionManager.ExecuteStepI(
		c.Request.Context(),
		subscription.ID,
		identityProofRequest,
	)
	if err != nil {
		if gauthErr, ok := err.(*gauth.GAuthError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   gauthErr.Code,
				"message": gauthErr.Message,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "step_i_failed",
				"message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"subscription_id": subscription.ID,
		"status":          "pending",
		"created_at":      subscription.CreatedAt,
		"message":         "Step I completed - Owner's authorizer identity verified",
	})
}

// GetSubscription handles GET /api/v1/rfc0111/subscriptions/:id
func (h *SubscriptionHandlers) GetSubscription(c *gin.Context) {
	subscriptionID := c.Param("id")

	subscription, err := h.subscriptionStore.GetSubscription(c.Request.Context(), subscriptionID)
	if err != nil {
		if err == gauth.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Subscription not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subscription.ID,
		"status":          subscription.Status,
		"created_at":      subscription.CreatedAt,
		"updated_at":      subscription.UpdatedAt,
	})
}

// ListSubscriptions handles GET /api/v1/rfc0111/subscriptions?client_id=xxx
func (h *SubscriptionHandlers) ListSubscriptions(c *gin.Context) {
	clientID := c.Query("client_id")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "client_id query parameter required",
		})
		return
	}

	subscriptions, err := h.subscriptionStore.ListSubscriptions(c.Request.Context(), clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": err.Error(),
		})
		return
	}

	result := make([]gin.H, len(subscriptions))
	for i, sub := range subscriptions {
		result[i] = gin.H{
			"subscription_id": sub.ID,
			"status":          sub.Status,
			"created_at":      sub.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": result,
		"count":         len(result),
	})
}

// ExecuteStepII handles POST /api/v1/rfc0111/subscriptions/:id/step-ii
// RFC-0111 Step II: Owner's Authorizer Authorization Proof
func (h *SubscriptionHandlers) ExecuteStepII(c *gin.Context) {
	subscriptionID := c.Param("id")

	var req struct {
		CommercialRegisterRef string `json:"commercial_register_ref" binding:"required"`
		Jurisdiction          string `json:"jurisdiction" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	err := h.subscriptionManager.ExecuteStepII(
		c.Request.Context(),
		subscriptionID,
		req.CommercialRegisterRef,
		req.Jurisdiction,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if gauthErr, ok := err.(*gauth.GAuthError); ok {
			if gauthErr.Code == "step_ii_prerequisite_failed" {
				code = http.StatusBadRequest
			}
		}
		c.JSON(code, gin.H{
			"error":   "step_ii_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subscriptionID,
		"step":            "II",
		"status":          "completed",
		"message":         "Owner's authorizer authorization verified - proceed to Step III",
	})
}

// ExecuteStepIII handles POST /api/v1/rfc0111/subscriptions/:id/step-iii
// RFC-0111 Step III: Client Owner Identity Proof
func (h *SubscriptionHandlers) ExecuteStepIII(c *gin.Context) {
	subscriptionID := c.Param("id")

	var req struct {
		SubjectID     string                 `json:"subject_id" binding:"required"`
		IdentityType  string                 `json:"identity_type" binding:"required"`
		ProofMethod   string                 `json:"proof_method" binding:"required"`
		ProofData     map[string]interface{} `json:"proof_data" binding:"required"`
		RequiredLevel string                 `json:"required_level" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	identityProofRequest := &gauth.IdentityProofRequest{
		SubjectID:     req.SubjectID,
		IdentityType:  req.IdentityType,
		ProofMethod:   req.ProofMethod,
		ProofData:     req.ProofData,
		RequiredLevel: req.RequiredLevel,
	}

	err := h.subscriptionManager.ExecuteStepIII(
		c.Request.Context(),
		subscriptionID,
		identityProofRequest,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if gauthErr, ok := err.(*gauth.GAuthError); ok {
			code = http.StatusBadRequest
			c.JSON(code, gin.H{
				"error":   gauthErr.Code,
				"message": gauthErr.Message,
			})
		} else {
			c.JSON(code, gin.H{
				"error":   "step_iii_failed",
				"message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"step":    "III",
		"message": "Client owner identity verified",
	})
}

// ExecuteStepIV handles POST /api/v1/rfc0111/subscriptions/:id/step-iv
// RFC-0111 Step IV: Client Owner Authorization Proof
func (h *SubscriptionHandlers) ExecuteStepIV(c *gin.Context) {
	subscriptionID := c.Param("id")

	var req struct {
		AuthorizationChain *gauth.AuthorizationChain `json:"authorization_chain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	err := h.subscriptionManager.ExecuteStepIV(
		c.Request.Context(),
		subscriptionID,
		req.AuthorizationChain,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if gauthErr, ok := err.(*gauth.GAuthError); ok {
			if gauthErr.Code == "step_iv_prerequisite_failed" {
				code = http.StatusBadRequest
			}
		}
		c.JSON(code, gin.H{
			"error":   "step_iv_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subscriptionID,
		"step":            "IV",
		"status":          "completed",
		"message":         "Client owner authorization verified - proceed to Step V",
	})
}

// ExecuteStepV handles POST /api/v1/rfc0111/subscriptions/:id/step-v
// RFC-0111 Step V: Client Authorization
func (h *SubscriptionHandlers) ExecuteStepV(c *gin.Context) {
	subscriptionID := c.Param("id")

	var req struct {
		ClientID              string      `json:"client_id" binding:"required"`
		PoACredential         interface{} `json:"poa_credential" binding:"required"`
		EnableIdentitySharing bool        `json:"enable_identity_sharing"`
		EnablePrompting       bool        `json:"enable_prompting"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Create a minimal mock PoA for demonstration purposes
	// In production, this would parse the full PoADefinition from req.PoACredential
	mockPoA := &poa.PoADefinition{
		Parties: poa.Parties{
			Principal: poa.Principal{
				Type:     "organization",
				Identity: "client-owner-67890",
			},
			AuthorizedClient: poa.AuthorizedClient{
				Type:              "ai_system",
				TypeEnum:          poa.ClientTypeLLM,
				Identity:          req.ClientID,
				OperationalStatus: "active",
				StatusEnum:        poa.OperationalStatusActive,
				CapabilityLevel:   poa.CapabilityL3,
			},
		},
		Authorization: poa.AuthorizationScope{
			AuthorizedActions: poa.AuthorizedActions{
				NonPhysicalActions: []poa.ActionTypeNonPhysical{
					taxonomy.ActionNonPhysicalAnalyzing,
					taxonomy.ActionNonPhysicalDocumenting,
				},
			},
		},
		Requirements: poa.Requirements{
			JurisdictionLaw: poa.JurisdictionLaw{
				Language:            "en",
				GoverningLaw:        "EU-GDPR",
				PlaceOfJurisdiction: "Germany",
				AttachedDocuments:   []string{},
			},
		},
	}

	err := h.subscriptionManager.ExecuteStepV(
		c.Request.Context(),
		subscriptionID,
		req.ClientID,
		mockPoA,
		req.EnableIdentitySharing,
		req.EnablePrompting,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if gauthErr, ok := err.(*gauth.GAuthError); ok {
			if gauthErr.Code == "step_v_prerequisite_failed" {
				code = http.StatusBadRequest
			}
		}
		c.JSON(code, gin.H{
			"error":   "step_v_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subscriptionID,
		"step":            "V",
		"status":          "completed",
		"message":         "Client authorization granted - proceed to Step VI",
	})
}

// ExecuteStepVI handles POST /api/v1/rfc0111/subscriptions/:id/step-vi
// RFC-0111 Step VI: Resource Owner Identity Proof
func (h *SubscriptionHandlers) ExecuteStepVI(c *gin.Context) {
	subscriptionID := c.Param("id")

	var req struct {
		SubjectID     string                 `json:"subject_id" binding:"required"`
		IdentityType  string                 `json:"identity_type" binding:"required"`
		ProofMethod   string                 `json:"proof_method" binding:"required"`
		ProofData     map[string]interface{} `json:"proof_data" binding:"required"`
		RequiredLevel string                 `json:"required_level" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	identityProofRequest := &gauth.IdentityProofRequest{
		SubjectID:     req.SubjectID,
		IdentityType:  req.IdentityType,
		ProofMethod:   req.ProofMethod,
		ProofData:     req.ProofData,
		RequiredLevel: req.RequiredLevel,
	}

	err := h.subscriptionManager.ExecuteStepVI(
		c.Request.Context(),
		subscriptionID,
		identityProofRequest,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if gauthErr, ok := err.(*gauth.GAuthError); ok {
			code = http.StatusBadRequest
			c.JSON(code, gin.H{
				"error":   gauthErr.Code,
				"message": gauthErr.Message,
			})
		} else {
			c.JSON(code, gin.H{
				"error":   "step_vi_failed",
				"message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"step":    "VI",
		"message": "Resource owner identity verified",
	})
}

// ExecuteStepVII handles POST /api/v1/rfc0111/subscriptions/:id/step-vii
// RFC-0111 Step VII: Resource Owner Authorization Proof
func (h *SubscriptionHandlers) ExecuteStepVII(c *gin.Context) {
	subscriptionID := c.Param("id")

	var req struct {
		AuthorizationChain *gauth.AuthorizationChain `json:"authorization_chain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	err := h.subscriptionManager.ExecuteStepVII(
		c.Request.Context(),
		subscriptionID,
		req.AuthorizationChain,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if gauthErr, ok := err.(*gauth.GAuthError); ok {
			if gauthErr.Code == "step_vii_prerequisite_failed" {
				code = http.StatusBadRequest
			}
		}
		c.JSON(code, gin.H{
			"error":   "step_vii_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subscriptionID,
		"step":            "VII",
		"status":          "completed",
		"message":         "Resource owner authorization verified - proceed to Step VIII",
	})
}

// ExecuteStepVIII handles POST /api/v1/rfc0111/subscriptions/:id/step-viii
// RFC-0111 Step VIII: Resource Server Authorization
func (h *SubscriptionHandlers) ExecuteStepVIII(c *gin.Context) {
	subscriptionID := c.Param("id")

	var req struct {
		ResourceServerID  string   `json:"resource_server_id" binding:"required"`
		ServerEndpoint    string   `json:"server_endpoint" binding:"required"`
		ResourceTypes     []string `json:"resource_types" binding:"required"`
		AllowedOperations []string `json:"allowed_operations" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	err := h.subscriptionManager.ExecuteStepVIII(
		c.Request.Context(),
		subscriptionID,
		req.ResourceServerID,
		req.ServerEndpoint,
		req.ResourceTypes,
		req.AllowedOperations,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if gauthErr, ok := err.(*gauth.GAuthError); ok {
			if gauthErr.Code == "step_viii_prerequisite_failed" {
				code = http.StatusBadRequest
			}
		}
		c.JSON(code, gin.H{
			"error":   "step_viii_failed",
			"message": err.Error(),
		})
		return
	}

	// Get the completed subscription to generate token
	subscription, err := h.subscriptionManager.GetSubscriptionStatus(c.Request.Context(), subscriptionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_generation_failed",
			"message": "Failed to retrieve subscription details",
		})
		return
	}

	// Generate extended token JWT
	token, err := generateExtendedTokenFromSubscription(subscription)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_generation_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subscriptionID,
		"step":            "VIII",
		"status":          "completed",
		"message":         "Subscription flow complete - all steps verified",
		"token":           token,
		"access_token":    token,
	})
}

// Helper function to generate extended token from completed subscription
func generateExtendedTokenFromSubscription(sub *gauth.Subscription) (string, error) {
	now := time.Now()
	exp := now.Add(24 * time.Hour) // Token valid for 24 hours

	// Extract client and resource information
	var clientID, clientOwnerID, resourceOwnerID, resourceServerID string
	if sub.ClientAuthorizationGrant != nil {
		clientID = sub.ClientAuthorizationGrant.ClientID
		clientOwnerID = sub.ClientAuthorizationGrant.ClientOwnerID
	}
	if sub.ResourceOwnerIdentity != nil {
		resourceOwnerID = sub.ResourceOwnerIdentity.SubjectID
	}
	if sub.ResourceServerAuth != nil {
		resourceServerID = sub.ResourceServerAuth.ServerID
	}

	// Get issuer from environment (matches gauth service configuration)
	issuer := os.Getenv("GAUTH_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:8080" // Default for dev
	}

	// Build RFC-0111 extended token claims
	claims := jwt.MapClaims{
		"iss":             issuer,
		"sub":             clientOwnerID,
		"aud":             []string{resourceServerID},
		"exp":             exp.Unix(),
		"iat":             now.Unix(),
		"jti":             sub.ID,
		"subscription_id": sub.ID,
		"client_id":       clientID,
		"client_owner":    clientOwnerID,
		"resource_owner":  resourceOwnerID,
		"resource_server": resourceServerID,
		"scope":           "extended_authorization",
		"poa_credential":  sub.ClientAuthorizationGrant != nil && sub.ClientAuthorizationGrant.PoACredential != nil,
		"token_type":      "extended",
	}

	// Use the same signing key as the rest of the application
	// This ensures tokens can be validated consistently
	signingKey := getJWTSigningKey()

	// Create and sign JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// getJWTSigningKey returns the JWT signing key from environment or default
func getJWTSigningKey() []byte {
	secret := os.Getenv("GAUTH_JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-demo-00000000000000000000000000000000"
	}
	return []byte(secret)
}
