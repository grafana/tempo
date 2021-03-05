package v2

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	}

	for _, tc := range tests {
		outputBuffer := make([]byte, len(tc.expected)+int(totalHeaderSize))

		restOfPage, err := marshalHeaderToPage(outputBuffer)
		require.NoError(t, err)
		copy(restOfPage, tc.expected)

		page, err := unmarshalPageFromBytes(outputBuffer)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
	}
}
