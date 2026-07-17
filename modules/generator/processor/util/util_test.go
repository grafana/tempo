package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func TestFindServiceName(t *testing.T) {
	testCases := []struct {
		name                string
		attributes          []v1_common.KeyValue
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
			[]v1_common.KeyValue{
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
			[]v1_common.KeyValue{
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
			[]v1_common.KeyValue{
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
			[]v1_common.KeyValue{
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

func TestFindServiceLabels(t *testing.T) {
	strAttr := func(key, value string) v1_common.KeyValue {
		return v1_common.KeyValue{
			Key: key,
			Value: &v1_common.AnyValue{
				Value: &v1_common.AnyValue_StringValue{StringValue: value},
			},
		}
	}

	testCases := []struct {
		name               string
		attributes         []v1_common.KeyValue
		expectedSvcName    string
		expectedJobName    string
		expectedInstanceID string
	}{
		{
			name: "empty attributes",
		},
		{
			name:            "service name only",
			attributes:      []v1_common.KeyValue{strAttr("service.name", "my-service")},
			expectedSvcName: "my-service",
			expectedJobName: "my-service",
		},
		{
			name: "service name and namespace",
			attributes: []v1_common.KeyValue{
				strAttr("service.namespace", "my-namespace"),
				strAttr("service.name", "my-service"),
			},
			expectedSvcName: "my-service",
			expectedJobName: "my-namespace/my-service",
		},
		{
			name:       "namespace without service name yields empty job",
			attributes: []v1_common.KeyValue{strAttr("service.namespace", "my-namespace")},
		},
		{
			name: "instance id",
			attributes: []v1_common.KeyValue{
				strAttr("service.name", "my-service"),
				strAttr("service.instance.id", "instance-1"),
			},
			expectedSvcName:    "my-service",
			expectedJobName:    "my-service",
			expectedInstanceID: "instance-1",
		},
		{
			name: "first occurrence wins",
			attributes: []v1_common.KeyValue{
				strAttr("service.name", "first"),
				strAttr("service.name", "second"),
				strAttr("service.instance.id", "instance-1"),
				strAttr("service.instance.id", "instance-2"),
			},
			expectedSvcName:    "first",
			expectedJobName:    "first",
			expectedInstanceID: "instance-1",
		},
		{
			name: "non-string values are stringified",
			attributes: []v1_common.KeyValue{
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_BoolValue{BoolValue: false},
					},
				},
			},
			expectedSvcName: "false",
			expectedJobName: "false",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svcName, jobName, instanceID := FindServiceLabels(tc.attributes)

			assert.Equal(t, tc.expectedSvcName, svcName)
			assert.Equal(t, tc.expectedJobName, jobName)
			assert.Equal(t, tc.expectedInstanceID, instanceID)
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
		Attributes: []v1_common.KeyValue{
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

func TestGetSpanMultiplier_WithTraceState(t *testing.T) {
	ratioAttr := "sampling.ratio"

	makeSpan := func(traceState string, attrVal float64) *v1.Span {
		s := &v1.Span{
			TraceState: traceState,
		}
		if attrVal > 0 {
			s.Attributes = []v1_common.KeyValue{
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
