package web

import (
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/handlers/beta"
)

// RegisterBetaExternalServiceEndpoints registers HTTP endpoints that expose
// mock external services (PVP, Commercial Registry) for UI integration.
// This is part of Phase 2A Enhancement to convert UI mocks to real backend endpoints.
func (s *BetaServer) RegisterBetaExternalServiceEndpoints(components *gauth.RFC0111Components) {
	// Create handlers wrapping the RFC-0111 mock clients
	pvpHandler := beta.NewPVPHandler(components.PVPClient)
	registryHandler := beta.NewRegistryHandler(components.CommercialRegClient)

	// Register PVP endpoints under /api/v1/beta/pvp
	pvpGroup := s.router.Group("/api/v1/beta/pvp")
	{
		pvpGroup.POST("/verify", pvpHandler.HandleVerify)
	}

	// Register Commercial Registry endpoints under /api/v1/beta/registry
	registryGroup := s.router.Group("/api/v1/beta/registry")
	{
		registryGroup.POST("/verify-entity", registryHandler.HandleVerifyEntity)
		registryGroup.POST("/verify-signatory", registryHandler.HandleVerifySignatory)
	}
}
