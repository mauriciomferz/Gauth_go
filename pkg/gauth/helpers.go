package gauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// Helper functions for grant generation and validation

// generateGrantID creates a cryptographically secure random grant ID
func generateGrantID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random grant ID: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// TimeNow is a replaceable function to get the current time (useful for testing)
var TimeNow = func() TimeFunc {
	return timeNow{}
}

type TimeFunc interface {
	Unix() int64
}

type timeNow struct{}

func (timeNow) Unix() int64 {
	return time.Now().Unix()
}
