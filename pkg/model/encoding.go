package model

import (
	"fmt"

	"github.com/grafana/tempo/pkg/model/tracepb"
	v1 "github.com/grafana/tempo/pkg/model/v1"
	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/gogo/protobuf/proto"
)

// CurrentEncoding is a string representing the encoding that all new blocks should be created with
//   "" = tempopb.Trace
//   "v1" = tempopb.TraceBytes
const CurrentEncoding = v1.Encoding

// allEncodings is used for testing
var allEncodings = []string{
	v1.Encoding,
	tracepb.Encoding,
}

// jpe put in decoder?
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
