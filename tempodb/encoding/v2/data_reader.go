package v2

import (
	"bytes"
	"context"
	"fmt"
	"io"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dataReader struct {
	contextReader backend.ContextReader

	pageBuffer []byte

	pool                  ReaderPool
	compressedReader      io.Reader
	compressedPagesBuffer [][]byte
}

// constDataHeader is a singleton data header.  the data header is
//  stateless b/c there are no fields.  to very minorly reduce allocations all
//  data should just use this.
var constDataHeader = &dataHeader{}

// NewDataReader constructs a v2 DataReader that handles paged...reading
func NewDataReader(r backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	pool, err := getReaderPool(encoding)
	if err != nil {
		return nil, err
	}

	return &dataReader{
		contextReader: r,
		pool:          pool,
	}, nil
}

// Read implements common.DataReader
func (r *dataReader) Read(ctx context.Context, records []common.Record, buffer []byte) ([][]byte, []byte, error) {
	if len(records) == 0 {
		return nil, buffer, nil
	}

	start := records[0].Start
	length := uint32(0)
	for _, record := range records {
		length += record.Length
	}

	if cap(buffer) < int(length) {
		buffer = make([]byte, length)
	}
	buffer = buffer[:length]
	_, err := r.contextReader.ReadAt(ctx, buffer, int64(start))
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

	// read and strip page data
	compressedPages := make([][]byte, 0, len(slicePages))
	for _, v0Page := range slicePages {
		page, err := unmarshalPageFromBytes(v0Page, constDataHeader)
		if err != nil {
			return nil, nil, err
		}

		compressedPages = append(compressedPages, page.data)
	}

	// prepare compressed pages buffer
	if cap(r.compressedPagesBuffer) < len(compressedPages) {
		// extend r.compressedPagesBuffer
		diff := len(compressedPages) - cap(r.compressedPagesBuffer)
		r.compressedPagesBuffer = append(r.compressedPagesBuffer[:cap(r.compressedPagesBuffer)], make([][]byte, diff)...)
	} else {
		r.compressedPagesBuffer = r.compressedPagesBuffer[:len(compressedPages)]
	}

	// now decompress
	for i, page := range compressedPages {
		reader, err := r.getCompressedReader(page)
		if err != nil {
			return nil, nil, err
		}

		r.compressedPagesBuffer[i], err = tempo_io.ReadAllWithBuffer(reader, len(page), r.compressedPagesBuffer[i])
		if err != nil {
			return nil, nil, err
		}
	}

	return r.compressedPagesBuffer, buffer, nil
}

func (r *dataReader) Close() {
	if r.compressedReader != nil {
		r.pool.PutReader(r.compressedReader)
	}
}

// NextPage implements common.DataReader
func (r *dataReader) NextPage(buffer []byte) ([]byte, uint32, error) { // jpe what do with buffer?
	reader, err := r.contextReader.Reader()
	if err != nil {
		return nil, 0, err
	}

	page, err := unmarshalPageFromReader(reader, constDataHeader, r.pageBuffer)
	if err != nil {
		return nil, 0, err
	}
	r.pageBuffer = page.data // jpe eep these buffers

	compressedReader, err := r.getCompressedReader(page.data)
	if err != nil {
		return nil, 0, err
	}

	buffer, err = tempo_io.ReadAllWithBuffer(compressedReader, len(page.data), buffer) // jpe eep these buffers
	if err != nil {
		return nil, 0, err
	}
	return buffer, page.totalLength, nil

}

func (r *dataReader) getCompressedReader(page []byte) (io.Reader, error) {
	var err error
	if r.compressedReader == nil {
		r.compressedReader, err = r.pool.GetReader(bytes.NewReader(page))
	} else {
		r.compressedReader, err = r.pool.ResetReader(bytes.NewReader(page), r.compressedReader)
	}
	return r.compressedReader, err
}
