package model

import (
	"fmt"

	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/gogo/protobuf/proto"
)

// CurrentEncoding is a string representing the encoding that all new blocks should be created with
//   "" = tempopb.Trace
//   "v1" = tempopb.TraceBytes
const CurrentEncoding = "v1"

// TracePBEncoding is a string that represents the original TracePBEncoding. Pass this if you know that the
// bytes are encoded *tracepb.Trace
const TracePBEncoding = ""

// Unmarshal converts a byte slice of the passed encoding into a *tempopb.Trace
func Unmarshal(obj []byte, dataEncoding string) (*tempopb.Trace, error) {
	trace := &tempopb.Trace{}

	switch dataEncoding {
	case "":
		err := proto.Unmarshal(obj, trace)
		if err != nil {
			return nil, err
		}
	case "v1":
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

			trace.Batches = append(trace.Batches, innerTrace.Batches...) // todo(jpe) set trace.ID?
		}
	default:
		return nil, fmt.Errorf("unrecognized dataEncoding in Unmarshal %s", dataEncoding)
	}

	return trace, nil
}

// marshal converts a tempopb.Trace into a byte slice encoded using dataEncoding
func marshal(trace *tempopb.Trace, dataEncoding string) ([]byte, error) {
	switch dataEncoding {
	case "":
		return proto.Marshal(trace)
	case "v1":
		traceBytes := &tempopb.TraceBytes{}
		bytes, err := proto.Marshal(trace)
		if err != nil {
			return nil, err
		}

		traceBytes.Traces = append(traceBytes.Traces, bytes)

		return proto.Marshal(traceBytes)
	default:
		return nil, fmt.Errorf("unrecognized dataEncoding in Unmarshal %s", dataEncoding)
	}
}
