package v1

import "github.com/grafana/tempo/pkg/tempopb"

const Encoding = "v1"

type Decoder struct {
}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d *Decoder) ToProto(obj []byte) (*tempopb.Trace, error) {
	return nil, nil
}
func (d *Decoder) Matches(id []byte, obj []byte, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error) {
	return nil, nil
}
func (d *Decoder) Range(obj []byte) (uint32, uint32, error) {
	return 0, 0, nil
}
