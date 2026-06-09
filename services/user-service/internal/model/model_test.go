package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthContextHasPermission(t *testing.T) {
	ctx := AuthContext{Perms: []string{"user.read"}}
	require.True(t, ctx.HasPermission("user.read"))
	require.False(t, ctx.HasPermission("user.delete"))
}

func TestAuthContextWildcard(t *testing.T) {
	ctx := AuthContext{Perms: []string{"*"}}
	require.True(t, ctx.HasPermission("anything"))
}
