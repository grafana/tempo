package secretdetection

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kit/log"

	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

var (
	benchmarkFindings int
	benchmarkTraceIDs int
)

func BenchmarkCompiledPolicyDetect(b *testing.B) {
	compiler, err := secrets.NewPolicyCompiler(nil)
	if err != nil {
		b.Fatal(err)
	}
	policy, err := compiler.CompilePolicy(secrets.Policy{})
	if err != nil {
		b.Fatal(err)
	}

	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "clean", value: "https://api.example.com/v1/orders/123456?region=us-east-1"},
		{name: "secret", value: fakeSlackToken()},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.value)))
			for range b.N {
				benchmarkFindings = len(policy.Detect(tc.value).Matches)
			}
		})
	}
}

func BenchmarkCompiledPolicyDetectMaxCustomRules(b *testing.B) {
	rules := make([]secrets.CustomRule, 16)
	for i := range rules {
		rules[i] = secrets.CustomRule{
			ID:    fmt.Sprintf("benchmark-%d", i),
			Regex: fmt.Sprintf(`benchmark%02d-[A-Za-z0-9]{20}`, i),
		}
	}
	compiler, err := secrets.NewPolicyCompiler(nil)
	if err != nil {
		b.Fatal(err)
	}
	policy, err := compiler.CompilePolicy(secrets.Policy{CustomRules: rules})
	if err != nil {
		b.Fatal(err)
	}

	for _, tc := range []struct {
		name, value string
		matches     int
	}{
		{"clean", "clean-value", 0},
		{"last-custom", fmt.Sprintf("benchmark%02d-abcdefghijklmnopqrst", len(rules)-1), 1},
	} {
		if len(policy.Detect(tc.value).Matches) != tc.matches {
			b.Fatal("benchmark input does not satisfy its detection expectation")
		}
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				benchmarkFindings = len(policy.Detect(tc.value).Matches)
			}
		})
	}
}

func BenchmarkTraceIDsForFieldManyScopes(b *testing.B) {
	const scopeCount = 1000
	scopes := make([]*tracev1.ScopeSpans, scopeCount)
	for i := range scopes {
		traceID := make([]byte, 16)
		traceID[14] = byte(i >> 8)
		traceID[15] = byte(i)
		scopes[i] = &tracev1.ScopeSpans{Spans: []*tracev1.Span{{TraceId: traceID}}}
	}
	request := &tempopb.PushSpansRequest{
		Batches: []*tracev1.ResourceSpans{{ScopeSpans: scopes}},
	}
	field := secrets.TraceField{Location: secrets.TraceLocation{Resource: 0, Scope: scopeCount - 1}}
	direct := make([][]byte, 1)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		benchmarkTraceIDs = len(traceIDsForField(request, field, direct, make(map[traceSetKey][][]byte)))
	}
}

func BenchmarkProcessorPushSpans(b *testing.B) {
	for _, tc := range []struct {
		name         string
		valueProfile string
		secretEvery  int
		sameTrace    bool
	}{
		{name: "single-trace-short-clean", valueProfile: "short", sameTrace: true},
		{name: "single-trace-long-clean", valueProfile: "long", sameTrace: true},
		{name: "short-clean", valueProfile: "short"},
		{name: "mixed-clean", valueProfile: "mixed"},
		{name: "long-clean", valueProfile: "long"},
		{name: "one-secret", valueProfile: "long", secretEvery: 100},
		{name: "all-spans-secret", valueProfile: "long", secretEvery: 1},
	} {
		b.Run(tc.name, func(b *testing.B) {
			const (
				spanCount = 100
				attrCount = 8
			)
			req := benchmarkPushRequest(spanCount, attrCount, tc.valueProfile, tc.secretEvery, tc.sameTrace)
			cfg := Config{}
			p, err := New(cfg, "benchmark-"+tc.name, log.NewNopLogger(), nil)
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ReportMetric(spanCount, "spans/op")
			b.ReportMetric(spanCount*attrCount, "attrs/op")
			b.SetBytes(int64(req.Size()))
			b.ResetTimer()
			for range b.N {
				p.PushSpans(context.Background(), req)
			}
		})
	}
}

func BenchmarkProcessorPushSpansOpsProfiles(b *testing.B) {
	for _, profile := range []struct {
		name      string
		spanCount int
		attrCount int
	}{
		{name: "ops-p50-bytes-per-span", spanCount: 4, attrCount: 16},
		{name: "ops-p90-bytes-per-span", spanCount: 4, attrCount: 20},
		{name: "ops-p99-bytes-per-span", spanCount: 4, attrCount: 28},
	} {
		b.Run(profile.name, func(b *testing.B) {
			req := benchmarkPushRequest(profile.spanCount, profile.attrCount, "long", 0, true)
			p, err := New(Config{}, "benchmark-"+profile.name, log.NewNopLogger(), nil)
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.SetBytes(int64(req.Size()))
			b.ResetTimer()
			for range b.N {
				p.PushSpans(context.Background(), req)
			}
			b.ReportMetric(float64(profile.spanCount), "spans/op")
			b.ReportMetric(float64(profile.spanCount*profile.attrCount), "attrs/op")
			b.ReportMetric(float64(req.Size())/float64(profile.spanCount), "bytes/span")
		})
	}
}

func BenchmarkProcessorPushSpansOpsHighCardinalityValues(b *testing.B) {
	const (
		spanCount = 4
		attrCount = 16
	)
	req := benchmarkPushRequest(spanCount, attrCount, "long", 0, true)
	for spanIndex, span := range req.Batches[0].ScopeSpans[0].Spans {
		for attrIndex, attribute := range span.Attributes {
			attribute.Value = &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{
				StringValue: fmt.Sprintf("https://api.example.com/v1/orders/%d/%d", spanIndex, attrIndex),
			}}
		}
	}
	p, err := New(Config{}, "benchmark-ops-high-cardinality", log.NewNopLogger(), nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(req.Size()))
	b.ResetTimer()
	for range b.N {
		p.PushSpans(context.Background(), req)
	}
	b.ReportMetric(spanCount, "spans/op")
	b.ReportMetric(spanCount*attrCount, "attrs/op")
	b.ReportMetric(float64(req.Size())/spanCount, "bytes/span")
}

func BenchmarkWalkPushSpansRequest(b *testing.B) {
	const (
		spanCount = 100
		attrCount = 8
	)
	req := benchmarkPushRequest(spanCount, attrCount, "long", 0, true)

	b.ReportAllocs()
	b.ReportMetric(spanCount, "spans/op")
	b.ReportMetric(spanCount*attrCount, "attrs/op")
	b.SetBytes(int64(req.Size()))
	b.ResetTimer()
	for range b.N {
		fields := 0
		secrets.WalkPushSpansRequest(req, func(field secrets.TraceField) bool {
			fields += len(field.Value)
			return true
		})
		benchmarkFindings = fields
	}
}

func benchmarkPushRequest(spanCount, attrCount int, valueProfile string, secretEvery int, sameTrace bool) *tempopb.PushSpansRequest {
	longValues := []string{
		"https://api.example.com/v1/orders/123456",
		"production-us-east-1",
		"checkout-service-instance-42",
		"customer-request-completed",
	}
	shortValues := []string{"GET", "200", "prod", "ok"}

	spans := make([]*tracev1.Span, spanCount)
	for spanIdx := range spans {
		attrs := make([]*commonv1.KeyValue, attrCount)
		for attrIdx := range attrs {
			values := longValues
			if valueProfile == "short" || valueProfile == "mixed" && attrIdx%2 == 0 {
				values = shortValues
			}
			value := values[(spanIdx+attrIdx)%len(values)]
			if secretEvery > 0 && spanIdx%secretEvery == 0 && attrIdx == 0 {
				value = fakeSlackToken()
			}
			attrs[attrIdx] = benchmarkAttribute(fmt.Sprintf("benchmark.attr.%d", attrIdx), value)
		}
		traceIndex := spanIdx
		if sameTrace {
			traceIndex = 0
		}
		spans[spanIdx] = &tracev1.Span{
			TraceId:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, byte(traceIndex)},
			SpanId:     []byte{1, 2, 3, 4, 5, 6, 7, byte(spanIdx)},
			Attributes: attrs,
		}
	}

	return &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{{
		Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{
			benchmarkAttribute("service.name", "checkout-service"),
		}},
		ScopeSpans: []*tracev1.ScopeSpans{{Spans: spans}},
	}}}
}

func benchmarkAttribute(key, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{
			StringValue: value,
		}},
	}
}
