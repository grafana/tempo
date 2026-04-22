// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv1 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv1"

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jaegertracing/jaeger-idl/thrift-gen/zipkincore"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/otel/semconv/v1.38.0"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/internal/zipkin"
)

var (
	// ZipkinV1 friendly conversion errors
	msgZipkinV1JSONUnmarshalError = "zipkinv1"
	msgZipkinV1TraceIDError       = "zipkinV1 span traceId"
	msgZipkinV1SpanIDError        = "zipkinV1 span id"
	msgZipkinV1ParentIDError      = "zipkinV1 span parentId"
	// Generic hex to ID conversion errors
	errHexTraceIDWrongLen = errors.New("hex traceId span has wrong length (expected 16 or 32)")
	errHexTraceIDZero     = errors.New("traceId is zero")
	errHexIDWrongLen      = errors.New("hex Id has wrong length (expected 16)")
	errHexIDZero          = errors.New("ID is zero")
)

type jsonUnmarshaler struct {
	// ParseStringTags should be set to true if tags should be converted to numbers when possible.
	ParseStringTags bool
}

// UnmarshalTraces from JSON bytes.
func (j jsonUnmarshaler) UnmarshalTraces(buf []byte) (ptrace.Traces, error) {
	return jsonBatchToTraces(buf, j.ParseStringTags)
}

// NewJSONTracesUnmarshaler returns an unmarshaler for Zipkin JSON.
func NewJSONTracesUnmarshaler(parseStringTags bool) ptrace.Unmarshaler {
	return jsonUnmarshaler{ParseStringTags: parseStringTags}
}

// Trace translation from Zipkin V1 is a bit of special case since there is no model
// defined in golang for Zipkin V1 spans and there is no need to define one here, given
// that the jsonSpan defined below is as defined at:
// https://zipkin.io/zipkin-api/zipkin-api.yaml
type jsonSpan struct {
	TraceID           string              `json:"traceId"`
	Name              string              `json:"name,omitempty"`
	ParentID          string              `json:"parentId,omitempty"`
	ID                string              `json:"id"`
	Timestamp         int64               `json:"timestamp"`
	Duration          int64               `json:"duration"`
	Debug             bool                `json:"debug,omitempty"`
	Annotations       []*annotation       `json:"annotations,omitempty"`
	BinaryAnnotations []*binaryAnnotation `json:"binaryAnnotations,omitempty"`
}

// endpoint structure used by jsonSpan.
type endpoint struct {
	ServiceName string `json:"serviceName"`
	IPv4        string `json:"ipv4"`
	IPv6        string `json:"ipv6"`
	Port        int32  `json:"port"`
}

// annotation struct used by jsonSpan.
type annotation struct {
	Timestamp int64     `json:"timestamp"`
	Value     string    `json:"value"`
	Endpoint  *endpoint `json:"endpoint"`
}

// binaryAnnotation used by jsonSpan.
type binaryAnnotation struct {
	Key      string    `json:"key"`
	Value    string    `json:"value"`
	Endpoint *endpoint `json:"endpoint"`
}

// jsonBatchToTraces converts a JSON blob with a list of Zipkin v1 spans to ptrace.Traces.
func jsonBatchToTraces(blob []byte, parseStringTags bool) (ptrace.Traces, error) {
	var zSpans []*jsonSpan
	if err := json.Unmarshal(blob, &zSpans); err != nil {
		return ptrace.Traces{}, fmt.Errorf("%s: %w", msgZipkinV1JSONUnmarshalError, err)
	}

	spanAndEndpoints := make([]spanAndEndpoint, 0, len(zSpans))
	for _, zSpan := range zSpans {
		sae, err := jsonToSpanAndEndpoint(zSpan, parseStringTags)
		if err != nil {
			// error from internal package function, it already wraps the error to give better context.
			return ptrace.Traces{}, err
		}
		spanAndEndpoints = append(spanAndEndpoints, sae)
	}

	return zipkinToTraces(spanAndEndpoints)
}

type spanAndEndpoint struct {
	span     ptrace.Span
	endpoint *endpoint
}

func zipkinToTraces(spanAndEndpoints []spanAndEndpoint) (ptrace.Traces, error) {
	td := ptrace.NewTraces()
	// Service to batch maps the service name to the trace request with the corresponding node.
	svcToTD := make(map[string]ptrace.SpanSlice)
	for _, curr := range spanAndEndpoints {
		ss := getOrCreateNodeRequest(svcToTD, td, curr.endpoint)
		curr.span.MoveTo(ss.AppendEmpty())
	}
	return td, nil
}

func jsonToSpanAndEndpoint(zSpan *jsonSpan, parseStringTags bool) (spanAndEndpoint, error) {
	traceID, err := hexToTraceID(zSpan.TraceID)
	if err != nil {
		return spanAndEndpoint{}, fmt.Errorf("%s: %w", msgZipkinV1TraceIDError, err)
	}
	spanID, err := hexToSpanID(zSpan.ID)
	if err != nil {
		return spanAndEndpoint{}, fmt.Errorf("%s: %w", msgZipkinV1SpanIDError, err)
	}
	var parentID [8]byte
	if zSpan.ParentID != "" {
		parentID, err = hexToSpanID(zSpan.ParentID)
		if err != nil {
			return spanAndEndpoint{}, fmt.Errorf("%s: %w", msgZipkinV1ParentIDError, err)
		}
	}

	span, edpt := jsonAnnotationsToSpanAndEndpoint(zSpan.Annotations)
	localComponent := jsonBinAnnotationsToSpanAttributes(span, zSpan.BinaryAnnotations, parseStringTags)
	if edpt.ServiceName == unknownServiceName && localComponent != "" {
		edpt.ServiceName = localComponent
	}
	if zSpan.Timestamp != 0 {
		span.SetStartTimestamp(epochMicrosecondsToTimestamp(zSpan.Timestamp))
		span.SetEndTimestamp(epochMicrosecondsToTimestamp(zSpan.Timestamp + zSpan.Duration))
	}

	span.SetName(zSpan.Name)
	span.SetTraceID(traceID)
	span.SetSpanID(spanID)
	span.SetParentSpanID(parentID)
	setTimestampsIfUnset(span)

	return spanAndEndpoint{span: span, endpoint: edpt}, nil
}

func jsonBinAnnotationsToSpanAttributes(span ptrace.Span, binAnnotations []*binaryAnnotation, parseStringTags bool) string {
	var fallbackServiceName string
	if len(binAnnotations) == 0 {
		return fallbackServiceName
	}

	sMapper := &statusMapper{}
	var localComponent string
	for _, binAnnotation := range binAnnotations {
		if binAnnotation.Endpoint != nil && binAnnotation.Endpoint.ServiceName != "" {
			fallbackServiceName = binAnnotation.Endpoint.ServiceName
		}

		key := binAnnotation.Key
		if key == zipkincore.LOCAL_COMPONENT {
			// TODO: (@pjanotti) add reference to OpenTracing and change related tags to use them
			key = "component"
			localComponent = binAnnotation.Value
		}

		val := parseAnnotationValue(binAnnotation.Value, parseStringTags)
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

func parseAnnotationValue(value string, parseStringTags bool) pcommon.Value {
	if parseStringTags {
		switch zipkin.DetermineValueType(value) {
		case pcommon.ValueTypeInt:
			iValue, _ := strconv.ParseInt(value, 10, 64)
			return pcommon.NewValueInt(iValue)
		case pcommon.ValueTypeDouble:
			fValue, _ := strconv.ParseFloat(value, 64)
			return pcommon.NewValueDouble(fValue)
		case pcommon.ValueTypeBool:
			bValue, _ := strconv.ParseBool(value)
			return pcommon.NewValueBool(bValue)
		default:
			return pcommon.NewValueStr(value)
		}
	}

	return pcommon.NewValueStr(value)
}

// Unknown service name works both as a default value and a flag to indicate that a valid endpoint was found.
const unknownServiceName = "unknown-service"

func jsonAnnotationsToSpanAndEndpoint(annotations []*annotation) (ptrace.Span, *endpoint) {
	// Zipkin V1 annotations have a timestamp so they fit well with ptrace.SpanEvent
	earlyAnnotationTimestamp := int64(math.MaxInt64)
	lateAnnotationTimestamp := int64(math.MinInt64)
	var edpt *endpoint
	span := ptrace.NewSpan()

	// We want to set the span kind from the first annotation that contains information
	// about the span kind. This flags ensures we only set span kind once from
	// the first annotation.
	spanKindIsSet := false

	for _, currAnnotation := range annotations {
		if currAnnotation == nil || currAnnotation.Value == "" {
			continue
		}

		endpointName := unknownServiceName
		if currAnnotation.Endpoint != nil && currAnnotation.Endpoint.ServiceName != "" {
			endpointName = currAnnotation.Endpoint.ServiceName
		}

		// Check if annotation has span kind information.
		annotationHasSpanKind := false
		switch currAnnotation.Value {
		case "cs", "cr", "ms", "mr", "ss", "sr":
			annotationHasSpanKind = true
		}

		// Populate the endpoint if it is not already populated and current endpoint
		// has a service name and span kind.
		if edpt == nil && endpointName != unknownServiceName && annotationHasSpanKind {
			edpt = currAnnotation.Endpoint
		}

		if !spanKindIsSet && annotationHasSpanKind {
			// We have not yet populated span kind, do it now.
			// Translate from Zipkin span kind stored in Value field to Kind/ExternalKind
			// pair of internal fields.
			switch currAnnotation.Value {
			case "cs", "cr":
				span.SetKind(ptrace.SpanKindClient)
			case "ms":
				span.SetKind(ptrace.SpanKindProducer)
			case "mr":
				span.SetKind(ptrace.SpanKindConsumer)
			case "ss", "sr":
				span.SetKind(ptrace.SpanKindServer)
			}

			// Remember that we populated the span kind, so that we don't do it again.
			spanKindIsSet = true
		}

		ts := epochMicrosecondsToTimestamp(currAnnotation.Timestamp)
		if currAnnotation.Timestamp < earlyAnnotationTimestamp {
			earlyAnnotationTimestamp = currAnnotation.Timestamp
			span.SetStartTimestamp(ts)
		}
		if currAnnotation.Timestamp > lateAnnotationTimestamp {
			lateAnnotationTimestamp = currAnnotation.Timestamp
			span.SetEndTimestamp(ts)
		}

		if annotationHasSpanKind {
			// If this annotation is for the send/receive timestamps, no need to create the annotation
			continue
		}

		ev := span.Events().AppendEmpty()
		ev.SetTimestamp(ts)
		ev.SetName(currAnnotation.Value)
	}

	if edpt == nil {
		edpt = &endpoint{
			ServiceName: unknownServiceName,
		}
	}

	return span, edpt
}

func hexToTraceID(hexStr string) (pcommon.TraceID, error) {
	// Per info at https://zipkin.io/zipkin-api/zipkin-api.yaml it should be 16 or 32 characters
	hexLen := len(hexStr)
	if hexLen != 16 && hexLen != 32 {
		return pcommon.NewTraceIDEmpty(), errHexTraceIDWrongLen
	}

	var id [16]byte
	if hexLen == 16 {
		if _, err := hex.Decode(id[8:], []byte(hexStr)); err != nil {
			return pcommon.NewTraceIDEmpty(), err
		}
		if pcommon.TraceID(id).IsEmpty() {
			return pcommon.NewTraceIDEmpty(), errHexTraceIDZero
		}
		return id, nil
	}

	if _, err := hex.Decode(id[:], []byte(hexStr)); err != nil {
		return pcommon.NewTraceIDEmpty(), err
	}
	if pcommon.TraceID(id).IsEmpty() {
		return pcommon.NewTraceIDEmpty(), errHexTraceIDZero
	}
	return id, nil
}

func hexToSpanID(hexStr string) (pcommon.SpanID, error) {
	// Per info at https://zipkin.io/zipkin-api/zipkin-api.yaml it should be 16 characters
	if len(hexStr) != 16 {
		return pcommon.NewSpanIDEmpty(), errHexIDWrongLen
	}

	var id [8]byte
	if _, err := hex.Decode(id[:], []byte(hexStr)); err != nil {
		return pcommon.NewSpanIDEmpty(), err
	}
	if pcommon.SpanID(id).IsEmpty() {
		return pcommon.NewSpanIDEmpty(), errHexIDZero
	}
	return id, nil
}

func epochMicrosecondsToTimestamp(msecs int64) pcommon.Timestamp {
	if msecs <= 0 {
		return pcommon.Timestamp(0)
	}
	return pcommon.Timestamp(uint64(msecs) * 1e3)
}

func getOrCreateNodeRequest(m map[string]ptrace.SpanSlice, td ptrace.Traces, endpoint *endpoint) ptrace.SpanSlice {
	// this private function assumes that the caller never passes an nil endpoint
	nodeKey := endpoint.string()
	ss, found := m[nodeKey]
	if found {
		return ss
	}

	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr(string(conventions.ServiceNameKey), endpoint.ServiceName)
	endpoint.setAttributes(rs.Resource().Attributes())
	ss = rs.ScopeSpans().AppendEmpty().Spans()
	m[nodeKey] = ss
	return ss
}

func (ep *endpoint) string() string {
	return fmt.Sprintf("%s-%s-%s-%d", ep.ServiceName, ep.IPv4, ep.IPv6, ep.Port)
}

func (ep *endpoint) setAttributes(dest pcommon.Map) {
	if ep.IPv4 == "" && ep.IPv6 == "" && ep.Port == 0 {
		return
	}

	if ep.IPv4 != "" {
		dest.PutStr("ipv4", ep.IPv4)
	}
	if ep.IPv6 != "" {
		dest.PutStr("ipv6", ep.IPv6)
	}
	if ep.Port != 0 {
		dest.PutStr("port", strconv.Itoa(int(ep.Port)))
	}
}

func setTimestampsIfUnset(span ptrace.Span) {
	// zipkin allows timestamp to be unset, but opentelemetry-collector expects it to have a value.
	// If this is unset, the conversion from open census to the internal trace format breaks
	// what should be an identity transformation oc -> internal -> oc
	if span.StartTimestamp() == 0 {
		now := time.Now()
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(now))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(now))
		span.Attributes().PutBool(zipkin.StartTimeAbsent, true)
	}
}
