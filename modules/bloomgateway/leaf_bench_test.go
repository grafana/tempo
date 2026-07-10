package bloomgateway

import (
	"math/rand"
	"testing"
)

// referenceEntriesPerLeaf is DESIGN.md § Sizing's reference leaf
// population (entries per leaf ~596 at D=25 across the reference tenant's
// 20e9 (trace, block) pairs) — the population size these benchmarks report
// against, so their ns/op numbers are comparable to the design's own cost
// model ("~10 probes over a ~3.6 KiB array").
const referenceEntriesPerLeaf = 596

// newPopulatedLeaf builds a leaf with exactly n entries via
// InsertIfAbsent, retrying on the rare fp/handle collision (an
// insert-if-absent no-op) so the returned leaf always has exactly n
// entries.
func newPopulatedLeaf(n int, rng *rand.Rand) *Leaf {
	l := NewLeaf()
	for l.Len() < n {
		l.InsertIfAbsent(uint16(rng.Intn(1<<16)), Handle(rng.Uint32()))
	}
	return l
}

// BenchmarkLeafInsert is one of this plan's three named hot-path
// benchmarks (WP9). It measures InsertIfAbsent's memmove-heavy cost while
// holding the leaf at the reference ~596-entries/leaf sizing throughout the
// run: letting b.N inserts accumulate unboundedly (the leaf would end up
// with b.N+596 entries, often millions) would measure an ever-larger,
// unrepresentative memmove cost instead of the steady-state number
// DESIGN.md's own cost model is stated against. Every resetEvery
// iterations the leaf is rebuilt from scratch back to referenceEntriesPerLeaf
// under b.StopTimer/b.StartTimer, so the timed portion always inserts into
// a leaf sized within one reset batch of the reference — a small, accepted
// amount of size variation (596 to 596+resetEvery) traded against not
// rebuilding on every single iteration (which would dominate wall-clock
// runtime for large b.N without affecting the reported ns/op, since the
// rebuild itself is untimed, but would still make `go test` slow).
func BenchmarkLeafInsert(b *testing.B) {
	const resetEvery = 64

	rng := rand.New(rand.NewSource(1))

	// Pre-generate every (fp, handle) pair to insert so the timed loop
	// measures only InsertIfAbsent's own cost, never the RNG's.
	fps := make([]uint16, b.N)
	handles := make([]Handle, b.N)
	for i := range fps {
		fps[i] = uint16(rng.Intn(1 << 16))
		handles[i] = Handle(rng.Uint32())
	}

	var l *Leaf
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%resetEvery == 0 {
			b.StopTimer()
			l = newPopulatedLeaf(referenceEntriesPerLeaf, rng)
			b.StartTimer()
		}
		l.InsertIfAbsent(fps[i], handles[i])
	}
}

// BenchmarkLeafLookup is BenchmarkLeafInsert's companion, measuring the
// query path's binary search at the same reference sizing. Lookups don't
// mutate the leaf, so there is no analogous size-drift concern — one
// populated leaf serves the whole run. Half the probed fingerprints are
// drawn from the leaf's own entries (guaranteed hits, modeling an existing
// trace); half are independently random (~596/65536 ~ 0.9% accidental-hit
// chance, overwhelmingly misses, modeling the miss/reject-all path) — this
// mirrors the query path's real mix of outcomes (§ Query path).
func BenchmarkLeafLookup(b *testing.B) {
	rng := rand.New(rand.NewSource(2))
	l := newPopulatedLeaf(referenceEntriesPerLeaf, rng)
	present := make([]uint16, len(l.fps))
	copy(present, l.fps)

	probes := make([]uint16, b.N)
	for i := range probes {
		if i%2 == 0 {
			probes[i] = present[rng.Intn(len(present))]
		} else {
			probes[i] = uint16(rng.Intn(1 << 16))
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Lookup(probes[i])
	}
}
