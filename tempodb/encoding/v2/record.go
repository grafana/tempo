package v2

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/grafana/tempo/v2/pkg/validation"
)

// recordLength holds the size of a single record in bytes
const recordLength = 28 // 28 = 128 bit ID, 64bit start, 32bit length

type record struct{}

var staticRecord = record{}

func NewRecordReaderWriter() RecordReaderWriter {
	return staticRecord
}

// MarshalRecords converts a slice of records into a byte slice
func (r record) MarshalRecords(records []Record) ([]byte, error) {
	recordBytes := make([]byte, len(records)*recordLength)

	err := r.MarshalRecordsToBuffer(records, recordBytes)
	if err != nil {
		return nil, err
	}

	return recordBytes, nil
}

// MarshalRecordsToBuffer converts a slice of records and marshals them to an existing byte slice
func (record) MarshalRecordsToBuffer(records []Record, buffer []byte) error {
	if len(records)*recordLength > len(buffer) {
		return fmt.Errorf("buffer %d is not big enough for records %d", len(buffer), len(records)*recordLength)
	}

	for i, r := range records {
		buff := buffer[i*recordLength : (i+1)*recordLength]

		if !validation.ValidTraceID(r.ID) { // todo: remove this check.  maybe have a max id size of 128 bits?
			return errors.New("ids must be 128 bit")
		}

		marshalRecord(r, buff)
	}

	return nil
}

// RecordCount returns the number of records in a byte slice
func (record) RecordCount(b []byte) int {
	return len(b) / recordLength
}

func (record) RecordLength() int {
	return recordLength
}

// UnmarshalRecord creates a new record from the contents ofa byte slice
func (record) UnmarshalRecord(buff []byte) Record {
	r := Record{
		ID:     make([]byte, 16), // 128 bits
		Start:  0,
		Length: 0,
	}

	copy(r.ID, buff[:16])
	r.Start = binary.LittleEndian.Uint64(buff[16:24])
	r.Length = binary.LittleEndian.Uint32(buff[24:])

	return r
}

// marshalRecord writes a record to an existing byte slice
func marshalRecord(r Record, buff []byte) {
	copy(buff, r.ID)

	binary.LittleEndian.PutUint64(buff[16:24], r.Start)
	binary.LittleEndian.PutUint32(buff[24:], r.Length)
}
