package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const (
	// DefaultIDLength is the default number of bytes for token IDs
	DefaultIDLength = 32
)

// NewID generates a cryptographically secure random token ID
func NewID() string {
	b := make([]byte, DefaultIDLength)
	if _, err := rand.Read(b); err != nil {
		// If crypto/rand fails, this is a serious issue
		panic(fmt.Sprintf("failed to generate secure token ID: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.
