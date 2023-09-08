package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDurationMarshalJSON(t *testing.T) {
	type mockStruct struct {
		Duration *Duration `json:"duration"`
	}

	testCases := []struct {
		name     string
		mock     []byte
		expected mockStruct
		expErr   string
	}{
		{
			name: "unmarshal duration nanoseconds",
			mock: []byte(`{"duration":60000000000}`),
			expected: mockStruct{
				Duration: &Duration{60 * time.Second},
			},
			expErr: "",
		},
		{
			name: "unmarshal duration string",
			mock: []byte(`{"duration":"60s"}`),
			expected: mockStruct{
				Duration: &Duration{60 * time.Second},
			},
			expErr: "",
		},
		{
			name:     "unmarshal duration error",
			mock:     []byte(`{"duration":"foo"}`),
			expected: mockStruct{},
			expErr:   "time: invalid duration \"foo\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result mockStruct
			err := json.Unmarshal(tc.mock, &result)
			if tc.expErr != "" {
				assert.EqualError(t, err, tc.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, *tc.expected.Duration, *result.Duration)
			}
		})
	}
}
