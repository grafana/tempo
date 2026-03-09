package secretdetection

import (
	"context"
	"fmt"
	"sync"
	"testing"

	gklog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/pkg/sharedconfig"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
)

// capturingLogger records all log calls for assertion.
type capturingLogger struct {
	mu      sync.Mutex
	entries []map[string]string
}

func (l *capturingLogger) Log(keyvals ...interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := make(map[string]string)
	for i := 0; i+1 < len(keyvals); i += 2 {
		key, _ := keyvals[i].(string)
		val := fmt.Sprintf("%v", keyvals[i+1])
		entry[key] = val
	}
	l.entries = append(l.entries, entry)
	return nil
}

func (l *capturingLogger) Entries() []map[string]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	cp := make([]map[string]string, len(l.entries))
	copy(cp, l.entries)
	return cp
}

func newTestProcessor(t *testing.T) *Processor {
	t.Helper()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	p, err := New(cfg, "test-tenant", gklog.NewNopLogger())
	require.NoError(t, err)
	return p
}

func newTestProcessorWithLogger(t *testing.T, logger *capturingLogger) *Processor {
	t.Helper()
	p, err := New(Config{}, "test-tenant", level.Warn(logger))
	require.NoError(t, err)
	// Override logger with high rate limit for tests.
	p.logger = tempo_log.NewRateLimitedLogger(1000, level.Warn(logger))
	return p
}

func newTestProcessorWithSpanMetrics(t *testing.T, info SpanMetricsInfo) *Processor {
	t.Helper()
	cfg := Config{SpanMetricsInfo: info}
	p, err := New(cfg, "test-tenant", gklog.NewNopLogger())
	require.NoError(t, err)
	return p
}

func newTestProcessorWithSpanMetricsAndLogger(t *testing.T, info SpanMetricsInfo, logger *capturingLogger) *Processor {
	t.Helper()
	cfg := Config{SpanMetricsInfo: info}
	p, err := New(cfg, "test-tenant", level.Warn(logger))
	require.NoError(t, err)
	// Override logger with high rate limit for tests.
	p.logger = tempo_log.NewRateLimitedLogger(1000, level.Warn(logger))
	return p
}

// Test secrets are built via concatenation so no complete token literal appears
// in source — this avoids triggering GitHub push protection.
func fakeSlackToken() string { return "xoxb-" + "1234567890-1234567890123-abcdefghijklmnopqrstuvwx" }
func fakeStripeKey() string  { return "sk_live_" + "1234567890abcdefghijklmnop" }

func TestName(t *testing.T) {
	p := newTestProcessor(t)
	assert.Equal(t, processor.SecretDetectionName, p.Name())
}

func TestDetectsSlackBotToken(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	p := newTestProcessor(t)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("slack.token", fakeSlackToken()),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Greater(t, count, 0.0, "expected secret detection for Slack bot token")
}

func TestDetectsStripeKey(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	p := newTestProcessor(t)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("payment.key", fakeStripeKey()),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Greater(t, count, 0.0, "expected secret detection for Stripe key")
}

func TestNoSecretNoDetection(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	p := newTestProcessor(t)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("http.method", "GET"),
									test.MakeAttribute("http.url", "https://example.com/api/users"),
									test.MakeAttribute("service.name", "my-service"),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Equal(t, 0.0, count, "expected no secret detections for clean attributes")
}

func TestDetectsSecretInResourceAttributes(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	p := newTestProcessor(t)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{
					Attributes: []*common_v1.KeyValue{
						test.MakeAttribute("deployment.token", fakeSlackToken()),
					},
				},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeResource))
	assert.Greater(t, count, 0.0, "expected secret detection in resource attributes")
}

func TestDetectsSecretInEventAttributes(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	p := newTestProcessor(t)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Events: []*trace_v1.Span_Event{
									{
										Attributes: []*common_v1.KeyValue{
											test.MakeAttribute("error.detail", fakeStripeKey()),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeEvent))
	assert.Greater(t, count, 0.0, "expected secret detection in event attributes")
}

func TestDetectsSecretInLinkAttributes(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	p := newTestProcessor(t)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Links: []*trace_v1.Span_Link{
									{
										Attributes: []*common_v1.KeyValue{
											test.MakeAttribute("link.token", fakeSlackToken()),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeLink))
	assert.Greater(t, count, 0.0, "expected secret detection in link attributes")
}

func TestSkipsEmptyAndNilValues(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	p := newTestProcessor(t)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("empty", ""),
									{Key: "nil_value", Value: nil},
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Equal(t, 0.0, count, "expected no detections for empty/nil values")
}

func TestLogOutputContainsExpectedFields(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	logger := &capturingLogger{}
	p := newTestProcessorWithLogger(t, logger)

	traceID := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00, 0xee, 0xff}
	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: traceID,
								SpanId:  spanID,
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("slack.token", fakeSlackToken()),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	entries := logger.Entries()
	require.NotEmpty(t, entries, "expected at least one log entry")

	entry := entries[0]
	assert.Equal(t, "secret detected in span attribute", entry["msg"])
	assert.Equal(t, "test-tenant", entry["tenant"])
	assert.Equal(t, "aabbccdd11223344556677889900eeff", entry["traceID"])
	assert.Equal(t, "0102030405060708", entry["spanID"])
	assert.Equal(t, "slack.token", entry["attr_key"])
	assert.Equal(t, "slack-bot-token", entry["rule"])
	assert.Equal(t, "span", entry["scope"])
	assert.Equal(t, "false", entry["in_metrics"])
	assert.Equal(t, "", entry["metric_series"])
	assert.NotEmpty(t, entry["ts"], "expected timestamp in log")
}

func TestShutdown(t *testing.T) {
	p := newTestProcessor(t)
	p.Shutdown(context.Background())
}

// --- Metrics-exposure detection tests ---

func TestSecretInSpanMetricsDimension(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	info := NewSpanMetricsInfo(
		[]string{"slack.token", "other.attr"},
		nil, false, nil,
	)
	p := newTestProcessorWithSpanMetrics(t, info)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("slack.token", fakeSlackToken()),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Greater(t, count, 0.0, "expected detection")

	inMetricsCount := testutil.ToFloat64(metricSecretDetectionsInMetricsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Greater(t, inMetricsCount, 0.0, "expected in-metrics detection for dimension attribute")
}

func TestSecretInResourceWithTargetInfoEnabled(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	info := NewSpanMetricsInfo(nil, nil, true, nil)
	p := newTestProcessorWithSpanMetrics(t, info)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{
					Attributes: []*common_v1.KeyValue{
						test.MakeAttribute("deployment.token", fakeSlackToken()),
					},
				},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	inMetricsCount := testutil.ToFloat64(metricSecretDetectionsInMetricsTotal.WithLabelValues("test-tenant", scopeResource))
	assert.Greater(t, inMetricsCount, 0.0, "expected in-metrics detection for resource attr with target_info enabled")
}

func TestSecretInResourceExcludedFromTargetInfo(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	info := NewSpanMetricsInfo(nil, nil, true, []string{"deployment.token"})
	p := newTestProcessorWithSpanMetrics(t, info)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{
					Attributes: []*common_v1.KeyValue{
						test.MakeAttribute("deployment.token", fakeSlackToken()),
					},
				},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	// Detection still fires, but NOT in-metrics since the key is excluded from target_info
	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeResource))
	assert.Greater(t, count, 0.0, "expected detection")

	inMetricsCount := testutil.ToFloat64(metricSecretDetectionsInMetricsTotal.WithLabelValues("test-tenant", scopeResource))
	assert.Equal(t, 0.0, inMetricsCount, "expected no in-metrics detection for excluded attribute")
}

func TestSecretInDimensionMappingSourceLabel(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	info := NewSpanMetricsInfo(
		nil,
		[]sharedconfig.DimensionMappings{
			{Name: "mapped_dim", SourceLabel: []string{"slack.token", "other.attr"}},
		},
		false, nil,
	)
	p := newTestProcessorWithSpanMetrics(t, info)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("slack.token", fakeSlackToken()),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	inMetricsCount := testutil.ToFloat64(metricSecretDetectionsInMetricsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Greater(t, inMetricsCount, 0.0, "expected in-metrics detection for dimension mapping source label")
}

func TestSecretNotInMetricsWhenNotDimension(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	// Configure dimensions that don't include the secret-bearing attribute
	info := NewSpanMetricsInfo(
		[]string{"http.method", "http.status_code"},
		nil, false, nil,
	)
	p := newTestProcessorWithSpanMetrics(t, info)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("slack.token", fakeSlackToken()),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Greater(t, count, 0.0, "expected detection")

	inMetricsCount := testutil.ToFloat64(metricSecretDetectionsInMetricsTotal.WithLabelValues("test-tenant", scopeSpan))
	assert.Equal(t, 0.0, inMetricsCount, "expected no in-metrics detection when attribute is not a dimension")
}

func TestSecretInMetricsLogContainsSeriesAndTimestamp(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	logger := &capturingLogger{}
	info := NewSpanMetricsInfo(
		[]string{"slack.token"},
		nil, false, nil,
	)
	p := newTestProcessorWithSpanMetricsAndLogger(t, info, logger)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("slack.token", fakeSlackToken()),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	entries := logger.Entries()
	require.NotEmpty(t, entries)

	entry := entries[0]
	assert.Equal(t, "true", entry["in_metrics"])
	assert.Contains(t, entry["metric_series"], seriesCallsTotal)
	assert.Contains(t, entry["metric_series"], seriesLatency)
	assert.Contains(t, entry["metric_series"], seriesSizeTotal)
	assert.NotEmpty(t, entry["ts"])
}

func TestSecretInResourceWithTargetInfoLogContainsSeries(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	logger := &capturingLogger{}
	info := NewSpanMetricsInfo(nil, nil, true, nil)
	p := newTestProcessorWithSpanMetricsAndLogger(t, info, logger)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{
					Attributes: []*common_v1.KeyValue{
						test.MakeAttribute("deployment.token", fakeSlackToken()),
					},
				},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	entries := logger.Entries()
	require.NotEmpty(t, entries)

	entry := entries[0]
	assert.Equal(t, "true", entry["in_metrics"])
	assert.Equal(t, seriesTargetInfo, entry["metric_series"])
}

func TestEventAndLinkSecretsNeverInMetrics(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	// Enable everything — dimensions, target_info — events/links should still not trigger in_metrics
	info := NewSpanMetricsInfo(
		[]string{"link.token", "error.detail"},
		nil, true, nil,
	)
	p := newTestProcessorWithSpanMetrics(t, info)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Events: []*trace_v1.Span_Event{
									{
										Attributes: []*common_v1.KeyValue{
											test.MakeAttribute("error.detail", fakeStripeKey()),
										},
									},
								},
								Links: []*trace_v1.Span_Link{
									{
										Attributes: []*common_v1.KeyValue{
											test.MakeAttribute("link.token", fakeSlackToken()),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	eventInMetrics := testutil.ToFloat64(metricSecretDetectionsInMetricsTotal.WithLabelValues("test-tenant", scopeEvent))
	linkInMetrics := testutil.ToFloat64(metricSecretDetectionsInMetricsTotal.WithLabelValues("test-tenant", scopeLink))
	assert.Equal(t, 0.0, eventInMetrics, "event attributes should never be in metrics")
	assert.Equal(t, 0.0, linkInMetrics, "link attributes should never be in metrics")
}

func TestCheckMetricsExposureDeduplicatesWhenBothDimensionAndMapping(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	metricSecretDetectionsInMetricsTotal.Reset()

	logger := &capturingLogger{}
	// slack.token is both a dimension AND a dimension mapping source label
	info := NewSpanMetricsInfo(
		[]string{"slack.token"},
		[]sharedconfig.DimensionMappings{
			{Name: "mapped", SourceLabel: []string{"slack.token"}},
		},
		false, nil,
	)
	p := newTestProcessorWithSpanMetricsAndLogger(t, info, logger)

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			{
				Resource: &resource_v1.Resource{},
				ScopeSpans: []*trace_v1.ScopeSpans{
					{
						Spans: []*trace_v1.Span{
							{
								TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
								SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
								Attributes: []*common_v1.KeyValue{
									test.MakeAttribute("slack.token", fakeSlackToken()),
								},
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), req)

	entries := logger.Entries()
	require.NotEmpty(t, entries)

	// Should have exactly 3 unique series, not 6 duplicated
	expected := seriesCallsTotal + "," + seriesLatency + "," + seriesSizeTotal
	assert.Equal(t, expected, entries[0]["metric_series"])
}
