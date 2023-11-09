package listtomap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestListToMapMarshalOperationsYAML(t *testing.T) {
	testCases := []struct {
		name                  string
		inputYAML             string
		expectedListToMapYAML ListToMap
		marshalledYAML        string
	}{
		{
			name:                  "empty map",
			inputYAML:             "",
			expectedListToMapYAML: ListToMap{},
			marshalledYAML:        "null\n",
		},
		{
			name:      "map with entries",
			inputYAML: "- foo",
			expectedListToMapYAML: ListToMap{
				"foo": {},
			},
			marshalledYAML: "- foo\n",
		},
		{
			name:      "explicit string entries",
			inputYAML: "- \"foo\"",
			expectedListToMapYAML: ListToMap{
				"foo": {},
			},
			marshalledYAML: "- foo\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// YAML to struct
			var l ListToMap
			assert.NoError(t, yaml.Unmarshal([]byte(tc.inputYAML), &l))
			assert.NotNil(t, l.GetMap())
			assert.Equal(t, tc.expectedListToMapYAML, l)

			// struct to YAML
			bytes, err := yaml.Marshal(tc.expectedListToMapYAML)
			assert.NoError(t, err)
			assert.Equal(t, tc.marshalledYAML, string(bytes))
		})
	}
}

func TestListToMapMarshalOperationsJSON(t *testing.T) {
	testCases := []struct {
		name                  string
		inputJSON             string
		expectedListToMapJSON ListToMap
		marshalledJSON        string
	}{
		{
			name:                  "empty map",
			inputJSON:             "[]",
			expectedListToMapJSON: ListToMap{},
			marshalledJSON:        "[]",
		},
		{
			name:      "map with entries",
			inputJSON: "[\"foo\"]",
			expectedListToMapJSON: ListToMap{
				"foo": {},
			},
			marshalledJSON: "[\"foo\"]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// JSON to struct
			var l ListToMap
			assert.NoError(t, json.Unmarshal([]byte(tc.inputJSON), &l))
			assert.NotNil(t, l.GetMap())
			assert.Equal(t, tc.expectedListToMapJSON, l)

			// struct to JSON
			bytes, err := json.Marshal(tc.expectedListToMapJSON)
			assert.NoError(t, err)
			assert.Equal(t, tc.marshalledJSON, string(bytes))
		})
	}
}

func TestMerge(t *testing.T) {
	testCases := []struct {
		name           string
		m1, m2, merged ListToMap
	}{
		{
			"merge keys from both maps",
			map[string]struct{}{"key1": {}, "key3": {}},
			map[string]struct{}{"key2": {}, "key3": {}},
			map[string]struct{}{"key1": {}, "key2": {}, "key3": {}},
		},
		{
			"nil map",
			nil,
			map[string]struct{}{"key1": {}},
			map[string]struct{}{"key1": {}},
		},
		{
			"both nil",
			nil,
			nil,
			map[string]struct{}{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.merged, Merge(tc.m1, tc.m2))
		})
	}
}
