package frontendv1pb

import (
	"bytes"

	"github.com/grafana/dskit/httpgrpc"
)

// HTTPRequestEnvelope and HTTPResponseEnvelope carry the dskit
// (gogo-generated) httpgrpc types through wiresmith-generated messages via
// the (wiresmith.options.customtype) extension: the envelope owns the wire
// encoding of the field payload and delegates to the gogo Marshal/Unmarshal
// methods, so the bytes on the wire are identical to the old gogo codegen.
//
// Caveat: wiresmith omits a singular customtype field whose SizeWiresmith()
// is 0, so a non-nil but empty HTTPRequest/HTTPResponse decodes as nil on
// the other side. Tempo's frontend protocol never sends empty messages in
// these fields.

type HTTPRequestEnvelope struct {
	Req *httpgrpc.HTTPRequest
}

func (e *HTTPRequestEnvelope) SizeWiresmith() int {
	if e == nil || e.Req == nil {
		return 0
	}
	return e.Req.Size()
}

func (e *HTTPRequestEnvelope) MarshalWiresmith(buf []byte) (int, error) {
	if e == nil || e.Req == nil {
		return 0, nil
	}
	return e.Req.MarshalTo(buf)
}

func (e *HTTPRequestEnvelope) UnmarshalWiresmith(buf []byte) error {
	if e.Req == nil {
		e.Req = &httpgrpc.HTTPRequest{}
	}
	return e.Req.Unmarshal(buf)
}

func (e *HTTPRequestEnvelope) EqualWiresmith(other any) bool {
	o, ok := envelopeValue[HTTPRequestEnvelope](other)
	if !ok {
		return false
	}
	return e.Req.Equal(o.Req)
}

func (e *HTTPRequestEnvelope) CompareWiresmith(other any) int {
	o, ok := envelopeValue[HTTPRequestEnvelope](other)
	if !ok {
		return -1
	}
	if e.Req == nil || o.Req == nil {
		return boolCompare(e.Req != nil, o.Req != nil)
	}
	return compareWire(e.Req, o.Req)
}

type HTTPResponseEnvelope struct {
	Resp *httpgrpc.HTTPResponse
}

func (e *HTTPResponseEnvelope) SizeWiresmith() int {
	if e == nil || e.Resp == nil {
		return 0
	}
	return e.Resp.Size()
}

func (e *HTTPResponseEnvelope) MarshalWiresmith(buf []byte) (int, error) {
	if e == nil || e.Resp == nil {
		return 0, nil
	}
	return e.Resp.MarshalTo(buf)
}

func (e *HTTPResponseEnvelope) UnmarshalWiresmith(buf []byte) error {
	if e.Resp == nil {
		e.Resp = &httpgrpc.HTTPResponse{}
	}
	return e.Resp.Unmarshal(buf)
}

func (e *HTTPResponseEnvelope) EqualWiresmith(other any) bool {
	o, ok := envelopeValue[HTTPResponseEnvelope](other)
	if !ok {
		return false
	}
	return e.Resp.Equal(o.Resp)
}

func (e *HTTPResponseEnvelope) CompareWiresmith(other any) int {
	o, ok := envelopeValue[HTTPResponseEnvelope](other)
	if !ok {
		return -1
	}
	if e.Resp == nil || o.Resp == nil {
		return boolCompare(e.Resp != nil, o.Resp != nil)
	}
	return compareWire(e.Resp, o.Resp)
}

// WrapHTTPRequests adapts a dskit request slice to the envelope slice the
// generated FrontendToClient field uses.
func WrapHTTPRequests(reqs []*httpgrpc.HTTPRequest) []HTTPRequestEnvelope {
	out := make([]HTTPRequestEnvelope, len(reqs))
	for i, r := range reqs {
		out[i] = HTTPRequestEnvelope{Req: r}
	}
	return out
}

// UnwrapHTTPRequests is the inverse of WrapHTTPRequests.
func UnwrapHTTPRequests(envs []HTTPRequestEnvelope) []*httpgrpc.HTTPRequest {
	out := make([]*httpgrpc.HTTPRequest, len(envs))
	for i, e := range envs {
		out[i] = e.Req
	}
	return out
}

// WrapHTTPResponses adapts a dskit response slice to the envelope slice the
// generated ClientToFrontend field uses.
func WrapHTTPResponses(resps []*httpgrpc.HTTPResponse) []HTTPResponseEnvelope {
	out := make([]HTTPResponseEnvelope, len(resps))
	for i, r := range resps {
		out[i] = HTTPResponseEnvelope{Resp: r}
	}
	return out
}

// UnwrapHTTPResponses is the inverse of WrapHTTPResponses.
func UnwrapHTTPResponses(envs []HTTPResponseEnvelope) []*httpgrpc.HTTPResponse {
	out := make([]*httpgrpc.HTTPResponse, len(envs))
	for i, e := range envs {
		out[i] = e.Resp
	}
	return out
}

// envelopeValue normalizes the value/pointer forms the generated Equal and
// Compare methods may hand us.
func envelopeValue[T any](other any) (*T, bool) {
	switch v := other.(type) {
	case T:
		return &v, true
	case *T:
		if v == nil {
			return nil, false
		}
		return v, true
	default:
		return nil, false
	}
}

// boolCompare orders absent (nil) before present, mirroring gogo's
// Compare nil handling.
func boolCompare(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	default:
		return 1
	}
}

// compareWire gives a deterministic total order over two gogo messages by
// comparing their marshaled bytes (Compare is only used for ordering, the
// exact order does not matter as long as it is stable).
func compareWire(a, b interface {
	Marshal() ([]byte, error)
},
) int {
	ab, err := a.Marshal()
	if err != nil {
		return -1
	}
	bb, err := b.Marshal()
	if err != nil {
		return 1
	}
	return bytes.Compare(ab, bb)
}
