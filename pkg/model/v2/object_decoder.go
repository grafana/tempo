package v2

import (
	"fmt"
	"math"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

const Encoding = "v2"

type ObjectDecoder struct {
}

var staticDecoder = &ObjectDecoder{}

func NewObjectDecoder() *ObjectDecoder {
	return staticDecoder
}

func (d *ObjectDecoder) PrepareForRead(obj []byte) (*tempopb.Trace, error) {
	if len(obj) == 0 {
		return &tempopb.Trace{}, nil
	}

	// jpe - start/end time
	obj, _, _, err := stripStartEnd(obj)
	if err != nil {
		return nil, err
	}

	trace := &tempopb.Trace{}
	traceBytes := &tempopb.TraceBytes{}
	err = proto.Unmarshal(obj, traceBytes)
	if err != nil {
		return nil, err
	}

	for _, bytes := range traceBytes.Traces {
		innerTrace := &tempopb.Trace{}
		err = proto.Unmarshal(bytes, innerTrace)
		if err != nil {
			return nil, err
		}

		trace.Batches = append(trace.Batches, innerTrace.Batches...)
	}
	return trace, err
}

func (d *ObjectDecoder) Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error) {
	start, end, err := d.FastRange(obj)
	if err != nil {
		return nil, err
	}

	if !(req.Start <= end && req.End >= start) {
		return nil, nil
	}

	t, err := d.PrepareForRead(obj)
	if err != nil {
		return nil, err
	}

	return trace.MatchesProto(id, t, req)
}

func (d *ObjectDecoder) Combine(objs ...[]byte) ([]byte, error) {
	var minStart, maxEnd uint32
	minStart = math.MaxUint32

	var combinedTrace *tempopb.Trace
	for _, obj := range objs {
		t, err := d.PrepareForRead(obj)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling trace: %w", err)
		}

		if len(obj) != 0 {
			start, end, err := d.FastRange(obj)
			if err != nil {
				return nil, fmt.Errorf("error getting range: %w", err)
			}

			if start < minStart {
				minStart = start
			}
			if end > maxEnd {
				maxEnd = end
			}
		}

		combinedTrace, _ = trace.CombineTraceProtos(combinedTrace, t)
	}

	combinedBytes, err := d.marshal(combinedTrace, minStart, maxEnd)
	if err != nil {
		return nil, fmt.Errorf("error marshaling combinedBytes: %w", err)
	}

	return combinedBytes, nil
}

func (d *ObjectDecoder) FastRange(buff []byte) (uint32, uint32, error) {
	_, start, end, err := stripStartEnd(buff)
	return start, end, err
}

func (d *ObjectDecoder) marshal(t *tempopb.Trace, start, end uint32) ([]byte, error) {
	traceBytes := &tempopb.TraceBytes{}
	bytes, err := proto.Marshal(t)
	if err != nil {
		return nil, err
	}

	traceBytes.Marshal()
	traceBytes.Traces = append(traceBytes.Traces, bytes)

	return marshalWithStartEnd(traceBytes, start, end)
}
