package v2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type pagedIterator struct {
	dataReader  DataReader
	indexReader IndexReader
	objectRW    ObjectReaderWriter

	currentIndex int
	maxIndexPage int

	chunkSizeBytes uint32
	pages          [][]byte
	activePage     []byte

	buffer []byte
}

// newPagedIterator returns a backend.Iterator.  This iterator is used to iterate
// through objects stored in object storage.
func newPagedIterator(chunkSizeBytes uint32, indexReader IndexReader, dataReader DataReader, objectRW ObjectReaderWriter) BytesIterator {
	return &pagedIterator{
		dataReader:     dataReader,
		indexReader:    indexReader,
		chunkSizeBytes: chunkSizeBytes,
		objectRW:       objectRW,
		currentIndex:   0,
		maxIndexPage:   math.MaxInt,
	}
}

// newPartialPagedIterator returns a backend.Iterator.  This iterator is used to iterate
// through a contiguous and limited set of pages in object storage.
func newPartialPagedIterator(chunkSizeBytes uint32, indexReader IndexReader, dataReader DataReader, objectRW ObjectReaderWriter, startIndexPage, totalIndexPages int) BytesIterator {
	return &pagedIterator{
		dataReader:     dataReader,
		indexReader:    indexReader,
		chunkSizeBytes: chunkSizeBytes,
		objectRW:       objectRW,
		currentIndex:   startIndexPage,
		maxIndexPage:   startIndexPage + totalIndexPages,
	}
}

// For performance reasons the ID and object slices returned from this method are owned by
// the iterator.  If you have need to keep these values for longer than a single iteration
// you need to make a copy of them.
func (i *pagedIterator) NextBytes(ctx context.Context) (common.ID, []byte, error) {
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
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, fmt.Errorf("error unmarshalling active page, err: %w", err)
	} else if !errors.Is(err, io.EOF) {
		return id, object, nil
	}

	if i.currentIndex >= i.maxIndexPage {
		return nil, nil, io.EOF
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
	records := make([]Record, 0, 5) // 5?  why not?
	for currentRecord != nil {
		// see if we can fit this record in.  we have to get at least one record in
		if (length+currentRecord.Length > i.chunkSizeBytes || i.currentIndex >= i.maxIndexPage) && len(records) != 0 {
			break
		}

		// add currentRecord to the batch
		records = append(records, *currentRecord)
		length += currentRecord.Length

		// get next
		i.currentIndex++
		currentRecord, err = i.indexReader.At(ctx, i.currentIndex)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting next record, err: %w", err)
		}
	}

	i.pages, i.buffer, err = i.dataReader.Read(ctx, records, i.pages, i.buffer)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading objects for records, err: %w", err)
	}
	if len(i.pages) == 0 {
		return nil, nil, fmt.Errorf("unexpected 0 length pages in pagedIterator, err: %w", err)
	}

	i.activePage = i.pages[0]
	i.pages = i.pages[1:] // advance pages

	// attempt to get next object from objects
	i.activePage, id, object, err = i.objectRW.UnmarshalAndAdvanceBuffer(i.activePage)
	if err != nil {
		return nil, nil, fmt.Errorf("error unmarshalling active page from new records, err: %w", err)
	}

	return id, object, nil
}

func (i *pagedIterator) Close() {
	i.dataReader.Close()
}
