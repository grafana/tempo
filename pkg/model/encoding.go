package model

import (
	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/gogo/protobuf/proto"
)

// CurrentEncoding is a string representing the encoding that all new blocks should be created with
const CurrentEncoding = ""

// TracePBEncoding is a string that represents the original TracePBEncoding. Pass this if you know that the
// bytes are encoded *tracepb.Trace
const TracePBEncoding = ""

// Unmarshal converts a byte slice of the passed encoding into a *tempopb.Trace
func Unmarshal(obj []byte, dataEncoding string) (*tempopb.Trace, error) {
	trace := &tempopb.Trace{}
	err := proto.Unmarshal(obj, trace)
	return trace, err
}

// Marshal converts a tempopb.Trace into a byte slice encoded using dataEncoding
// nolint: interfacer
func Marshal(trace *tempopb.Trace, dataEncoding string) ([]byte, error) {
	return proto.Marshal(trace)
}
