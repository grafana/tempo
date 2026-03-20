package util

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
	"github.com/stretchr/testify/assert"

	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func TestFindServiceName(t *testing.T) {
	testCases := []struct {
		name                string
		attributes          []*v1_common.KeyValue
		expectedServiceName string
		expectedOk          bool
	}{
		{
			"empty attributes",
			nil,
			"",
			false,
		},
		{
			"service name",
			[]*v1_common.KeyValue{
				{
					Key: "cluster",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "test",
						},
					},
				},
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "my-service",
						},
					},
				},
			},
			"my-service",
			true,
		},
		{
			"service name",
			[]*v1_common.KeyValue{
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "",
						},
					},
				},
			},
			"",
			true,
		},
		{
			"no service name",
			[]*v1_common.KeyValue{
				{
					Key: "cluster",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "test",
						},
					},
				},
			},
			"",
			false,
		},
		{
			"service name is other type",
			[]*v1_common.KeyValue{
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_BoolValue{
							BoolValue: false,
						},
					},
				},
			},
			"false",
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svcName, ok := FindServiceName(tc.attributes)

			assert.Equal(t, tc.expectedOk, ok)
			assert.Equal(t, tc.expectedServiceName, svcName)
		})
	}
}

func TestGetSpanMultiplierFromTraceState(t *testing.T) {
	tests := []struct {
		name       string
		traceState string
		expected   float64
	}{
		{
			name:       "empty tracestate",
			traceState: "",
			expected:   0,
		},
		{
			name:       "th:0 means always sampled",
			traceState: "ot=th:0",
			expected:   1.0,
		},
		{
			name:       "th:8 means 50% sampling, multiplier 2",
			traceState: "ot=th:8",
			expected:   2.0,
		},
		{
			name:       "th:c means 25% sampling, multiplier 4",
			traceState: "ot=th:c",
			expected:   4.0,
		},
		{
			name:       "th:fd70a4 means ~1% sampling, multiplier ~100",
			traceState: "ot=th:fd70a4",
			expected:   100.0,
		},
		{
			name:       "multiple vendors in tracestate",
			traceState: "vendor1=value1,ot=th:8,vendor2=value2",
			expected:   2.0,
		},
		{
			name:       "ot value with multiple subkeys",
			traceState: "ot=rv:00112233445566;th:8",
			expected:   2.0,
		},
		{
			name:       "invalid hex in threshold",
			traceState: "ot=th:xyz",
			expected:   0,
		},
		{
			name:       "no ot key",
			traceState: "vendor1=value1,vendor2=value2",
			expected:   0,
		},
		{
			name:       "ot without th subkey",
			traceState: "ot=rv:00112233445566",
			expected:   0,
		},
		{
			name:       "threshold too long",
			traceState: "ot=th:123456789abcdef",
			expected:   0,
		},
		{
			name:       "empty threshold value",
			traceState: "ot=th:",
			expected:   0,
		},
		{
			name:       "vendor key ending with ot",
			traceState: "not=foo:bar",
			expected:   0,
		},
		{
			name:       "vendor key ending with ot with th",
			traceState: "not=th:c",
			expected:   0,
		},
		{
			name:       "not and ot vendor keys each with th",
			traceState: "not=th:8,ot=th:c",
			expected:   4.00,
		},
		{
			name:       "not and ot vendor keys each with th and whitespace",
			traceState: "not=th:8, ot=th:c",
			expected:   4.00,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getSpanMultiplierFromTraceState(tc.traceState)
			assert.InDelta(t, tc.expected, result, 0.001, "tracestate: %s", tc.traceState)
		})
	}
}

func BenchmarkGetSpanMultiplier(b *testing.B) {
	rs := &v1_resource.Resource{}

	spanWithTraceState := &v1.Span{
		TraceState: "ot=th:8",
	}
	spanWithoutTraceState := &v1.Span{
		TraceState: "xx=yy:zz",
		Attributes: []*v1_common.KeyValue{
			{
				Key: "sampling.ratio",
				Value: &v1_common.AnyValue{
					Value: &v1_common.AnyValue_DoubleValue{DoubleValue: 0.5},
				},
			},
		},
	}

	b.Run("with otel tracestate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GetSpanMultiplier("", spanWithTraceState, rs, true)
		}
	})

	b.Run("without otel tracestate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GetSpanMultiplier("sampling.ratio", spanWithoutTraceState, rs, true)
		}
	})

	b.Run("otel racestate disabled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GetSpanMultiplier("sampling.ratio", spanWithoutTraceState, rs, false)
		}
	})
}

func FuzzGetSpanMultiplierFromTraceState(f *testing.F) {
	f.Add("")
	f.Add("ot=th:8")
	f.Add("ot=th:c")
	f.Add("ot=th:")
	f.Add("vendor1=value1,ot=th:8,vendor2=value2")
	f.Add("ot=rv:00112233445566;th:8")
	f.Add("not=th:8,ot=th:c")
	f.Add("not=th:8, ot=th:c")
	f.Add("ot=th:xyz")
	f.Add("vendor1=value1,vendor2=value2")
	f.Add("  ot=th:8  ")
	f.Add(",,,")
	f.Add("ot=")
	f.Add("=value")

	f.Fuzz(func(t *testing.T, traceState string) {
		// Verify that our multiplier matches what is done from the sampling package.
		result := getSpanMultiplierFromTraceState(traceState)
		w3c, err := sampling.NewW3CTraceState(traceState)
		if err == nil {
			assert.Equal(t, w3c.OTelValue().AdjustedCount(), result, "traceState: %s", traceState)
		}
		// We are looser with trace state errors where we can still parse ot=,
		// so no assertions when there is an error as the results may differ.
	})
}

func TestGetSpanMultiplier_WithTraceState(t *testing.T) {
	ratioAttr := "sampling.ratio"

	makeSpan := func(traceState string, attrVal float64) *v1.Span {
		s := &v1.Span{
			TraceState: traceState,
		}
		if attrVal > 0 {
			s.Attributes = []*v1_common.KeyValue{
				{
					Key: ratioAttr,
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_DoubleValue{DoubleValue: attrVal},
					},
				},
			}
		}
		return s
	}

	rs := &v1_resource.Resource{}

	tests := []struct {
		name             string
		enableTraceState bool
		span             *v1.Span
		ratioKey         string
		expected         float64
	}{
		{
			name:             "tracestate disabled, uses attribute",
			enableTraceState: false,
			span:             makeSpan("ot=th:8", 0.5),
			ratioKey:         ratioAttr,
			expected:         2.0,
		},
		{
			name:             "tracestate enabled, tracestate wins over attribute",
			enableTraceState: true,
			span:             makeSpan("ot=th:c", 0.5),
			ratioKey:         ratioAttr,
			expected:         4.0, // from tracestate (25% sampling), not 2.0 from attribute
		},
		{
			name:             "tracestate enabled but empty, falls back to attribute",
			enableTraceState: true,
			span:             makeSpan("", 0.5),
			ratioKey:         ratioAttr,
			expected:         2.0,
		},
		{
			name:             "tracestate enabled but invalid, falls back to attribute",
			enableTraceState: true,
			span:             makeSpan("ot=th:xyz", 0.25),
			ratioKey:         ratioAttr,
			expected:         4.0,
		},
		{
			name:             "tracestate enabled, no attribute, no tracestate",
			enableTraceState: true,
			span:             makeSpan("", 0),
			ratioKey:         "",
			expected:         1.0,
		},
		{
			name:             "tracestate disabled, ignores tracestate",
			enableTraceState: false,
			span:             makeSpan("ot=th:8", 0),
			ratioKey:         "",
			expected:         1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := GetSpanMultiplier(tc.ratioKey, tc.span, rs, tc.enableTraceState)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}
