package v2

import (
	"bytes"
	"context"
	"fmt"
	"io"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/klauspost/compress/zstd"
)

type dataReader struct {
	contextReader backend.ContextReader

	pageBuffer []byte

	pool             ReaderPool
	compressedReader io.Reader
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
func (r *dataReader) Read(ctx context.Context, records []common.Record, pagesBuffer [][]byte, buffer []byte) ([][]byte, []byte, error) {
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

	compressedPagesBuffer := make([][]byte, len(records))

	cursor := uint32(0)
	previousEnd := uint64(0)
	for i, record := range records {
		end := cursor + record.Length
		if end > uint32(len(buffer)) {
			return nil, nil, fmt.Errorf("record out of bounds while reading pages: %d, %d, %d, %d", cursor, record.Length, end, len(buffer))
		}

		if previousEnd != record.Start && previousEnd != 0 {
			return nil, nil, fmt.Errorf("non-contiguous pages requested from dataReader: %d, %+v", previousEnd, record)
		}

		compressedPagesBuffer[i] = buffer[cursor:end]
		cursor += record.Length
		previousEnd = record.Start + uint64(record.Length)
	}

	// read and strip page data
	compressedPages := make([][]byte, 0, len(compressedPagesBuffer))
	for _, v0Page := range compressedPagesBuffer {
		page, err := unmarshalPageFromBytes(v0Page, constDataHeader)
		if err != nil {
			return nil, nil, err
		}

		compressedPages = append(compressedPages, page.data)
	}

	// prepare pagesBuffer
	if cap(pagesBuffer) < len(compressedPages) {
		// extend pagesBuffer
		diff := len(compressedPages) - cap(pagesBuffer)
		pagesBuffer = append(pagesBuffer[:cap(pagesBuffer)], make([][]byte, diff)...)
	} else {
		pagesBuffer = pagesBuffer[:len(compressedPages)]
	}

	// now decompress
	for i, page := range compressedPages {
		reader, err := r.getCompressedReader(page)
		if err != nil {
			return nil, nil, err
		}

		decoder, ok := reader.(*zstd.Decoder)
		if ok {
			pagesBuffer[i], err = decoder.DecodeAll(page, pagesBuffer[i][:0])
		} else {
			pagesBuffer[i], err = tempo_io.ReadAllWithBuffer(reader, len(page), pagesBuffer[i])
		}
		if err != nil {
			return nil, nil, err
		}
	}

	return pagesBuffer, buffer, nil
}

func (r *dataReader) Close() {
	if r.compressedReader != nil {
		r.pool.PutReader(r.compressedReader)
	}
}

// NextPage implements common.DataReader
func (r *dataReader) NextPage(buffer []byte) ([]byte, uint32, error) {
	reader, err := r.contextReader.Reader()
	if err != nil {
		return nil, 0, err
	}

	page, err := unmarshalPageFromReader(reader, constDataHeader, r.pageBuffer)
	if err != nil {
		return nil, 0, err
	}
	r.pageBuffer = page.data

	compressedReader, err := r.getCompressedReader(page.data)
	if err != nil {
		return nil, 0, err
	}

	decoder, ok := reader.(*zstd.Decoder)
	if ok {
		buffer, err = decoder.DecodeAll(page.data, buffer[:0])
	} else {
		buffer, err = tempo_io.ReadAllWithBuffer(compressedReader, len(page.data), buffer)
	}

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
