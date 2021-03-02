package v1

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

type pageReader struct {
	v0PageReader common.PageReader

	pool             ReaderPool
	compressedReader io.Reader
}

// NewPageReader constructs a v1 PageReader that handles compression
func NewPageReader(r backend.ReaderAtContext, encoding backend.Encoding) (common.PageReader, error) {
	pool, err := getReaderPool(encoding)
	if err != nil {
		return nil, err
	}

	return &pageReader{
		v0PageReader: v0.NewPageReader(r),
		pool:         pool,
	}, nil
}

// Read returns the pages requested in the passed records.  It
// assumes that if there are multiple records they are ordered
// and contiguous
func (r *pageReader) Read(ctx context.Context, records []*common.Record) ([][]byte, error) {
	compressedPages, err := r.v0PageReader.Read(ctx, records)
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

func (r *pageReader) Close() {
	r.v0PageReader.Close()

	if r.compressedReader != nil {
		r.pool.PutReader(r.compressedReader)
	}
}
