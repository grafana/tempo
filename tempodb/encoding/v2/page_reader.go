package v2

import (
	"context"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
)

type pageReader struct {
	v0PageReader common.PageReader
}

// NewPageReader constructs a v2 PageReader that handles paged...reading
func NewPageReader(r backend.ContextReader, encoding backend.Encoding) (common.PageReader, error) {
	v2PageReader := &pageReader{
		v0PageReader: v0.NewPageReader(r),
	}

	v1PageReader, err := v1.NewPageReaderWithReader(v2PageReader, encoding)
	if err != nil {
		return nil, err
	}

	return v1PageReader, nil
}

func (r *pageReader) Read(ctx context.Context, records []*common.Record) ([][]byte, error) {
	v0Pages, err := r.v0PageReader.Read(ctx, records)
	if err != nil {
		return nil, err
	}

	pages := make([][]byte, 0, len(v0Pages))
	for _, v0Page := range v0Pages {
		page, err := unmarshalPageFromBytes(v0Page)
		if err != nil {
			return nil, err
		}

		pages = append(pages, page.data)
	}

	return pages, nil
}

func (r *pageReader) Close() {
	r.v0PageReader.Close()
}
