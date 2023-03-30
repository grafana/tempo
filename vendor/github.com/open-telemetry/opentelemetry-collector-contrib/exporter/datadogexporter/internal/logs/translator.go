// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/logs"

import (
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes"
	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes/source"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"go.uber.org/zap"
)

const (
	// This set of constants specify the keys of the attributes that will be used to preserve
	// the original OpenTelemetry logs attributes.
	otelNamespace      = "otel"
	otelTraceID        = otelNamespace + ".trace_id"
	otelSpanID         = otelNamespace + ".span_id"
	otelSeverityNumber = otelNamespace + ".severity_number"
	otelSeverityText   = otelNamespace + ".severity_text"
	otelTimestamp      = otelNamespace + ".timestamp"
)
const (
	// This set of constants specify the keys of the attributes that will be used to represent Datadog
	// counterparts to the OpenTelemetry Logs attributes.
	ddNamespace = "dd"
	ddTraceID   = ddNamespace + ".trace_id"
	ddSpanID    = ddNamespace + ".span_id"
	ddStatus    = "status"
	ddTimestamp = "@timestamp"
)

const (
	logLevelTrace = "trace"
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
	logLevelFatal = "fatal"
)

// otelTag specifies a tag to be added to all logs sent from the Datadog exporter
const otelTag = "otel_source:datadog_exporter"

// Transform converts the log record in lr, which came in with the resource in res to a Datadog log item.
// the variable specifies if the log body should be sent as an attribute or as a plain message.
func Transform(lr plog.LogRecord, res pcommon.Resource, logger *zap.Logger) datadogV2.HTTPLogItem {
	host, service := extractHostNameAndServiceName(res.Attributes(), lr.Attributes())

	l := datadogV2.HTTPLogItem{
		AdditionalProperties: make(map[string]string),
	}
	if host != "" {
		l.Hostname = datadog.PtrString(host)
	}
	if service != "" {
		l.Service = datadog.PtrString(service)
	}

	// we need to set log attributes as AdditionalProperties
	// AdditionalProperties are treated as Datadog Log Attributes
	var status string
	lr.Attributes().Range(func(k string, v pcommon.Value) bool {
		switch strings.ToLower(k) {
		// set of remapping are taken from Datadog Backend
		case "msg", "message", "log":
			l.Message = v.AsString()
		case "status", "severity", "level", "syslog.severity":
			status = v.AsString()
		case "traceid", "contextmap.traceid", "oteltraceid":
			traceID, err := decodeTraceID(v.AsString())
			if err != nil {
				logger.Warn("failed to decode trace id",
					zap.String("trace_id", v.AsString()),
					zap.Error(err))
				break
			}
			if l.AdditionalProperties[ddTraceID] == "" {
				l.AdditionalProperties[ddTraceID] = strconv.FormatUint(traceIDToUint64(traceID), 10)
				l.AdditionalProperties[otelTraceID] = v.AsString()
			}
		case "spanid", "contextmap.spanid", "otelspanid":
			spanID, err := decodeSpanID(v.AsString())
			if err != nil {
				logger.Warn("failed to decode span id",
					zap.String("span_id", v.AsString()),
					zap.Error(err))
				break
			}
			if l.AdditionalProperties[ddSpanID] == "" {
				l.AdditionalProperties[ddSpanID] = strconv.FormatUint(spanIDToUint64(spanID), 10)
				l.AdditionalProperties[otelSpanID] = v.AsString()
			}
		case "ddtags":
			var tags = append(attributes.TagsFromAttributes(res.Attributes()), v.AsString(), otelTag)
			tagStr := strings.Join(tags, ",")
			l.Ddtags = datadog.PtrString(tagStr)
		default:
			l.AdditionalProperties[k] = v.AsString()
		}
		return true
	})
	if traceID := lr.TraceID(); !traceID.IsEmpty() {
		l.AdditionalProperties[ddTraceID] = strconv.FormatUint(traceIDToUint64(traceID), 10)
		l.AdditionalProperties[otelTraceID] = hex.EncodeToString(traceID[:])
	}
	if spanID := lr.SpanID(); !spanID.IsEmpty() {
		l.AdditionalProperties[ddSpanID] = strconv.FormatUint(spanIDToUint64(spanID), 10)
		l.AdditionalProperties[otelSpanID] = hex.EncodeToString(spanID[:])
	}

	// we want to use the serverity that client has set on the log and let Datadog backend
	// decide the appropriate level
	if lr.SeverityText() != "" {
		if status == "" {
			status = lr.SeverityText()
		}
		l.AdditionalProperties[otelSeverityText] = lr.SeverityText()
	}
	if lr.SeverityNumber() != 0 {
		if status == "" {
			status = statusFromSeverityNumber(lr.SeverityNumber())
		}
		l.AdditionalProperties[otelSeverityNumber] = strconv.Itoa(int(lr.SeverityNumber()))
	}
	l.AdditionalProperties[ddStatus] = status
	// for Datadog to use the same timestamp we need to set the additional property of "@timestamp"
	if lr.Timestamp() != 0 {
		// we are retaining the nano second precision in this property
		l.AdditionalProperties[otelTimestamp] = strconv.FormatInt(lr.Timestamp().AsTime().UnixNano(), 10)
		l.AdditionalProperties[ddTimestamp] = lr.Timestamp().AsTime().Format(time.RFC3339)
	}
	if l.Message == "" {
		// set the Message to the Body in case it wasn't already parsed as part of the attributes
		l.Message = lr.Body().AsString()
	}

	if !l.HasDdtags() {
		var tags = append(attributes.TagsFromAttributes(res.Attributes()), otelTag)
		tagStr := strings.Join(tags, ",")
		l.Ddtags = datadog.PtrString(tagStr)
	}

	return l
}

func extractHostNameAndServiceName(resourceAttrs pcommon.Map, logAttrs pcommon.Map) (host string, service string) {
	if src, ok := attributes.SourceFromAttributes(resourceAttrs, true); ok && src.Kind == source.HostnameKind {
		host = src.Identifier
	}
	// hostName is blank from resource
	// we need to derive from log attributes
	if host == "" {
		if src, ok := attributes.SourceFromAttributes(logAttrs, true); ok && src.Kind == source.HostnameKind {
			host = src.Identifier
		}
	}
	if s, ok := resourceAttrs.Get(conventions.AttributeServiceName); ok {
		service = s.AsString()
	}
	// serviceName is blank from resource
	// we need to derive from log attributes
	if service == "" {
		if s, ok := logAttrs.Get(conventions.AttributeServiceName); ok {
			service = s.AsString()
		}
	}
	return host, service
}

func decodeTraceID(traceID string) ([16]byte, error) {
	var ret [16]byte
	_, err := hex.Decode(ret[:], []byte(traceID))
	return ret, err
}

func decodeSpanID(spanID string) ([8]byte, error) {
	var ret [8]byte
	_, err := hex.Decode(ret[:], []byte(spanID))
	return ret, err
}

// traceIDToUint64 converts 128bit traceId to 64 bit uint64
func traceIDToUint64(b [16]byte) uint64 {
	return binary.BigEndian.Uint64(b[len(b)-8:])
}

// spanIDToUint64 converts byte array to uint64
func spanIDToUint64(b [8]byte) uint64 {
	return binary.BigEndian.Uint64(b[:])
}

// statusFromSeverityNumber converts the severity number to log level
// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/data-model.md#field-severitynumber
// this is not exactly datadog log levels , but derived from range name from above link
// see https://docs.datadoghq.com/logs/log_configuration/processors/?tab=ui#log-status-remapper for details on how it maps to datadog level
func statusFromSeverityNumber(severity plog.SeverityNumber) string {
	switch {
	case severity <= 4:
		return logLevelTrace
	case severity <= 8:
		return logLevelDebug
	case severity <= 12:
		return logLevelInfo
	case severity <= 16:
		return logLevelWarn
	case severity <= 20:
		return logLevelError
	case severity <= 24:
		return logLevelFatal
	default:
		// By default, treat this as error
		return logLevelError
	}
}
