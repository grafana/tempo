package backend

import (
	"encoding/json"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func Test_getDedicatedColumnsFromCache(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		isCached bool
		expected DedicatedColumns
	}{
		{
			name:     "null",
			jsonStr:  `null`,
			expected: nil,
		},
		{
			name:     "empty",
			jsonStr:  `[]`,
			expected: DedicatedColumns{},
		},
		{
			name:     "not empty",
			jsonStr:  `[ {"s": "resource", "n": "namespace"}, {"n": "http.method"}, {"n": "namespace"} ]`,
			isCached: true,
			expected: DedicatedColumns{
				{Scope: "resource", Name: "namespace", Type: "string"},
				{Scope: "span", Name: "http.method", Type: "string"},
				{Scope: "span", Name: "namespace", Type: "string"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes := []byte(tc.jsonStr)

			// clear the map so we start with a clean slate
			dedicatedColumnsCache.Purge()

			if tc.isCached {
				v, ok := getDedicatedColumnsFromCache(jsonBytes)
				require.False(t, ok)
				require.Nil(t, v)
			}

			dcs := DedicatedColumns{}
			err := json.Unmarshal(jsonBytes, &dcs)
			require.NoError(t, err)
			require.Equal(t, tc.expected, dcs)

			v, ok := getDedicatedColumnsFromCache(jsonBytes)
			require.True(t, ok)
			require.Equal(t, dcs, v)
			if tc.isCached {
				require.Equal(t, unsafe.Pointer(&dcs[0]), unsafe.Pointer(&v[0])) // check v was taken from the cache (pointers are the same)
			}
		})
	}
}
