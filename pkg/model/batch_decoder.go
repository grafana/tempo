package model

import (
	"fmt"

	v1 "github.com/grafana/tempo/pkg/model/v1"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/tempopb"
)

// BatchDecoder is used by the distributor/ingester to aggregate and pass batches of traces
type BatchDecoder interface {
	// PrepareForWrite takes a trace pointer and returns a record prepared for writing to an ingester
	PrepareForWrite(trace *tempopb.Trace, start uint32, end uint32) ([]byte, error)
	// PrepareForRead converts a set of batches created using PrepareForWrite. These batches
	//  are converted into a tempo.Trace. This operation can be quite costly and should be called for reading
	PrepareForRead(batches [][]byte) (*tempopb.Trace, error)
	// ToObject converts a set of batches into an object ready to be written to the tempodb backend.
	//  The resultant byte slice can then be manipulated using the corresponding ObjectDecoder.
	//  ToObject is on the write path and should do as little as possible.
	ToObject(batches [][]byte) ([]byte, error)
}

// NewBatchDecoder returns a Decoder given the passed string.
func NewBatchDecoder(dataEncoding string) (BatchDecoder, error) {
	switch dataEncoding {
	case v1.Encoding:
		return v1.NewBatchDecoder(), nil
	case v2.Encoding:
		return v2.NewBatchDecoder(), nil
	}

	return nil, fmt.Errorf("unknown encoding %s. Supported encodings %v", dataEncoding, AllEncodings)
}

// MustNewBatchDecoder creates a new encoding or it panics
func MustNewBatchDecoder(dataEncoding string) BatchDecoder {
	decoder, err := NewBatchDecoder(dataEncoding)

	if err != nil {
		panic(err)
	}

	return decoder
}
