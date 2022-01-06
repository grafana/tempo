package tracepb

import (
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
)

const Encoding = ""

type Decoder struct {
}

var decoder = &Decoder{}

func NewDecoder() *Decoder {
	return decoder
}

func (d *Decoder) Unmarshal(obj []byte) (*tempopb.Trace, error) {
	trace := &tempopb.Trace{}
	err := proto.Unmarshal(obj, trace)
	if err != nil {
		return nil, err
	}
	return trace, err
}

func (d *Decoder) Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error) {
	return nil, nil
}
func (d *Decoder) Range(obj []byte) (uint32, uint32, error) {
	return 0, 0, nil
}
