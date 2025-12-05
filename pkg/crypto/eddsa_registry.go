package crypto

import "log"

// GlobalEdDSARegistry holds a reference to a key manager for external publication (e.g., JWKS).
// Placed in internal/crypto to allow both gauth and web packages to interact without creating an import cycle.
//
// Deprecated: GlobalEdDSARegistry is deprecated and will be removed in a future version.
// Use dependency injection to pass crypto.Manager instances instead.
var GlobalEdDSARegistry *Manager

// RegisterGlobalEdDSAManager sets the global manager reference if not already set.
// It ignores nil inputs. Once set, subsequent calls retain the initial manager.
//
// Deprecated: RegisterGlobalEdDSAManager is deprecated. Use dependency injection instead.
func RegisterGlobalEdDSAManager(m *Manager) {
	log.Println("WARNING: RegisterGlobalEdDSAManager is deprecated. Please migrate to dependency injection.")
	if m == nil {
		return
	}
	if GlobalEdDSARegistry == nil {
		GlobalEdDSARegistry = m
	}
}

// GetGlobalEdDSAManager returns the global manager reference.
//
// Deprecated: GetGlobalEdDSAManager is deprecated. Use dependency injection instead.
func GetGlobalEdDSAManager() *Manager {
	log.Println("WARNING: GetGlobalEdDSAManager is deprecated. Please migrate to dependency injection.")
	return GlobalEdDSARegistry
}
