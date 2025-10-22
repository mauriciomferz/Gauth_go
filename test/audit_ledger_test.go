package test

import (
	"testing"

	a "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/stretchr/testify/require"
)

func TestLedgerAppendAndVerify(t *testing.T) {
	l := a.NewLedger()
	_, err := l.Append("actor", "action", "res", "details")
	require.NoError(t, err)
	_, err = l.Append("actor2", "action2", "res2", "details2")
	require.NoError(t, err)
	require.NoError(t, l.Verify())
}

func TestLedgerSeal(t *testing.T) {
	l := a.NewLedger()
	l.Seal()
	_, err := l.Append("a", "b", "c", "d")
	require.Error(t, err)
}
