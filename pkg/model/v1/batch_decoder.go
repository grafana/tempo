package v1

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

type BatchDecoder struct {
}

var batchDecoder = &BatchDecoder{}

// NewBatchDecoder() returns a v1 batch decoder.
func NewBatchDecoder() *BatchDecoder {
	return batchDecoder
}

func (d *BatchDecoder) PrepareForWrite(trace *tempopb.Trace, start uint32, end uint32) ([]byte, error) {
	// v1 encoding doesn't support start/end
	return proto.Marshal(trace)
}

func (d *BatchDecoder) PrepareForRead(batches [][]byte) (*tempopb.Trace, error) {
	// each slice is a marshalled tempopb.Trace, unmarshal and combine
	var combinedTrace *tempopb.Trace
	for _, batch := range batches {
		t := &tempopb.Trace{}
		err := proto.Unmarshal(batch, t)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling trace: %w", err)
		}

		combinedTrace, _ = trace.CombineTraceProtos(combinedTrace, t)
	}

	return combinedTrace, nil
}

func (d *BatchDecoder) ToObject(batches [][]byte) ([]byte, error) {
	// wrap byte slices in a tempopb.TraceBytes and marshal
	wrapper := &tempopb.TraceBytes{
		Traces: append([][]byte(nil), batches...),
	}
	return proto.Marshal(wrapper)
}
