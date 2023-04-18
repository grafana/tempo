package model

import (
	"fmt"

	v1 "github.com/grafana/tempo/pkg/model/v1"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/tempopb"
)

// CurrentEncoding is a string representing the encoding that all new blocks should be created with
const CurrentEncoding = v2.Encoding

// AllEncodings is used for testing
var AllEncodings = []string{
	v1.Encoding,
	v2.Encoding,
}

// ObjectDecoder is used to work with opaque byte slices that contain trace data in the backend
type ObjectDecoder interface {
	// PrepareForRead converts the byte slice into a tempopb.Trace for reading. This can be very expensive
	//  and should only be used when surfacing a byte slice from tempodb and preparing it for reads.
	PrepareForRead(obj []byte) (*tempopb.Trace, error)

	// Combine combines the passed byte slice
	Combine(objs ...[]byte) ([]byte, error)
	// FastRange returns the start and end unix epoch timestamp of the trace. If its not possible to efficiently get these
	// values from the underlying encoding then it should return decoder.ErrUnsupported
	FastRange(obj []byte) (uint32, uint32, error)
}

// NewObjectDecoder returns a Decoder given the passed string.
func NewObjectDecoder(dataEncoding string) (ObjectDecoder, error) {
	switch dataEncoding {
	case v1.Encoding:
		return v1.NewObjectDecoder(), nil
	case v2.Encoding:
		return v2.NewObjectDecoder(), nil
	}

	return nil, fmt.Errorf("unknown encoding %s. Supported encodings %v", dataEncoding, AllEncodings)
}

// MustNewObjectDecoder creates a new encoding or it panics
func MustNewObjectDecoder(dataEncoding string) ObjectDecoder {
	decoder, err := NewObjectDecoder(dataEncoding)

	if err != nil {
		panic(err)
	}

	return decoder
}
