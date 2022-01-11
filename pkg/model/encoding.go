package model

import (
	"fmt"

	v1 "github.com/grafana/tempo/pkg/model/v1"
	"github.com/grafana/tempo/pkg/tempopb"
)

// CurrentEncoding is a string representing the encoding that all new blocks should be created with
//   "" = tempopb.Trace
//   "v1" = tempopb.TraceBytes
const CurrentEncoding = v1.Encoding

// allEncodings is used for testing
var allEncodings = []string{
	v1.Encoding,
}

// Decoder is used to work with opaque byte slices that contain trace data
type Decoder interface {
	// PrepareForRead converts the byte slice into a tempopb.Trace for reading. This can be very expensive
	//  and should only be used when surfacing a byte slice from tempodb and preparing it for reads.
	PrepareForRead(obj []byte) (*tempopb.Trace, error)
	// Matches tests the passed byte slice and id to determine if it matches the criteria in tempopb.SearchRequest
	Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error)
	// Combine combines the passed byte slice
	Combine(objs ...[]byte) ([]byte, error)
}

// encoderDecoder is an internal interface to assist with testing in this package
type encoderDecoder interface {
	Decoder
	Marshal(t *tempopb.Trace) ([]byte, error)
}

// NewDecoder returns a Decoder given the passed string.
func NewDecoder(dataEncoding string) (Decoder, error) {
	switch dataEncoding {
	case v1.Encoding:
		return v1.NewDecoder(), nil
	}

	return nil, fmt.Errorf("unknown encoding %s. Supported encodings %v", dataEncoding, allEncodings)
}

// MustNewDecoder creates a new encoding or it panics
func MustNewDecoder(dataEncoding string) Decoder {
	decoder, err := NewDecoder(dataEncoding)

	if err != nil {
		panic(err)
	}

	return decoder
}
