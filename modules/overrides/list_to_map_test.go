package overrides

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestListToMapMarshalOperations(t *testing.T) {
	testCases := []struct {
		name           string
		original       ListToMap
		marshalledYAML string
		marshalledJSON string
	}{
		{
			name:           "empty map",
			original:       ListToMap{},
			marshalledYAML: "",
			marshalledJSON: "[]",
		},
		{
			name: "map with entries",
			original: ListToMap{
				m: map[string]struct{}{
					"foo": {},
				},
			},
			marshalledYAML: "- foo",
			marshalledJSON: "[\"foo\"]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bytes, err := yaml.Marshal(tc.original)
			assert.NoError(t, err)
			assert.Equal(t, tc.marshalledYAML, string(bytes))

			l := ListToMap{}
			assert.NoError(t, yaml.Unmarshal([]byte(tc.marshalledYAML), &l))
			assert.Equal(t, tc.original, l)

			bytes, err = json.Marshal(tc.original)
			assert.NoError(t, err)
			assert.Equal(t, tc.marshalledJSON, string(bytes))

			l2 := ListToMap{}
			assert.NoError(t, json.Unmarshal([]byte(tc.marshalledJSON), &l2))
			assert.Equal(t, tc.original, l2)
		})
	}
}
