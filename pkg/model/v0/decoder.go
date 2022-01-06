package v0

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

const Encoding = ""

type Encoder struct {
}

var staticEncoding = &Encoder{}

func NewEncoding() *Encoder {
	return staticEncoding
}

func (d *Encoder) Unmarshal(obj []byte) (*tempopb.Trace, error) {
	trace := &tempopb.Trace{}
	err := proto.Unmarshal(obj, trace)
	if err != nil {
		return nil, err
	}
	return trace, err
}

func (d *Encoder) Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error) {
	t, err := d.Unmarshal(obj)
	if err != nil {
		return nil, err
	}

	return trace.MatchesProto(id, t, req)
}

func (d *Encoder) Combine(objs ...[]byte) ([]byte, error) {
	var combinedTrace *tempopb.Trace
	for _, obj := range objs {
		t, err := d.Unmarshal(obj)
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

func (d *Encoder) Marshal(t *tempopb.Trace) ([]byte, error) {
	return proto.Marshal(t)
}
