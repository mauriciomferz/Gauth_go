// Sign implements the crypto.Signer interface for compatibility
package token

import (
	"context"
	"crypto"
	"io"
	"sync"
)

// MockStore provides a mock implementation of the Store interface for testing
type MockStore struct {
	mu     sync.RWMutex
	tokens map[string]*Token
	err    error
}

// NewMockStore creates a new mock store instance
func NewMockStore() *MockStore {
	return &MockStore{
		tokens: make(map[string]*Token),
	}
}

// SetError sets an error that will be returned by all operations
func (m *MockStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Reset clears all tokens and errors
func (m *MockStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens = make(map[string]*Token)
	m.err = nil
}

// Save mocks storing a token
func (m *MockStore) Save(ctx context.Context, key string, token *Token) error {
	if err := m.checkError(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[key] = copyToken(token)
	return nil
}

// Get mocks retrieving a token
func (m *MockStore) Get(ctx context.Context, key string) (*Token, error) {
	if err := m.checkError(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	token, exists := m.tokens[key]
	if !exists {
		return nil, ErrTokenNotFound
	}
	return copyToken(token), nil
}

// Delete mocks removing a token
func (m *MockStore) Delete(ctx context.Context, key string) error {
	if err := m.checkError(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tokens[key]; !exists {
		return ErrTokenNotFound
	}

	delete(m.tokens, key)
	return nil
}

// List mocks listing tokens
func (m *MockStore) List(ctx context.Context, filter Filter) ([]*Token, error) {
	if err := m.checkError(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []*Token
	for _, token := range m.tokens {
		if matchesFilter(token, filter) {
			matches = append(matches, copyToken(token))
		}
	}
	return matches, nil
}

// Rotate mocks token rotation
func (m *MockStore) Rotate(ctx context.Context, old, new *Token) error {
	if err := m.checkError(); err != nil {
		return err
	}

	if err := m.Delete(ctx, old.ID); err != nil {
		return err
	}

	return m.Save(ctx, new.ID, new)
}

// Revoke mocks token revocation
func (m *MockStore) Revoke(ctx context.Context, token *Token) error {
	if err := m.checkError(); err != nil {
		return err
	}

	return m.Delete(ctx, token.ID)
}

// Validate mocks token validation
func (m *MockStore) Validate(ctx context.Context, token *Token) error {
	if err := m.checkError(); err != nil {
		return err
	}

	stored, err := m.Get(ctx, token.ID)
	if err != nil {
		return err
	}

	if stored.Value != token.Value {
		return ErrInvalidToken
	}

	return nil
}

// Refresh mocks token refresh
func (m *MockStore) Refresh(ctx context.Context, refreshToken *Token) (*Token, error) {
	if err := m.checkError(); err != nil {
		return nil, err
	}

	if refreshToken.Type != Refresh {
		return nil, ErrInvalidType
	}

	return &Token{
		ID:        GenerateID(),
		Type:      Access,
		Subject:   refreshToken.Subject,
		Issuer:    refreshToken.Issuer,
		Audience:  refreshToken.Audience,
		Scopes:    refreshToken.Scopes,
		Algorithm: refreshToken.Algorithm,
	}, nil
}

func (m *MockStore) checkError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.err
}

// MockSigner provides a mock implementation for testing token signing
type MockSigner struct {
	SignFunc   func(token *Token) (string, error)
	VerifyFunc func(tokenString string) (*Token, error)
	err        error
}

// Sign implements the crypto.Signer interface for compatibility
func (m *MockSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	// Return a dummy signature for testing
	return []byte("mock-signature"), nil
}

// Sign implements the crypto.Signer interface for compatibility; returns dummy signature
// (moved below the struct definition)

// Public returns a dummy public key for compatibility with crypto.Signer
func (m *MockSigner) Public() crypto.PublicKey {
	return nil
}

// NewMockSigner creates a new mock signer instance
func NewMockSigner() *MockSigner {
	return &MockSigner{
		SignFunc: func(token *Token) (string, error) {
			return "mock.signed." + token.ID, nil
		},
		VerifyFunc: func(tokenString string) (*Token, error) {
			return &Token{
				ID:    "mock-token",
				Value: tokenString,
				Type:  Access,
			}, nil
		},
	}
}

// SetError sets an error that will be returned by operations
func (m *MockSigner) SetError(err error) {
	m.err = err
}

// SignToken implements token signing
func (m *MockSigner) SignToken(token *Token) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.SignFunc(token)
}

// VerifyToken implements token verification
func (m *MockSigner) VerifyToken(tokenString string) (*Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.VerifyFunc(tokenString)
}
