package backend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionsString(t *testing.T) {
	str := func(o map[string]string) string { return (&BlockMeta{Options: o}).OptionsString() }

	require.Empty(t, str(nil))
	require.Empty(t, str(map[string]string{}))
	require.Equal(t, "[a=1]", str(map[string]string{"a": "1"}))

	for range 20 { // sorted by key, so map iteration order must not matter
		require.Equal(t, "[a=1, b=2, c=3]", str(map[string]string{"c": "3", "b": "2", "a": "1"}))
	}

	a := str(map[string]string{"a": "1", "b": "2"})
	require.NotEqual(t, a, str(map[string]string{"a": "1", "b": "3"}))
	require.NotEqual(t, a, str(map[string]string{"a": "1"}))
	// key/value boundaries are significant
	require.NotEqual(t, str(map[string]string{"ab": "c"}), str(map[string]string{"a": "bc"}))

	t.Run("delimiters in keys and values cannot make different options equal", func(t *testing.T) {
		require.Equal(t, `[a=1\, b\=2]`, str(map[string]string{"a": "1, b=2"}))
		require.NotEqual(t, str(map[string]string{"a": "1, b=2"}), str(map[string]string{"a": "1", "b": "2"}))
		require.NotEqual(t, str(map[string]string{"a=b": "c"}), str(map[string]string{"a": "b=c"}))
		require.NotEqual(t, str(map[string]string{`a\`: "b"}), str(map[string]string{"a": `\b`}))
		require.NotEqual(t, str(map[string]string{"a": `1\`, "b": "2"}), str(map[string]string{"a": `1\, b=2`}))
	})
}
