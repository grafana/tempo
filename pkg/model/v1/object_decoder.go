package v1

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/decoder"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

const Encoding = "v1"

type ObjectDecoder struct {
}

var staticDecoder = &ObjectDecoder{}

func NewObjectDecoder() *ObjectDecoder {
	return staticDecoder
}

func (d *ObjectDecoder) PrepareForRead(obj []byte) (*tempopb.Trace, error) {
	trace := &tempopb.Trace{}
	traceBytes := &tempopb.TraceBytes{}
	err := proto.Unmarshal(obj, traceBytes)
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

func (d *ObjectDecoder) Combine(objs ...[]byte) ([]byte, error) {
	c := trace.NewCombiner()
	for i, obj := range objs {
		t, err := staticDecoder.PrepareForRead(obj)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling trace: %w", err)
		}

		c.ConsumeWithFinal(t, i == len(obj)-1)
	}
	combinedTrace, _ := c.Result()

	combinedBytes, err := d.Marshal(combinedTrace)
	if err != nil {
		return nil, fmt.Errorf("error marshaling combinedBytes: %w", err)
	}

	return combinedBytes, nil
}

func (d *ObjectDecoder) FastRange([]byte) (uint32, uint32, error) {
	return 0, 0, decoder.ErrUnsupported
}

func (d *ObjectDecoder) Marshal(t *tempopb.Trace) ([]byte, error) {
	traceBytes := &tempopb.TraceBytes{}
	bytes, err := proto.Marshal(t)
	if err != nil {
		return nil, err
	}

	traceBytes.Traces = append(traceBytes.Traces, bytes)

	return proto.Marshal(traceBytes)
}
