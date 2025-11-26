package test

import (
	"testing"
	"time"

	v "github.com/mauriciomferz/Gauth_go/pkg/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicValidatorClaims(t *testing.T) {
	val := v.NewBasicValidator()
	err := val.ValidateClaims(v.Claims{Subject: "s", Issuer: "iss", Audience: []string{"a"}, ExpiresAt: time.Now().Add(time.Hour)})
	require.NoError(t, err)
}

func TestBasicValidatorClaimsErrors(t *testing.T) {
	val := v.NewBasicValidator()
	err := val.ValidateClaims(v.Claims{Subject: "", Issuer: "", Audience: []string{}, ExpiresAt: time.Now().Add(-time.Minute)})
	require.Error(t, err)
}

func TestSafeSlice(t *testing.T) {
	s := v.SafeSlice("こんにちは世界", 4)
	// SafeSlice returns exactly n runes plus ellipsis when truncated.
	assert.Equal(t, "こんにち...", s)
}
