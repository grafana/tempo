package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v2"
)

type mockStruct struct {
	Duration *Duration `json:"duration"`
}

func TestDuration_MarshalJSON(t *testing.T) {
	testCases := []struct {
		name     string
		input    mockStruct
		expected []byte
		expErr   string
	}{
		{
			name:     "marshal duration",
			input:    mockStruct{&Duration{60 * time.Second}},
			expected: []byte("{\"duration\":\"1m0s\"}"),
			expErr:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := json.Marshal(tc.input)
			if tc.expErr != "" {
				assert.EqualError(t, err, tc.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestDuration_MarshalYAML guards the admin /status/overrides/{tenant} page,
// which yaml.Marshal()s a struct containing *Duration fields. Without a
// MarshalYAML method, yaml.v2 emits "duration: 5m0s" (the embedded
// time.Duration as a nested map) instead of the flat scalar "5m0s".
func TestDuration_MarshalYAML(t *testing.T) {
	in := mockStruct{&Duration{60 * time.Second}}
	out, err := yaml.Marshal(in)
	assert.NoError(t, err)
	assert.Equal(t, "duration: 1m0s\n", string(out))

	var decoded mockStruct
	assert.NoError(t, yaml.Unmarshal(out, &decoded))
	assert.Equal(t, in.Duration.Duration, decoded.Duration.Duration)
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected mockStruct
		expErr   string
	}{
		{
			name:  "unmarshal duration nanoseconds",
			input: []byte(`{"duration":60000000000}`),
			expected: mockStruct{
				Duration: &Duration{60 * time.Second},
			},
			expErr: "",
		},
		{
			name:  "unmarshal duration string",
			input: []byte(`{"duration":"60s"}`),
			expected: mockStruct{
				Duration: &Duration{60 * time.Second},
			},
			expErr: "",
		},
		{
			name:     "unmarshal duration error",
			input:    []byte(`{"duration":"foo"}`),
			expected: mockStruct{},
			expErr:   "time: invalid duration \"foo\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result mockStruct
			err := json.Unmarshal(tc.input, &result)
			if tc.expErr != "" {
				assert.EqualError(t, err, tc.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, *tc.expected.Duration, *result.Duration)
			}
		})
	}
}
