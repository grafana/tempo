package v2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/grafana/tempo/v2/pkg/model"
	"github.com/grafana/tempo/v2/pkg/model/trace"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	"github.com/grafana/tempo/v2/tempodb/encoding/common"
)

type dedupingIterator struct {
	iter          BytesIterator
	combiner      model.ObjectCombiner
	currentID     []byte
	currentObject []byte
	dataEncoding  string              // used for .NextBytes()
	decoder       model.ObjectDecoder // used for .Next()
}

// NewDedupingIterator returns a dedupingIterator.  This iterator is used to wrap another
// iterator.  It will dedupe consecutive objects with the same id using the ObjectCombiner.
func NewDedupingIterator(iter BytesIterator, combiner model.ObjectCombiner, dataEncoding string) (BytesIterator, error) {
	var decoder model.ObjectDecoder
	var err error
	if dataEncoding != "" {
		decoder, err = model.NewObjectDecoder(dataEncoding)
		if err != nil {
			return nil, err
		}
	}

	i := &dedupingIterator{
		iter:         iter,
		combiner:     combiner,
		dataEncoding: dataEncoding,
		decoder:      decoder,
	}

	i.currentID, i.currentObject, err = i.iter.NextBytes(context.Background())
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	return i, nil
}

// Next implements BytesIterator
func (i *dedupingIterator) NextBytes(ctx context.Context) (common.ID, []byte, error) {
	dedupedID, currentObjects, err := i.next(ctx)
	if err != nil {
		return nil, nil, err
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

// Next implements common.Iterator
func (i *dedupingIterator) Next(ctx context.Context) (common.ID, *tempopb.Trace, error) {
	if i.decoder == nil {
		return nil, nil, fmt.Errorf("dedupingIterator.Next() called but no decoder set")
	}

	dedupedID, currentObjects, err := i.next(ctx)
	if err != nil {
		return nil, nil, err
	}

	if len(currentObjects) == 1 {
		tr, err := i.decoder.PrepareForRead(currentObjects[0])
		if err != nil {
			return nil, nil, err
		}

		return dedupedID, tr, nil
	}

	combiner := trace.NewCombiner(0)
	for j, obj := range currentObjects {
		tr, err := i.decoder.PrepareForRead(obj)
		if err != nil {
			return nil, nil, err
		}

		_, err = combiner.ConsumeWithFinal(tr, j == len(currentObjects)-1)
		if err != nil {
			return nil, nil, err
		}
	}

	tr, _ := combiner.Result()

	return dedupedID, tr, nil
}

func (i *dedupingIterator) next(ctx context.Context) (common.ID, [][]byte, error) {
	if i.currentID == nil {
		return nil, nil, io.EOF
	}

	var dedupedID []byte
	currentObjects := [][]byte{i.currentObject}

	for {
		id, obj, err := i.iter.NextBytes(ctx)
		if err != nil && !errors.Is(err, io.EOF) {
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

	return dedupedID, currentObjects, nil
}

// Close implements Iterator
func (i *dedupingIterator) Close() {
	i.iter.Close()
}
