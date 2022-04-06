package v2

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type idIterator struct {
	dr     common.DataReader
	or     common.ObjectReaderWriter
	iter   Iterator
	buffer []byte
}

var _ common.IDIterator = (*idIterator)(nil)

func NewIDIterator(cr backend.ContextReader, enc backend.Encoding) (*idIterator, error) {
	dr, err := NewDataReader(cr, enc)
	if err != nil {
		return nil, fmt.Errorf("error creating data reader: %w", err)
	}

	i := &idIterator{
		dr: dr,
		or: NewObjectReaderWriter(),
	}
	return i, nil
}

func (i *idIterator) Next(ctx context.Context) (id common.ID, err error) {

	// Read from current iterator until done
	if i.iter != nil {
		id, _, err = i.iter.Next(ctx)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading from iterator: %w", err)
		}
		if id != nil {
			return id, nil
		}
	}

	// Get next page/iterator
	i.buffer, _, err = i.dr.NextPage(i.buffer)
	if err == io.EOF {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("error reading page from datareader: %w", err)
	}

	i.iter = NewIterator(bytes.NewReader(i.buffer), i.or)

	id, _, err = i.iter.Next(ctx)
	return id, err
}

func (i *idIterator) Close() {
	i.dr.Close()
}
