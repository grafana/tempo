package bloomgateway

import (
	"testing"

	"github.com/cespare/xxhash/v2"
)

// addressNoPool is a deliberately-duplicated, non-pooled copy of Address's
// hot path (plain xxhash.NewWithSeed per call) kept ONLY to let
// BenchmarkAddress/NoPool measure what Address would cost without
// digestPool — i.e. to decide the pooling question empirically rather than
// by assertion (§0 D4). It must not be used outside this benchmark.
func addressNoPool(traceID []byte, hashSeed uint64, d, f uint8) (leafIdx uint32, fingerprint uint32) {
	canonical := traceID
	if len(canonical) != 16 {
		canonical = paddedCopyForBench(canonical)
	}

	dig := xxhash.NewWithSeed(hashSeed)
	_, _ = dig.Write(canonical)
	sum := dig.Sum64()

	leafIdx = uint32(sum >> (64 - uint64(d)))
	fingerprint = uint32((sum >> (64 - uint64(d) - uint64(f))) & fingerprintMask(f))
	return leafIdx, fingerprint
}

func paddedCopyForBench(traceID []byte) []byte {
	padded := make([]byte, 16)
	if len(traceID) > 16 {
		copy(padded, traceID[len(traceID)-16:])
	} else {
		copy(padded[16-len(traceID):], traceID)
	}
	return padded
}

// BenchmarkAddress is sized at the reference ~200k-hashes-per-Add rate (§
// Event processing cost). It reports ns/op and allocs/op for both the
// pooled implementation Address actually uses and a non-pooled variant, so
// the digestPool decision documented in hash.go is backed by measurement.
func BenchmarkAddress(b *testing.B) {
	seed := HashSeed([]byte("bench-seed"))
	traceID := make([]byte, 16)
	for i := range traceID {
		traceID[i] = byte(i)
	}

	b.Run("Pooled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Address(traceID, seed, 25, 16)
		}
	})

	b.Run("NoPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			addressNoPool(traceID, seed, 25, 16)
		}
	})

	// A batch of 200_000 hashes approximates one instance's share of a
	// single AddChunk apply at reference sizing (§ Event processing:
	// "~200k hashes... per instance").
	b.Run("Pooled/200kBatch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 200_000; j++ {
				Address(traceID, seed, 25, 16)
			}
		}
	})
}

// BenchmarkAddress_Parallel exercises the pool under concurrent access from
// the worker pool (§ Write path: fixed N=16 workers applying in parallel),
// which is the scenario a sync.Pool is specifically good at (per-P local
// caches) and a single shared *xxhash.Digest would not be.
func BenchmarkAddress_Parallel(b *testing.B) {
	seed := HashSeed([]byte("bench-seed"))
	traceID := make([]byte, 16)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Address(traceID, seed, 25, 16)
		}
	})
}
