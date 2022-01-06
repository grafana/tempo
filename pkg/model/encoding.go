package model

import (
	"fmt"

	v0 "github.com/grafana/tempo/pkg/model/v0"
	v1 "github.com/grafana/tempo/pkg/model/v1"
	"github.com/grafana/tempo/pkg/tempopb"
)

// CurrentEncoding is a string representing the encoding that all new blocks should be created with
//   "" = tempopb.Trace
//   "v1" = tempopb.TraceBytes
const CurrentEncoding = v1.Encoding

// allEncodings is used for testing
var allEncodings = []string{
	v0.Encoding,
	v1.Encoding,
}

type Encoding interface {
	// Unmarshal the byte slice to a tempopb trace
	Unmarshal(obj []byte) (*tempopb.Trace, error)
	// Marshal a tempopb.Trace to a byte slice
	Marshal(t *tempopb.Trace) ([]byte, error)
	// Matches tests the passed byte slice and id to determine if it matches the criteria in tempopb.SearchRequest
	Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error)
	// Combine combines the passed byte slice
	Combine(objs ...[]byte) ([]byte, error)
}

// NewEncoding returns an Encoding given the passed string.
func NewEncoding(dataEncoding string) (Encoding, error) {
	switch dataEncoding {
	case v0.Encoding:
		return v0.NewEncoding(), nil
	case v1.Encoding:
		return v1.NewEncoding(), nil
	}

	return nil, fmt.Errorf("unknown encoding %s. Supported encodings %v", dataEncoding, allEncodings)
}

// MustNewEncoding creates a new encoding or it panics
func MustNewEncoding(dataEncoding string) Encoding {
	decoder, err := NewEncoding(dataEncoding)

	if err != nil {
		panic(err)
	}

	return decoder
}
