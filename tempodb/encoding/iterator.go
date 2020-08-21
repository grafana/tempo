package encoding

import (
	"io"
)

type Iterator interface {
	Next() (ID, []byte, error)
}

type iterator struct {
	reader io.Reader
}

func NewIterator(reader io.Reader) Iterator {
	return &iterator{
		reader: reader,
	}
}

func (i *iterator) Next() (ID, []byte, error) {
	return unmarshalObjectFromReader(i.reader)
}
