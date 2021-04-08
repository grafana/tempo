package v1

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dataReader struct {
	dataReader common.DataReader

	pool             ReaderPool
	compressedReader io.Reader
}

// NewDataReader is useful for nesting compression inside of a different reader
func NewDataReader(r common.DataReader, encoding backend.Encoding) (common.DataReader, error) {
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
func (r *dataReader) Read(ctx context.Context, records []*common.Record) ([][]byte, error) {
	compressedPages, err := r.dataReader.Read(ctx, records)
	if err != nil {
		return nil, err
	}

	// now decompress
	decompressedPages := make([][]byte, 0, len(compressedPages))
	for _, page := range compressedPages {
		if r.compressedReader == nil {
			r.compressedReader, err = r.pool.GetReader(bytes.NewReader(page))
		} else {
			r.compressedReader, err = r.pool.ResetReader(bytes.NewReader(page), r.compressedReader)
		}
		if err != nil {
			return nil, err
		}

		page, err := ioutil.ReadAll(r.compressedReader)
		if err != nil {
			return nil, err
		}

		decompressedPages = append(decompressedPages, page)
	}

	return decompressedPages, nil
}

func (r *dataReader) Close() {
	r.dataReader.Close()

	if r.compressedReader != nil {
		r.pool.PutReader(r.compressedReader)
	}
}

// NextPage implements common.DataReader (kind of)
func (r *dataReader) NextPage() ([]byte, error) {
	return nil, common.ErrUnsupported
}
