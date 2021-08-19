package util

import (
	"errors"
	"testing"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"

	"github.com/stretchr/testify/assert"
)

func TestHexStringToTraceID(t *testing.T) {
	tc := []struct {
		id          string
		expected    []byte
		expectError error
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
			expectError: errors.New("trace IDs can't be larger than 128 bits"),
		},
		{
			id:       "234567890abcdef", // odd length
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		},
		{
			id:          "1234567890abcdef ", // trailing space
			expected:    nil,
			expectError: errors.New("trace IDs can only contain hex characters: invalid character ' ' at position 17"),
		},
	}

	for _, tt := range tc {
		t.Run(tt.id, func(t *testing.T) {
			actual, err := HexStringToTraceID(tt.id)

			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, actual)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestTraceIDToHexString(t *testing.T) {
	tc := []struct {
		byteID  []byte
		traceID string
	}{
		{
			byteID:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12},
			traceID: "12",
		},
		{
			byteID:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			traceID: "1234567890abcdef", // 64 bit
		},
		{
			byteID:  []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			traceID: "1234567890abcdef1234567890abcdef", // 128 bit
		},
		{
			byteID:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0xa0},
			traceID: "12a0", // trailing zero
		},
	}

	for _, tt := range tc {
		t.Run(tt.traceID, func(t *testing.T) {
			actual := TraceIDToHexString(tt.byteID)

			assert.Equal(t, tt.traceID, actual)
		})
	}
}

// For licensing reasons they strings exist in two packages. This test exists to make sure they don't
// drift.
func TestEquality(t *testing.T) {
	assert.Equal(t, AcceptHeaderKey, tempo.AcceptHeaderKey)
	assert.Equal(t, ProtobufTypeHeaderValue, tempo.ProtobufTypeHeaderValue)
}
