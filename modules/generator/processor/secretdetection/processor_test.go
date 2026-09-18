package secretdetection

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	gklog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
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

type capturingObserver struct {
	value float64
}

func (o *capturingObserver) Observe(value float64) {
	o.value = value
}

func newTestProcessor(t *testing.T) *Processor {
	t.Helper()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	p, err := New(cfg, "test-tenant", gklog.NewNopLogger(), nil)
	require.NoError(t, err)
	p.processFindingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	p.processCoverageLogLimiter = rate.NewLimiter(rate.Inf, 0)
	return p
}

func newTestProcessorWithLogger(t *testing.T, logger *capturingLogger) *Processor {
	t.Helper()
	p, err := New(Config{}, "test-tenant", level.Warn(logger), nil)
	require.NoError(t, err)
	p.logFinding = logger.Log
	p.findingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	p.processFindingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	p.processCoverageLogLimiter = rate.NewLimiter(rate.Inf, 0)
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

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeSpan))
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

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeSpan))
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

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeSpan))
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

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeResource))
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

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeEvent))
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

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeLink))
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

	count := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeSpan))
	assert.Equal(t, 0.0, count, "expected no detections for empty/nil values")
}

func TestLogOutputContainsExpectedFields(t *testing.T) {
	metricSecretDetectionsTotal.Reset()
	logger := &capturingLogger{}
	p := newTestProcessorWithLogger(t, logger)
	p.now = func() time.Time { return time.Unix(100, 0) }

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

	assert.Equal(t, []map[string]string{{
		"msg":        "secret detected in trace field",
		"tenant":     "test-tenant",
		"traceID":    "aabbccdd11223344556677889900eeff",
		"spanID":     "0102030405060708",
		"field_kind": string(secrets.FieldKindSpanAttribute),
		"rule":       "slack-bot-token",
		"ts":         "1970-01-01T00:01:40Z",
	}}, logger.Entries())
}

func TestTraceFindingStatesPreserveInlineStateWhenGrowing(t *testing.T) {
	var states traceFindingStates
	field := secrets.TraceField{}
	for index := range 10 {
		traceID := make([]byte, 16)
		traceID[15] = byte(index)
		states.stateFor(traceID, field).findingLogs = index + 1
	}

	for index := range 10 {
		traceID := make([]byte, 16)
		traceID[15] = byte(index)
		assert.Equal(t, index+1, states.stateFor(traceID, field).findingLogs)
	}
}

func TestDetectionScansBeyondFormerPerTraceCeilings(t *testing.T) {
	assertFindingAfter := func(t *testing.T, attributes []*common_v1.KeyValue) {
		t.Helper()
		logger := &capturingLogger{}
		p := newTestProcessorWithLogger(t, logger)
		request := &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{{
			Resource: &resource_v1.Resource{},
			ScopeSpans: []*trace_v1.ScopeSpans{{Spans: []*trace_v1.Span{{
				TraceId:    []byte{1},
				SpanId:     []byte{1},
				Attributes: attributes,
			}}}},
		}}}

		p.PushSpans(context.Background(), request)

		found := false
		for _, entry := range logger.Entries() {
			found = found || entry["msg"] == "secret detected in trace field" &&
				entry["traceID"] == "01" &&
				entry["rule"] == "slack-bot-token"
		}
		assert.True(t, found)
	}

	t.Run("fields", func(t *testing.T) {
		const formerMaxFieldsPerTrace = 10_000
		attributes := make([]*common_v1.KeyValue, formerMaxFieldsPerTrace+1)
		for index := range formerMaxFieldsPerTrace {
			attributes[index] = test.MakeAttribute("field", "safe")
		}
		attributes[formerMaxFieldsPerTrace] = test.MakeAttribute("slack.token", fakeSlackToken())
		assertFindingAfter(t, attributes)
	})

	t.Run("bytes", func(t *testing.T) {
		const formerMaxBytesPerTrace = 1 << 20
		attributes := []*common_v1.KeyValue{
			test.MakeAttribute("large.field", strings.Repeat("x", formerMaxBytesPerTrace)),
			test.MakeAttribute("slack.token", fakeSlackToken()),
		}
		assertFindingAfter(t, attributes)
	})
}

func TestFindingLogLimitIsIsolatedPerTrace(t *testing.T) {
	logger := &capturingLogger{}
	p := newTestProcessorWithLogger(t, logger)

	firstAttributes := make([]*common_v1.KeyValue, maxFindingLogsPerTrace+1)
	for index := range firstAttributes {
		firstAttributes[index] = test.MakeAttribute(fmt.Sprintf("field.%d", index), fakeSlackToken())
	}
	request := &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{{
		Resource: &resource_v1.Resource{},
		ScopeSpans: []*trace_v1.ScopeSpans{{Spans: []*trace_v1.Span{
			{TraceId: []byte{1}, SpanId: []byte{1}, Attributes: firstAttributes},
			{TraceId: []byte{2}, SpanId: []byte{2}, Attributes: []*common_v1.KeyValue{
				test.MakeAttribute("slack.token", fakeSlackToken()),
			}},
		}}},
	}}}

	p.PushSpans(context.Background(), request)

	firstFindings := 0
	var firstLimitGap, secondFinding bool
	for _, entry := range logger.Entries() {
		if entry["msg"] == "secret detected in trace field" && entry["traceID"] == "01" {
			firstFindings++
		}
		firstLimitGap = firstLimitGap || entry["reason"] == "finding_log_limit_exceeded" && entry["traceID"] == "01"
		secondFinding = secondFinding || entry["msg"] == "secret detected in trace field" && entry["traceID"] == "02"
	}
	assert.Equal(t, maxFindingLogsPerTrace, firstFindings)
	assert.True(t, firstLimitGap)
	assert.True(t, secondFinding)
}

func newConfiguredProcessor(t *testing.T, policy secrets.Policy, logger *capturingLogger) *Processor {
	t.Helper()
	compiler, err := secrets.NewPolicyCompiler(nil)
	require.NoError(t, err)
	compiled, err := compiler.CompilePolicy(policy)
	require.NoError(t, err)
	p, err := New(Config{CompiledPolicy: compiled}, "test-tenant", level.Warn(logger), nil)
	require.NoError(t, err)
	p.logFinding = logger.Log
	p.findingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	p.processFindingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	p.processCoverageLogLimiter = rate.NewLimiter(rate.Inf, 0)
	return p
}

func spanAttrReq(attrs ...[2]string) *tempopb.PushSpansRequest {
	kvs := make([]*common_v1.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		kvs = append(kvs, test.MakeAttribute(a[0], a[1]))
	}
	return &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{{
		Resource: &resource_v1.Resource{},
		ScopeSpans: []*trace_v1.ScopeSpans{{Spans: []*trace_v1.Span{{
			TraceId:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanId:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
			Attributes: kvs,
		}}}},
	}}}
}

func ruleSet(entries []map[string]string) map[string]bool {
	m := map[string]bool{}
	for _, e := range entries {
		if r, ok := e["rule"]; ok {
			m[r] = true
		}
	}
	return m
}

func TestProcessorFallbackRespectsNativeSelection(t *testing.T) {
	for _, tc := range []struct {
		name string
		ids  []string
		want map[string]bool
	}{
		{"selected", []string{"stripe-access-token"}, map[string]bool{"stripe-access-token": true}},
		{"empty", []string{}, map[string]bool{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compiler, err := secrets.NewPolicyCompiler(&tc.ids)
			require.NoError(t, err)
			logger := &capturingLogger{}
			p, err := New(Config{PolicyCompiler: compiler}, "test-tenant", logger, nil)
			require.NoError(t, err)
			p.logFinding = logger.Log
			p.findingLogLimiter = rate.NewLimiter(rate.Inf, 0)
			p.processFindingLogLimiter = rate.NewLimiter(rate.Inf, 0)
			p.processCoverageLogLimiter = rate.NewLimiter(rate.Inf, 0)
			p.PushSpans(context.Background(), spanAttrReq(
				[2]string{"first", fakeSlackToken()},
				[2]string{"second", "sk_test_" + "0123456789abcdefghijklmn"},
			))
			require.Equal(t, tc.want, ruleSet(logger.Entries()))
		})
	}
}

func TestCustomRuleExtendsProductionCatalog(t *testing.T) {
	logger := &capturingLogger{}
	p := newConfiguredProcessor(t, secrets.Policy{
		CustomRules: []secrets.CustomRule{{ID: "acme-key", Regex: `ACME-[A-Z0-9]{10}`}},
	}, logger)
	p.PushSpans(context.Background(), spanAttrReq([2]string{"auth", "ACME-AB12CD34EF"}, [2]string{"slack", fakeSlackToken()}))
	rules := ruleSet(logger.Entries())
	assert.True(t, rules["acme-key"])
	assert.True(t, rules["slack-bot-token"])
}

func TestAttributeNamesAreNotDetectionInput(t *testing.T) {
	logger := &capturingLogger{}
	p := newConfiguredProcessor(t, secrets.Policy{}, logger)
	p.PushSpans(context.Background(), spanAttrReq([2]string{fakeSlackToken(), "safe"}))
	assert.Empty(t, logger.Entries())
}

func TestFindingLogOmitsSecretAttributeNamesAndValues(t *testing.T) {
	logger := &capturingLogger{}
	p := newConfiguredProcessor(t, secrets.Policy{}, logger)
	secret := fakeSlackToken()
	p.PushSpans(context.Background(), spanAttrReq([2]string{secret, secret}))

	entries := logger.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, string(secrets.FieldKindSpanAttribute), entries[0]["field_kind"])
	assert.NotContains(t, entries[0], "attr_key")
	for _, value := range entries[0] {
		assert.NotContains(t, value, secret)
	}
}

func TestFindingLogLimitReportsDegradedCoverage(t *testing.T) {
	const tenant = "finding-log-limit-tenant"
	logger := &capturingLogger{}
	p, err := New(Config{}, tenant, logger, nil)
	require.NoError(t, err)
	p.logFinding = logger.Log
	p.processFindingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	p.findingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	p.processCoverageLogLimiter = rate.NewLimiter(rate.Inf, 0)
	p.now = func() time.Time { return time.Unix(100, 0) }

	attrs := make([][2]string, maxFindingLogsPerTrace+1)
	for i := range attrs {
		attrs[i] = [2]string{fmt.Sprintf("field.%d", i), fakeSlackToken()}
	}
	p.PushSpans(context.Background(), spanAttrReq(attrs...))

	entries := logger.Entries()
	require.Len(t, entries, maxFindingLogsPerTrace+1)
	assert.Equal(t, map[string]string{
		"msg":        "secret detection coverage gap",
		"tenant":     tenant,
		"traceID":    "0102030405060708090a0b0c0d0e0f10",
		"field_kind": string(secrets.FieldKindSpanAttribute),
		"reason":     "finding_log_limit_exceeded",
		"ts":         "1970-01-01T00:01:40Z",
	}, entries[len(entries)-1])
}

func TestFindingLogRateLimitReportsDegradedCoverage(t *testing.T) {
	logger := &capturingLogger{}
	p := newTestProcessorWithLogger(t, logger)
	p.findingLogLimiter = rate.NewLimiter(0, 0)
	p.now = func() time.Time { return time.Unix(100, 0) }

	p.PushSpans(context.Background(), spanAttrReq([2]string{"authorization", fakeSlackToken()}))

	assert.Equal(t, []map[string]string{{
		"msg":        "secret detection coverage gap",
		"tenant":     "test-tenant",
		"traceID":    "0102030405060708090a0b0c0d0e0f10",
		"field_kind": string(secrets.FieldKindSpanAttribute),
		"reason":     "finding_log_rate_limit_exceeded",
		"ts":         "1970-01-01T00:01:40Z",
	}}, logger.Entries())
}

func TestFindingLogProcessLimitIsSharedAcrossProcessors(t *testing.T) {
	firstLogger := &capturingLogger{}
	first, err := New(Config{}, "first-tenant", firstLogger, nil)
	require.NoError(t, err)
	first.logFinding = firstLogger.Log
	first.findingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	secondLogger := &capturingLogger{}
	second, err := New(Config{}, "second-tenant", secondLogger, nil)
	require.NoError(t, err)
	second.logFinding = secondLogger.Log
	second.findingLogLimiter = rate.NewLimiter(rate.Inf, 0)
	first.processCoverageLogLimiter = rate.NewLimiter(rate.Inf, 0)
	second.processCoverageLogLimiter = rate.NewLimiter(rate.Inf, 0)

	limiter := rate.NewLimiter(0, 1)
	first.processFindingLogLimiter = limiter
	second.processFindingLogLimiter = limiter
	first.PushSpans(context.Background(), spanAttrReq([2]string{"authorization", fakeSlackToken()}))
	second.PushSpans(context.Background(), spanAttrReq([2]string{"authorization", fakeSlackToken()}))

	assert.Len(t, firstLogger.Entries(), 1)
	require.Len(t, secondLogger.Entries(), 1)
	assert.Equal(t, "finding_log_rate_limit_exceeded", secondLogger.Entries()[0]["reason"])
}

func TestCoverageLogProcessLimitIsSharedAcrossProcessors(t *testing.T) {
	firstLogger := &capturingLogger{}
	first, err := New(Config{}, "first-tenant", firstLogger, nil)
	require.NoError(t, err)
	secondLogger := &capturingLogger{}
	second, err := New(Config{}, "second-tenant", secondLogger, nil)
	require.NoError(t, err)

	limiter := rate.NewLimiter(0, 1)
	first.processCoverageLogLimiter = limiter
	second.processCoverageLogLimiter = limiter
	first.findingLogLimiter = rate.NewLimiter(0, 0)
	second.findingLogLimiter = rate.NewLimiter(0, 0)
	first.PushSpans(context.Background(), spanAttrReq([2]string{"authorization", fakeSlackToken()}))
	second.PushSpans(context.Background(), spanAttrReq([2]string{"authorization", fakeSlackToken()}))

	require.Len(t, firstLogger.Entries(), 1)
	assert.Equal(t, "finding_log_rate_limit_exceeded", firstLogger.Entries()[0]["reason"])
	assert.Empty(t, secondLogger.Entries())
}

func TestTenantMetricsCountRuleFieldMatchesAndBatches(t *testing.T) {
	testRegistry := registry.NewTestRegistry()
	tenantMetrics := NewTenantMetrics(testRegistry, "sampler-ingest")
	compiler, err := secrets.NewPolicyCompiler(&[]string{})
	require.NoError(t, err)
	compiled, err := compiler.CompilePolicy(secrets.Policy{CustomRules: []secrets.CustomRule{
		{ID: "first-rule", Regex: `CUSTOMER-[0-9]+`},
		{ID: "second-rule", Regex: `CUSTOMER-[0-9]+`},
	}})
	require.NoError(t, err)
	p, err := New(Config{CompiledPolicy: compiled, SourceStream: "sampler-ingest"}, "test-tenant", gklog.NewNopLogger(), tenantMetrics)
	require.NoError(t, err)
	p.findingLogLimiter = rate.NewLimiter(0, 0)
	p.processCoverageLogLimiter = rate.NewLimiter(rate.Inf, 0)
	detectionsBefore := testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeResource))
	pushesBefore := testutil.ToFloat64(metricSecretDetectionPushesTotal.WithLabelValues("sampler-ingest"))

	// Repeated occurrences and fan-out to two traces do not multiply rule-field matches.
	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{{
		Resource: &resource_v1.Resource{Attributes: []*common_v1.KeyValue{
			test.MakeAttribute("authorization", "CUSTOMER-1 CUSTOMER-2"),
		}},
		ScopeSpans: []*trace_v1.ScopeSpans{{Spans: []*trace_v1.Span{
			{TraceId: []byte{1}},
			{TraceId: []byte{2}},
		}}},
	}}})
	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{})

	sourceLabels := labels.FromStrings("source_stream", "sampler-ingest")
	detectionLabels := labels.FromStrings("attribute_scope", scopeResource, "source_stream", "sampler-ingest")
	assert.Equal(t, 2.0, testRegistry.Query(tenantMetricPushes, sourceLabels))
	assert.Equal(t, 2.0, testRegistry.Query(tenantMetricDetections, detectionLabels))
	assert.Equal(t, pushesBefore+2, testutil.ToFloat64(metricSecretDetectionPushesTotal.WithLabelValues("sampler-ingest")))
	assert.Equal(t, detectionsBefore+2, testutil.ToFloat64(metricSecretDetectionsTotal.WithLabelValues(scopeResource)))
}

func TestFindingLogErrorsProduceBoundedSafeDiagnostics(t *testing.T) {
	logger := &capturingLogger{}
	p := newTestProcessorWithLogger(t, logger)
	p.coverageLogLimiter = rate.NewLimiter(0, 1)
	p.now = func() time.Time { return time.Unix(100, 0) }
	failures := 0
	p.logFinding = func(keyvals ...interface{}) error {
		if keyvals[1] == "secret detected in trace field" {
			failures++
			return fmt.Errorf("cannot write %s", fakeSlackToken())
		}
		return logger.Log(keyvals...)
	}

	p.PushSpans(context.Background(), spanAttrReq(
		[2]string{"authorization", fakeSlackToken()},
		[2]string{"authorization", fakeSlackToken()},
		[2]string{"authorization", fakeSlackToken()},
	))

	assert.Equal(t, 3, failures)
	assert.Equal(t, []map[string]string{{
		"msg":        "secret detection coverage gap",
		"tenant":     "test-tenant",
		"traceID":    "0102030405060708090a0b0c0d0e0f10",
		"field_kind": string(secrets.FieldKindSpanAttribute),
		"reason":     "finding_log_error",
		"ts":         "1970-01-01T00:01:40Z",
	}}, logger.Entries())
}

func TestPushSpansObservesCompletedScanDuration(t *testing.T) {
	p := newTestProcessor(t)
	started := time.Unix(100, 0)
	times := []time.Time{started, started.Add(2 * time.Second)}
	p.now = func() time.Time {
		now := times[0]
		times = times[1:]
		return now
	}
	duration := &capturingObserver{}
	p.duration = duration

	p.PushSpans(context.Background(), &tempopb.PushSpansRequest{})

	assert.Equal(t, 2.0, duration.value)
}
