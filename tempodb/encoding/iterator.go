package encoding

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// Iterator is capable of iterating through a set of objects
type Iterator interface {
	Next(context.Context) (common.ID, []byte, error)
	Close()
}

type iterator struct {
	reader io.Reader
}

// NewIterator returns the most basic iterator.  It iterates over
// raw objects.
func NewIterator(reader io.Reader) Iterator {
	return &iterator{
		reader: reader,
	}
}

func (i *iterator) Next(_ context.Context) (common.ID, []byte, error) {
	return v0.UnmarshalObjectFromReader(i.reader)
}

func (i *iterator) Close() {
}
