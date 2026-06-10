package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCountOverTimeAggregator_Extrapolate verifies that, with extrapolation
// enabled, each observed span contributes its IntrinsicSpanMultiplier value
// instead of 1.
func TestCountOverTimeAggregator_Extrapolate(t *testing.T) {
	cases := []struct {
		name       string
		multiplier float64
		want       float64
	}{
		{"no multiplier attribute -> 1.0", 0, 5}, // 5 spans, no multiplier => 5
		{"multiplier 2.0 -> 2x", 2, 10},
		{"multiplier 4.0 -> 4x", 4, 20},
		{"multiplier 100.0 -> 100x", 100, 500},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agg := NewCountOverTimeAggregator()
			agg.SetExtrapolate(true)

			for i := 0; i < 5; i++ {
				sp := newMockSpan(nil)
				if tc.multiplier > 0 {
					sp.attributes[IntrinsicSpanMultiplierAttribute] = NewStaticFloat(tc.multiplier)
				}
				agg.Observe(sp)
			}
			require.Equal(t, tc.want, agg.Sample())
		})
	}
}

// TestCountOverTimeAggregator_DefaultUnaffected ensures the default (no
// extrapolation) path is unchanged.
func TestCountOverTimeAggregator_DefaultUnaffected(t *testing.T) {
	agg := NewCountOverTimeAggregator()

	for i := 0; i < 5; i++ {
		sp := newMockSpan(nil)
		sp.attributes[IntrinsicSpanMultiplierAttribute] = NewStaticFloat(4.0)
		agg.Observe(sp)
	}
	require.Equal(t, 5.0, agg.Sample(), "extrapolation must be opt-in")
}

// benchSpan mimics the production vparquet5 span: AttributeFor short-circuits
// the IntrinsicSpanMultiplier lookup to a direct field access, instead of the
// generic map lookup used by mockSpan. This makes the benchmark more
// representative of real per-span hot-path cost.
type benchSpan struct {
	mult float64
}

func (b *benchSpan) AttributeFor(a Attribute) (Static, bool) {
	if a.Intrinsic == IntrinsicSpanMultiplier {
		return NewStaticFloat(b.mult), true
	}
	return StaticNil, false
}

func (b *benchSpan) AllAttributes() map[Attribute]Static       { return nil }
func (b *benchSpan) AllAttributesFunc(func(Attribute, Static)) {}
func (b *benchSpan) ID() []byte                                { return nil }
func (b *benchSpan) StartTimeUnixNanos() uint64                { return 0 }
func (b *benchSpan) DurationNanos() uint64                     { return 0 }
func (b *benchSpan) SiblingOf([]Span, []Span, bool, bool, []Span) []Span {
	return nil
}

func (b *benchSpan) DescendantOf([]Span, []Span, bool, bool, bool, []Span) []Span {
	return nil
}

func (b *benchSpan) ChildOf([]Span, []Span, bool, bool, bool, []Span) []Span {
	return nil
}

// BenchmarkCountOverTimeAggregator measures the hot-path cost of
// extrapolation toggled off vs on, using both a generic mockSpan and a
// production-like benchSpan.
func BenchmarkCountOverTimeAggregator(b *testing.B) {
	mock := newMockSpan(nil)
	mock.attributes[IntrinsicSpanMultiplierAttribute] = NewStaticFloat(2.0)

	prod := &benchSpan{mult: 2.0}
	prodNoMult := &benchSpan{mult: 0}

	b.Run("default (no extrapolation)", func(b *testing.B) {
		agg := NewCountOverTimeAggregator()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			agg.Observe(prod)
		}
	})

	b.Run("extrapolation on, prod-like span", func(b *testing.B) {
		agg := NewCountOverTimeAggregator()
		agg.SetExtrapolate(true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			agg.Observe(prod)
		}
	})

	b.Run("extrapolation on, prod-like, no multiplier", func(b *testing.B) {
		agg := NewCountOverTimeAggregator()
		agg.SetExtrapolate(true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			agg.Observe(prodNoMult)
		}
	})

	b.Run("extrapolation on, mockSpan (map lookup)", func(b *testing.B) {
		agg := NewCountOverTimeAggregator()
		agg.SetExtrapolate(true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			agg.Observe(mock)
		}
	})
}
