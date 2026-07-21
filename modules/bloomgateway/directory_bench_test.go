package bloomgateway

import (
	"math/rand"
	"testing"
)

// BenchmarkDirectoryInsertLive measures InsertLive's cost through Directory
// -- unlike BenchmarkLeafInsert (leaf_bench_test.go), which calls
// Leaf.InsertIfAbsent directly and so never exercises Directory's stripe
// lock or its atomic entries counter (added alongside the entries_total
// gauge). InsertLive is the actual write-path hot loop (events.go's
// ApplyAddChunk calls it once per trace ID), so this is the benchmark that
// must not measurably regress when a per-insert atomic Add is introduced.
// Same reset-to-reference-size shape as BenchmarkLeafInsert, for the same
// reason (measure steady-state cost, not an ever-growing leaf).
func BenchmarkDirectoryInsertLive(b *testing.B) {
	const resetEvery = 64
	const idx = uint32(0)

	rng := rand.New(rand.NewSource(3))

	fps := make([]uint16, b.N)
	handles := make([]Handle, b.N)
	for i := range fps {
		fps[i] = uint16(rng.Intn(1 << 16))
		handles[i] = Handle(rng.Uint32())
	}

	dir := NewDirectory(1) // 2 slots; only idx 0 is ever touched
	resetLeaf := func() {
		dir.Shed(idx)
		leaf, started := dir.BeginConstructing(idx)
		if !started {
			b.Fatal("BeginConstructing must succeed on a freshly shed slot")
		}
		// Sequential setup, no concurrent access yet (InsertLive hasn't been
		// called on this idx this round) -- bypassing the lock here is safe,
		// matching BenchmarkSweepPass's own identical convention.
		for leaf.Len() < referenceEntriesPerLeaf {
			leaf.InsertIfAbsent(uint16(rng.Intn(1<<16)), Handle(rng.Uint32()))
		}
		if err := dir.Complete(idx, leaf); err != nil {
			b.Fatal(err)
		}
	}
	resetLeaf()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%resetEvery == 0 && i > 0 {
			b.StopTimer()
			resetLeaf()
			b.StartTimer()
		}
		dir.InsertLive(idx, fps[i], handles[i])
	}
}
