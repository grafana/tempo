package encoding

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type recordlessIterator struct {
	o common.ObjectReaderWriter
	d common.DataReader

	currentPage []byte
}

// NewRecordlessIterator iterates over pages from a datareader directly without requiring records
func NewRecordlessIterator(d common.DataReader, o common.ObjectReaderWriter) Iterator {
	return &recordlessIterator{
		o: o,
		d: d,
	}
}

func (i *recordlessIterator) Next(_ context.Context) (common.ID, []byte, error) {
	var (
		id  common.ID
		obj []byte
		err error
	)
	i.currentPage, id, obj, err = i.o.UnmarshalAndAdvanceBuffer(i.currentPage)
	if err == io.EOF {
		i.currentPage, err = i.d.NextPage(i.currentPage)
		if err != nil {
			return nil, nil, err
		}
		i.currentPage, id, obj, err = i.o.UnmarshalAndAdvanceBuffer(i.currentPage)
	}
	return id, obj, err
}

func (i *recordlessIterator) Close() {
	i.d.Close()
}
