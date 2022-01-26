package v1

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

type SegmentDecoder struct {
}

var segmentDecoder = &SegmentDecoder{}

// NewSegmentDecoder() returns a v1 segment decoder.
func NewSegmentDecoder() *SegmentDecoder {
	return segmentDecoder
}

func (d *SegmentDecoder) PrepareForWrite(trace *tempopb.Trace, start uint32, end uint32) ([]byte, error) {
	// v1 encoding doesn't support start/end
	return proto.Marshal(trace)
}

func (d *SegmentDecoder) PrepareForRead(segments [][]byte) (*tempopb.Trace, error) {
	// each slice is a marshalled tempopb.Trace, unmarshal and combine
	var combinedTrace *tempopb.Trace
	for _, s := range segments {
		t := &tempopb.Trace{}
		err := proto.Unmarshal(s, t)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling trace: %w", err)
		}

		combinedTrace, _ = trace.CombineTraceProtos(combinedTrace, t)
	}

	return combinedTrace, nil
}

func (d *SegmentDecoder) ToObject(segments [][]byte) ([]byte, error) {
	// wrap byte slices in a tempopb.TraceBytes and marshal
	wrapper := &tempopb.TraceBytes{
		Traces: append([][]byte(nil), segments...),
	}
	return proto.Marshal(wrapper)
}
