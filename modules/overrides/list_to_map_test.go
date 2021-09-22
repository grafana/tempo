package overrides

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestListToMapMarshalOperations(t *testing.T) {
	testCases := []struct {
		name                  string
		inputYAML             string
		inputJSON             string
		expectedListToMapYAML ListToMap
		expectedListToMapJSON ListToMap
		marshalledYAML        string
		marshalledJSON        string
	}{
		{
			name:                  "empty map",
			inputYAML:             "null",
			expectedListToMapYAML: ListToMap{},
			marshalledYAML:        "null\n",
			inputJSON:             "[]",
			expectedListToMapJSON: ListToMap{
				m: map[string]struct{}{},
			},
			marshalledJSON: "[]",
		},
		{
			name:      "map with entries",
			inputYAML: "- foo",
			expectedListToMapYAML: ListToMap{
				m: map[string]struct{}{
					"foo": {},
				},
			},
			marshalledYAML: "- foo\n",
			inputJSON:      "[\"foo\"]",
			expectedListToMapJSON: ListToMap{
				m: map[string]struct{}{
					"foo": {},
				},
			},
			marshalledJSON: "[\"foo\"]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// YAML to struct
			var l ListToMap
			assert.NoError(t, yaml.Unmarshal([]byte(tc.inputYAML), &l))
			assert.Equal(t, tc.expectedListToMapYAML, l)

			// struct to YAML
			bytes, err := yaml.Marshal(tc.expectedListToMapYAML)
			assert.NoError(t, err)
			assert.Equal(t, tc.marshalledYAML, string(bytes))

			// JSON to struct
			var l2 ListToMap
			assert.NoError(t, json.Unmarshal([]byte(tc.inputJSON), &l2))
			assert.Equal(t, tc.expectedListToMapJSON, l2)

			// struct to JSON
			bytes, err = json.Marshal(tc.expectedListToMapJSON)
			assert.NoError(t, err)
			assert.Equal(t, tc.marshalledJSON, string(bytes))
		})
	}
}
