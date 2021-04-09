package encoding

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type walIterator struct {
	o common.ObjectReaderWriter
	d common.DataReader

	currentPage []byte
}

// NewWALIterator iterates over pages from a datareader directly like one would for the WAL
func NewWALIterator(d common.DataReader, o common.ObjectReaderWriter) Iterator {
	return &walIterator{
		o: o,
		d: d,
	}
}

func (i *walIterator) Next(_ context.Context) (common.ID, []byte, error) {
	var (
		id  common.ID
		obj []byte
		err error
	)
	i.currentPage, id, obj, err = i.o.UnmarshalAndAdvanceBuffer(i.currentPage)
	if err == io.EOF {
		i.currentPage, err = i.d.NextPage()
		if err != nil {
			return nil, nil, err
		}
		i.currentPage, id, obj, err = i.o.UnmarshalAndAdvanceBuffer(i.currentPage)
	}
	return id, obj, err
}

func (i *walIterator) Close() {
	i.d.Close()
}
