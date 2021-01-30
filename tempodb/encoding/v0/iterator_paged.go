package v0

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

type pagedIterator struct {
	r common.PageReader

	chunkSizeBytes uint32
	indexBuffer    []byte
	pages          [][]byte
	activePage     []byte
}

// NewPagedIterator returns a backendIterator.  This iterator is used to iterate
//  through objects stored in object storage.
func NewPagedIterator(chunkSizeBytes uint32, index []byte, pageReader common.PageReader) common.Iterator {
	return &pagedIterator{
		r:              pageReader,
		indexBuffer:    index,
		chunkSizeBytes: chunkSizeBytes,
	}
}

// For performance reasons the ID and object slices returned from this method are owned by
// the iterator.  If you have need to keep these values for longer than a single iteration
// you need to make a copy of them.
func (i *pagedIterator) Next() (common.ID, []byte, error) {
	var err error
	var id common.ID
	var object []byte

	// if the current page is empty advance to the next one
	if len(i.activePage) == 0 && len(i.pages) > 0 {
		i.activePage = i.pages[0]
		i.pages = i.pages[1:] // advance pages
	}

	i.activePage, id, object, err = unmarshalAndAdvanceBuffer(i.activePage)
	if err != nil && err != io.EOF {
		return nil, nil, errors.Wrap(err, "error iterating through object in backend")
	} else if err != io.EOF {
		return id, object, nil
	}

	// objects reader was empty, check the index
	// if no index left, EOF
	if len(i.indexBuffer) == 0 {
		return nil, nil, io.EOF
	}

	// pull next n bytes into objects
	var length uint32
	records := make([]*common.Record, 0, 5) // 5?  why not?
	for len(i.indexBuffer) > 0 {
		record := unmarshalRecord(i.indexBuffer[:recordLength])

		// see if we can fit this record in.  we have to get at least one record in
		if length+record.Length > i.chunkSizeBytes && len(records) != 0 {
			break
		}
		// advance index buffer
		i.indexBuffer = i.indexBuffer[recordLength:]
		records = append(records, record)
		length += record.Length
	}

	i.pages, err = i.r.Read(records)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error iterating through object in backend")
	}
	if len(i.pages) == 0 {
		return nil, nil, errors.Wrap(err, "unexpected 0 length pages in pagedIterator")
	}

	i.activePage = i.pages[0]
	i.pages = i.pages[1:] // advance pages

	// attempt to get next object from objects
	i.activePage, id, object, err = unmarshalAndAdvanceBuffer(i.activePage)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error iterating through object in backend")
	}

	return id, object, nil
}
