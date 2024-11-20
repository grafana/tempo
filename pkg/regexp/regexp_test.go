package regexp

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
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

func TestShouldMemoize(t *testing.T) {
	tcs := []struct {
		regex         string
		shouldMemoize bool
	}{
		{
			regex:         ".*",
			shouldMemoize: false,
		},
		{
			regex:         "foo.*",
			shouldMemoize: false,
		},
		{
			regex:         ".*bar",
			shouldMemoize: false,
		},
		{
			regex:         "foo|bar",
			shouldMemoize: false,
		},
		{
			regex:         ".*bar.*", // creates a containsStringMatcher so should not memoize
			shouldMemoize: false,
		},
		{
			regex:         ".*bar.*foo.*", // calls contains in order
			shouldMemoize: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.regex, func(t *testing.T) {
			m, err := labels.NewFastRegexMatcher(tc.regex)
			require.NoError(t, err)
			require.Equal(t, tc.shouldMemoize, shouldMemoize(m))
		})
	}
}
