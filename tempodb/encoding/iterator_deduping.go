package encoding

import (
	"bytes"
	"io"
)

type ObjectCombiner interface {
	Combine(objA []byte, objB []byte) []byte
}

type dedupingIterator struct {
	iter          Iterator
	combiner      ObjectCombiner
	currentID     []byte
	currentObject []byte
}

// NewDedupingIterator returns a dedupingIterator.  This iterator is used to wrap another
//  iterator.  It will dedupe consecutive objects with the same id using the ObjectCombiner.
func NewDedupingIterator(iter Iterator, combiner ObjectCombiner) (Iterator, error) {
	i := &dedupingIterator{
		iter:     iter,
		combiner: combiner,
	}

	var err error
	i.currentID, i.currentObject, err = i.iter.Next()
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (i *dedupingIterator) Next() (ID, []byte, error) {
	if i.currentID == nil {
		return nil, nil, io.EOF
	}

	var dedupedID []byte
	var dedupedObject []byte

	for {
		id, obj, err := i.iter.Next()
		if err == io.EOF {
			i.currentID = nil
			i.currentObject = nil
		}
		if err != nil {
			return nil, nil, err
		}

		if !bytes.Equal(i.currentID, id) {
			dedupedID = i.currentID
			dedupedObject = i.currentObject

			i.currentID = id
			i.currentObject = obj
			break
		}

		i.currentID = id
		i.currentObject = i.combiner.Combine(i.currentObject, obj)
	}

	return dedupedID, dedupedObject, nil
}
