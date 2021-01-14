package v0

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding"
)

type iterator struct {
	reader io.Reader
}

func NewIterator(reader io.Reader) encoding.Iterator {
	return &iterator{
		reader: reader,
	}
}

func (i *iterator) Next() (encoding.ID, []byte, error) {
	return unmarshalObjectFromReader(i.reader)
}
