package crypto

// KeyProvider abstracts key management for dependency injection
type KeyProvider interface {
	ActiveSigner() (Signer, error)
	PublicKey(keyID string) (keyBytes []byte, algo string, err error)
	VerifyWith(msg, sig []byte, keyID string) error
}
