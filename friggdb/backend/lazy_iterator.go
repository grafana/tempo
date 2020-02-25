package backend

import (
	"io"
	"math"

	"github.com/google/uuid"
)

type lazyIterator struct {
	tenantID string
	blockID  uuid.UUID
	r        Reader

	indexBuffer   []byte
	objectsBuffer []byte
}

func NewLazyIterator(tenantID string, blockID uuid.UUID, chunkSizeBytes uint32, reader Reader) (Iterator, error) {
	index, err := reader.Index(blockID, tenantID)
	if err != nil {
		return nil, err
	}

	return &lazyIterator{
		tenantID:    tenantID,
		blockID:     blockID,
		r:           reader,
		indexBuffer: index,
	}, err
}

func (i *lazyIterator) Next() (ID, []byte, error) {
	var err error
	var id ID
	var object []byte

	i.objectsBuffer, id, object, err = unmarshalAndAdvanceBuffer(i.objectsBuffer)
	if err != nil && err != io.EOF {
		return nil, nil, err
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
	i.objectsBuffer = make([]byte, length)
	err = i.r.Object(i.blockID, i.tenantID, start, i.objectsBuffer)
	if err != nil {
		return nil, nil, err
	}

	// attempt to get next object from objects
	i.objectsBuffer, id, object, err = unmarshalAndAdvanceBuffer(i.objectsBuffer)
	if err != nil {
		return nil, nil, err
	}

	return id, object, nil
}
