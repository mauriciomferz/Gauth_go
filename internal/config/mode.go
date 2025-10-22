package config

import (
	"os"
	"strings"
)

const (
	ModeEnv  = "GAUTH_MODE"
	ModeDev  = "development"
	ModeProd = "production"
	ModeTest = "test"
)

// ReadMode returns the normalized mode value (development if unset/unknown).
func ReadMode() string {
	m := strings.ToLower(strings.TrimSpace(os.Getenv(ModeEnv)))
	switch m {
	case ModeProd, ModeDev, ModeTest:
		return m
	default:
		return ModeDev
	}
}

// IsProduction returns true if GAUTH_MODE=production.
func IsProduction() bool { return ReadMode() == ModeProd }
