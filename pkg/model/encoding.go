package model

import (
	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/gogo/protobuf/proto"
)

// CurrentEncoding is a string representing the encoding that all new blocks should be created with
const CurrentEncoding = ""
const BaseEncoding = ""

// Unmarshal converts a byte slice of the passed encoding into a *tempopb.Trace
func Unmarshal(obj []byte, dataEncoding string) (*tempopb.Trace, error) {
	trace := &tempopb.Trace{}
	err := proto.Unmarshal(obj, trace)
	return trace, err
}
