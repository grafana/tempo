// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datadogreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/datadogreceiver"

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/collector/semconv/v1.16.0"
)

func addResourceData(req *http.Request, rs *pcommon.Resource) {
	attrs := rs.Attributes()
	attrs.Clear()
	attrs.EnsureCapacity(3)
	attrs.PutStr("telemetry.sdk.name", "Datadog")
	ddTracerVersion := req.Header.Get("Datadog-Meta-Tracer-Version")
	if ddTracerVersion != "" {
		attrs.PutStr("telemetry.sdk.version", "Datadog-"+ddTracerVersion)
	}
	ddTracerLang := req.Header.Get("Datadog-Meta-Lang")
	if ddTracerLang != "" {
		attrs.PutStr("telemetry.sdk.language", ddTracerLang)
	}
}

func toTraces(payload *pb.TracerPayload, req *http.Request) ptrace.Traces {
	var traces pb.Traces
	for _, p := range payload.GetChunks() {
		traces = append(traces, p.GetSpans())
	}
	dest := ptrace.NewTraces()
	resSpans := dest.ResourceSpans().AppendEmpty()
	resource := resSpans.Resource()
	addResourceData(req, &resource)
	resSpans.SetSchemaUrl(semconv.SchemaURL)

	for _, trace := range traces {
		ils := resSpans.ScopeSpans().AppendEmpty()
		ils.Scope().SetName("Datadog-" + req.Header.Get("Datadog-Meta-Lang"))
		ils.Scope().SetVersion(req.Header.Get("Datadog-Meta-Tracer-Version"))
		spans := ptrace.NewSpanSlice()
		spans.EnsureCapacity(len(trace))
		for _, span := range trace {
			newSpan := spans.AppendEmpty()

			newSpan.SetTraceID(uInt64ToTraceID(0, span.TraceID))
			newSpan.SetSpanID(uInt64ToSpanID(span.SpanID))
			newSpan.SetStartTimestamp(pcommon.Timestamp(span.Start))
			newSpan.SetEndTimestamp(pcommon.Timestamp(span.Start + span.Duration))
			newSpan.SetParentSpanID(uInt64ToSpanID(span.ParentID))
			newSpan.SetName(span.Resource)

			if span.Error > 0 {
				newSpan.Status().SetCode(ptrace.StatusCodeError)
			} else {
				newSpan.Status().SetCode(ptrace.StatusCodeOk)
			}

			attrs := newSpan.Attributes()
			attrs.EnsureCapacity(len(span.GetMeta()) + 1)
			attrs.PutStr(semconv.AttributeServiceName, span.Service)
			for k, v := range span.GetMeta() {
				k = translateDataDogKeyToOtel(k)
				if len(k) > 0 {
					attrs.PutStr(k, v)
				}
			}

			if span.Meta["span.kind"] != "" {
				switch span.Meta["span.kind"] {
				case "server":
					newSpan.SetKind(ptrace.SpanKindServer)
				case "client":
					newSpan.SetKind(ptrace.SpanKindClient)
				case "producer":
					newSpan.SetKind(ptrace.SpanKindProducer)
				case "consumer":
					newSpan.SetKind(ptrace.SpanKindConsumer)
				case "internal":
					newSpan.SetKind(ptrace.SpanKindInternal)
				default:
					newSpan.SetKind(ptrace.SpanKindUnspecified)
				}
			} else {
				switch span.Type {
				case "web":
					newSpan.SetKind(ptrace.SpanKindServer)
				case "http":
					newSpan.SetKind(ptrace.SpanKindClient)
				default:
					newSpan.SetKind(ptrace.SpanKindUnspecified)
				}
			}
		}
		spans.MoveAndAppendTo(ils.Spans())
	}

	return dest
}

func translateDataDogKeyToOtel(k string) string {
	switch strings.ToLower(k) {
	case "env":
		return semconv.AttributeDeploymentEnvironment
	case "version":
		return semconv.AttributeServiceVersion
	case "container_id":
		return semconv.AttributeContainerID
	case "container_name":
		return semconv.AttributeContainerName
	case "image_name":
		return semconv.AttributeContainerImageName
	case "image_tag":
		return semconv.AttributeContainerImageTag
	case "process_id":
		return semconv.AttributeProcessPID
	case "error.stacktrace":
		return semconv.AttributeExceptionStacktrace
	case "error.msg":
		return semconv.AttributeExceptionMessage
	default:
		return k
	}

}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	return buffer
}

func putBuffer(buffer *bytes.Buffer) {
	bufferPool.Put(buffer)
}

func handlePayload(req *http.Request) (tp *pb.TracerPayload, err error) {
	switch {
	case strings.HasPrefix(req.URL.Path, "/v0.7"):
		buf := getBuffer()
		defer putBuffer(buf)
		if _, err = io.Copy(buf, req.Body); err != nil {
			return nil, err
		}
		var tracerPayload pb.TracerPayload
		_, err = tracerPayload.UnmarshalMsg(buf.Bytes())
		return &tracerPayload, err
	case strings.HasPrefix(req.URL.Path, "/v0.5"):
		buf := getBuffer()
		defer putBuffer(buf)
		if _, err = io.Copy(buf, req.Body); err != nil {
			return nil, err
		}
		var traces pb.Traces
		err = traces.UnmarshalMsgDictionary(buf.Bytes())
		return &pb.TracerPayload{
			LanguageName:    req.Header.Get("Datadog-Meta-Lang"),
			LanguageVersion: req.Header.Get("Datadog-Meta-Lang-Version"),
			Chunks:          traceChunksFromTraces(traces),
			TracerVersion:   req.Header.Get("Datadog-Meta-Tracer-Version"),
		}, err
	case strings.HasPrefix(req.URL.Path, "/v0.1"):
		var spans []pb.Span
		if err = json.NewDecoder(req.Body).Decode(&spans); err != nil {
			return nil, err
		}
		return &pb.TracerPayload{
			LanguageName:    req.Header.Get("Datadog-Meta-Lang"),
			LanguageVersion: req.Header.Get("Datadog-Meta-Lang-Version"),
			Chunks:          traceChunksFromSpans(spans),
			TracerVersion:   req.Header.Get("Datadog-Meta-Tracer-Version"),
		}, nil

	default:
		var traces pb.Traces
		if err = decodeRequest(req, &traces); err != nil {
			return nil, err
		}
		return &pb.TracerPayload{
			LanguageName:    req.Header.Get("Datadog-Meta-Lang"),
			LanguageVersion: req.Header.Get("Datadog-Meta-Lang-Version"),
			Chunks:          traceChunksFromTraces(traces),
			TracerVersion:   req.Header.Get("Datadog-Meta-Tracer-Version"),
		}, err
	}
}

func decodeRequest(req *http.Request, dest *pb.Traces) (err error) {
	switch mediaType := getMediaType(req); mediaType {
	case "application/msgpack":
		buf := getBuffer()
		defer putBuffer(buf)
		_, err = io.Copy(buf, req.Body)
		if err != nil {
			return err
		}
		_, err = dest.UnmarshalMsg(buf.Bytes())
		return err
	case "application/json":
		fallthrough
	case "text/json":
		fallthrough
	case "":
		err = json.NewDecoder(req.Body).Decode(&dest)
		return err
	default:
		// do our best
		if err1 := json.NewDecoder(req.Body).Decode(&dest); err1 != nil {
			buf := getBuffer()
			defer putBuffer(buf)
			_, err2 := io.Copy(buf, req.Body)
			if err2 != nil {
				return err2
			}
			_, err2 = dest.UnmarshalMsg(buf.Bytes())
			return err2
		}
		return nil
	}
}

func traceChunksFromSpans(spans []pb.Span) []*pb.TraceChunk {
	traceChunks := []*pb.TraceChunk{}
	byID := make(map[uint64][]*pb.Span)
	for i, s := range spans {
		byID[s.TraceID] = append(byID[s.TraceID], &spans[i])
	}
	for _, t := range byID {
		traceChunks = append(traceChunks, &pb.TraceChunk{
			Priority: int32(0),
			Spans:    t,
		})
	}
	return traceChunks
}

func traceChunksFromTraces(traces pb.Traces) []*pb.TraceChunk {
	traceChunks := make([]*pb.TraceChunk, 0, len(traces))
	for _, trace := range traces {
		traceChunks = append(traceChunks, &pb.TraceChunk{
			Priority: int32(0),
			Spans:    trace,
		})
	}

	return traceChunks
}

func getMediaType(req *http.Request) string {
	mt, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		return "application/json"
	}
	return mt
}

func uInt64ToTraceID(high, low uint64) pcommon.TraceID {
	traceID := [16]byte{}
	binary.BigEndian.PutUint64(traceID[:8], high)
	binary.BigEndian.PutUint64(traceID[8:], low)
	return traceID
}

func uInt64ToSpanID(id uint64) pcommon.SpanID {
	spanID := [8]byte{}
	binary.BigEndian.PutUint64(spanID[:], id)
	return spanID
}
