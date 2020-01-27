package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/grafana/frigg/pkg/util/validation"
)

type ID []byte

type Record struct {
	ID     []byte
	Start  uint64
	Length uint32
}

type recordSorter struct {
	records []*Record
}

func SortRecords(records []*Record) {
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

// todo: move encoding/decoding to a seperate util area?  is the index too large?  need an io.Reader?
func MarshalRecords(records []*Record, bloomFP float64) ([]byte, []byte, error) {
	recordBytes := make([]byte, len(records)*28) // 28 = 128 bit ID, 64bit start, 32bit length
	bf := bloom.NewBloomFilter(float64(len(records)), bloomFP)

	for i, r := range records {
		buff := recordBytes[i*28 : (i+1)*28]

		if !validation.ValidTraceID(r.ID) {
			return nil, nil, fmt.Errorf("Trace Ids must be 128 bit")
		}

		bf.Add(farm.Fingerprint64(r.ID))
		encodeRecord(r, buff)
	}

	bloomBytes := bf.JSONMarshal()

	return recordBytes, bloomBytes, nil
}

func UnmarshalRecords(recordBytes []byte) ([]*Record, error) {
	numRecords := len(recordBytes) / 28
	records := make([]*Record, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		buff := recordBytes[i*28 : (i+1)*28]

		r := newRecord()
		decodeRecord(buff, r)

		records = append(records, r)
	}

	return records, nil
}

// binary search the bytes.  records are not compressed and ordered
func FindRecord(id ID, recordBytes []byte) (*Record, error) {
	numRecords := len(recordBytes) / 28
	record := newRecord()

	i := sort.Search(numRecords, func(i int) bool {
		buff := recordBytes[i*28 : (i+1)*28]
		decodeRecord(buff, record)

		return bytes.Compare(record.ID, id) >= 0
	})

	if i >= 0 && i < numRecords {
		buff := recordBytes[i*28 : (i+1)*28]
		decodeRecord(buff, record)

		if bytes.Equal(id, record.ID) {
			return record, nil
		}
	}

	return nil, nil
}

func encodeRecord(r *Record, buff []byte) {
	copy(buff, r.ID)

	binary.LittleEndian.PutUint64(buff[16:24], r.Start)
	binary.LittleEndian.PutUint32(buff[24:], r.Length)
}

func decodeRecord(buff []byte, r *Record) {
	copy(r.ID, buff[:16])
	r.Start = binary.LittleEndian.Uint64(buff[16:24])
	r.Length = binary.LittleEndian.Uint32(buff[24:])
}

func newRecord() *Record {
	return &Record{
		ID:     make([]byte, 16), // 128 bits
		Start:  0,
		Length: 0,
	}
}
