package collector

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopedDistinct(t *testing.T) {
	tcs := []struct {
		in       map[string][]string
		expected map[string][]string
		limit    int
		exceeded bool
	}{
		{
			in: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1", "val2"},
			},
			expected: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1", "val2"},
			},
		},
		{
			in: map[string][]string{
				"scope1": {"val1", "val2", "val1"},
				"scope2": {"val1", "val2", "val2"},
			},
			expected: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1", "val2"},
			},
		},
		{
			in: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1", "val2"},
			},
			expected: map[string][]string{
				"scope1": {"val1", "val2"},
				"scope2": {"val1"},
			},
			limit:    13,
			exceeded: true,
		},
	}

	for _, tc := range tcs {
		c := NewScopedDistinctString(tc.limit)

		// get and sort keys so we can deterministically add values
		keys := []string{}
		for k := range tc.in {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		for _, k := range keys {
			v := tc.in[k]
			for _, val := range v {
				c.Collect(k, val)
			}
		}

		require.Equal(t, tc.exceeded, c.Exceeded())

		actual := c.Strings()
		require.Equal(t, len(tc.expected), len(actual))

		for k, v := range tc.expected {
			require.Equal(t, v, actual[k])
		}
	}
}

// test diff
