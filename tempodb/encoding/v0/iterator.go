package v0

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type iterator struct {
	reader io.Reader
}

func NewIterator(reader io.Reader) common.Iterator {
	return &iterator{
		reader: reader,
	}
}

func (i *iterator) Next() (common.ID, []byte, error) {
	return unmarshalObjectFromReader(i.reader)
}

func (i *iterator) Close() {
}
