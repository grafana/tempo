package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
	"github.com/stretchr/testify/assert"
)

func TestMultiplierFromTraceState(t *testing.T) {
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
			result := MultiplierFromTraceState(tc.traceState)
			assert.InDelta(t, tc.expected, result, 0.001, "tracestate: %s", tc.traceState)
		})
	}
}

func BenchmarkMultiplierFromTraceState(b *testing.B) {
	b.Run("ot=th:8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			MultiplierFromTraceState("ot=th:8")
		}
	})
	b.Run("no ot segment", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			MultiplierFromTraceState("vendor1=value1,vendor2=value2")
		}
	})
	b.Run("empty", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			MultiplierFromTraceState("")
		}
	})
}

func FuzzMultiplierFromTraceState(f *testing.F) {
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
		result := MultiplierFromTraceState(traceState)
		w3c, err := sampling.NewW3CTraceState(traceState)
		if err == nil {
			assert.Equal(t, w3c.OTelValue().AdjustedCount(), result, "traceState: %s", traceState)
		}
	})
}
