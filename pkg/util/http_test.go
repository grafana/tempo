package util

import (
	"testing"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"

	"github.com/stretchr/testify/assert"
)

func TestHexStringToTraceID(t *testing.T) {

	tc := []struct {
		id          string
		expected    []byte
		expectError bool
	}{
		{
			id:       "12",
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12},
		},
		{
			id:       "1234567890abcdef", // 64 bit
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		},
		{
			id:       "1234567890abcdef1234567890abcdef", // 128 bit
			expected: []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		},
		{
			id:          "121234567890abcdef1234567890abcdef", // value too long
			expected:    []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			expectError: true,
		},
		{
			id:       "234567890abcdef", // odd length
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		},
	}

	for _, tt := range tc {
		t.Run(tt.id, func(t *testing.T) {
			actual, err := hexStringToTraceID(tt.id)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, actual)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// For licensing reasons they strings exist in two packages. This test exists to make sure they don't
// drift.
func TestEquality(t *testing.T) {
	assert.Equal(t, AcceptHeaderKey, tempo.AcceptHeaderKey)
	assert.Equal(t, ProtobufTypeHeaderValue, tempo.ProtobufTypeHeaderValue)
}
