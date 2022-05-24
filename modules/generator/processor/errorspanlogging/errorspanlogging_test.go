package errorspanlogging

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-kit/log"

	"github.com/grafana/tempo/pkg/tempopb"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestErrorSpanLogging(t *testing.T) {
	cfg := Config{}

	buf := &bytes.Buffer{}
	logger := log.NewLogfmtLogger(buf)
	p := New(cfg, logger)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(1, nil)
	batch.InstrumentationLibrarySpans[0].Spans[0].Status.Code = trace_v1.Status_STATUS_CODE_ERROR
	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	logResult := buf.String()
	if logResult == "" {
		t.Error("Expected log result to be non-empty")
	}

	if !strings.Contains(logResult, "msg=error_spans_received") {
		t.Errorf("Expected log result to contain 'msg=error_spans_received' but got '%s'", logResult)
	}

	if !strings.Contains(logResult, "span_service_name=test-service") {
		t.Errorf("Expected log result to contain span_service_name=test-service, got %s", logResult)
	}

	if !strings.Contains(logResult, "span_name=test") {
		t.Errorf("Expected log result to contain span_name=test, got %s", logResult)
	}

	if !strings.Contains(logResult, "span_kind=SPAN_KIND_CLIENT") {
		t.Errorf("Expected log result to contain span_kind=SPAN_KIND_CLIENT, got %s", logResult)
	}

	if !strings.Contains(logResult, "span_status=STATUS_CODE_ERROR") {
		t.Errorf("Expected log result to contain span_status=STATUS_CODE_ERROR, got %s", logResult)
	}
}

func TestErrorSpanLogging_status_code_is_not_error_should_ignore(t *testing.T) {
	cfg := Config{}

	buf := &bytes.Buffer{}
	logger := log.NewLogfmtLogger(buf)
	p := New(cfg, logger)
	defer p.Shutdown(context.Background())

	batch := test.MakeBatch(2, nil)
	batch.InstrumentationLibrarySpans[0].Spans[0].Status.Code = trace_v1.Status_STATUS_CODE_OK
	batch.InstrumentationLibrarySpans[0].Spans[1].Status.Code = trace_v1.Status_STATUS_CODE_UNSET

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{batch}})

	logResult := buf.String()
	if logResult != "" {
		t.Errorf("Expected log result to be empty, got %s", logResult)
	}
}
