// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions112 "go.opentelemetry.io/otel/semconv/v1.12.0"
	conventions "go.opentelemetry.io/otel/semconv/v1.15.0"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/occonventions"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/tracetranslator"
	idutils "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/core/xidutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/internal/zipkin"
)

// ToTranslator converts from Zipkin data model to pdata.
type ToTranslator struct {
	// ParseStringTags should be set to true if tags should be converted to numbers when possible.
	ParseStringTags bool
}

// ToTraces translates Zipkin v2 spans into ptrace.Traces.
func (t ToTranslator) ToTraces(zipkinSpans []*zipkinmodel.SpanModel) (ptrace.Traces, error) {
	traceData := ptrace.NewTraces()
	if len(zipkinSpans) == 0 {
		return traceData, nil
	}

	sort.Sort(byOTLPTypes(zipkinSpans))

	rss := traceData.ResourceSpans()
	prevServiceName := ""
	prevInstrLibName := ""
	ilsIsNew := true
	var curRscSpans ptrace.ResourceSpans
	var curILSpans ptrace.ScopeSpans
	var curSpans ptrace.SpanSlice
	for _, zspan := range zipkinSpans {
		if zspan == nil {
			continue
		}
		tags := copySpanTags(zspan.Tags)
		localServiceName := extractLocalServiceName(zspan)
		if localServiceName != prevServiceName {
			prevServiceName = localServiceName
			curRscSpans = rss.AppendEmpty()
			populateResourceFromZipkinSpan(tags, localServiceName, curRscSpans.Resource())
			prevInstrLibName = ""
			ilsIsNew = true
		}
		instrLibName := extractInstrumentationLibrary(zspan)
		if instrLibName != prevInstrLibName || ilsIsNew {
			prevInstrLibName = instrLibName
			curILSpans = curRscSpans.ScopeSpans().AppendEmpty()
			ilsIsNew = false
			populateILFromZipkinSpan(tags, instrLibName, curILSpans.Scope())
			curSpans = curILSpans.Spans()
		}
		err := zSpanToInternal(zspan, tags, curSpans.AppendEmpty(), t.ParseStringTags)
		if err != nil {
			return traceData, err
		}
	}

	return traceData, nil
}

func getResourceSemanticConventionAttributeNames() []string {
	return []string{
		string(conventions.CloudProviderKey),
		string(conventions.CloudAccountIDKey),
		string(conventions.CloudRegionKey),
		string(conventions.CloudAvailabilityZoneKey),
		string(conventions.CloudPlatformKey),
		string(conventions.AWSECSContainerARNKey),
		string(conventions.AWSECSClusterARNKey),
		string(conventions.AWSECSLaunchtypeKey),
		string(conventions.AWSECSTaskARNKey),
		string(conventions.AWSECSTaskFamilyKey),
		string(conventions.AWSECSTaskRevisionKey),
		string(conventions.AWSEKSClusterARNKey),
		string(conventions.AWSLogGroupNamesKey),
		string(conventions.AWSLogGroupARNsKey),
		string(conventions.AWSLogStreamNamesKey),
		string(conventions.AWSLogStreamARNsKey),
		string(conventions.ContainerNameKey),
		string(conventions.ContainerIDKey),
		string(conventions.ContainerRuntimeKey),
		string(conventions.ContainerImageNameKey),
		string(conventions.ContainerImageTagKey),
		string(conventions.DeploymentEnvironmentKey),
		string(conventions.DeviceIDKey),
		string(conventions.DeviceModelIdentifierKey),
		string(conventions.DeviceModelNameKey),
		string(conventions.DeviceManufacturerKey),
		string(conventions.FaaSNameKey),
		string(conventions.FaaSIDKey),
		string(conventions.FaaSVersionKey),
		string(conventions.FaaSInstanceKey),
		string(conventions.FaaSMaxMemoryKey),
		string(conventions.HostIDKey),
		string(conventions.HostNameKey),
		string(conventions.HostTypeKey),
		string(conventions.HostArchKey),
		string(conventions.HostImageNameKey),
		string(conventions.HostImageIDKey),
		string(conventions.HostImageVersionKey),
		string(conventions.K8SClusterNameKey),
		string(conventions.K8SNodeNameKey),
		string(conventions.K8SNodeUIDKey),
		string(conventions.K8SNamespaceNameKey),
		string(conventions.K8SPodUIDKey),
		string(conventions.K8SPodNameKey),
		string(conventions.K8SContainerNameKey),
		string(conventions.K8SContainerRestartCountKey),
		string(conventions.K8SReplicaSetUIDKey),
		string(conventions.K8SReplicaSetNameKey),
		string(conventions.K8SDeploymentUIDKey),
		string(conventions.K8SDeploymentNameKey),
		string(conventions.K8SStatefulSetUIDKey),
		string(conventions.K8SStatefulSetNameKey),
		string(conventions.K8SDaemonSetUIDKey),
		string(conventions.K8SDaemonSetNameKey),
		string(conventions.K8SJobUIDKey),
		string(conventions.K8SJobNameKey),
		string(conventions.K8SCronJobUIDKey),
		string(conventions.K8SCronJobNameKey),
		string(conventions.OSTypeKey),
		string(conventions.OSDescriptionKey),
		string(conventions.OSNameKey),
		string(conventions.OSVersionKey),
		string(conventions.ProcessPIDKey),
		string(conventions.ProcessExecutableNameKey),
		string(conventions.ProcessExecutablePathKey),
		string(conventions.ProcessCommandKey),
		string(conventions.ProcessCommandLineKey),
		string(conventions.ProcessCommandArgsKey),
		string(conventions.ProcessOwnerKey),
		string(conventions.ProcessRuntimeNameKey),
		string(conventions.ProcessRuntimeVersionKey),
		string(conventions.ProcessRuntimeDescriptionKey),
		string(conventions.ServiceNameKey),
		string(conventions.ServiceNamespaceKey),
		string(conventions.ServiceInstanceIDKey),
		string(conventions.ServiceVersionKey),
		string(conventions.TelemetrySDKNameKey),
		string(conventions.TelemetrySDKLanguageKey),
		string(conventions.TelemetrySDKVersionKey),
		string(conventions.TelemetryAutoVersionKey),
		string(conventions.WebEngineNameKey),
		string(conventions.WebEngineVersionKey),
		string(conventions.WebEngineDescriptionKey),
	}
}

var nonSpanAttributes = func() map[string]struct{} {
	attrs := make(map[string]struct{})
	for _, key := range getResourceSemanticConventionAttributeNames() {
		attrs[key] = struct{}{}
	}
	attrs[zipkin.TagServiceNameSource] = struct{}{}
	attrs[string(conventions.OtelLibraryNameKey)] = struct{}{}
	attrs[string(conventions.OtelLibraryVersionKey)] = struct{}{}
	attrs[occonventions.AttributeProcessStartTime] = struct{}{}
	attrs[occonventions.AttributeExporterVersion] = struct{}{}
	attrs[string(conventions.ProcessPIDKey)] = struct{}{}
	attrs[occonventions.AttributeResourceType] = struct{}{}
	return attrs
}()

// Custom Sort on
type byOTLPTypes []*zipkinmodel.SpanModel

func (b byOTLPTypes) Len() int {
	return len(b)
}

func (b byOTLPTypes) Less(i, j int) bool {
	diff := strings.Compare(extractLocalServiceName(b[i]), extractLocalServiceName(b[j]))
	if diff != 0 {
		return diff <= 0
	}
	diff = strings.Compare(extractInstrumentationLibrary(b[i]), extractInstrumentationLibrary(b[j]))
	return diff <= 0
}

func (b byOTLPTypes) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func zSpanToInternal(zspan *zipkinmodel.SpanModel, tags map[string]string, dest ptrace.Span, parseStringTags bool) error {
	dest.SetTraceID(idutils.UInt64ToTraceID(zspan.TraceID.High, zspan.TraceID.Low))
	dest.SetSpanID(idutils.UInt64ToSpanID(uint64(zspan.ID)))
	if value, ok := tags[tracetranslator.TagW3CTraceState]; ok {
		dest.TraceState().FromRaw(value)
		delete(tags, tracetranslator.TagW3CTraceState)
	}
	parentID := zspan.ParentID
	if parentID != nil && *parentID != zspan.ID {
		dest.SetParentSpanID(idutils.UInt64ToSpanID(uint64(*parentID)))
	}

	dest.SetName(zspan.Name)
	dest.SetKind(zipkinKindToSpanKind(zspan.Kind, tags))

	populateSpanStatus(tags, dest.Status())
	if err := zTagsToSpanLinks(tags, dest.Links()); err != nil {
		return err
	}

	attrs := dest.Attributes()
	attrs.EnsureCapacity(len(tags))
	if err := zTagsToInternalAttrs(zspan, tags, attrs, parseStringTags); err != nil {
		return err
	}

	setTimestampsV2(zspan, dest, attrs)

	err := populateSpanEvents(zspan, dest.Events())
	return err
}

func populateSpanStatus(tags map[string]string, status ptrace.Status) {
	if value, ok := tags[string(conventions.OtelStatusCodeKey)]; ok {
		status.SetCode(ptrace.StatusCode(statusCodeValue[value]))
		delete(tags, string(conventions.OtelStatusCodeKey))
		if value, ok := tags[string(conventions.OtelStatusDescriptionKey)]; ok {
			status.SetMessage(value)
			delete(tags, string(conventions.OtelStatusDescriptionKey))
		}
	}

	if val, ok := tags[tracetranslator.TagError]; ok {
		status.SetCode(ptrace.StatusCodeError)
		if val == "true" {
			delete(tags, tracetranslator.TagError)
		}
	}
}

func zipkinKindToSpanKind(kind zipkinmodel.Kind, tags map[string]string) ptrace.SpanKind {
	switch kind {
	case zipkinmodel.Client:
		return ptrace.SpanKindClient
	case zipkinmodel.Server:
		return ptrace.SpanKindServer
	case zipkinmodel.Producer:
		return ptrace.SpanKindProducer
	case zipkinmodel.Consumer:
		return ptrace.SpanKindConsumer
	default:
		if value, ok := tags[tracetranslator.TagSpanKind]; ok {
			delete(tags, tracetranslator.TagSpanKind)
			if value == "internal" {
				return ptrace.SpanKindInternal
			}
		}
		return ptrace.SpanKindUnspecified
	}
}

func zTagsToSpanLinks(tags map[string]string, dest ptrace.SpanLinkSlice) error {
	for i := 0; i < 128; i++ {
		key := fmt.Sprintf("otlp.link.%d", i)
		val, ok := tags[key]
		if !ok {
			return nil
		}
		delete(tags, key)

		parts := strings.Split(val, "|")
		partCnt := len(parts)
		if partCnt < 5 {
			continue
		}
		link := dest.AppendEmpty()

		// Convert trace id.
		rawTrace := [16]byte{}
		errTrace := unmarshalJSON(rawTrace[:], []byte(parts[0]))
		if errTrace != nil {
			return errTrace
		}
		link.SetTraceID(rawTrace)

		// Convert span id.
		rawSpan := [8]byte{}
		errSpan := unmarshalJSON(rawSpan[:], []byte(parts[1]))
		if errSpan != nil {
			return errSpan
		}
		link.SetSpanID(rawSpan)

		link.TraceState().FromRaw(parts[2])

		var jsonStr string
		if partCnt == 5 {
			jsonStr = parts[3]
		} else {
			jsonParts := parts[3 : partCnt-1]
			jsonStr = strings.Join(jsonParts, "|")
		}
		var attrs map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &attrs); err != nil {
			return err
		}
		if err := jsonMapToAttributeMap(attrs, link.Attributes()); err != nil {
			return err
		}

		dropped, errDropped := strconv.ParseUint(parts[partCnt-1], 10, 32)
		if errDropped != nil {
			return errDropped
		}
		link.SetDroppedAttributesCount(uint32(dropped))
	}
	return nil
}

func populateSpanEvents(zspan *zipkinmodel.SpanModel, events ptrace.SpanEventSlice) error {
	events.EnsureCapacity(len(zspan.Annotations))
	for _, anno := range zspan.Annotations {
		event := events.AppendEmpty()
		event.SetTimestamp(pcommon.NewTimestampFromTime(anno.Timestamp))

		parts := strings.Split(anno.Value, "|")
		partCnt := len(parts)
		event.SetName(parts[0])
		if partCnt < 3 {
			continue
		}

		var jsonStr string
		if partCnt == 3 {
			jsonStr = parts[1]
		} else {
			jsonParts := parts[1 : partCnt-1]
			jsonStr = strings.Join(jsonParts, "|")
		}
		var attrs map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &attrs); err != nil {
			return err
		}
		if err := jsonMapToAttributeMap(attrs, event.Attributes()); err != nil {
			return err
		}

		dropped, errDropped := strconv.ParseUint(parts[partCnt-1], 10, 32)
		if errDropped != nil {
			return errDropped
		}
		event.SetDroppedAttributesCount(uint32(dropped))
	}
	return nil
}

func jsonMapToAttributeMap(attrs map[string]any, dest pcommon.Map) error {
	for key, val := range attrs {
		if s, ok := val.(string); ok {
			dest.PutStr(key, s)
		} else if d, ok := val.(float64); ok {
			if math.Mod(d, 1.0) == 0.0 {
				dest.PutInt(key, int64(d))
			} else {
				dest.PutDouble(key, d)
			}
		} else if b, ok := val.(bool); ok {
			dest.PutBool(key, b)
		}
	}
	return nil
}

func zTagsToInternalAttrs(zspan *zipkinmodel.SpanModel, tags map[string]string, dest pcommon.Map, parseStringTags bool) error {
	parseErr := tagsToAttributeMap(tags, dest, parseStringTags)
	if zspan.LocalEndpoint != nil {
		if zspan.LocalEndpoint.IPv4 != nil {
			dest.PutStr(string(conventions112.NetHostIPKey), zspan.LocalEndpoint.IPv4.String())
		}
		if zspan.LocalEndpoint.IPv6 != nil {
			dest.PutStr(string(conventions112.NetHostIPKey), zspan.LocalEndpoint.IPv6.String())
		}
		if zspan.LocalEndpoint.Port > 0 {
			dest.PutInt(string(conventions.NetHostPortKey), int64(zspan.LocalEndpoint.Port))
		}
	}
	if zspan.RemoteEndpoint != nil {
		if zspan.RemoteEndpoint.ServiceName != "" {
			dest.PutStr(string(conventions.PeerServiceKey), zspan.RemoteEndpoint.ServiceName)
		}
		if zspan.RemoteEndpoint.IPv4 != nil {
			dest.PutStr(string(conventions112.NetPeerIPKey), zspan.RemoteEndpoint.IPv4.String())
		}
		if zspan.RemoteEndpoint.IPv6 != nil {
			dest.PutStr(string(conventions112.NetPeerIPKey), zspan.RemoteEndpoint.IPv6.String())
		}
		if zspan.RemoteEndpoint.Port > 0 {
			dest.PutInt(string(conventions.NetPeerPortKey), int64(zspan.RemoteEndpoint.Port))
		}
	}
	return parseErr
}

func tagsToAttributeMap(tags map[string]string, dest pcommon.Map, parseStringTags bool) error {
	var parseErr error
	for key, val := range tags {
		if _, ok := nonSpanAttributes[key]; ok {
			continue
		}

		if parseStringTags {
			switch zipkin.DetermineValueType(val) {
			case pcommon.ValueTypeInt:
				iValue, _ := strconv.ParseInt(val, 10, 64)
				dest.PutInt(key, iValue)
			case pcommon.ValueTypeDouble:
				fValue, _ := strconv.ParseFloat(val, 64)
				dest.PutDouble(key, fValue)
			case pcommon.ValueTypeBool:
				bValue, _ := strconv.ParseBool(val)
				dest.PutBool(key, bValue)
			default:
				dest.PutStr(key, val)
			}
		} else {
			dest.PutStr(key, val)
		}
	}
	return parseErr
}

func populateResourceFromZipkinSpan(tags map[string]string, localServiceName string, resource pcommon.Resource) {
	if localServiceName == tracetranslator.ResourceNoServiceName {
		return
	}

	if len(tags) == 0 {
		resource.Attributes().PutStr(string(conventions.ServiceNameKey), localServiceName)
		return
	}

	snSource := tags[zipkin.TagServiceNameSource]
	if snSource == "" {
		resource.Attributes().PutStr(string(conventions.ServiceNameKey), localServiceName)
	} else {
		resource.Attributes().PutStr(snSource, localServiceName)
	}
	delete(tags, zipkin.TagServiceNameSource)

	for key := range nonSpanAttributes {
		if key == string(conventions.OtelLibraryNameKey) || key == string(conventions.OtelLibraryVersionKey) {
			continue
		}
		if value, ok := tags[key]; ok {
			resource.Attributes().PutStr(key, value)
			delete(tags, key)
		}
	}
}

func populateILFromZipkinSpan(tags map[string]string, instrLibName string, library pcommon.InstrumentationScope) {
	if instrLibName == "" {
		return
	}
	if value, ok := tags[string(conventions.OtelLibraryNameKey)]; ok {
		library.SetName(value)
		delete(tags, string(conventions.OtelLibraryNameKey))
	}
	if value, ok := tags[string(conventions.OtelLibraryVersionKey)]; ok {
		library.SetVersion(value)
		delete(tags, string(conventions.OtelLibraryVersionKey))
	}
}

func copySpanTags(tags map[string]string) map[string]string {
	dest := make(map[string]string, len(tags))
	for key, val := range tags {
		dest[key] = val
	}
	return dest
}

func extractLocalServiceName(zspan *zipkinmodel.SpanModel) string {
	if zspan == nil || zspan.LocalEndpoint == nil || zspan.LocalEndpoint.ServiceName == "" {
		return tracetranslator.ResourceNoServiceName
	}
	return zspan.LocalEndpoint.ServiceName
}

func extractInstrumentationLibrary(zspan *zipkinmodel.SpanModel) string {
	if zspan == nil || len(zspan.Tags) == 0 {
		return ""
	}
	return zspan.Tags[string(conventions.OtelLibraryNameKey)]
}

func setTimestampsV2(zspan *zipkinmodel.SpanModel, dest ptrace.Span, destAttrs pcommon.Map) {
	// zipkin allows timestamp to be unset, but otel span expects startTimestamp to have a value.
	// unset gets converted to zero on the zspan object during json deserialization because
	// time.Time (the type of Timestamp field) cannot be nil.  If timestamp is zero, the
	// conversion from this internal format back to zipkin format in zipkin exporter fails.
	// Instead, set to *unix* time zero, and convert back in traces_to_zipkinv2.go
	if zspan.Timestamp.IsZero() {
		unixTimeZero := pcommon.NewTimestampFromTime(time.Unix(0, 0))
		zeroPlusDuration := pcommon.NewTimestampFromTime(time.Unix(0, 0).Add(zspan.Duration))
		dest.SetStartTimestamp(unixTimeZero)
		dest.SetEndTimestamp(zeroPlusDuration)

		destAttrs.PutBool(zipkin.StartTimeAbsent, true)
	} else {
		dest.SetStartTimestamp(pcommon.NewTimestampFromTime(zspan.Timestamp))
		dest.SetEndTimestamp(pcommon.NewTimestampFromTime(zspan.Timestamp.Add(zspan.Duration)))
	}
}

// unmarshalJSON inflates trace id from hex string, possibly enclosed in quotes.
// TODO: Find a way to avoid this duplicate code. Consider to expose this in pdata.
func unmarshalJSON(dst []byte, src []byte) error {
	if l := len(src); l >= 2 && src[0] == '"' && src[l-1] == '"' {
		src = src[1 : l-1]
	}
	nLen := len(src)
	if nLen == 0 {
		return nil
	}

	if len(dst) != hex.DecodedLen(nLen) {
		return errors.New("invalid length for ID")
	}

	_, err := hex.Decode(dst, src)
	if err != nil {
		return fmt.Errorf("cannot unmarshal ID from string '%s': %w", string(src), err)
	}
	return nil
}

// TODO: Find a way to avoid this duplicate code. Consider to expose this in pdata.
var statusCodeValue = map[string]int32{
	"STATUS_CODE_UNSET": 0,
	"STATUS_CODE_OK":    1,
	"STATUS_CODE_ERROR": 2,
	// As reported in https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/14965
	// The Zipkin exporter used a different set of names when serializing span state.
	"Unset": 0,
	"Ok":    1,
	"Error": 2,
}
