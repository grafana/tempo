package v1

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

const Encoding = "v1"

type Decoder struct {
}

var staticDecoder = &Decoder{}

func NewDecoder() *Decoder {
	return staticDecoder
}

func (d *Decoder) PrepareForRead(obj []byte) (*tempopb.Trace, error) {
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

func (d *Decoder) Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error) {
	t, err := d.PrepareForRead(obj)
	if err != nil {
		return nil, err
	}

	return trace.MatchesProto(id, t, req)
}

func (d *Decoder) Combine(objs ...[]byte) ([]byte, error) {
	var combinedTrace *tempopb.Trace
	for _, obj := range objs {
		t, err := d.PrepareForRead(obj)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling trace: %w", err)
		}

		combinedTrace, _ = trace.CombineTraceProtos(combinedTrace, t)
	}

	combinedBytes, err := d.Marshal(combinedTrace)
	if err != nil {
		return nil, fmt.Errorf("error marshaling combinedBytes: %w", err)
	}

	return combinedBytes, nil
}

func (d *Decoder) Marshal(t *tempopb.Trace) ([]byte, error) {
	traceBytes := &tempopb.TraceBytes{}
	bytes, err := proto.Marshal(t)
	if err != nil {
		return nil, err
	}

	traceBytes.Traces = append(traceBytes.Traces, bytes)

	return proto.Marshal(traceBytes)
}
