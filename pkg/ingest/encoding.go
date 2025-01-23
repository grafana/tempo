// Package kafka provides encoding and decoding functionality for Tempo's Kafka integration.
package ingest

import (
	"errors"
	"fmt"
	math_bits "math/bits"
	"sync"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/twmb/franz-go/pkg/kgo"
)

var encoderPool = sync.Pool{
	New: func() any {
		return &tempopb.PushBytesRequest{}
	},
}

func encoderPoolGet() *tempopb.PushBytesRequest {
	x := encoderPool.Get()
	if x != nil {
		return x.(*tempopb.PushBytesRequest)
	}

	return &tempopb.PushBytesRequest{
		Traces: make([]tempopb.PreallocBytes, 0, 10),
		Ids:    make([][]byte, 0, 10),
	}
}

func encoderPoolPut(req *tempopb.PushBytesRequest) {
	req.Traces = req.Traces[:0]
	req.Ids = req.Ids[:0]
	encoderPool.Put(req)
}

func Encode(partitionID int32, tenantID string, req *tempopb.PushBytesRequest, maxSize int) ([]*kgo.Record, error) {
	reqSize := req.Size()

	// Fast path for small requests
	if reqSize <= maxSize {
		rec, err := marshalWriteRequestToRecord(partitionID, tenantID, req)
		if err != nil {
			return nil, err
		}
		return []*kgo.Record{rec}, nil
	}

	var records []*kgo.Record
	batch := encoderPoolGet()
	defer encoderPoolPut(batch)

	currentSize := 0

	for i, entry := range req.Traces {
		l := entry.Size() + len(req.Ids[i])
		// Size of the entry in the req
		entrySize := 1 + l + sovPush(uint64(l))

		// Check if a single entry is too big
		if entrySize > maxSize || (i == 0 && currentSize+entrySize > maxSize) {
			return nil, fmt.Errorf("single entry size (%d) exceeds maximum allowed size (%d)", entrySize, maxSize)
		}

		if currentSize+entrySize > maxSize {
			// Current req is full, create a record and start a new req
			if len(batch.Traces) > 0 {
				rec, err := marshalWriteRequestToRecord(partitionID, tenantID, batch)
				if err != nil {
					return nil, err
				}
				records = append(records, rec)
			}
			// Reset currentStream
			batch.Traces = batch.Traces[:0]
			batch.Ids = batch.Ids[:0]
			currentSize = 0
		}
		batch.Traces = append(batch.Traces, entry)
		batch.Ids = append(batch.Ids, req.Ids[i])
		currentSize += entrySize
	}

	// Handle any remaining entries
	if len(batch.Traces) > 0 {
		rec, err := marshalWriteRequestToRecord(partitionID, tenantID, batch)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	if len(records) == 0 {
		return nil, errors.New("no valid records created")
	}

	return records, nil
}

// marshalWriteRequestToRecord converts a PushBytesRequest to a Kafka record.
func marshalWriteRequestToRecord(partitionID int32, tenantID string, req *tempopb.PushBytesRequest) (*kgo.Record, error) {
	data, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	return &kgo.Record{
		Key:       []byte(tenantID),
		Value:     data,
		Partition: partitionID,
	}, nil
}

// Decoder is responsible for decoding Kafka record data back into logproto.Stream format.
// It caches parsed labels for efficiency.
type Decoder struct {
	req *tempopb.PushBytesRequest
}

func NewDecoder() *Decoder {
	return &Decoder{
		req: &tempopb.PushBytesRequest{}, // TODO - Pool?
	}
}

// Decode converts a Kafka record's byte data back into a tempopb.Trace.
func (d *Decoder) Decode(data []byte) (*tempopb.PushBytesRequest, error) {
	err := d.req.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}
	return d.req, nil
}

func (d *Decoder) Reset() {
	// Retain slice capacity
	d.req.Ids = d.req.Ids[:0]
	d.req.Traces = d.req.Traces[:0]
}

// sovPush calculates the size of varint-encoded uint64.
// It is used to determine the number of bytes needed to encode an uint64 value
// in Protocol Buffers' variable-length integer format.
func sovPush(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
