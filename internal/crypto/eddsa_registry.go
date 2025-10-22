package crypto

// GlobalEdDSARegistry holds a reference to a key manager for external publication (e.g., JWKS).
// Placed in internal/crypto to allow both gauth and web packages to interact without creating an import cycle.
var GlobalEdDSARegistry *Manager

// RegisterGlobalEdDSAManager sets the global manager reference if not already set.
// It ignores nil inputs. Once set, subsequent calls retain the initial manager.
func RegisterGlobalEdDSAManager(m *Manager) {
	if m == nil {
		return
	}
	if GlobalEdDSARegistry == nil {
		GlobalEdDSARegistry = m
	}
}
