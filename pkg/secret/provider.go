package secret

// Package secret provides a pluggable secret storage abstraction. Initial implementations:
// 1. memory: in-process map (non-persistent, for tests and dev)
// 2. vaultstub: placeholder adapter exposing intended surface without external dependency
//
// Future providers: HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager, Kubernetes secrets.

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// Provider defines minimal secret storage capabilities. Implementations should be safe for
// concurrent use. Errors SHOULD wrap context (provider name, key) for observability.
type Provider interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, opts ...Option) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Name() string
}

// Option can modify behavior of Set operations (e.g., TTL, encryption context, version tags).
type Option interface{ apply(*setOptions) }

type setOptions struct {
	ttlSeconds  int64
	ifNotExists bool
}

// WithTTL requests provider-specific expiry. Providers may ignore if unsupported.
func WithTTL(seconds int64) Option { return ttlOpt(seconds) }

type ttlOpt int64

func (t ttlOpt) apply(o *setOptions) { o.ttlSeconds = int64(t) }

// IfNotExists causes Set to fail if key already present (idempotent create semantics).
func IfNotExists() Option { return ifNotExistsOpt(true) }

type ifNotExistsOpt bool

func (i ifNotExistsOpt) apply(o *setOptions) { o.ifNotExists = bool(i) }

// MemoryProvider is an in-memory map implementation (NOT for production secret storage).
type MemoryProvider struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewMemory returns a new MemoryProvider.
func NewMemory() *MemoryProvider { return &MemoryProvider{data: make(map[string]string)} }

func (m *MemoryProvider) Name() string { return "memory" }

var (
	ErrNotFound     = errors.New("secret not found")
	ErrAlreadyExist = errors.New("secret already exists")
)

func (m *MemoryProvider) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (m *MemoryProvider) Set(_ context.Context, key, value string, opts ...Option) error {
	var so setOptions
	for _, o := range opts {
		o.apply(&so)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if so.ifNotExists {
		if _, exists := m.data[key]; exists {
			return ErrAlreadyExist
		}
	}
	m.data[key] = value
	return nil
}

func (m *MemoryProvider) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MemoryProvider) List(_ context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0)
	for k := range m.data {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, nil
}

// VaultStub is a placeholder for a future Vault provider. It delegates to an internal memory map
// while exposing distinct Name() for metrics separation and future incremental replacement.
type VaultStub struct{ *MemoryProvider }

func NewVaultStub() *VaultStub    { return &VaultStub{MemoryProvider: NewMemory()} }
func (v *VaultStub) Name() string { return "vaultstub" }
