package v2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testHeader struct {
	field uint32
}

func (h *testHeader) unmarshalHeader(b []byte) error {
	if len(b) != int(uint32Size) {
		return errors.New("err")
	}

	h.field = binary.LittleEndian.Uint32(b[:uint32Size])

	return nil
}

func (h *testHeader) headerLength() int {
	return int(uint32Size)
}

func (h *testHeader) marshalHeader(b []byte) error {
	if len(b) != int(uint32Size) {
		return errors.New("bad")
	}

	binary.LittleEndian.PutUint32(b, h.field)
	return nil
}

func TestPageMarshalToWriter(t *testing.T) {
	tests := []struct {
		expected []byte
		field    uint32
	}{
		{
			expected: []byte{0x01, 0x02, 0x03},
			field:    15,
		},
		{
			expected: []byte{},
			field:    0,
		},
	}

	for _, tc := range tests {
		buff := &bytes.Buffer{}

		header := &testHeader{
			field: tc.field,
		}

		bytesWritten, err := marshalPageToWriter(tc.expected, buff, header)
		require.NoError(t, err)
		assert.Equal(t, len(tc.expected)+header.headerLength()+int(uint32Size)+int(uint16Size), bytesWritten)

		page, err := unmarshalPageFromBytes(buff.Bytes(), &testHeader{})
		require.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
		assert.Equal(t, tc.field, page.header.(*testHeader).field)
	}
}

func TestPageMarshalToBuffer(t *testing.T) {
	tests := []struct {
		expected []byte
		field    uint32
	}{
		{
			expected: []byte{0x01, 0x02, 0x03},
			field:    15,
		},
		{
			expected: []byte{},
			field:    0,
		},
	}

	for _, tc := range tests {
		header := &testHeader{
			field: tc.field,
		}
		outputBuffer := make([]byte, len(tc.expected)+header.headerLength()+int(uint32Size)+int(uint16Size))

		restOfPage, err := marshalHeaderToPage(outputBuffer, header)
		require.NoError(t, err)
		copy(restOfPage, tc.expected)

		page, err := unmarshalPageFromBytes(outputBuffer, &testHeader{})
		require.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
		assert.Equal(t, tc.field, page.header.(*testHeader).field)
	}
}
