package v2

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageMarshalToWriter(t *testing.T) {
	tests := []struct {
		expected []byte
	}{
		{
			expected: []byte{0x01, 0x02, 0x03},
		},
		{
			expected: []byte{},
		},
	}

	for _, tc := range tests {
		buff := &bytes.Buffer{}

		bytesWritten, err := marshalPageToWriter(tc.expected, buff)
		assert.NoError(t, err)
		assert.Equal(t, len(tc.expected)+int(totalHeaderSize), bytesWritten)

		page, err := unmarshalPageFromBytes(buff.Bytes())
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
	}
}

func TestPageMarshalToBuffer(t *testing.T) {
	tests := []struct {
		expected     []byte
		expectsError bool
	}{
		{
			expected: []byte{0x01, 0x02, 0x03},
		},
		{
			expected: []byte{},
		},
		{
			expected:     []byte{},
			expectsError: true,
		},
	}

	for _, tc := range tests {
		outputBuffer := make([]byte, len(tc.expected)+int(totalHeaderSize))

		write := func(b []byte) error {
			copy(b, tc.expected)

			if tc.expectsError {
				return errors.New("eep")
			}

			return nil
		}

		bytesWritten, err := marshalPageToBuffer(write, outputBuffer)
		if tc.expectsError {
			assert.Error(t, err)
			continue
		}

		assert.NoError(t, err)
		assert.Equal(t, len(tc.expected)+int(totalHeaderSize), bytesWritten)

		page, err := unmarshalPageFromBytes(outputBuffer)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
	}
}
