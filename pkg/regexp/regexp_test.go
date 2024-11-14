package regexp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegexpMatch(t *testing.T) {
	r, err := NewRegexp([]string{"^a.*"}, true)
	require.NoError(t, err)

	require.True(t, r.Match([]byte("abc")))
	require.True(t, r.MatchString("abc"))
	require.False(t, r.MatchString("xyz"))

	r, err = NewRegexp([]string{"^a.*"}, false)
	require.NoError(t, err)

	require.False(t, r.Match([]byte("abc")))
	require.False(t, r.MatchString("abc"))
	require.True(t, r.MatchString("xyz"))
}
