package v0

import (
	"context"
	"io"
	"math"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

type backendIterator struct {
	tenantID string
	blockID  uuid.UUID
	r        backend.Reader

	indexBuffer         []byte
	objectsBuffer       []byte
	activeObjectsBuffer []byte
}

// NewBackendIterator returns a backendIterator.  This iterator is used to iterate
//  through objects stored in object storage.
func NewBackendIterator(tenantID string, blockID uuid.UUID, chunkSizeBytes uint32, reader backend.Reader) (common.Iterator, error) {
	index, err := reader.Read(context.TODO(), nameIndex, blockID, tenantID)
	if err != nil {
		return nil, err
	}

	return &backendIterator{
		tenantID:      tenantID,
		blockID:       blockID,
		r:             reader,
		indexBuffer:   index,
		objectsBuffer: make([]byte, chunkSizeBytes),
	}, err
}

// For performance reasons the ID and object slices returned from this method are owned by
// the iterator.  If you have need to keep these values for longer than a single iteration
// you need to make a copy of them.
func (i *backendIterator) Next() (common.ID, []byte, error) {
	var err error
	var id common.ID
	var object []byte

	i.activeObjectsBuffer, id, object, err = unmarshalAndAdvanceBuffer(i.activeObjectsBuffer)
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
	var start uint64
	var length uint32

	start = math.MaxUint64
	for len(i.indexBuffer) > 0 {
		record := unmarshalRecord(i.indexBuffer[:recordLength])

		// see if we can fit this record in.  we have to get at least one record in
		if length+record.Length > uint32(len(i.objectsBuffer)) && start != math.MaxUint64 {
			break
		}
		// advance index buffer
		i.indexBuffer = i.indexBuffer[recordLength:]

		if start == math.MaxUint64 {
			start = record.Start
		}
		length += record.Length
	}
	if length > uint32(len(i.objectsBuffer)) {
		i.objectsBuffer = make([]byte, length)
	}
	i.activeObjectsBuffer = i.objectsBuffer[:length]
	err = i.r.ReadRange(context.TODO(), nameObjects, i.blockID, i.tenantID, start, i.activeObjectsBuffer)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error iterating through object in backend")
	}

	// attempt to get next object from objects
	i.activeObjectsBuffer, id, object, err = unmarshalAndAdvanceBuffer(i.activeObjectsBuffer)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error iterating through object in backend")
	}

	return id, object, nil
}
