package config

// Centralized environment + ephemeral secret helpers for demos.
// NOTE: Demo only – not production hardened.

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Get returns env var or default.
func Get(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// GetInt parses an int64 env var (base 10) or returns default if missing/invalid or <=0.
func GetInt(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// GetDurationSeconds returns seconds as int64.
func GetDurationSeconds(key string, def int64) int64 { return GetInt(key, def) }

// EphemeralSecret returns provided env var if set; otherwise generates a random base64url string.
// A warning message is returned if generation occurred so caller can log.
func EphemeralSecret(envKey string, size int) (secret string, generated bool, warn string) {
	if val := os.Getenv(envKey); val != "" {
		return val, false, ""
	}
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ephemeral-%d", time.Now().UnixNano()), true, fmt.Sprintf("entropy failure generating %s: %v", envKey, err)
	}
	return base64.RawURLEncoding.EncodeToString(b), true, fmt.Sprintf("%s not set – generated ephemeral secret (demo only; set %s for stable runs)", envKey, envKey)
}
