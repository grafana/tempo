package v1

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
	dataReader common.DataReader

	pool             ReaderPool
	compressedReader io.Reader

	buffer                []byte
	compressedPagesBuffer [][]byte
}

// NewDataReader creates a datareader that supports compression
func NewDataReader(r backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	return NewNestedDataReader(v0.NewDataReader(r), encoding)
}

// NewNestedDataReader is useful for nesting compression inside of a different reader
func NewNestedDataReader(r common.DataReader, encoding backend.Encoding) (common.DataReader, error) {
	pool, err := getReaderPool(encoding)
	if err != nil {
		return nil, err
	}

	return &dataReader{
		dataReader: r,
		pool:       pool,
	}, nil
}

// Read returns the pages requested in the passed records.  It
// assumes that if there are multiple records they are ordered
// and contiguous
func (r *dataReader) Read(ctx context.Context, records []common.Record, pagesBuffer [][]byte, buffer []byte) ([][]byte, []byte, error) {
	var err error
	r.compressedPagesBuffer, buffer, err = r.dataReader.Read(ctx, records, r.compressedPagesBuffer, buffer)
	if err != nil {
		return nil, nil, err
	}

	// Reset/resize buffer
	if cap(pagesBuffer) < len(r.compressedPagesBuffer) {
		pagesBuffer = make([][]byte, len(r.compressedPagesBuffer))
	}
	pagesBuffer = pagesBuffer[:len(r.compressedPagesBuffer)]

	// now decompress
	for i, page := range r.compressedPagesBuffer {
		reader, err := r.getCompressedReader(page)
		if err != nil {
			return nil, nil, err
		}

		pagesBuffer[i], err = tempo_io.ReadAllWithBuffer(reader, len(page), pagesBuffer[i])
		if err != nil {
			return nil, nil, err
		}
		// TODO mdisibio - There is a lot of performance penalty here even with no compression.
		// Investigate further.
		//pagesBuffer[i] = page
	}

	return pagesBuffer, buffer, nil
}

func (r *dataReader) Close() {
	r.dataReader.Close()

	if r.compressedReader != nil {
		r.pool.PutReader(r.compressedReader)
	}
}

// NextPage implements common.DataReader (kind of)
func (r *dataReader) NextPage(buffer []byte) ([]byte, uint32, error) {
	page, pageLen, err := r.dataReader.NextPage(buffer)
	if err != nil {
		return nil, 0, err
	}
	reader, err := r.getCompressedReader(page)
	if err != nil {
		return nil, 0, err
	}
	r.buffer, err = tempo_io.ReadAllWithBuffer(reader, len(page), r.buffer)
	if err != nil {
		return nil, 0, err
	}
	return r.buffer, pageLen, nil
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
