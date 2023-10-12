// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv1 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv1"

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"

	jaegerzipkin "github.com/jaegertracing/jaeger/model/converter/thrift/zipkin"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/idutils"
)

type thriftUnmarshaler struct{}

// UnmarshalTraces from Thrift bytes.
func (t thriftUnmarshaler) UnmarshalTraces(buf []byte) (ptrace.Traces, error) {
	spans, err := jaegerzipkin.DeserializeThrift(context.TODO(), buf)
	if err != nil {
		return ptrace.Traces{}, err
	}
	return thriftBatchToTraces(spans)
}

// NewThriftTracesUnmarshaler returns an unmarshaler for Zipkin Thrift.
func NewThriftTracesUnmarshaler() ptrace.Unmarshaler {
	return thriftUnmarshaler{}
}

// thriftBatchToTraces converts Zipkin v1 spans to ptrace.Traces.
func thriftBatchToTraces(zSpans []*zipkincore.Span) (ptrace.Traces, error) {
	spanAndEndpoints := make([]spanAndEndpoint, 0, len(zSpans))
	for _, zSpan := range zSpans {
		spanAndEndpoints = append(spanAndEndpoints, thriftToSpanAndEndpoint(zSpan))
	}

	return zipkinToTraces(spanAndEndpoints)
}

func thriftToSpanAndEndpoint(zSpan *zipkincore.Span) spanAndEndpoint {
	traceIDHigh := int64(0)
	if zSpan.TraceIDHigh != nil {
		traceIDHigh = *zSpan.TraceIDHigh
	}

	// TODO: (@pjanotti) ideally we should error here instead of generating invalid Traces
	// however per https://go.opentelemetry.io/collector/issues/349
	// failures on the receivers in general are silent at this moment, so letting them
	// proceed for now. We should validate the traceID, spanID and parentID are good with
	// OTLP requirements.
	traceID := idutils.UInt64ToTraceID(uint64(traceIDHigh), uint64(zSpan.TraceID))
	spanID := idutils.UInt64ToSpanID(uint64(zSpan.ID))
	var parentID pcommon.SpanID
	if zSpan.ParentID != nil {
		parentID = idutils.UInt64ToSpanID(uint64(*zSpan.ParentID))
	}

	span, edpt := thriftAnnotationsToSpanAndEndpoint(zSpan.Annotations)
	localComponent := thriftBinAnnotationsToSpanAttributes(span, zSpan.BinaryAnnotations)
	if edpt.ServiceName == unknownServiceName && localComponent != "" {
		edpt.ServiceName = localComponent
	}

	if zSpan.Timestamp != nil {
		span.SetStartTimestamp(epochMicrosecondsToTimestamp(*zSpan.Timestamp))
		var duration int64
		if zSpan.Duration != nil {
			duration = *zSpan.Duration
		}
		span.SetEndTimestamp(epochMicrosecondsToTimestamp(*zSpan.Timestamp + duration))
	}

	span.SetName(zSpan.Name)
	span.SetTraceID(traceID)
	span.SetSpanID(spanID)
	span.SetParentSpanID(parentID)

	return spanAndEndpoint{span: span, endpoint: edpt}
}

func thriftAnnotationsToSpanAndEndpoint(ztAnnotations []*zipkincore.Annotation) (ptrace.Span, *endpoint) {
	annotations := make([]*annotation, 0, len(ztAnnotations))
	for _, ztAnnot := range ztAnnotations {
		annot := &annotation{
			Timestamp: ztAnnot.Timestamp,
			Value:     ztAnnot.Value,
			Endpoint:  toTranslatorEndpoint(ztAnnot.Host),
		}
		annotations = append(annotations, annot)
	}
	return jsonAnnotationsToSpanAndEndpoint(annotations)
}

func toTranslatorEndpoint(e *zipkincore.Endpoint) *endpoint {
	if e == nil {
		return nil
	}

	var ipv4, ipv6 string
	if e.Ipv4 != 0 {
		ipv4 = net.IPv4(byte(e.Ipv4>>24), byte(e.Ipv4>>16), byte(e.Ipv4>>8), byte(e.Ipv4)).String()
	}
	if len(e.Ipv6) != 0 {
		ipv6 = net.IP(e.Ipv6).String()
	}
	return &endpoint{
		ServiceName: e.ServiceName,
		IPv4:        ipv4,
		IPv6:        ipv6,
		Port:        int32(e.Port),
	}
}

var trueByteSlice = []byte{1}

func thriftBinAnnotationsToSpanAttributes(span ptrace.Span, ztBinAnnotations []*zipkincore.BinaryAnnotation) string {
	var fallbackServiceName string
	if len(ztBinAnnotations) == 0 {
		return fallbackServiceName
	}

	sMapper := &statusMapper{}
	var localComponent string
	for _, binaryAnnotation := range ztBinAnnotations {
		val := pcommon.NewValueEmpty()
		binAnnotationType := binaryAnnotation.AnnotationType
		if binaryAnnotation.Host != nil {
			fallbackServiceName = binaryAnnotation.Host.ServiceName
		}
		switch binaryAnnotation.AnnotationType {
		case zipkincore.AnnotationType_BOOL:
			isTrue := bytes.Equal(binaryAnnotation.Value, trueByteSlice)
			val.SetBool(isTrue)
		case zipkincore.AnnotationType_BYTES:
			bytesStr := base64.StdEncoding.EncodeToString(binaryAnnotation.Value)
			val.SetStr(bytesStr)
		case zipkincore.AnnotationType_DOUBLE:
			if d, err := bytesFloat64ToFloat64(binaryAnnotation.Value); err != nil {
				strAttributeForError(val, err)
			} else {
				val.SetDouble(d)
			}
		case zipkincore.AnnotationType_I16:
			if i, err := bytesInt16ToInt64(binaryAnnotation.Value); err != nil {
				strAttributeForError(val, err)
			} else {
				val.SetInt(i)
			}
		case zipkincore.AnnotationType_I32:
			if i, err := bytesInt32ToInt64(binaryAnnotation.Value); err != nil {
				strAttributeForError(val, err)
			} else {
				val.SetInt(i)
			}
		case zipkincore.AnnotationType_I64:
			if i, err := bytesInt64ToInt64(binaryAnnotation.Value); err != nil {
				strAttributeForError(val, err)
			} else {
				val.SetInt(i)
			}
		case zipkincore.AnnotationType_STRING:
			val.SetStr(string(binaryAnnotation.Value))
		default:
			strAttributeForError(val, fmt.Errorf("unknown zipkin v1 binary annotation type (%d)", int(binAnnotationType)))
		}

		key := binaryAnnotation.Key
		if key == zipkincore.LOCAL_COMPONENT {
			// TODO: (@pjanotti) add reference to OpenTracing and change related tags to use them
			key = "component"
			localComponent = string(binaryAnnotation.Value)
		}

		if drop := sMapper.fromAttribute(key, val); drop {
			continue
		}

		val.CopyTo(span.Attributes().PutEmpty(key))
	}

	if fallbackServiceName == "" {
		fallbackServiceName = localComponent
	}

	sMapper.status(span.Status())
	return fallbackServiceName
}

var errNotEnoughBytes = errors.New("not enough bytes representing the number")

func bytesInt16ToInt64(b []byte) (int64, error) {
	const minSliceLength = 2
	if len(b) < minSliceLength {
		return 0, errNotEnoughBytes
	}
	return int64(binary.BigEndian.Uint16(b[:minSliceLength])), nil
}

func bytesInt32ToInt64(b []byte) (int64, error) {
	const minSliceLength = 4
	if len(b) < minSliceLength {
		return 0, errNotEnoughBytes
	}
	return int64(binary.BigEndian.Uint32(b[:minSliceLength])), nil
}

func bytesInt64ToInt64(b []byte) (int64, error) {
	const minSliceLength = 8
	if len(b) < minSliceLength {
		return 0, errNotEnoughBytes
	}
	return int64(binary.BigEndian.Uint64(b[:minSliceLength])), nil
}

func bytesFloat64ToFloat64(b []byte) (float64, error) {
	const minSliceLength = 8
	if len(b) < minSliceLength {
		return 0.0, errNotEnoughBytes
	}
	bits := binary.BigEndian.Uint64(b)
	return math.Float64frombits(bits), nil
}

func strAttributeForError(dest pcommon.Value, err error) {
	dest.SetStr("<" + err.Error() + ">")
}
