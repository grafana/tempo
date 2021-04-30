package v0

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dataReader struct {
	r backend.ContextReader
}

// NewDataReader returns a new v0 dataReader.  A v0 dataReader
// is basically a no-op.  It retrieves the requested byte
// ranges and returns them as is.
// A pages "format" is a contiguous collection of objects
// | -- object -- | -- object -- | ...
func NewDataReader(r backend.ContextReader) common.DataReader {
	return &dataReader{
		r: r,
	}
}

// Read returns the pages requested in the passed records.  It
// assumes that if there are multiple records they are ordered
// and contiguous
func (r *dataReader) Read(ctx context.Context, records []*common.Record, buffer []byte) ([][]byte, []byte, error) {
	if len(records) == 0 {
		return nil, buffer, nil
	}

	start := records[0].Start
	length := uint32(0)
	for _, record := range records {
		length += record.Length
	}

	buffer = make([]byte, length)
	_, err := r.r.ReadAt(ctx, buffer, int64(start))
	if err != nil {
		return nil, nil, err
	}

	slicePages := make([][]byte, 0, len(records))
	cursor := uint32(0)
	previousEnd := uint64(0)
	for _, record := range records {
		end := cursor + record.Length
		if end > uint32(len(buffer)) {
			return nil, nil, fmt.Errorf("record out of bounds while reading pages: %d, %d, %d, %d", cursor, record.Length, end, len(buffer))
		}

		if previousEnd != record.Start && previousEnd != 0 {
			return nil, nil, fmt.Errorf("non-contiguous pages requested from dataReader: %d, %+v", previousEnd, record)
		}

		slicePages = append(slicePages, buffer[cursor:end])
		cursor += record.Length
		previousEnd = record.Start + uint64(record.Length)
	}

	return slicePages, buffer, nil
}

// Close implements common.DataReader
func (r *dataReader) Close() {
}

// NextPage implements common.DataReader
func (r *dataReader) NextPage(buffer []byte) ([]byte, uint32, error) {
	reader, err := r.r.Reader()
	if err != nil {
		return nil, 0, err
	}

	// v0 pages are just single objects. this method will return one object at a time from the encapsulated reader
	var totalLength uint32
	err = binary.Read(reader, binary.LittleEndian, &totalLength)
	if err != nil {
		return nil, 0, err
	}

	if cap(buffer) < int(totalLength) {
		buffer = make([]byte, totalLength)
	} else {
		buffer = buffer[:totalLength]
	}
	binary.LittleEndian.PutUint32(buffer, totalLength)

	_, err = reader.Read(buffer[uint32Size:])
	if err != nil {
		return nil, 0, err
	}

	return buffer, totalLength, nil
}
