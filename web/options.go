package web

import (
	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// BetaServerOption allows configuring BetaServer.
type BetaServerOption func(*BetaServer)

// WithKeyProvider sets the key provider for the server.
func WithKeyProvider(kp crypto.KeyProvider) BetaServerOption {
	return func(s *BetaServer) {
		s.keyProvider = kp
	}
}
