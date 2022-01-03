package v2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testHeader struct {
	field uint32
}

func (h *testHeader) unmarshalHeader(b []byte) error {
	if len(b) != int(uint32Size) {
		return errors.New("testheader err")
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
		assert.Equal(t, len(tc.expected)+header.headerLength()+int(baseHeaderSize), bytesWritten)

		page, err := unmarshalPageFromBytes(buff.Bytes(), &testHeader{})
		require.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
		assert.Equal(t, tc.field, page.header.(*testHeader).field)

		page, err = unmarshalPageFromReader(buff, &testHeader{}, []byte{})
		require.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
		assert.Equal(t, uint32(bytesWritten), page.totalLength)
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
		outputBuffer := make([]byte, len(tc.expected)+header.headerLength()+int(baseHeaderSize))

		restOfPage, err := marshalHeaderToPage(outputBuffer, header)
		require.NoError(t, err)
		copy(restOfPage, tc.expected)

		page, err := unmarshalPageFromBytes(outputBuffer, &testHeader{})
		require.NoError(t, err)
		assert.Equal(t, tc.expected, page.data)
		assert.Equal(t, uint32(len(outputBuffer)), page.totalLength)
		assert.Equal(t, tc.field, page.header.(*testHeader).field)
	}
}

func TestCorruptHeader(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	buff := &bytes.Buffer{}

	bytesWritten, err := marshalPageToWriter(data, buff, constDataHeader)
	require.NoError(t, err)
	assert.Equal(t, len(data)+constDataHeader.headerLength()+int(baseHeaderSize), bytesWritten)

	buffBytes := buff.Bytes()

	// overwrite the base header with 0s
	zeroHeader := bytes.Repeat([]byte{0x00}, baseHeaderSize)
	copy(buffBytes, zeroHeader)

	page, err := unmarshalPageFromBytes(buffBytes, constDataHeader)
	assert.Nil(t, page)
	assert.EqualError(t, err, "expected data len -6 does not match actual 3")

	page, err = unmarshalPageFromReader(bytes.NewReader(buffBytes), constDataHeader, nil)
	assert.Nil(t, page)
	assert.EqualError(t, err, "unexpected negative dataLength unmarshalling page: -6")

	// overwrite the base header with FFs
	zeroHeader = bytes.Repeat([]byte{0xFF}, baseHeaderSize)
	copy(buffBytes, zeroHeader)

	page, err = unmarshalPageFromBytes(buffBytes, constDataHeader)
	assert.Nil(t, page)
	assert.EqualError(t, err, "headerLen 65535 greater than remaining len 3")

	page, err = unmarshalPageFromReader(bytes.NewReader(buffBytes), constDataHeader, nil)
	assert.Nil(t, page)
	assert.EqualError(t, err, "unexpected non-zero len data header")
}

func TestIncompletePageDetected(t *testing.T) {
	dataLen := 100
	del := 10
	buff := &bytes.Buffer{}

	_, err := marshalPageToWriter(make([]byte, dataLen), buff, constDataHeader)
	require.NoError(t, err)

	// Delete some bytes from the end of the page
	buff.Truncate(buff.Len() - del)

	_, err = unmarshalPageFromReader(buff, constDataHeader, nil)
	require.EqualError(t, err, fmt.Sprintf("unexpected incomplete page read: expected:%d read:%d", dataLen, dataLen-del))
}
