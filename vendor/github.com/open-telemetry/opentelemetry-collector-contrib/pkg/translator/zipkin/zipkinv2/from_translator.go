// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"strconv"
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventionsv112 "go.opentelemetry.io/otel/semconv/v1.12.0"
	conventionsv125 "go.opentelemetry.io/otel/semconv/v1.25.0"
	conventions "go.opentelemetry.io/otel/semconv/v1.38.0"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/tracetranslator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"
	idutils "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/core/xidutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/internal/zipkin"
)

const (
	spanEventDataFormat = "%s|%s|%d"
	spanLinkDataFormat  = "%s|%s|%s|%s|%d"
)

var sampled = true

// FromTranslator converts from pdata to Zipkin data model.
type FromTranslator struct{}

// FromTraces translates internal trace data into Zipkin v2 spans.
// Returns a slice of Zipkin SpanModel's.
func (FromTranslator) FromTraces(td ptrace.Traces) ([]*zipkinmodel.SpanModel, error) {
	resourceSpans := td.ResourceSpans()
	if resourceSpans.Len() == 0 {
		return nil, nil
	}

	zSpans := make([]*zipkinmodel.SpanModel, 0, td.SpanCount())

	for i := 0; i < resourceSpans.Len(); i++ {
		batch, err := resourceSpansToZipkinSpans(resourceSpans.At(i), td.SpanCount()/resourceSpans.Len())
		if err != nil {
			return zSpans, err
		}
		if batch != nil {
			zSpans = append(zSpans, batch...)
		}
	}

	return zSpans, nil
}

func resourceSpansToZipkinSpans(rs ptrace.ResourceSpans, estSpanCount int) ([]*zipkinmodel.SpanModel, error) {
	resource := rs.Resource()
	ilss := rs.ScopeSpans()

	if resource.Attributes().Len() == 0 && ilss.Len() == 0 {
		return nil, nil
	}

	localServiceName, zTags := resourceToZipkinEndpointServiceNameAndAttributeMap(resource)

	zSpans := make([]*zipkinmodel.SpanModel, 0, estSpanCount)
	for i := 0; i < ilss.Len(); i++ {
		ils := ilss.At(i)
		extractScopeTags(ils.Scope(), zTags)
		spans := ils.Spans()
		for j := 0; j < spans.Len(); j++ {
			zSpan, err := spanToZipkinSpan(spans.At(j), localServiceName, zTags)
			if err != nil {
				return zSpans, err
			}
			zSpans = append(zSpans, zSpan)
		}
	}

	return zSpans, nil
}

func extractScopeTags(il pcommon.InstrumentationScope, zTags map[string]string) {
	attrs := il.Attributes()
	for k, v := range attrs.All() {
		zTags[k] = v.AsString()
	}

	if ilName := il.Name(); ilName != "" {
		zTags[string(conventionsv125.OTelLibraryNameKey)] = ilName
	}
	if ilVer := il.Version(); ilVer != "" {
		zTags[string(conventionsv125.OTelLibraryVersionKey)] = ilVer
	}
}

func spanToZipkinSpan(
	span ptrace.Span,
	localServiceName string,
	zTags map[string]string,
) (*zipkinmodel.SpanModel, error) {
	tags := aggregateSpanTags(span, zTags)

	zs := &zipkinmodel.SpanModel{}

	if span.TraceID().IsEmpty() {
		return zs, errors.New("TraceID is invalid")
	}
	zs.TraceID = convertTraceID(span.TraceID())
	if span.SpanID().IsEmpty() {
		return zs, errors.New("SpanID is invalid")
	}
	zs.ID = convertSpanID(span.SpanID())

	traceState := span.TraceState().AsRaw()
	if traceState != "" {
		tags[tracetranslator.TagW3CTraceState] = traceState
	}

	if !span.ParentSpanID().IsEmpty() {
		id := convertSpanID(span.ParentSpanID())
		zs.ParentID = &id
	}

	zs.Sampled = &sampled
	zs.Name = span.Name()
	startTime := span.StartTimestamp().AsTime()

	// leave timestamp unset on zs (zipkin span) if
	// otel span startTime is zero.  Zipkin has a
	// case where startTime is not set on the span.
	// See handling of this (and setting of otel span
	// to unix time zero) in zipkinv2_to_traces.go
	if startTime.Unix() != 0 {
		zs.Timestamp = startTime
	}

	if span.EndTimestamp() != 0 {
		zs.Duration = time.Duration(span.EndTimestamp() - span.StartTimestamp())
	}
	zs.Kind = spanKindToZipkinKind(span.Kind())
	if span.Kind() == ptrace.SpanKindInternal {
		tags[tracetranslator.TagSpanKind] = "internal"
	}

	redundantKeys := make(map[string]bool, 8)
	zs.LocalEndpoint = zipkinEndpointFromTags(tags, localServiceName, false, redundantKeys)
	zs.RemoteEndpoint = zipkinEndpointFromTags(tags, "", true, redundantKeys)

	removeRedundantTags(redundantKeys, tags)
	populateStatus(span.Status(), zs, tags)

	if err := spanEventsToZipkinAnnotations(span.Events(), zs); err != nil {
		return nil, err
	}
	if err := spanLinksToZipkinTags(span.Links(), tags); err != nil {
		return nil, err
	}

	zs.Tags = tags

	return zs, nil
}

func populateStatus(status ptrace.Status, zs *zipkinmodel.SpanModel, tags map[string]string) {
	if status.Code() == ptrace.StatusCodeError {
		tags[tracetranslator.TagError] = "true"
	} else {
		// The error tag should only be set if Status is Error. If a boolean version
		// ({"error":false} or {"error":"false"}) is present, it SHOULD be removed.
		// Zipkin will treat any span with error sent as failed.
		delete(tags, tracetranslator.TagError)
	}

	// Per specs, Span Status MUST be reported as a key-value pair in tags to Zipkin, unless it is UNSET.
	// In the latter case it MUST NOT be reported.
	// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/sdk_exporters/zipkin.md#status
	if status.Code() == ptrace.StatusCodeUnset {
		return
	}

	tags[string(conventions.OTelStatusCodeKey)] = traceutil.StatusCodeStr(status.Code())
	if status.Message() != "" {
		tags[string(conventions.OTelStatusDescriptionKey)] = status.Message()
		zs.Err = fmt.Errorf("%s", status.Message())
	}
}

func aggregateSpanTags(span ptrace.Span, zTags map[string]string) map[string]string {
	tags := make(map[string]string)
	maps.Copy(tags, zTags)
	spanTags := attributeMapToStringMap(span.Attributes())
	maps.Copy(tags, spanTags)
	return tags
}

func spanEventsToZipkinAnnotations(events ptrace.SpanEventSlice, zs *zipkinmodel.SpanModel) error {
	if events.Len() > 0 {
		zAnnos := make([]zipkinmodel.Annotation, events.Len())
		for i := 0; i < events.Len(); i++ {
			event := events.At(i)
			if event.Attributes().Len() == 0 && event.DroppedAttributesCount() == 0 {
				zAnnos[i] = zipkinmodel.Annotation{
					Timestamp: event.Timestamp().AsTime(),
					Value:     event.Name(),
				}
			} else {
				jsonStr, err := json.Marshal(event.Attributes().AsRaw())
				if err != nil {
					return err
				}
				zAnnos[i] = zipkinmodel.Annotation{
					Timestamp: event.Timestamp().AsTime(),
					Value: fmt.Sprintf(spanEventDataFormat, event.Name(), jsonStr,
						event.DroppedAttributesCount()),
				}
			}
		}
		zs.Annotations = zAnnos
	}
	return nil
}

func spanLinksToZipkinTags(links ptrace.SpanLinkSlice, zTags map[string]string) error {
	for i := 0; i < links.Len(); i++ {
		link := links.At(i)
		key := fmt.Sprintf("otlp.link.%d", i)
		jsonStr, err := json.Marshal(link.Attributes().AsRaw())
		if err != nil {
			return err
		}
		zTags[key] = fmt.Sprintf(spanLinkDataFormat, traceutil.TraceIDToHexOrEmptyString(link.TraceID()),
			traceutil.SpanIDToHexOrEmptyString(link.SpanID()), link.TraceState().AsRaw(), jsonStr, link.DroppedAttributesCount())
	}
	return nil
}

func attributeMapToStringMap(attrMap pcommon.Map) map[string]string {
	rawMap := make(map[string]string)
	for k, v := range attrMap.All() {
		rawMap[k] = v.AsString()
	}
	return rawMap
}

func removeRedundantTags(redundantKeys map[string]bool, zTags map[string]string) {
	for k, v := range redundantKeys {
		if v {
			delete(zTags, k)
		}
	}
}

func resourceToZipkinEndpointServiceNameAndAttributeMap(
	resource pcommon.Resource,
) (serviceName string, zTags map[string]string) {
	zTags = make(map[string]string)
	attrs := resource.Attributes()
	if attrs.Len() == 0 {
		return tracetranslator.ResourceNoServiceName, zTags
	}

	for k, v := range attrs.All() {
		zTags[k] = v.AsString()
	}

	serviceName = extractZipkinServiceName(zTags)
	return serviceName, zTags
}

func extractZipkinServiceName(zTags map[string]string) string {
	var serviceName string
	if sn, ok := zTags[string(conventions.ServiceNameKey)]; ok {
		serviceName = sn
		delete(zTags, string(conventions.ServiceNameKey))
	} else if fn, ok := zTags[string(conventions.FaaSNameKey)]; ok {
		serviceName = fn
		delete(zTags, string(conventions.FaaSNameKey))
		zTags[zipkin.TagServiceNameSource] = string(conventions.FaaSNameKey)
	} else if fn, ok := zTags[string(conventions.K8SDeploymentNameKey)]; ok {
		serviceName = fn
		delete(zTags, string(conventions.K8SDeploymentNameKey))
		zTags[zipkin.TagServiceNameSource] = string(conventions.K8SDeploymentNameKey)
	} else if fn, ok := zTags[string(conventions.ProcessExecutableNameKey)]; ok {
		serviceName = fn
		delete(zTags, string(conventions.ProcessExecutableNameKey))
		zTags[zipkin.TagServiceNameSource] = string(conventions.ProcessExecutableNameKey)
	} else {
		serviceName = tracetranslator.ResourceNoServiceName
	}
	return serviceName
}

func spanKindToZipkinKind(kind ptrace.SpanKind) zipkinmodel.Kind {
	switch kind {
	case ptrace.SpanKindClient:
		return zipkinmodel.Client
	case ptrace.SpanKindServer:
		return zipkinmodel.Server
	case ptrace.SpanKindProducer:
		return zipkinmodel.Producer
	case ptrace.SpanKindConsumer:
		return zipkinmodel.Consumer
	default:
		return zipkinmodel.Undetermined
	}
}

func zipkinEndpointFromTags(
	zTags map[string]string,
	localServiceName string,
	remoteEndpoint bool,
	redundantKeys map[string]bool,
) (endpoint *zipkinmodel.Endpoint) {
	serviceName := localServiceName
	if peerSvc, ok := zTags[string(conventions.PeerServiceKey)]; ok && remoteEndpoint {
		serviceName = peerSvc
		redundantKeys[string(conventions.PeerServiceKey)] = true
	}

	var ipKey, portKey string
	if remoteEndpoint {
		ipKey, portKey = string(conventionsv112.NetPeerIPKey), string(conventionsv125.NetPeerPortKey)
	} else {
		ipKey, portKey = string(conventionsv112.NetHostIPKey), string(conventionsv125.NetHostPortKey)
	}

	var ip net.IP
	ipv6Selected := false
	if ipStr, ok := zTags[ipKey]; ok {
		ipv6Selected = isIPv6Address(ipStr)
		ip = net.ParseIP(ipStr)
		redundantKeys[ipKey] = true
	}

	var port uint64
	if portStr, ok := zTags[portKey]; ok {
		port, _ = strconv.ParseUint(portStr, 10, 16)
		redundantKeys[portKey] = true
	}

	if serviceName == "" && ip == nil {
		return nil
	}

	zEndpoint := &zipkinmodel.Endpoint{
		ServiceName: serviceName,
		Port:        uint16(port),
	}
	if ipv6Selected {
		zEndpoint.IPv6 = ip
	} else {
		zEndpoint.IPv4 = ip
	}

	return zEndpoint
}

func isIPv6Address(ipStr string) bool {
	for i := 0; i < len(ipStr); i++ {
		if ipStr[i] == ':' {
			return true
		}
	}
	return false
}

func convertTraceID(t pcommon.TraceID) zipkinmodel.TraceID {
	h, l := idutils.TraceIDToUInt64Pair(t)
	return zipkinmodel.TraceID{High: h, Low: l}
}

func convertSpanID(s pcommon.SpanID) zipkinmodel.ID {
	return zipkinmodel.ID(idutils.SpanIDToUInt64(s))
}
