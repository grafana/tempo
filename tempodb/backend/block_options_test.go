package backend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionsHash(t *testing.T) {
	hash := func(o map[string]string) uint64 { return (&BlockMeta{Options: o}).OptionsHash() }

	require.Zero(t, hash(nil))
	require.Zero(t, hash(map[string]string{}))

	a := hash(map[string]string{"a": "1", "b": "2"})
	require.NotZero(t, a)
	for range 20 { // map iteration order must not matter
		require.Equal(t, a, hash(map[string]string{"b": "2", "a": "1"}))
	}

	require.NotEqual(t, a, hash(map[string]string{"a": "1", "b": "3"}))
	require.NotEqual(t, a, hash(map[string]string{"a": "1"}))
	// key/value boundaries are significant
	require.NotEqual(t, hash(map[string]string{"ab": "c"}), hash(map[string]string{"a": "bc"}))
}
