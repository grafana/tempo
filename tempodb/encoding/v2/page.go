package v2

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	uint32Size      = uint32(4) // jpe review:  lots of casting
	uint16Size      = uint32(2)
	totalHeaderSize = uint16Size + uint32Size + 0
)

// jpe - any headers to add in the first version of this?
//     - checksum? crc?  for indexes only?
//     - min/max ids for record?
type page struct {
	data []byte
}

/*
  |                 -- totalLength --                          |
  |             |            | -- headerLength -- |            |
  |   32 bits   |   16 bits  |                    |            |
  | totalLength | header len | header fields      | page bytes |
*/
func unmarshalPageFromBytes(b []byte) (*page, error) {
	if len(b) < int(totalHeaderSize) {
		return nil, fmt.Errorf("page of size %d too small", len(b))
	}

	totalLength := binary.LittleEndian.Uint32(b[:uint32Size])
	b = b[uint32Size:]
	headerLength := binary.LittleEndian.Uint16(b[:uint16Size])
	b = b[uint16Size:]

	// no header fields yet
	if headerLength != 0 {
		return nil, fmt.Errorf("headerLen unexpectedly %d while reading a page", headerLength)
	}

	dataLength := totalLength - uint32Size - uint16Size - uint32(headerLength)
	if len(b) != int(dataLength) {
		return nil, fmt.Errorf("page size %d but read %d", totalLength, dataLength)
	}

	return &page{
		data: b,
	}, nil
}

// marshalPageToWriter marshals the page bytes to the passed writer
func marshalPageToWriter(b []byte, w io.Writer) (int, error) {
	var headerLength uint16
	var totalLength uint32

	headerLength = 0
	totalLength = uint32(headerLength) + uint32Size + uint16Size + uint32(len(b))

	err := binary.Write(w, binary.LittleEndian, totalLength)
	if err != nil {
		return 0, err
	}

	err = binary.Write(w, binary.LittleEndian, headerLength)
	if err != nil {
		return 0, err
	}

	// header fields?

	_, err = w.Write(b)
	if err != nil {
		return 0, err
	}

	return int(totalLength), nil
}

// marshalPageToBuffer marshals a page to the passed buffer.  It uses the
// writePage method to write the bytes directly.  writePage method is expected
// to error if the buffer is too small
func marshalPageToBuffer(writePage func([]byte) error, buffer []byte) (int, error) {
	var headerLength uint16
	var totalLength uint32

	if len(buffer) < int(totalHeaderSize) {
		return 0, fmt.Errorf("page of size %d too small", len(buffer))
	}

	headerLength = 0
	totalLength = uint32(len(buffer))

	binary.LittleEndian.PutUint32(buffer[:uint32Size], totalLength)
	buffer = buffer[uint32Size:]
	binary.LittleEndian.PutUint16(buffer[:uint16Size], headerLength)
	buffer = buffer[uint16Size:]

	err := writePage(buffer)
	if err != nil {
		return 0, err
	}

	return int(totalLength), nil
}

func objectsPerPage(objectSizeBytes int, pageSizeBytes int) int {
	if objectSizeBytes == 0 {
		return 0
	}

	return (pageSizeBytes - int(totalHeaderSize)) / objectSizeBytes
}

func totalPages(totalObjects int, objectsPerPage int) int {
	if objectsPerPage == 0 {
		return 0
	}

	pages := totalObjects / objectsPerPage
	if totalObjects%objectsPerPage != 0 {
		pages++
	}
	return pages
}
