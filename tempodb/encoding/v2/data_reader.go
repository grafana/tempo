package v2

import (
	"bytes"
	"context"
	"io"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

type dataReader struct {
	contextReader backend.ContextReader
	dataReader    common.DataReader

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
		dataReader:    v0.NewDataReader(r),
		pool:          pool,
	}, nil
}

// Read implements common.DataReader
func (r *dataReader) Read(ctx context.Context, records []common.Record, buffer []byte) ([][]byte, []byte, error) {
	v0Pages, buffer, err := r.dataReader.Read(ctx, records, buffer)
	if err != nil {
		return nil, nil, err
	}

	// read and strip page data
	compressedPages := make([][]byte, 0, len(v0Pages))
	for _, v0Page := range v0Pages {
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
	r.dataReader.Close()

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
