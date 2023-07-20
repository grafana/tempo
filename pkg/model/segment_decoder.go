package model

import (
	"fmt"

	v1 "github.com/grafana/tempo/pkg/model/v1"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/tempopb"
)

// SegmentDecoder is used by the distributor/ingester to aggregate and pass segments of traces. The distributor
// creates the segments using PrepareForWrite which can then be consumed and organized by traceid in the ingester.
//
// The ingester then holds these in memory until either:
//   - The trace id is queried. In this case it uses PrepareForRead to turn the segments into a tempopb.Trace for
//     return on the query path.
//   - It needs to push them into tempodb. For this it uses ToObject() to create a single byte slice from the
//     segments that is then completely handled by an ObjectDecoder of the same version
type SegmentDecoder interface {
	// PrepareForWrite takes a trace pointer and returns a record prepared for writing to an ingester
	PrepareForWrite(trace *tempopb.Trace, start uint32, end uint32) ([]byte, error)
	// PrepareForRead converts a set of segments created using PrepareForWrite. These segments
	//  are converted into a tempopb.Trace. This operation can be quite costly and should be called only for reading
	PrepareForRead(segments [][]byte) (*tempopb.Trace, error)
	// ToObject converts a set of segments into an object ready to be written to the tempodb backend.
	//  The resultant byte slice can then be manipulated using the corresponding ObjectDecoder.
	//  ToObject is on the write path and should do as little as possible.
	ToObject(segments [][]byte) ([]byte, error)
	// FastRange returns the start and end unix epoch timestamp of the provided segment. If its not possible to efficiently get these
	// values from the underlying encoding then it should return decoder.ErrUnsupported
	FastRange(segment []byte) (uint32, uint32, error)
}

// NewSegmentDecoder returns a Decoder given the passed string.
func NewSegmentDecoder(dataEncoding string) (SegmentDecoder, error) {
	switch dataEncoding {
	case v1.Encoding:
		return v1.NewSegmentDecoder(), nil
	case v2.Encoding:
		return v2.NewSegmentDecoder(), nil
	}

	return nil, fmt.Errorf("unknown encoding %s. Supported encodings %v", dataEncoding, AllEncodings)
}

// MustNewSegmentDecoder creates a new encoding or it panics
func MustNewSegmentDecoder(dataEncoding string) SegmentDecoder {
	decoder, err := NewSegmentDecoder(dataEncoding)
	if err != nil {
		panic(err)
	}

	return decoder
}
