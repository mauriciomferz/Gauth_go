package web

import (
	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
	"github.com/mauriciomferz/AgentAuth/web/handlers/beta"
)

// RegisterBetaExternalServiceEndpoints registers HTTP endpoints that expose
// mock external services (PVP, Commercial Registry, PoA) for UI integration.
// This is part of Phase 2A Enhancement to convert UI mocks to real backend endpoints.
func (s *BetaServer) RegisterBetaExternalServiceEndpoints(components *gauth.AAP001Components) {
	// Create handlers wrapping the RFC-0111 mock clients
	pvpHandler := beta.NewPVPHandler(components.PVPClient)
	registryHandler := beta.NewRegistryHandler(components.CommercialRegClient)
	poaHandler := beta.NewPoAHandler()

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

	// Register Power of Attorney CRUD endpoints under /api/v1/beta/poa
	poaGroup := s.router.Group("/api/v1/beta/poa")
	{
		poaGroup.POST("", poaHandler.HandleCreate)
		poaGroup.GET("/:id", poaHandler.HandleGet)
		poaGroup.GET("", poaHandler.HandleList)
		poaGroup.PUT("/:id", poaHandler.HandleUpdate)
		poaGroup.DELETE("/:id", poaHandler.HandleDelete)
		poaGroup.POST("/:id/validate", poaHandler.HandleValidate)
	}
}
