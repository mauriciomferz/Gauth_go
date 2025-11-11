// Package web - RFC-0111 Disclosure API Routes
package web

import (
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/handlers/disclosure"
)

// RegisterDisclosureRoutes registers RFC-0111 disclosure/transparency endpoints
func (s *BetaServer) RegisterDisclosureRoutes(disclosureService *gauth.DisclosureService) {
	handler := disclosure.NewHandler(disclosureService)

	// Public disclosure endpoints for transparency
	api := s.router.Group("/api/v1/disclosure")
	{
		// List active authorizations for a resource owner
		// GET /api/v1/disclosure/authorizations?resource_owner_id=xxx
		api.GET("/authorizations", handler.ListActiveAuthorizationsHandler)

		// Get detailed information about a specific authorization
		// GET /api/v1/disclosure/authorizations/:id?resource_owner_id=xxx
		api.GET("/authorizations/:id", handler.GetAuthorizationDetailHandler)

		// Revoke an authorization
		// POST /api/v1/disclosure/authorizations/:id/revoke
		api.POST("/authorizations/:id/revoke", handler.RevokeAuthorizationHandler)

		// Get audit trail for an authorization
		// GET /api/v1/disclosure/authorizations/:id/audit?resource_owner_id=xxx
		api.GET("/authorizations/:id/audit", handler.GetAuditTrailHandler)
	}
}
