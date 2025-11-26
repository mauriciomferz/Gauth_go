package test

import (
	"testing"

	c "github.com/mauriciomferz/Gauth_go/pkg/compliance"
	"github.com/stretchr/testify/require"
)

func TestComplianceRegistry(t *testing.T) {
	r := c.NewRegistry()
	r.Register(c.Flow{ID: "f1", Source: "svcA", Destination: "svcB", DataTypes: []c.DataClass{c.DataClassPersonal}, Purpose: "demo", Retention: "30d"})
	flows := r.List()
	require.Len(t, flows, 1)
	require.Equal(t, "f1", flows[0].ID)
}
