// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv1 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv1"

import (
	"errors"
	"math"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/otel/semconv/v1.15.0"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/tracetranslator"
)

type status struct {
	codePtr *ptrace.StatusCode
	message string
}

const (
	tagZipkinCensusCode    = "census.status_code"
	tagZipkinCensusMsg     = "census.status_description"
	tagZipkinOpenCensusMsg = "opencensus.status_description"
)

// statusMapper contains codes translated from different sources to OC status codes
type statusMapper struct {
	// oc status code extracted from "status.code" tags
	fromStatus status
	// oc status code extracted from "census.status_code" tags
	fromCensus status
	// oc status code extracted from "http.status_code" tags
	fromHTTP status
	// oc status code extracted from "error" tags
	fromErrorTag status
	// oc status code 'unknown' when the "error" tag exists but is invalid
	fromErrorTagUnknown status
}

// status fills the given ptrace.Status from the best possible extraction source.
// It'll first try to return status extracted from "census.status_code" to account for zipkin
// then fallback on code extracted from "status.code" tags
// and finally fallback on code extracted and translated from "http.status_code"
// status must be called after all tags/attributes are processed with the `fromAttribute` method.
func (m *statusMapper) status(dest ptrace.Status) {
	var s status
	switch {
	case m.fromCensus.codePtr != nil:
		s = m.fromCensus
	case m.fromStatus.codePtr != nil:
		s = m.fromStatus
	case m.fromErrorTag.codePtr != nil:
		s = m.fromErrorTag
		if m.fromCensus.message != "" {
			s.message = m.fromCensus.message
		} else if m.fromStatus.message != "" {
			s.message = m.fromStatus.message
		}
	case m.fromHTTP.codePtr != nil:
		s = m.fromHTTP
	default:
		s = m.fromErrorTagUnknown
	}

	if s.codePtr != nil {
		code := ptrace.StatusCodeUnset
		if s.codePtr != nil {
			code = *s.codePtr
		}
		dest.SetCode(code)
		dest.SetMessage(s.message)
	}
}

func (m *statusMapper) fromAttribute(key string, attrib pcommon.Value) bool {
	switch key {
	case tagZipkinCensusCode:
		ocCode, err := attribToStatusCode(attrib)
		if err == nil {
			// Convert oc status (OK == 0) to otel (OK == 1), anything else is error.
			code := ptrace.StatusCodeOk
			if ocCode != 0 {
				code = ptrace.StatusCodeError
			}
			m.fromCensus.codePtr = &code
		}
		return true

	case tagZipkinCensusMsg, tagZipkinOpenCensusMsg:
		m.fromCensus.message = attrib.Str()
		return true

	case string(conventions.OtelStatusCodeKey):
		// Keep the code as is, even if unknown. Since we are allowed to receive unknown values for enums.
		code, err := attribToStatusCode(attrib)
		if err == nil {
			m.fromStatus.codePtr = (*ptrace.StatusCode)(&code)
		}
		return true

	case string(conventions.OtelStatusDescriptionKey):
		m.fromStatus.message = attrib.Str()
		return true

	case string(conventions.HTTPStatusCodeKey):
		httpCode, err := attribToStatusCode(attrib)
		if err == nil {
			code := statusCodeFromHTTP(httpCode)
			m.fromHTTP.codePtr = &code
		}

	case tracetranslator.TagHTTPStatusMsg:
		m.fromHTTP.message = attrib.Str()

	case tracetranslator.TagError:
		// The status is stored with the "error" key, but otel does not care about the old census values.
		// See https://github.com/census-instrumentation/opencensus-go/blob/1eb9a13c7dd02141e065a665f6bf5c99a090a16a/exporter/zipkin/zipkin.go#L160-L165
		code := ptrace.StatusCodeError
		m.fromErrorTag.codePtr = &code
		return true
	}
	return false
}

// attribToStatusCode maps an integer or string attribute value to a status code.
// The function return nil if the value is of another type or cannot be converted to an int32 value.
func attribToStatusCode(attr pcommon.Value) (int32, error) {
	switch attr.Type() {
	case pcommon.ValueTypeEmpty:
		return 0, errors.New("nil attribute")
	case pcommon.ValueTypeInt:
		return toInt32(attr.Int())
	case pcommon.ValueTypeStr:
		i, err := strconv.ParseInt(attr.Str(), 10, 64)
		if err != nil {
			return 0, err
		}
		return toInt32(i)
	}
	return 0, errors.New("invalid attribute type")
}

func toInt32(i int64) (int32, error) {
	if i <= math.MaxInt32 && i >= math.MinInt32 {
		return int32(i), nil
	}
	return 0, errors.New("outside of the int32 range")
}

// statusCodeFromHTTP takes an HTTP status code and return the appropriate OpenTelemetry status code
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/http.md
func statusCodeFromHTTP(code int32) ptrace.StatusCode {
	if code >= 100 && code < 400 {
		return ptrace.StatusCodeUnset
	}
	return ptrace.StatusCodeError
}
