package beta

import (
	"context"
	"net/http"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/gin-gonic/gin"
)

// EntityVerifyRequest represents the request for entity verification
type EntityVerifyRequest struct {
	EntityID   string `json:"entity_id" binding:"required"`
	EntityName string `json:"entity_name" binding:"required"`
	EntityType string `json:"entity_type" binding:"required"` // corporation, partnership, llc, sole_proprietorship
	Jurisdiction string `json:"jurisdiction" binding:"required"` // ISO 3166-1 alpha-2 code
}

// SignatoryVerifyRequest represents the request for signatory verification
type SignatoryVerifyRequest struct {
	EntityID string `json:"entity_id" binding:"required"`
	PersonID string `json:"person_id" binding:"required"`
	Role     string `json:"role" binding:"required"` // director, ceo, authorized_signatory, officer
}

// EntityVerifyResponse represents the response from entity verification
type EntityVerifyResponse struct {
	Success  bool          `json:"success"`
	Verified bool          `json:"verified"`
	Entity   *EntityDetails `json:"entity,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// SignatoryVerifyResponse represents the response from signatory verification
type SignatoryVerifyResponse struct {
	Success     bool              `json:"success"`
	Verified    bool              `json:"verified"`
	Signatory   *SignatoryDetails `json:"signatory,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// EntityDetails contains verified entity information
type EntityDetails struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"entity_type"`
	Jurisdiction string    `json:"jurisdiction"`
	Status       string    `json:"status"`
	RegisteredAt time.Time `json:"registered_at"`
	Authority    string    `json:"authority"`
}

// SignatoryDetails contains verified signatory information
type SignatoryDetails struct {
	EntityID    string    `json:"entity_id"`
	PersonID    string    `json:"person_id"`
	Role        string    `json:"role"`
	Authorized  bool      `json:"authorized"`
	ValidFrom   time.Time `json:"valid_from"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	Authority   string    `json:"authority"`
}

// RegistryHandler wraps a CommercialRegisterClient for HTTP exposure
type RegistryHandler struct {
	registryClient gauth.CommercialRegisterClient
}

// NewRegistryHandler creates a new Commercial Registry HTTP handler
func NewRegistryHandler(registryClient gauth.CommercialRegisterClient) *RegistryHandler {
	return &RegistryHandler{
		registryClient: registryClient,
	}
}

// HandleVerifyEntity processes entity verification requests
//
// POST /api/v1/beta/registry/verify-entity
//
// Request Body:
//   {
//     "entity_id": "12345678",
//     "entity_name": "Acme Corporation",
//     "entity_type": "corporation",
//     "jurisdiction": "US"
//   }
//
// Success Response (200 OK):
//   {
//     "success": true,
//     "verified": true,
//     "entity": {
//       "id": "12345678",
//       "name": "Acme Corporation",
//       "entity_type": "corporation",
//       "jurisdiction": "US",
//       "status": "active",
//       "registered_at": "2020-01-15T00:00:00Z",
//       "authority": "Commercial Register"
//     }
//   }
func (h *RegistryHandler) HandleVerifyEntity(c *gin.Context) {
	var req EntityVerifyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, EntityVerifyResponse{
			Success:  false,
			Verified: false,
			Error:    "invalid payload: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Call the Commercial Registry client
	// Note: Arguments are (jurisdiction, companyID) not (companyID, jurisdiction)
	companyInfo, err := h.registryClient.VerifyCompany(ctx, req.Jurisdiction, req.EntityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, EntityVerifyResponse{
			Success:  false,
			Verified: false,
			Error:    "registry service error: " + err.Error(),
		})
		return
	}

	// Check if entity is active
	if !companyInfo.Active {
		c.JSON(http.StatusOK, EntityVerifyResponse{
			Success:  true,
			Verified: false,
			Error:    "entity is not active",
		})
		return
	}

	// Success: return verified entity details
	response := EntityVerifyResponse{
		Success:  true,
		Verified: true,
		Entity: &EntityDetails{
			ID:           companyInfo.CompanyID,
			Name:         companyInfo.LegalName,
			Type:         companyInfo.LegalForm,
			Jurisdiction: companyInfo.Jurisdiction,
			Status:       companyInfo.Status,
			RegisteredAt: companyInfo.RegistrationDate,
			Authority:    "Commercial Register",
		},
	}

	c.JSON(http.StatusOK, response)
}

// HandleVerifySignatory processes signatory verification requests
//
// POST /api/v1/beta/registry/verify-signatory
//
// Request Body:
//   {
//     "entity_id": "12345678",
//     "person_id": "uuid-person",
//     "role": "ceo"
//   }
//
// Success Response (200 OK):
//   {
//     "success": true,
//     "verified": true,
//     "signatory": {
//       "entity_id": "12345678",
//       "person_id": "uuid-person",
//       "role": "ceo",
//       "authorized": true,
//       "valid_from": "2023-01-01T00:00:00Z",
//       "authority": "Commercial Register"
//     }
//   }
func (h *RegistryHandler) HandleVerifySignatory(c *gin.Context) {
	var req SignatoryVerifyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, SignatoryVerifyResponse{
			Success:  false,
			Verified: false,
			Error:    "invalid payload: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Call the Commercial Registry client for director verification
	// Note: Arguments are (companyID, personID)
	directorInfo, err := h.registryClient.VerifyManagingDirector(ctx, req.EntityID, req.PersonID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SignatoryVerifyResponse{
			Success:  false,
			Verified: false,
			Error:    "registry service error: " + err.Error(),
		})
		return
	}

	// Check if director is active
	if !directorInfo.Active {
		c.JSON(http.StatusOK, SignatoryVerifyResponse{
			Success:  true,
			Verified: false,
			Error:    "signatory is not active",
		})
		return
	}

	// Success: return verified signatory details
	response := SignatoryVerifyResponse{
		Success:  true,
		Verified: true,
		Signatory: &SignatoryDetails{
			EntityID:   req.EntityID,
			PersonID:   directorInfo.PersonID,
			Role:       directorInfo.Role,
			Authorized: true,
			ValidFrom:  directorInfo.AppointmentDate,
			ValidUntil: nil, // No expiration in mock
			Authority:  "Commercial Register",
		},
	}

	c.JSON(http.StatusOK, response)
}
