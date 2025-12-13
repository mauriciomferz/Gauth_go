package token

import "github.com/mauriciomferz/Gauth_go/pkg/gauth"

// SetGAuthService sets the GAuth service for RFC compliant operations
func (h *Handler) SetGAuthService(service gauth.GAuth) {
	h.GAuthService = service
}
