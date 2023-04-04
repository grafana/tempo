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

// ObjectDecoder translates between opaque byte slices and tempopb.Trace
// Object format:
// | uint32 | uint32 | variable length               |
// | start  | end    | marshalled tempopb.TraceBytes |
// start and end are unix epoch seconds. The byte slices in tempopb.TraceBytes are marshalled tempopb.Trace's
func NewObjectDecoder() *ObjectDecoder {
	return staticDecoder
}

func (d *ObjectDecoder) PrepareForRead(obj []byte) (*tempopb.Trace, error) {
	if len(obj) == 0 {
		return &tempopb.Trace{}, nil
	}

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
	return trace, nil
}

func (d *ObjectDecoder) Combine(objs ...[]byte) ([]byte, error) {
	var minStart, maxEnd uint32
	minStart = math.MaxUint32

	c := trace.NewCombiner()
	for i, obj := range objs {
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

		c.ConsumeWithFinal(t, i == len(objs)-1)
	}

	combinedTrace, _ := c.Result()

	traceBytes := &tempopb.TraceBytes{}
	bytes, err := proto.Marshal(combinedTrace)
	if err != nil {
		return nil, fmt.Errorf("error marshaling traceBytes: %w", err)
	}
	traceBytes.Traces = append(traceBytes.Traces, bytes)

	return marshalWithStartEnd(traceBytes, minStart, maxEnd)
}

func (d *ObjectDecoder) FastRange(buff []byte) (uint32, uint32, error) {
	_, start, end, err := stripStartEnd(buff)
	return start, end, err
}
