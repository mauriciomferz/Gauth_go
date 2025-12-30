package token

import "github.com/mauriciomferz/Gauth_go/pkg/gauth"

// SetAgentAuthService sets the AgentAuth service for RFC compliant operations
func (h *Handler) SetAgentAuthService(service gauth.AgentAuth) {
	h.AgentAuthService = service
}
