// This is copied over from Jaeger and modified to work for Tempo

// Upgrading to grpc 1.38.0 broke compatibility with gogoproto.customtype. (https://github.com/grpc/grpc-go/issues/4192)
// We use a customtype in the ingesters to pre-allocate byte slices that are reused for requests.
// Similarly Jaeger and Cortex use gogo's custom types for efficiency.
// gogoproto codec is needed only if a custom type (for ex: PreallocBytes) is used directly in a request-response object.
// The codec defined in this package allows us to choose gogo marshalling/unmarshalling for specific structs (Tempo/Jaeger/Cortex) only.

package gogocodec

import (
	"reflect"
	"strings"

	gogoproto "github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/proto"
)

const (
	tempoProtoGenPkgPath  = "github.com/grafana/tempo/pkg/tempopb"
	cortexPath            = "github.com/cortexproject/cortex"
	jaegerProtoGenPkgPath = "github.com/jaegertracing/jaeger/proto-gen"
	jaegerModelPkgPath    = "github.com/jaegertracing/jaeger/model"
	otelProtoPkgPath      = "go.opentelemetry.io/collector"
)

func init() {
	encoding.RegisterCodec(newCodec())
}

// gogoCodec forces the use of gogo proto marshalling/unmarshalling for Tempo/Cortex/Jaeger structs
type gogoCodec struct {
}

var _ encoding.Codec = (*gogoCodec)(nil)

func newCodec() *gogoCodec {
	return &gogoCodec{}
}

// Name implements encoding.Codec
func (c *gogoCodec) Name() string {
	return "proto"
}

// Marshal implements encoding.Codec
func (c *gogoCodec) Marshal(v interface{}) ([]byte, error) {
	t := reflect.TypeOf(v)
	elem := t.Elem()
	// use gogo proto only for Tempo/Cortex/Jaeger types
	if useGogo(elem) {
		return gogoproto.Marshal(v.(gogoproto.Message))
	}
	return proto.Marshal(v.(proto.Message))
}

// Unmarshal implements encoding.Codec
func (c *gogoCodec) Unmarshal(data []byte, v interface{}) error {
	t := reflect.TypeOf(v)
	elem := t.Elem()
	// use gogo proto only for Tempo/Cortex/Jaeger types
	if useGogo(elem) {
		return gogoproto.Unmarshal(data, v.(gogoproto.Message))
	}
	return proto.Unmarshal(data, v.(proto.Message))
}

// useGogo checks if the element belongs to Tempo/Cortex/Jaeger packages
func useGogo(t reflect.Type) bool {
	if t == nil {
		return false
	}
	pkgPath := t.PkgPath()
	return strings.HasPrefix(pkgPath, tempoProtoGenPkgPath) || strings.HasPrefix(pkgPath, cortexPath) || strings.HasPrefix(pkgPath, jaegerProtoGenPkgPath) || strings.HasPrefix(pkgPath, jaegerModelPkgPath) || strings.HasPrefix(pkgPath, otelProtoPkgPath)
}
