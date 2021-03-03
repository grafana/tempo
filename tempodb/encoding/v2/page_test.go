package v2

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPage(t *testing.T) {
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
		assert.Equal(t, len(tc.expected)+int(uint32Size)+int(uint16Size), bytesWritten)

		page, err := unmarshalPageFromBytes(buff.Bytes())
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
	}
}
