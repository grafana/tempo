package backend

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/google/uuid"
)

type lazyIterator struct {
	tenantID string
	blockID  uuid.UUID
	r        Reader

	indexBuffer    []byte
	objectsBuffer  []byte
	chunkSizeBytes uint32
}

func NewLazyIterator(tenantID string, blockID uuid.UUID, chunkSizeBytes uint32, reader Reader) (Iterator, error) {
	index, err := reader.Index(blockID, tenantID)
	if err != nil {
		return nil, err
	}

	return &lazyIterator{
		tenantID:       tenantID,
		blockID:        blockID,
		r:              reader,
		chunkSizeBytes: chunkSizeBytes,
		indexBuffer:    index,
	}, err
}

func (i *lazyIterator) Next() (ID, []byte, error) {
	var err error

	if len(i.objectsBuffer) == 0 {
		// if no index left, EOF
		if len(i.indexBuffer) == 0 {
			return nil, nil, io.EOF
		}

		// pull next n bytes into objects
		var start uint64
		var length uint32

		start = math.MaxUint64
		for length < i.chunkSizeBytes && len(i.indexBuffer) > 0 {

			var rec *Record
			rec, i.indexBuffer = UnmarshalRecordAndAdvance(i.indexBuffer)

			if start == math.MaxUint64 {
				start = rec.Start
			}
			length += rec.Length
		}

		i.objectsBuffer, err = i.r.Object(i.blockID, i.tenantID, start, length) // jpe : pass in
		if err != nil {
			return nil, nil, err
		}
	}

	// attempt to get next object from objects
	objectReader := bytes.NewReader(i.objectsBuffer)
	id, object, err := UnmarshalObjectFromReader(objectReader) // jpe UnmarshalObjectAndAdvance?s
	if err != nil {
		return nil, nil, err
	}

	// advance the objects buffer
	bytesRead := objectReader.Size() - int64(objectReader.Len())
	if bytesRead < 0 || bytesRead > int64(len(i.objectsBuffer)) {
		return nil, nil, fmt.Errorf("bad object read during compaction")
	}
	i.objectsBuffer = i.objectsBuffer[bytesRead:]

	return id, object, nil
}
