package backend

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getDedicatedColumns(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected *DedicatedColumns
	}{
		{
			name:    "case1",
			jsonStr: `[ {"s": "resource", "n": "namespace"}, {"n": "http.method"}, {"n": "namespace"} ]`,
			expected: &DedicatedColumns{
				{Scope: "resource", Name: "namespace", Type: "string"},
				{Scope: "span", Name: "http.method", Type: "string"},
				{Scope: "span", Name: "namespace", Type: "string"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// clear the map so we start with a clean slate
			clear(dedicatedColumnsKeeper)

			v := getDedicatedColumns(tc.jsonStr)
			require.Nil(t, v)

			dcs := &DedicatedColumns{}
			err := json.Unmarshal([]byte(tc.jsonStr), dcs)
			require.NoError(t, err)
			require.Equal(t, tc.expected, dcs)

			v = getDedicatedColumns(tc.jsonStr)
			require.Equal(t, dcs, v)

			p1 := fmt.Sprintf("%p", dcs)
			p2 := fmt.Sprintf("%p", v)
			require.Equal(t, p1, p2)

			dcs2 := &DedicatedColumns{}
			err = json.Unmarshal([]byte(tc.jsonStr), dcs2)
			require.NoError(t, err)
			require.Equal(t, tc.expected, dcs)

			require.Equal(t, dcs, dcs2)
		})
	}
}
