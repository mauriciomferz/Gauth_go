// Package gnap provides HTTP handlers for RFC 9635 GNAP endpoints.
package gnap

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/gauthplus"
	"github.com/mauriciomferz/Gauth_go/pkg/gnap"
)

// AuditLogger interface for logging audit events.
type AuditLogger interface {
	Log(ctx context.Context, entry interface{}) error
}

// Handler manages GNAP grant operations.
type Handler struct {
	Store       gnap.GrantStore
	TokenStore  gnap.TokenStore          // Token lifecycle management
	RSStore     gnap.ResourceServerStore // RS lifecycle Management (RFC 9767)
	AuditLogger AuditLogger              // Optional audit logging
	Verif       gauthplus.VerificationService
	BaseURL     string // Base URL for continuation URIs
	DefaultWait int    // Default wait seconds for polling
}

// NewHandler creates a GNAP handler.
func NewHandler(store gnap.GrantStore, verif gauthplus.VerificationService, baseURL string) *Handler {
	return &Handler{
		Store:       store,
		Verif:       verif,
		BaseURL:     strings.TrimSuffix(baseURL, "/"),
		DefaultWait: 30,
	}
}

// logAudit logs a GNAP audit event if logger is configured.
func (h *Handler) logAudit(c *gin.Context, eventType audit.EventType, action, result, grantID string, metadata map[string]interface{}) {
	if h.AuditLogger == nil {
		return
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["grant_id"] = grantID

	event := &audit.Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Action:    action,
		Result:    result,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Metadata:  metadata,
		Severity:  "info",
	}
	_ = h.AuditLogger.Log(c.Request.Context(), event)
}

// RegisterRoutes adds GNAP endpoints to the router.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Discovery endpoint (RFC 9635 §9)
	r.GET("/.well-known/gnap-as-rs", h.Discovery)

	// Grant transaction endpoint (RFC 9635 §3)
	r.POST("/gnap/tx", h.GrantRequest)

	// Continuation endpoint (RFC 9635 §5)
	r.POST("/gnap/continue/:id", h.Continue)
	r.PATCH("/gnap/continue/:id", h.ContinueUpdate)
	r.DELETE("/gnap/continue/:id", h.ContinueCancel)

	// Token management (RFC 9635 §6)
	r.POST("/gnap/token/:id", h.TokenRotate)
	r.DELETE("/gnap/token/:id", h.TokenRevoke)

	// Resource Server Connection (RFC 9767)
	r.POST("/gnap/rs/register", h.RegisterRS)
	r.POST("/gnap/rs/introspect", h.IntrospectRS)
}

// DiscoveryResponse contains AS metadata per RFC 9635 §9.
type DiscoveryResponse struct {
	GrantRequestEndpoint string   `json:"grant_request_endpoint"`
	DeviceAuthEndpoint   string   `json:"device_authorization_endpoint,omitempty"`
	InteractionStart     []string `json:"interaction_start_modes_supported,omitempty"`
	InteractionFinish    []string `json:"interaction_finish_modes_supported,omitempty"`
	KeyProofs            []string `json:"key_proofs_supported,omitempty"`
	SubjectFormats       []string `json:"subject_formats_supported,omitempty"`
	Assertions           []string `json:"assertions_supported,omitempty"`
}

// Discovery handles GET /.well-known/gnap-as-rs (§9).
func (h *Handler) Discovery(c *gin.Context) {
	c.JSON(http.StatusOK, DiscoveryResponse{
		GrantRequestEndpoint: h.BaseURL + "/gnap/tx",
		DeviceAuthEndpoint:   h.BaseURL + "/device/authorize",
		InteractionStart:     []string{"redirect", "user_code", "user_code_uri"},
		InteractionFinish:    []string{"redirect", "push"},
		KeyProofs:            []string{"httpsig"},
		SubjectFormats:       []string{"opaque", "email", "iss_sub"},
		Assertions:           []string{"id_token", "saml2"},
	})
}

// GrantRequest handles POST /gnap/tx (§3).
func (h *Handler) GrantRequest(c *gin.Context) {
	var req gnap.GrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gnap.GrantResponse{
			Error: &gnap.GrantError{
				Code:        gnap.ErrorInvalidRequest,
				Description: "invalid JSON payload",
			},
		})
		return
	}

	// Validate required fields
	if req.AccessToken == nil && req.Subject == nil {
		c.JSON(http.StatusBadRequest, gnap.GrantResponse{
			Error: &gnap.GrantError{
				Code:        gnap.ErrorInvalidRequest,
				Description: "access_token or subject required",
			},
		})
		return
	}

	// Validate PoACredentialRef if provided (GAuth Extension)
	if req.PoACredentialRef != "" && h.Verif != nil {
		poaVerif, err := h.Verif.VerifyPowerOfAttorney(c.Request.Context(), req.PoACredentialRef)
		if err != nil || !poaVerif.Valid {
			c.JSON(http.StatusBadRequest, gnap.GrantResponse{
				Error: &gnap.GrantError{
					Code:        gnap.ErrorInvalidRequest,
					Description: "invalid power_of_attorney_ref",
				},
			})
			return
		}
	}

	// Create grant
	grant, err := h.Store.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gnap.GrantResponse{
			Error: &gnap.GrantError{
				Code:        gnap.ErrorInvalidRequest,
				Description: err.Error(),
			},
		})
		return
	}

	// Build response based on request
	resp := h.buildGrantResponse(c.Request.Context(), grant, &req)

	// Determine HTTP status
	status := http.StatusOK
	if grant.State == gnap.GrantStatePending {
		status = http.StatusOK // 200 with interact
	}

	c.JSON(status, resp)
}

// Continue handles POST /gnap/continue/:id (§5.1).
func (h *Handler) Continue(c *gin.Context) {
	grantID := c.Param("id")

	// Verify authorization header (continuation token)
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "GNAP ") {
		c.JSON(http.StatusUnauthorized, gnap.GrantResponse{
			Error: &gnap.GrantError{Code: gnap.ErrorInvalidRequest, Description: "missing GNAP token"},
		})
		return
	}
	token := strings.TrimPrefix(authHeader, "GNAP ")

	grant, err := h.Store.Get(grantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gnap.GrantResponse{
			Error: &gnap.GrantError{Code: gnap.ErrorUnknownRequest},
		})
		return
	}

	// Verify token
	if grant.ContinueToken != token {
		c.JSON(http.StatusUnauthorized, gnap.GrantResponse{
			Error: &gnap.GrantError{Code: gnap.ErrorInvalidRequest, Description: "invalid token"},
		})
		return
	}

	// Check if can continue
	if !grant.CanContinue() {
		c.JSON(http.StatusBadRequest, gnap.GrantResponse{
			Error: &gnap.GrantError{Code: gnap.ErrorInvalidRequest, Description: "grant cannot be continued"},
		})
		return
	}

	// Parse continuation request body
	var contReq struct {
		InteractRef string `json:"interact_ref"`
	}
	_ = c.ShouldBindJSON(&contReq)

	// If interaction reference provided, validate and approve
	if contReq.InteractRef != "" {
		if grant.InteractRef != "" && grant.InteractRef == contReq.InteractRef {
			// Interaction completed successfully - approve grant
			if err := grant.Transition(gnap.GrantStateApproved); err == nil {
				_ = h.Store.Update(grant)
			}
		}
	}

	// Build response
	resp := h.buildContinueResponse(grant)
	c.JSON(http.StatusOK, resp)
}

// ContinueUpdate handles PATCH /gnap/continue/:id (§5.3).
func (h *Handler) ContinueUpdate(c *gin.Context) {
	// Similar to Continue but allows modifying the grant request
	// For now, delegate to Continue
	h.Continue(c)
}

// ContinueCancel handles DELETE /gnap/continue/:id (§5.4).
func (h *Handler) ContinueCancel(c *gin.Context) {
	grantID := c.Param("id")

	grant, err := h.Store.Get(grantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gnap.GrantResponse{
			Error: &gnap.GrantError{Code: gnap.ErrorUnknownRequest},
		})
		return
	}

	// Transition to denied
	_ = grant.Transition(gnap.GrantStateDenied)
	_ = h.Store.Update(grant)

	c.Status(http.StatusNoContent)
}

// TokenRotate handles POST /gnap/token/:id (§6.1).
func (h *Handler) TokenRotate(c *gin.Context) {
	tokenValue := c.Param("id")

	if h.TokenStore == nil {
		c.JSON(http.StatusNotImplemented, gnap.GrantResponse{
			Error: &gnap.GrantError{Code: gnap.ErrorInvalidRequest, Description: "token store not configured"},
		})
		return
	}

	newToken, err := h.TokenStore.Rotate(tokenValue)
	if err != nil {
		c.JSON(http.StatusBadRequest, gnap.GrantResponse{
			Error: &gnap.GrantError{Code: gnap.ErrorInvalidRequest, Description: err.Error()},
		})
		return
	}

	// Return new token info
	c.JSON(http.StatusOK, gnap.GrantResponse{
		AccessToken: &gnap.AccessToken{
			Value:     newToken.Value,
			Access:    newToken.Access,
			ExpiresIn: int64(time.Until(newToken.ExpiresAt).Seconds()),
			IssuedAt:  newToken.IssuedAt,
			Flags:     newToken.Flags,
			Manage: &gnap.TokenManagement{
				URI: h.BaseURL + "/gnap/token/" + newToken.Value,
			},
		},
	})
}

// TokenRevoke handles DELETE /gnap/token/:id (§6.2).
func (h *Handler) TokenRevoke(c *gin.Context) {
	tokenValue := c.Param("id")

	if h.TokenStore == nil {
		c.Status(http.StatusNoContent)
		return
	}

	_ = h.TokenStore.Revoke(tokenValue)
	c.Status(http.StatusNoContent)
}

// buildGrantResponse constructs response for initial grant request.
func (h *Handler) buildGrantResponse(ctx context.Context, grant *gnap.Grant, req *gnap.GrantRequest) *gnap.GrantResponse {
	resp := &gnap.GrantResponse{}

	// Always include continuation for non-terminal states
	if !grant.IsTerminal() {
		resp.Continue = &gnap.ContinuationInfo{
			URI:         h.BaseURL + "/gnap/continue/" + grant.ID,
			AccessToken: &gnap.ContinuationToken{Value: grant.ContinueToken},
			Wait:        h.DefaultWait,
		}
	}

	// Check if interaction is needed
	if req.Interact != nil {
		// Generate interaction response
		grant.InteractNonce = gnap.GenerateID("int_")
		grant.InteractRef = gnap.GenerateID("ref_")
		_ = grant.Transition(gnap.GrantStatePending)
		_ = h.Store.Update(grant)

		resp.Interact = &gnap.InteractionResponse{
			Finish:    grant.InteractNonce,
			ExpiresIn: 300, // 5 minutes
		}

		// Add interaction start modes
		for _, mode := range req.Interact.Start {
			switch mode {
			case gnap.InteractionStartRedirect:
				resp.Interact.Redirect = h.BaseURL + "/gnap/interact/" + grant.ID
			case gnap.InteractionStartUserCode:
				resp.Interact.UserCode = generateShortCode()
			}
		}
	} else {
		// No interaction needed - immediate approval
		_ = grant.Transition(gnap.GrantStateApproved)
		_ = h.Store.Update(grant)

		// Issue access token
		resp.AccessToken = h.issueToken(grant, req)
	}

	// GAuth extension: Populate PoA info and AuthorizationChain
	h.linkGAuthContext(ctx, resp, req)

	// Assign client instance ID if not present
	if req.Client != nil && req.Client.InstanceID == "" {
		resp.InstanceID = gnap.GenerateID("cli_")
	}

	return resp
}

// buildContinueResponse constructs response for continuation.
func (h *Handler) buildContinueResponse(grant *gnap.Grant) *gnap.GrantResponse {
	resp := &gnap.GrantResponse{}

	// If approved, issue token
	if grant.State == gnap.GrantStateApproved {
		resp.AccessToken = h.issueToken(grant, grant.Request)

		// Transition to finalized
		_ = grant.Transition(gnap.GrantStateFinalized)
		_ = h.Store.Update(grant)
	} else {
		// Still pending - include continue info
		resp.Continue = &gnap.ContinuationInfo{
			URI:         h.BaseURL + "/gnap/continue/" + grant.ID,
			AccessToken: &gnap.ContinuationToken{Value: grant.ContinueToken},
			Wait:        h.DefaultWait,
		}
	}

	return resp
}

// issueToken creates an access token for an approved grant.
func (h *Handler) issueToken(grant *gnap.Grant, req *gnap.GrantRequest) *gnap.AccessToken {
	token := &gnap.AccessToken{
		Value:     gnap.GenerateID("gauth_gnap_"),
		ExpiresIn: 3600, // 1 hour default
		IssuedAt:  time.Now().UTC(),
	}

	// Copy requested access
	if req.AccessToken != nil {
		token.Access = req.AccessToken.Access
		token.Label = req.AccessToken.Label
		token.Flags = req.AccessToken.Flags
	}

	// Add token management
	token.Manage = &gnap.TokenManagement{
		URI: h.BaseURL + "/gnap/token/" + grant.ID,
	}

	// GAuth extension: link PoA if present
	if req.PoACredentialRef != "" {
		token.PoAID = req.PoACredentialRef
	}

	return token
}

// generateShortCode creates a user-friendly code for device flow.
func generateShortCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Avoid ambiguous chars
	code := make([]byte, 8)
	for i := range code {
		code[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond) // Ensure different values
	}
	return string(code[:4]) + "-" + string(code[4:])
}

// linkGAuthContext populates GNAP response with GAuth-specific context.
func (h *Handler) linkGAuthContext(ctx context.Context, resp *gnap.GrantResponse, req *gnap.GrantRequest) {
	if h.Verif == nil || req.PoACredentialRef == "" {
		return
	}

	// Determine representative action for verification report
	var action gauthplus.Action
	if req.AccessToken != nil && len(req.AccessToken.Access) > 0 {
		ar := req.AccessToken.Access[0]
		action.Type = "transaction" // Default
		if ar.Type != "" {
			action.Type = ar.Type
		}
		action.Resource = ar.Identifier
		if len(ar.Actions) > 0 {
			action.Operation = ar.Actions[0]
		}
	} else {
		action.Type = "verification"
		action.Operation = "status_check"
	}

	// Generate comprehensive verification report
	report, err := h.Verif.GenerateVerificationReport(ctx, req.PoACredentialRef, action)
	if err != nil {
		return
	}

	// Populate PoA Ref
	resp.PowerOfAttorney = &gnap.PowerOfAttorneyRef{
		PoAID:   report.PoAID,
		Issuer:  report.PoAVerification.IssuerID,
		Grantee: report.PoAVerification.GranteeID,
		Scope:   report.PoAVerification.Scope,
	}

	// Fetch and transform Authorization Chain with enrichment
	gnapChain := make([]gnap.ChainLink, len(report.ChainOfAuthority))
	for i, link := range report.ChainOfAuthority {
		entityType := "human"
		if !link.IsHuman {
			entityType = "ai_agent"
		}

		gnapChain[i] = gnap.ChainLink{
			Entity:     link.GranteeID,
			EntityType: entityType,
			Authority:  link.IssuerID,
			Verified:   true, // Part of the verified report
		}
	}
	resp.AuthorizationChain = gnapChain

	// Determine compliance level
	compliance := "basic"
	if report.OverallValid {
		compliance = "high"
		if report.FiduciaryCompliance != nil && !report.FiduciaryCompliance.Compliant {
			compliance = "degraded"
		}
		if report.CapabilityCheck != nil && !report.CapabilityCheck.Sufficient {
			compliance = "conditional"
		}
	} else {
		compliance = "non_compliant"
	}
	resp.ComplianceLevel = compliance
}
