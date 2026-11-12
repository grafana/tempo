package sampling

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			expected:   -1,
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
			expected:   -1,
		},
		{
			name:       "no ot key",
			traceState: "vendor1=value1,vendor2=value2",
			expected:   -1,
		},
		{
			name:       "ot without th subkey",
			traceState: "ot=rv:00112233445566",
			expected:   -1,
		},
		{
			name:       "threshold too long",
			traceState: "ot=th:123456789abcdef",
			expected:   -1,
		},
		{
			name:       "empty threshold value",
			traceState: "ot=th:",
			expected:   -1,
		},
		{
			name:       "vendor key ending with ot",
			traceState: "not=foo:bar",
			expected:   -1,
		},
		{
			name:       "vendor key ending with ot with th",
			traceState: "not=th:c",
			expected:   -1,
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

	// Integer multipliers with an R-value stay integer (frac == 0, no
	// rounding decision needed).
	t.Run("integer multiplier with rv is unchanged", func(t *testing.T) {
		require.InDelta(t, 2.0, MultiplierFromTraceState("ot=th:8;rv:12345678901234"), 0.001)
		require.InDelta(t, 4.0, MultiplierFromTraceState("ot=th:c;rv:12345678901234"), 0.001)
	})

	// For non-integer multipliers with an R-value present, verify each
	// result is floor/ceil and E[result] converges to the raw multiplier.
	t.Run("fractional multiplier is stochastically rounded and unbiased", func(t *testing.T) {
		// th:4 → probability = (2^56 - 4*2^52) / 2^56 = 0.75 → multiplier = 1.333...
		raw := MultiplierFromTraceState("ot=th:4")
		require.InDelta(t, 4.0/3.0, raw, 1e-9)

		const n = 4096
		var sum float64
		for i := 0; i < n; i++ {
			// Vary the low bits of the R-value across the full 56-bit range.
			rv := uint64(i) * (1 << 44)
			ts := fmt.Sprintf("ot=th:4;rv:%014x", rv&((1<<56)-1))
			r := MultiplierFromTraceState(ts)
			// Every result must be either floor(4/3)=1 or ceil(4/3)=2.
			require.True(t, r == 1 || r == 2, "expected 1 or 2, got %v for ts=%q", r, ts)
			sum += r
		}
		mean := sum / float64(n)
		require.InDelta(t, raw, mean, 0.02, "mean over %d samples should track the raw multiplier", n)
	})

	// Same R-value → same rounding decision.
	t.Run("deterministic for a given rv", func(t *testing.T) {
		ts := "ot=th:4;rv:0123456789abcd"
		first := MultiplierFromTraceState(ts)
		for i := 0; i < 10; i++ {
			require.Equal(t, first, MultiplierFromTraceState(ts))
		}
	})
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
		if err != nil {
			return
		}
		// Our fast path finds the first `ot=` key; the strict parser applies
		// list-vendor semantics. When multiple `ot=` entries exist the two
		// paths can pick different values — skip cross-validation there.
		if strings.Count(traceState, "ot=") > 1 {
			return
		}
		// zc is not known to the strict parser; its handling is covered by table tests.
		if strings.Contains(traceState, "zc:1") {
			return
		}
		raw := w3c.OTelValue().AdjustedCount()
		if raw == 0 {
			assert.Equal(t, -1.0, result, "traceState: %s", traceState)
			return
		}
		// When there's no R-value, our result equals the raw AdjustedCount.
		// When an R-value is present, our result is stochastically rounded
		// and must be floor(raw) or ceil(raw). E[result] still equals raw
		// but each individual call may differ.
		if _, ok := w3c.OTelValue().RValueRandomness(); !ok {
			assert.Equal(t, raw, result, "traceState: %s", traceState)
			return
		}
		floor, ceil := math.Floor(raw), math.Floor(raw)
		if raw > floor {
			ceil = floor + 1
		}
		assert.True(t, result == floor || result == ceil, "traceState: %s — expected %v or %v, got %v", traceState, floor, ceil, result)
	})
}

func TestMultiplierFromTraceStateZeroCount(t *testing.T) {
	tests := []struct {
		name       string
		traceState string
		expected   float64
	}{
		{"no ot key", "vendor=zc:1", -1},
		{"zc only", "ot=zc:1", 0},
		{"zc with other subkeys after", "ot=zc:1;rv:6e6d1a75832a2f", 0},
		{"zc with other subkeys before", "ot=rv:6e6d1a75832a2f;zc:1", 0},
		{"zc takes precedence over th", "ot=th:8;zc:1", 0},
		{"zc in other vendor key", "not=zc:1,ot=th:8", 2},
		{"zc with other vendor keys", "a=b, ot=zc:1,c=d", 0},
		{"zc with value other than 1", "ot=zc:0", -1},
		{"zc with empty value", "ot=zc:", -1},
		{"subkey ending in zc", "ot=xzc:1", -1},
		{"subkey with zc prefix", "ot=zc:10", -1},
		{"empty subkey before zc", "ot=th:8;;zc:1", 0},
		{"empty subkey after zc", "ot=zc:1;;th:8", 0},
		{"trailing separator", "ot=zc:1;", 0},
		{"leading separator", "ot=;zc:1", 0},
		{"zc with invalid th", "ot=th:xyz;zc:1", 0},
		{"zc with too long th", "ot=th:123456789abcdef;zc:1", 0},
		{"only the first ot entry is used", "ot=th:8,ot=zc:1", 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, MultiplierFromTraceState(tc.traceState), "tracestate: %s", tc.traceState)
		})
	}
}
