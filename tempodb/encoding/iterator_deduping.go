package encoding

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dedupingIterator struct {
	iter          Iterator
	combiner      common.ObjectCombiner
	currentID     []byte
	currentObject []byte
	dataEncoding  string
}

// NewDedupingIterator returns a dedupingIterator.  This iterator is used to wrap another
//  iterator.  It will dedupe consecutive objects with the same id using the ObjectCombiner.
func NewDedupingIterator(iter Iterator, combiner common.ObjectCombiner, dataEncoding string) (Iterator, error) {
	i := &dedupingIterator{
		iter:         iter,
		combiner:     combiner,
		dataEncoding: dataEncoding,
	}

	var err error
	i.currentID, i.currentObject, err = i.iter.Next(context.Background())
	if err != nil && err != io.EOF {
		return nil, err
	}

	return i, nil
}

// jpe - test
func (i *dedupingIterator) Next(ctx context.Context) (common.ID, []byte, error) {
	if i.currentID == nil {
		return nil, nil, io.EOF
	}

	var dedupedID []byte
	currentObjects := [][]byte{i.currentObject}

	for {
		id, obj, err := i.iter.Next(ctx)
		if err != nil && err != io.EOF {
			return nil, nil, err
		}

		if !bytes.Equal(i.currentID, id) {
			dedupedID = i.currentID

			i.currentID = id
			i.currentObject = obj
			break
		}

		i.currentID = id
		currentObjects = append(currentObjects, obj)
	}

	if len(currentObjects) == 1 {
		return dedupedID, currentObjects[0], nil
	}

	dedupedObject, _, err := i.combiner.Combine(i.dataEncoding, currentObjects...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to combine while Nexting: %w", err)
	}

	return dedupedID, dedupedObject, nil
}

func (i *dedupingIterator) Close() {
	i.iter.Close()
}
