package v2

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	uint64Size     = 8
	uint32Size     = 4
	uint16Size     = 2
	baseHeaderSize = uint16Size + uint32Size
)

type page struct {
	data   []byte
	header pageHeader
}

/*
  |                 -- totalLength --                          |
  |             |            | -- headerLength -- |            |
  |   32 bits   |   16 bits  |                    |            |
  | totalLength | header len | header fields      | page bytes |
*/
func unmarshalPageFromBytes(b []byte, header pageHeader) (*page, error) {
	totalHeaderSize := baseHeaderSize + header.headerLength()
	if len(b) < totalHeaderSize {
		return nil, fmt.Errorf("page of size %d too small", len(b))
	}

	totalLength := binary.LittleEndian.Uint32(b[:uint32Size])
	b = b[uint32Size:]
	headerLength := binary.LittleEndian.Uint16(b[:uint16Size])
	b = b[uint16Size:]
	err := header.unmarshalHeader(b[:headerLength])
	if err != nil {
		return nil, err
	}
	b = b[headerLength:]

	dataLength := int(totalLength) - totalHeaderSize
	if len(b) != dataLength {
		return nil, fmt.Errorf("expected data len %d does not match actual %d", dataLength, len(b))
	}

	return &page{
		data:   b,
		header: header,
	}, nil
}

// marshalPageToWriter marshals the page bytes to the passed writer
func marshalPageToWriter(b []byte, w io.Writer, header pageHeader) (int, error) {
	var headerLength uint16
	var totalLength uint32

	headerLength = uint16(header.headerLength())
	totalLength = uint32(headerLength) + baseHeaderSize + uint32(len(b))

	err := binary.Write(w, binary.LittleEndian, totalLength)
	if err != nil {
		return 0, err
	}

	err = binary.Write(w, binary.LittleEndian, headerLength)
	if err != nil {
		return 0, err
	}

	if headerLength != 0 {
		headerBuff := make([]byte, headerLength)
		err = header.marshalHeader(headerBuff)
		if err != nil {
			return 0, err
		}

		_, err := w.Write(headerBuff)
		if err != nil {
			return 0, err
		}
	}

	_, err = w.Write(b)
	if err != nil {
		return 0, err
	}

	return int(totalLength), nil
}

// marshalHeaderToPage marshals the header only to the passed in page and then returns
//  the rest of the page slice for the caller to finish
func marshalHeaderToPage(page []byte, header pageHeader) ([]byte, error) {
	var headerLength uint16
	var totalLength uint32

	totalHeaderSize := baseHeaderSize + uint32(header.headerLength())
	if len(page) < int(totalHeaderSize) {
		return nil, fmt.Errorf("page of size %d too small", len(page))
	}

	headerLength = uint16(header.headerLength()) // jpe casting
	totalLength = uint32(len(page))

	binary.LittleEndian.PutUint32(page[:uint32Size], totalLength)
	page = page[uint32Size:]
	binary.LittleEndian.PutUint16(page[:uint16Size], headerLength)
	page = page[uint16Size:]

	err := header.marshalHeader(page[:headerLength])
	if err != nil {
		return nil, err
	}

	return page[headerLength:], nil
}

func objectsPerPage(objectSizeBytes int, pageSizeBytes int, headerSize int) int {
	if objectSizeBytes == 0 {
		return 0
	}

	// headerSize only accounts for the custom header size.  also subtract base
	return (pageSizeBytes - headerSize - int(baseHeaderSize)) / objectSizeBytes
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
