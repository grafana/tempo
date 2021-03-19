package encoding

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

type pagedIterator struct {
	dataReader   common.DataReader
	indexReader  common.IndexReader
	objectRW     common.ObjectReaderWriter
	currentIndex int

	chunkSizeBytes uint32
	pages          [][]byte
	activePage     []byte
}

// newPagedIterator returns a backendIterator.  This iterator is used to iterate
//  through objects stored in object storage.
func newPagedIterator(chunkSizeBytes uint32, indexReader common.IndexReader, dataReader common.DataReader, objectRW common.ObjectReaderWriter) Iterator {
	return &pagedIterator{
		dataReader:     dataReader,
		indexReader:    indexReader,
		chunkSizeBytes: chunkSizeBytes,
		objectRW:       objectRW,
	}
}

// For performance reasons the ID and object slices returned from this method are owned by
// the iterator.  If you have need to keep these values for longer than a single iteration
// you need to make a copy of them.
func (i *pagedIterator) Next(ctx context.Context) (common.ID, []byte, error) {
	var err error
	var id common.ID
	var object []byte

	// if the current page is empty advance to the next one
	if len(i.activePage) == 0 && len(i.pages) > 0 {
		i.activePage = i.pages[0]
		i.pages = i.pages[1:] // advance pages
	}

	// dataReader returns pages in the raw format, so this works
	i.activePage, id, object, err = i.objectRW.UnmarshalAndAdvanceBuffer(i.activePage)
	if err != nil && err != io.EOF {
		return nil, nil, errors.Wrap(err, "error iterating through object in backend")
	} else if err != io.EOF {
		return id, object, nil
	}

	// objects reader was empty, check the index
	// if no index left, EOF
	currentRecord, err := i.indexReader.At(ctx, i.currentIndex)
	if err != nil {
		return nil, nil, err
	}
	if currentRecord == nil {
		return nil, nil, io.EOF
	}

	// pull next n bytes into objects
	var length uint32
	records := make([]*common.Record, 0, 5) // 5?  why not?
	for currentRecord != nil {
		// see if we can fit this record in.  we have to get at least one record in
		if length+currentRecord.Length > i.chunkSizeBytes && len(records) != 0 {
			break
		}

		// add currentRecord to the batch
		records = append(records, currentRecord)
		length += currentRecord.Length

		// get next
		i.currentIndex++
		currentRecord, err = i.indexReader.At(ctx, i.currentIndex)
		if err != nil {
			return nil, nil, err
		}
	}

	i.pages, err = i.dataReader.Read(ctx, records)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error iterating through object in backend")
	}
	if len(i.pages) == 0 {
		return nil, nil, errors.Wrap(err, "unexpected 0 length pages in pagedIterator")
	}

	i.activePage = i.pages[0]
	i.pages = i.pages[1:] // advance pages

	// attempt to get next object from objects
	i.activePage, id, object, err = i.objectRW.UnmarshalAndAdvanceBuffer(i.activePage)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error iterating through object in backend")
	}

	return id, object, nil
}

func (i *pagedIterator) Close() {
	i.dataReader.Close()
}
