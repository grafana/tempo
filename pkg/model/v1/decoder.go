package v1

import (
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/tracepb"
	"github.com/grafana/tempo/pkg/tempopb"
)

const Encoding = "v1"

type Decoder struct {
}

var decoder = &Decoder{}

func NewDecoder() *Decoder {
	return decoder
}

func (d *Decoder) Unmarshal(obj []byte) (*tempopb.Trace, error) {
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
	trace, err := d.Unmarshal(obj)
	if err != nil {
		return nil, err
	}

	return tracepb.MatchesProto(id, trace, req)
}

func (d *Decoder) Range(obj []byte) (uint32, uint32, error) {
	return 0, 0, nil // jpe unsupported
}
