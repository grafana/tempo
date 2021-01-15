package v0

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/index"
)

type iterator struct {
	reader io.Reader
}

func NewIterator(reader io.Reader) index.Iterator {
	return &iterator{
		reader: reader,
	}
}

func (i *iterator) Next() (index.ID, []byte, error) {
	return unmarshalObjectFromReader(i.reader)
}
