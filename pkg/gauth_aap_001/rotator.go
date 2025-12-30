package gauth_aap_001

import (
	"context"

	"github.com/mauriciomferz/Gauth_go/pkg/crypto/keyring"
)

// keyRingRotator adapts a KeyRing to the scheduler.Rotator interface.
type keyRingRotator struct {
	kr *keyring.KeyRing
}

func (r *keyRingRotator) Rotate(ctx context.Context) error {
	r.kr.Rotate()
	return nil
}
