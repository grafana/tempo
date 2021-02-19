package encoding

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
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
	return v0.UnmarshalObjectFromReader(i.reader) // jpe pagereader
}

func (i *iterator) Close() {
}
