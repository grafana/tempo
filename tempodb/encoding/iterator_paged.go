package encoding

import (
	"context"
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type pagedIterator struct {
	meta         *backend.BlockMeta
	dataReader   common.DataReader
	indexReader  common.IndexReader
	objectRW     common.ObjectReaderWriter
	currentIndex int

	currentChunkSize  uint32
	maxChunkSizeBytes uint32
	pages             [][]byte
	activePage        []byte

	buffer []byte
}

// newPagedIterator returns a backendIterator.  This iterator is used to iterate
//  through objects stored in object storage.
func newPagedIterator(meta *backend.BlockMeta, chunkSizeBytes uint32, indexReader common.IndexReader, dataReader common.DataReader, objectRW common.ObjectReaderWriter) Iterator {
	return &pagedIterator{
		meta:              meta,
		dataReader:        dataReader,
		indexReader:       indexReader,
		currentChunkSize:  1024,
		maxChunkSizeBytes: chunkSizeBytes,
		objectRW:          objectRW,
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
		return nil, nil, fmt.Errorf("error unmarshalling active page, blockID: %s, err: %w", i.meta.BlockID.String(), err)
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
	records := make([]common.Record, 0, 5) // 5?  why not?
	for currentRecord != nil {
		// see if we can fit this record in.  we have to get at least one record in
		if length+currentRecord.Length > i.currentChunkSize && len(records) != 0 {
			break
		}

		// increase currentChunkSize JPE you added this
		if i.currentChunkSize < i.maxChunkSizeBytes {
			i.currentChunkSize = i.currentChunkSize * 2
			if i.currentChunkSize > i.maxChunkSizeBytes {
				i.currentChunkSize = i.maxChunkSizeBytes
			}
		}

		// add currentRecord to the batch
		records = append(records, *currentRecord)
		length += currentRecord.Length

		// get next
		i.currentIndex++
		currentRecord, err = i.indexReader.At(ctx, i.currentIndex)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting next record, blockID: %s, err: %w", i.meta.BlockID.String(), err)
		}
	}

	i.pages, i.buffer, err = i.dataReader.Read(ctx, records, i.pages, i.buffer)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading objects for records, blockID: %s, err: %w", i.meta.BlockID.String(), err)
	}
	if len(i.pages) == 0 {
		return nil, nil, fmt.Errorf("unexpected 0 length pages in pagedIterator, blockID: %s, err: %w", i.meta.BlockID.String(), err)
	}

	i.activePage = i.pages[0]
	i.pages = i.pages[1:] // advance pages

	// attempt to get next object from objects
	i.activePage, id, object, err = i.objectRW.UnmarshalAndAdvanceBuffer(i.activePage)
	if err != nil {
		return nil, nil, fmt.Errorf("error unmarshalling active page from new records, blockID: %s, err: %w", i.meta.BlockID.String(), err)
	}

	return id, object, nil
}

func (i *pagedIterator) Close() {
	i.dataReader.Close()
}
