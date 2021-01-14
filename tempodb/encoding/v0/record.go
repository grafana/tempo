package v0

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb/encoding"
)

const recordLength = 28 // 28 = 128 bit ID, 64bit start, 32bit length

type recordSorter struct {
	records []*encoding.Record
}

func sortRecords(records []*encoding.Record) {
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

// todo: move encoding/decoding to a separate util area?  is the index too large?  need an io.Reader?
func MarshalRecords(records []*encoding.Record) ([]byte, error) {
	recordBytes := make([]byte, len(records)*recordLength)

	for i, r := range records {
		buff := recordBytes[i*recordLength : (i+1)*recordLength]

		if !validation.ValidTraceID(r.ID) { // todo: remove this check.  maybe have a max id size of 128 bits?
			return nil, fmt.Errorf("Ids must be 128 bit")
		}

		marshalRecord(r, buff)
	}

	return recordBytes, nil
}

func UnmarshalRecords(recordBytes []byte) ([]*encoding.Record, error) {
	mod := len(recordBytes) % recordLength
	if mod != 0 {
		return nil, fmt.Errorf("records are an unexpected number of bytes %d", mod)
	}

	numRecords := RecordCount(recordBytes)
	records := make([]*encoding.Record, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		buff := recordBytes[i*recordLength : (i+1)*recordLength]

		r := unmarshalRecord(buff)

		records = append(records, r)
	}

	return records, nil
}

// binary search the bytes.  records are not compressed and ordered
func FindRecord(id encoding.ID, recordBytes []byte) (*encoding.Record, error) {
	mod := len(recordBytes) % recordLength
	if mod != 0 {
		return nil, fmt.Errorf("records are an unexpected number of bytes %d", mod)
	}

	numRecords := RecordCount(recordBytes)
	var record *encoding.Record

	i := sort.Search(numRecords, func(i int) bool {
		buff := recordBytes[i*recordLength : (i+1)*recordLength]
		record = unmarshalRecord(buff)

		return bytes.Compare(record.ID, id) >= 0
	})

	if i >= 0 && i < numRecords {
		buff := recordBytes[i*recordLength : (i+1)*recordLength]
		record = unmarshalRecord(buff)

		return record, nil
	}

	return nil, nil
}

func RecordCount(b []byte) int {
	return len(b) / recordLength
}

func marshalRecord(r *encoding.Record, buff []byte) {
	copy(buff, r.ID)

	binary.LittleEndian.PutUint64(buff[16:24], r.Start)
	binary.LittleEndian.PutUint32(buff[24:], r.Length)
}

func unmarshalRecord(buff []byte) *encoding.Record {
	r := newRecord()

	copy(r.ID, buff[:16])
	r.Start = binary.LittleEndian.Uint64(buff[16:24])
	r.Length = binary.LittleEndian.Uint32(buff[24:])

	return r
}

func newRecord() *encoding.Record {
	return &encoding.Record{
		ID:     make([]byte, 16), // 128 bits
		Start:  0,
		Length: 0,
	}
}
