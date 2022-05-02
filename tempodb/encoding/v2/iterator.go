package v2

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type iterator struct {
	reader io.Reader
	o      common.ObjectReaderWriter
}

// NewIterator returns the most basic iterator.  It iterates over
// raw objects.
func NewIterator(reader io.Reader, o common.ObjectReaderWriter) common.Iterator {
	return &iterator{
		reader: reader,
		o:      o,
	}
}

func (i *iterator) Next(_ context.Context) (common.ID, []byte, error) {
	return i.o.UnmarshalObjectFromReader(i.reader)
}

func (i *iterator) Close() {
}
