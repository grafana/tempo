package encoding

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

type iterator struct {
	reader io.Reader
}

// NewIterator returns the most basic iterator.  It iterates over
// raw objects.
func NewIterator(reader io.Reader) common.Iterator {
	return &iterator{
		reader: reader,
	}
}

func (i *iterator) Next() (common.ID, []byte, error) {
	return v0.UnmarshalObjectFromReader(i.reader)
}

func (i *iterator) Close() {
}
