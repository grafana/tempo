package v2

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	uint32Size = uint32(4) // jpe review:  lots of casting
	uint16Size = uint32(2)
)

// jpe - any headers to add in the first version of this?
//     - checksum?
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
	if len(b) < int(uint32Size)+int(uint16Size) {
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
