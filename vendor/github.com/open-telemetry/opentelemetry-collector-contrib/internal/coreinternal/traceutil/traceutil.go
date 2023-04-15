// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package traceutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"

import (
	"encoding/hex"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// SpanKindStr returns a string representation of the SpanKind as it's defined in the proto.
// The function provides old behavior of ptrace.SpanKind.String() to support graceful adoption of
// https://github.com/open-telemetry/opentelemetry-collector/pull/6250.
func SpanKindStr(sk ptrace.SpanKind) string {
	switch sk {
	case ptrace.SpanKindUnspecified:
		return "SPAN_KIND_UNSPECIFIED"
	case ptrace.SpanKindInternal:
		return "SPAN_KIND_INTERNAL"
	case ptrace.SpanKindServer:
		return "SPAN_KIND_SERVER"
	case ptrace.SpanKindClient:
		return "SPAN_KIND_CLIENT"
	case ptrace.SpanKindProducer:
		return "SPAN_KIND_PRODUCER"
	case ptrace.SpanKindConsumer:
		return "SPAN_KIND_CONSUMER"
	}
	return ""
}

// StatusCodeStr returns a string representation of the StatusCode as it's defined in the proto.
// The function provides old behavior of ptrace.StatusCode.String() to support graceful adoption of
// https://github.com/open-telemetry/opentelemetry-collector/pull/6250.
func StatusCodeStr(sk ptrace.StatusCode) string {
	switch sk {
	case ptrace.StatusCodeUnset:
		return "STATUS_CODE_UNSET"
	case ptrace.StatusCodeOk:
		return "STATUS_CODE_OK"
	case ptrace.StatusCodeError:
		return "STATUS_CODE_ERROR"
	}
	return ""
}

// SpanIDToHexOrEmptyString returns a hex string from SpanID.
// An empty string is returned, if SpanID is empty.
func SpanIDToHexOrEmptyString(id pcommon.SpanID) string {
	if id.IsEmpty() {
		return ""
	}
	return hex.EncodeToString(id[:])
}

// TraceIDToHexOrEmptyString returns a hex string from TraceID.
// An empty string is returned, if TraceID is empty.
func TraceIDToHexOrEmptyString(id pcommon.TraceID) string {
	if id.IsEmpty() {
		return ""
	}
	return hex.EncodeToString(id[:])
}
