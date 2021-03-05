package v2

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type pageHeader interface {
	unmarshalHeader([]byte) error
	headerLength() int
	marshalHeader([]byte) error
}

// DataHeaderLength is the length in bytes for the data header
const DataHeaderLength = 0

// IndexHeaderLength is the length in bytes for the record header
const IndexHeaderLength = int(uint32Size) + 16 + 16 // crc32 + 2 128 bit ids

// dataHeader implements a pageHeader that has no fields
type dataHeader struct {
}

func (h *dataHeader) unmarshalHeader(b []byte) error {
	if len(b) != 0 {
		return errors.New("unexpected non-zero len data header")
	}

	return nil
}

func (h *dataHeader) headerLength() int {
	return DataHeaderLength
}

func (h *dataHeader) marshalHeader(b []byte) error {
	return nil
}

// indexHeader implements a pageHeader that has index fields
//   fnvChecksum
//   min id
//   max id
type indexHeader struct {
	fnvChecksum uint32    // jpe - test crc failures
	maxID       common.ID // 128 bits/16 bytes : inclusive
	minID       common.ID // 128 bits/16 bytes : exclusive
}

func (h *indexHeader) unmarshalHeader(b []byte) error {
	if len(b) != IndexHeaderLength {
		return fmt.Errorf("unexpected index header len of %d", len(b))
	}

	h.fnvChecksum = binary.LittleEndian.Uint32(b[:uint32Size])
	b = b[uint32Size:]
	h.maxID = b[:16]
	b = b[16:]
	h.minID = b[:16]
	//b = b[16:]

	return nil
}

func (h *indexHeader) headerLength() int {
	return IndexHeaderLength
}

func (h *indexHeader) marshalHeader(b []byte) error {
	if len(b) != IndexHeaderLength {
		return fmt.Errorf("unexpected index header len of %d", len(b))
	}

	binary.LittleEndian.PutUint32(b, h.fnvChecksum)
	b = b[uint32Size:]
	copy(b, h.maxID)
	b = b[16:]
	copy(b, h.minID)
	// b = b[16:]

	return nil
}
