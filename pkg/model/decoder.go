package model

import (
	"fmt"

	"github.com/grafana/tempo/pkg/model/tracepb"
	v1 "github.com/grafana/tempo/pkg/model/v1"
	"github.com/grafana/tempo/pkg/tempopb"
)

// jpe : distributor needs to marshal somehow
type Decoder interface {
	ToProto(obj []byte) (*tempopb.Trace, error)
	Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error)
	Range(obj []byte) (uint32, uint32, error)
}

func NewDecoder(dataEncoding string) (Decoder, error) {
	switch dataEncoding {
	case tracepb.Encoding:
		return tracepb.NewDecoder(), nil
	case v1.Encoding:
		return v1.NewDecoder(), nil
	}

	return nil, fmt.Errorf("unknown encoding %s. Supported encodings %v", dataEncoding, allEncodings)
}
