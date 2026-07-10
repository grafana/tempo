package deltauuid

import (
	"fmt"
	"math/rand"
	"testing"
)

// benchmarkSizes matches the plan's requested N values. 100 and 1,000 land
// in the "small window" regime IMPLEMENTATION_PLAN.md §0 D13 flags as not
// achieving meaningful delta compression (gaps stay near full-width even
// though the input is sorted); 100,000 approaches DESIGN.md's §Sizing
// reference "100k blocks unscoped" case.
var benchmarkSizes = []int{100, 1000, 100000}

// BenchmarkEncodeSortedDeltas reports ns/op, allocs/op, and (via
// b.ReportMetric) the actual measured bytes/entry at each N — the
// IMPLEMENTATION_PLAN.md §0 D13 "honesty finding": do not assume
// DESIGN.md's ~14 B/UUID figure holds uniformly, measure it.
func BenchmarkEncodeSortedDeltas(b *testing.B) {
	for _, n := range benchmarkSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			rng := rand.New(rand.NewSource(int64(n)))
			ids := randomAscendingUUIDs(rng, n)

			var encoded []byte
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				encoded = EncodeSortedDeltas(ids)
			}
			b.StopTimer()

			if n > 0 {
				b.ReportMetric(float64(len(encoded))/float64(n), "B/entry")
			}
		})
	}
}

// BenchmarkDecodeSortedDeltas is the encode benchmark's mirror image on the
// read side (the future query-frontend decoder's cost).
func BenchmarkDecodeSortedDeltas(b *testing.B) {
	for _, n := range benchmarkSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			rng := rand.New(rand.NewSource(int64(n)))
			ids := randomAscendingUUIDs(rng, n)
			encoded := EncodeSortedDeltas(ids)

			var decoded [][16]byte
			var err error
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				decoded, err = DecodeSortedDeltas(encoded)
			}
			b.StopTimer()

			if err != nil {
				b.Fatal(err)
			}
			if len(decoded) != n {
				b.Fatalf("decoded %d entries, want %d", len(decoded), n)
			}
			if n > 0 {
				b.ReportMetric(float64(len(encoded))/float64(n), "B/entry")
			}
		})
	}
}
