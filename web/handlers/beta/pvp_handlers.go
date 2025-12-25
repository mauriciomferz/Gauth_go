// Package beta provides HTTP handlers for Beta API endpoints.
// These handlers expose mock external services (PVP, Registry, PoA) as HTTP endpoints
// for UI integration during Phase 2A Enhancement.
package beta

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

// PVPVerifyRequest represents the request payload for PVP identity verification
type PVPVerifyRequest struct {
	DocumentType   string `json:"document_type" binding:"required"`   // passport, national_id, driver_license
	DocumentNumber string `json:"document_number" binding:"required"` // Document ID number
	FirstName      string `json:"first_name" binding:"required"`
	LastName       string `json:"last_name" binding:"required"`
	DateOfBirth    string `json:"date_of_birth" binding:"required"` // YYYY-MM-DD format
	Country        string `json:"country" binding:"required"`       // ISO 3166-1 alpha-2 code
}

// PVPVerifyResponse represents the response from PVP identity verification
type PVPVerifyResponse struct {
	Success  bool                 `json:"success"`
	Verified bool                 `json:"verified"`
	Person   *PersonDetails       `json:"person,omitempty"`
	Details  *VerificationDetails `json:"verification_details,omitempty"`
	Error    string               `json:"error,omitempty"`
}

// PersonDetails contains verified person information
type PersonDetails struct {
	ID          string `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	Nationality string `json:"nationality"`
}

// VerificationDetails contains metadata about the verification
type VerificationDetails struct {
	Method     string    `json:"method"`
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"`
	TrustLevel string    `json:"trust_level"`
}

// PVPHandler wraps a PowerVerificationPoint for HTTP exposure
type PVPHandler struct {
	pvpClient gauth.PowerVerificationPoint
}

// NewPVPHandler creates a new PVP HTTP handler
func NewPVPHandler(pvpClient gauth.PowerVerificationPoint) *PVPHandler {
	return &PVPHandler{
		pvpClient: pvpClient,
	}
}

// HandleVerify processes PVP identity verification requests
//
// POST /api/v1/beta/pvp/verify
//
// Request Body:
//
//	{
//	  "document_type": "passport",
//	  "document_number": "AB123456",
//	  "first_name": "John",
//	  "last_name": "Doe",
//	  "date_of_birth": "1990-01-15",
//	  "country": "US"
//	}
//
// Success Response (200 OK):
//
//	{
//	  "success": true,
//	  "verified": true,
//	  "person": {
//	    "id": "uuid-here",
//	    "first_name": "John",
//	    "last_name": "Doe",
//	    "date_of_birth": "1990-01-15",
//	    "nationality": "US"
//	  },
//	  "verification_details": {
//	    "method": "PVP",
//	    "timestamp": "2025-11-15T19:00:00Z",
//	    "confidence": 0.95,
//	    "trust_level": "high"
//	  }
//	}
//
// Error Response (400 Bad Request / 500 Internal Server Error):
//
//	{
//	  "success": false,
//	  "verified": false,
//	  "error": "error message"
//	}
func (h *PVPHandler) HandleVerify(c *gin.Context) {
	var req PVPVerifyRequest

	// Bind and validate JSON payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PVPVerifyResponse{
			Success:  false,
			Verified: false,
			Error:    "invalid payload: " + err.Error(),
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Convert HTTP request to PVP IdentityProofRequest
	// Note: Using a simplified subject ID based on document number
	// In production, this would be more sophisticated
	subjectID := req.DocumentNumber

	pvpRequest := &gauth.IdentityProofRequest{
		SubjectID:     subjectID,
		IdentityType:  "natural_person",
		ProofMethod:   "government_id",
		RequiredLevel: "high",
		ProofData: map[string]interface{}{
			"document_type":   req.DocumentType,
			"document_number": req.DocumentNumber,
			"first_name":      req.FirstName,
			"last_name":       req.LastName,
			"date_of_birth":   req.DateOfBirth,
			"country":         req.Country,
		},
	}

	// Call the PVP client
	result, err := h.pvpClient.VerifyIdentityProof(ctx, pvpRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, PVPVerifyResponse{
			Success:  false,
			Verified: false,
			Error:    "verification service error: " + err.Error(),
		})
		return
	}

	// Handle verification failure
	if !result.Valid {
		c.JSON(http.StatusOK, PVPVerifyResponse{
			Success:  true,
			Verified: false,
			Error:    result.FailureReason,
			Details: &VerificationDetails{
				Method:    "PVP",
				Timestamp: result.VerifiedAt,
			},
		})
		return
	}

	// Success: return verified person details
	response := PVPVerifyResponse{
		Success:  true,
		Verified: true,
		Person: &PersonDetails{
			ID:          result.SubjectID,
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			DateOfBirth: req.DateOfBirth,
			Nationality: req.Country,
		},
		Details: &VerificationDetails{
			Method:     "PVP",
			Timestamp:  result.VerifiedAt,
			Confidence: 0.95, // Mock confidence score
			TrustLevel: result.TrustLevel,
		},
	}

	c.JSON(http.StatusOK, response)
}
