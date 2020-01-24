package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/grafana/frigg/pkg/util/validation"
	"github.com/willf/bloom"
)

type TraceID []byte

type TraceRecord struct {
	TraceID []byte
	Start   uint64
	Length  uint32
}

type traceRecordSorter struct {
	records []*TraceRecord
}

func SortRecords(records []*TraceRecord) {
	sort.Sort(&traceRecordSorter{
		records: records,
	})
}

func (t *traceRecordSorter) Len() int {
	return len(t.records)
}

func (t *traceRecordSorter) Less(i, j int) bool {
	a := t.records[i]
	b := t.records[j]

	return bytes.Compare(a.TraceID, b.TraceID) == -1
}

func (t *traceRecordSorter) Swap(i, j int) {
	t.records[i], t.records[j] = t.records[j], t.records[i]
}

// todo: move encoding/decoding to a seperate util area?  is the index too large?  need an io.Reader?
func EncodeRecords(records []*TraceRecord) ([]byte, []byte, error) {
	// 28 = 128 bit traceid, 64bit start, 32bit length
	recordBytes := make([]byte, len(records)*28)
	// todo: choose a bloom filter size to hit a configurable false positive rate
	//   the following assumes 1000 traces/second cut every 2 hours.  n = 7200000 with a target fp rate of 1%
	//   https://hur.st/bloomfilter/ m = 69069274, k = 7
	bloom := bloom.New(69069274, 7)

	for i, r := range records {
		buff := recordBytes[i*28 : (i+1)*28]

		if !validation.ValidTraceID(r.TraceID) {
			return nil, nil, fmt.Errorf("Trace Ids must be 128 bit")
		}

		bloom.Add(r.TraceID)
		encodeRecord(r, buff)
	}

	bloomBytes, err := bloom.GobEncode()
	if err != nil {
		return nil, nil, err
	}

	return recordBytes, bloomBytes, nil
}

func DecodeRecords(recordBytes []byte) ([]*TraceRecord, error) {
	numRecords := len(recordBytes) / 28
	records := make([]*TraceRecord, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		buff := recordBytes[i*28 : (i+1)*28]

		r := newTraceRecord()
		decodeRecord(buff, r)

		records = append(records, r)
	}

	return records, nil
}

// binary search the bytes.  records are not compressed and ordered
func FindRecord(id TraceID, recordBytes []byte) (*TraceRecord, error) {
	numRecords := len(recordBytes) / 28
	record := newTraceRecord()

	i := sort.Search(numRecords, func(i int) bool {
		buff := recordBytes[i*28 : (i+1)*28]
		decodeRecord(buff, record)

		return bytes.Compare(record.TraceID, id) >= 0
	})

	if i >= 0 && i < numRecords {
		buff := recordBytes[i*28 : (i+1)*28]
		decodeRecord(buff, record)

		if bytes.Equal(id, record.TraceID) {
			return record, nil
		}
	}

	return nil, nil
}

func encodeRecord(r *TraceRecord, buff []byte) {
	copy(buff, r.TraceID)

	binary.LittleEndian.PutUint64(buff[16:24], r.Start)
	binary.LittleEndian.PutUint32(buff[24:], r.Length)
}

func decodeRecord(buff []byte, r *TraceRecord) {
	copy(r.TraceID, buff[:16])
	r.Start = binary.LittleEndian.Uint64(buff[16:24])
	r.Length = binary.LittleEndian.Uint32(buff[24:])
}

func newTraceRecord() *TraceRecord {
	return &TraceRecord{
		TraceID: make([]byte, 16), // 128 bits
		Start:   0,
		Length:  0,
	}
}
