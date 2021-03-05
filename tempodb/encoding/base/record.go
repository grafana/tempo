package base

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// RecordLength holds the size of a single record in bytes
const RecordLength = 28 // 28 = 128 bit ID, 64bit start, 32bit length

type recordSorter struct {
	records []*common.Record
}

// SortRecords sorts a slice of record pointers
func SortRecords(records []*common.Record) {
	sort.Sort(&recordSorter{
		records: records,
	})
}

func (t *recordSorter) Len() int {
	return len(t.records)
}

func (t *recordSorter) Less(i, j int) bool {
	a := t.records[i]
	b := t.records[j]

	return bytes.Compare(a.ID, b.ID) == -1
}

func (t *recordSorter) Swap(i, j int) {
	t.records[i], t.records[j] = t.records[j], t.records[i]
}

// MarshalRecords converts a slice of records into a byte slice
func MarshalRecords(records []*common.Record) ([]byte, error) {
	recordBytes := make([]byte, len(records)*RecordLength)

	err := MarshalRecordsToBuffer(records, recordBytes)
	if err != nil {
		return nil, err
	}

	return recordBytes, nil
}

// MarshalRecordsToBuffer converts a slice of records and marshals them to an existing byte slice
func MarshalRecordsToBuffer(records []*common.Record, buffer []byte) error {
	if len(records)*RecordLength > len(buffer) {
		return fmt.Errorf("buffer %d is not big enough for records %d", len(buffer), len(records)*RecordLength)
	}

	for i, r := range records {
		buff := buffer[i*RecordLength : (i+1)*RecordLength]

		if !validation.ValidTraceID(r.ID) { // todo: remove this check.  maybe have a max id size of 128 bits?
			return errors.New("ids must be 128 bit")
		}

		marshalRecord(r, buff)
	}

	return nil
}

func unmarshalRecords(recordBytes []byte) ([]*common.Record, error) {
	mod := len(recordBytes) % RecordLength
	if mod != 0 {
		return nil, fmt.Errorf("records are an unexpected number of bytes %d", mod)
	}

	numRecords := RecordCount(recordBytes)
	records := make([]*common.Record, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		buff := recordBytes[i*RecordLength : (i+1)*RecordLength]

		r := UnmarshalRecord(buff)

		records = append(records, r)
	}

	return records, nil
}

// RecordCount returns the number of records in a byte slice
func RecordCount(b []byte) int {
	return len(b) / RecordLength
}

// marshalRecord writes a record to an existing byte slice
func marshalRecord(r *common.Record, buff []byte) {
	copy(buff, r.ID)

	binary.LittleEndian.PutUint64(buff[16:24], r.Start)
	binary.LittleEndian.PutUint32(buff[24:], r.Length)
}

// UnmarshalRecord creates a new record from the contents ofa byte slice
func UnmarshalRecord(buff []byte) *common.Record {
	r := newRecord()

	copy(r.ID, buff[:16])
	r.Start = binary.LittleEndian.Uint64(buff[16:24])
	r.Length = binary.LittleEndian.Uint32(buff[24:])

	return r
}

func newRecord() *common.Record {
	return &common.Record{
		ID:     make([]byte, 16), // 128 bits
		Start:  0,
		Length: 0,
	}
}
