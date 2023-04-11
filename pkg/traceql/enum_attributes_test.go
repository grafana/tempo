package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllAttributesString(t *testing.T) {
	for _, s := range AllAttributeScopes() {
		actual := AttributeScopeFromString(s.String())
		require.Equal(t, s, actual)
	}

	require.Equal(t, AttributeScopeUnknown, AttributeScopeFromString("foo"))
	require.Equal(t, AttributeScopeNone, AttributeScopeFromString(""))
}
