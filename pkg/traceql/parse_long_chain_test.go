package traceql

import (
	"strings"
	"testing"
	"time"
)

// Regression test for grafana/tempo#7815.
//
// A long left-associative arithmetic chain used to parse in O(n^3) because
// newBinaryOperation ran uncached whole-subtree walks (referencesSpan,
// validate/impliedType) on every bottom-up reduction. At 4,000 terms the
// unfixed parser took ~3.5 minutes.
//
// Two triggers are covered:
//   - a chain that cannot be constant-folded because it contains a division by
//     zero (exercises the validate + execute fold path), and
//   - a chain whose leading operand references the span (exercises the
//     referencesSpan-only path, since the fold is skipped for span-referencing
//     expressions).
//
// The wall-clock bounds are deliberately generous to avoid CI flakiness. Under
// the O(n^3) behavior these term counts would take many minutes to hours.
func TestParseLongArithmeticChain(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		maxSeconds float64
	}{
		{
			name:       "unfoldable-constant-chain",
			query:      "{1/0+" + strings.Repeat("1+", 7998) + "1}",
			maxSeconds: 30,
		},
		{
			name:       "span-referencing-chain",
			query:      "{nestedSetLeft" + strings.Repeat("+1", 20000) + "}",
			maxSeconds: 10,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			maxDuration := time.Duration(tc.maxSeconds * float64(time.Second))
			start := time.Now()
			expr, err := ParseNoOptimizations(tc.query)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("ParseNoOptimizations returned error: %v", err)
			}
			if expr == nil {
				t.Fatal("ParseNoOptimizations returned a nil expression")
			}
			if elapsed > maxDuration {
				t.Fatalf("parsing %d bytes took %v (exceeds %v); O(n^3) regression",
					len(tc.query), elapsed, maxDuration)
			}
			t.Logf("parsed %d bytes in %v", len(tc.query), elapsed)
		})
	}
}

func BenchmarkParseArithmeticChain(b *testing.B) {
	sizes := []int{250, 500, 1000, 2000, 4000, 8000}
	for _, n := range sizes {
		q := "{1/0+" + strings.Repeat("1+", n-2) + "1}"
		b.Run(itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = ParseNoOptimizations(q)
			}
		})
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
