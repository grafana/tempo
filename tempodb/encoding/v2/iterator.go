package v2

import (
	"context"
	"io"

	"github.com/grafana/tempo/v2/tempodb/encoding/common"
)

type BytesIterator interface {
	NextBytes(ctx context.Context) (common.ID, []byte, error)
	Close()
}

type iterator struct {
	reader io.Reader
	o      ObjectReaderWriter
}

// NewIterator returns the most basic iterator.  It iterates over
// raw objects.
func NewIterator(reader io.Reader, o ObjectReaderWriter) BytesIterator {
	return &iterator{
		reader: reader,
		o:      o,
	}
}

func (i *iterator) NextBytes(_ context.Context) (common.ID, []byte, error) {
	return i.o.UnmarshalObjectFromReader(i.reader)
}

func (i *iterator) Close() {
}
