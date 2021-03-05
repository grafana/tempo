package v2

import "errors"

type pageHeader interface {
	unmarshalHeader([]byte) error
	headerLength() int
	marshalHeader([]byte) error
}

// DataHeaderLength is the length in bytes for the data header
const DataHeaderLength = 0

// IndexHeaderLength is the length in bytes for the record header
const IndexHeaderLength = 0

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
//   crc
//   min id
//   max id
type indexHeader struct {
	//jpe todo
}

func (h *indexHeader) unmarshalHeader(b []byte) error {
	return nil
}

func (h *indexHeader) headerLength() int {
	return IndexHeaderLength
}

func (h *indexHeader) marshalHeader(b []byte) error {
	return nil
}
