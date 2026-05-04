// This is copied over from Jaeger and modified to work for Tempo

// Upgrading to grpc 1.38.0 broke compatibility with gogoproto.customtype. (https://github.com/grpc/grpc-go/issues/4192)
// We use a customtype in the ingesters to pre-allocate byte slices that are reused for requests.
// Similarly Jaeger and Cortex use gogo's custom types for efficiency.
// gogoproto codec is needed only if a custom type (for ex: PreallocBytes) is used directly in a request-response object.
// The codec defined in this package allows us to choose gogo marshalling/unmarshalling for specific structs (Tempo/Jaeger/Cortex) only.

package gogocodec

import (
	"fmt"
	"reflect"
	"strings"

	gogoproto "github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/proto"
)

const (
	frontendProtoGenPkgPath = "github.com/grafana/tempo/modules/frontend"
	tempoProtoGenPkgPath    = "github.com/grafana/tempo/pkg/tempopb"
	jaegerProtoGenPkgPath   = "github.com/jaegertracing/jaeger-idl/proto-gen"
	jaegerModelPkgPath      = "github.com/jaegertracing/jaeger-idl/model"
	jaegerStorageV1PkgPath  = "github.com/grafana/tempo/cmd/tempo-query/jaeger/storage_v1"
	// etcd path can be removed once upgrade to grpc >v1.38 is released (tentatively next release from v3.5.1)
	etcdAPIProtoPkgPath = "go.etcd.io/etcd/api/v3"
)

// GogoCodec forces the use of gogo proto marshalling/unmarshalling for Tempo/Cortex/Jaeger/etcd structs
type GogoCodec struct{}

var _ encoding.Codec = (*GogoCodec)(nil)

// This mirrors OTEL collector's internal, unexported otelEncoder interface:
// go.opentelemetry.io/collector/pdata/internal/otelgrpc/encoding.go
// We cannot import it here because it's unexported and under an internal package.
type otelProtoMessage interface {
	SizeProto() int
	MarshalProto([]byte) int
	UnmarshalProto([]byte) error
}

func NewCodec() *GogoCodec {
	return &GogoCodec{}
}

// Name implements encoding.Codec
func (c *GogoCodec) Name() string {
	return "proto"
}

// Marshal implements encoding.Codec
func (c *GogoCodec) Marshal(v any) ([]byte, error) {
	if m, ok := v.(otelProtoMessage); ok {
		size := m.SizeProto()
		buf := make([]byte, size)
		n := m.MarshalProto(buf)
		return buf[:n], nil
	}

	// use gogo proto only for Tempo/Cortex/Jaeger/etcd types
	if useGogo(reflect.TypeOf(v)) {
		if msg, ok := v.(gogoproto.Message); ok {
			return gogoproto.Marshal(msg)
		}
	}

	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("gogocodec: unsupported marshal type %T", v)
	}
	return proto.Marshal(msg)
}

// Unmarshal implements encoding.Codec
func (c *GogoCodec) Unmarshal(data []byte, v any) error {
	if m, ok := v.(otelProtoMessage); ok {
		return m.UnmarshalProto(data)
	}

	// use gogo proto only for Tempo/Cortex/Jaeger/etcd types
	if useGogo(reflect.TypeOf(v)) {
		if msg, ok := v.(gogoproto.Message); ok {
			return gogoproto.Unmarshal(data, msg)
		}
	}

	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("gogocodec: unsupported unmarshal type %T", v)
	}
	return proto.Unmarshal(data, msg)
}

// useGogo checks if the element belongs to Tempo/Cortex/Jaeger/etcd packages
func useGogo(t reflect.Type) bool {
	if t == nil {
		return false
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		if t == nil {
			return false
		}
	}
	pkgPath := t.PkgPath()
	return strings.HasPrefix(pkgPath, frontendProtoGenPkgPath) || strings.HasPrefix(pkgPath, tempoProtoGenPkgPath) || strings.HasPrefix(pkgPath, jaegerProtoGenPkgPath) || strings.HasPrefix(pkgPath, jaegerModelPkgPath) || strings.HasPrefix(pkgPath, jaegerStorageV1PkgPath) || strings.HasPrefix(pkgPath, etcdAPIProtoPkgPath)
}
