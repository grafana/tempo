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

// jpe : distributor needs to marshal somehow
type Decoder interface {
	Unmarshal(obj []byte) (*tempopb.Trace, error)
	Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error)
	Combine(objs ...[]byte) ([]byte, error) // jpe combine tests?
	Marshal(t *tempopb.Trace) ([]byte, error)
	Range(obj []byte) (uint32, uint32, error) // jpe remove for now?
}

func NewDecoder(dataEncoding string) (Decoder, error) {
	switch dataEncoding {
	case v0.Encoding:
		return v0.NewDecoder(), nil
	case v1.Encoding:
		return v1.NewDecoder(), nil
	}

	return nil, fmt.Errorf("unknown encoding %s. Supported encodings %v", dataEncoding, allEncodings)
}

func MustNewDecoder(dataEncoding string) Decoder {
	decoder, err := NewDecoder(dataEncoding)

	if err != nil {
		panic(err)
	}

	return decoder
}
