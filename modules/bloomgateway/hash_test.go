package bloomgateway

import (
	"encoding/hex"
	"math/rand"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustDecodeHex is a small test helper so the vector table below can be
// written as hex strings (matching how trace IDs are usually printed/
// discussed) instead of Go byte-slice literals.
func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

// TestHash_ConformanceVectors is the load-bearing conformance point named
// in the implementation plan's invariant-to-test map (§7, invariant #6):
// canonical 16-byte zero-padded trace-ID hashing, with fixed (seed, d, f,
// traceID) tuples and their frozen (leafIdx, fingerprint) outputs. Values
// were captured from a single real run of this package's own Address (a
// from-scratch "hand computation" of xxhash64 itself is not attempted —
// that primitive has its own upstream tests; what this file freezes and
// cross-checks is THIS package's bit-slicing on top of it, which is the
// part WP5 actually owns and could get wrong). TestAddress_IndependentBit
// SlicingCrossCheck below re-derives the same outputs via a differently
// written formula, so a mis-slice would have to be wrong in the same way
// in two independently authored expressions to slip past both tests.
func TestHash_ConformanceVectors(t *testing.T) {
	const seedAlpha = "test-seed-alpha"
	hashSeed := HashSeed([]byte(seedAlpha))
	require.Equal(t, uint64(14667966163368906271), hashSeed, "HashSeed derivation must not drift: every gateway instance (and later, every producer/QF) must reproduce it byte-for-byte")

	tests := []struct {
		name     string
		traceID  []byte
		d, f     uint8
		wantLeaf uint32
		wantFP   uint32
	}{
		{
			name:     "16-byte all-zero",
			traceID:  mustDecodeHex(t, "00000000000000000000000000000000"),
			d:        25,
			f:        16,
			wantLeaf: 13424600,
			wantFP:   7324,
		},
		{
			name:     "16-byte sequential",
			traceID:  mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f10"),
			d:        25,
			f:        16,
			wantLeaf: 6566244,
			wantFP:   20887,
		},
		{
			name:     "16-byte all-ff",
			traceID:  mustDecodeHex(t, "ffffffffffffffffffffffffffffffff"),
			d:        25,
			f:        16,
			wantLeaf: 27912552,
			wantFP:   24545,
		},
		{
			name:     "8-byte Jaeger-style id",
			traceID:  mustDecodeHex(t, "0102030405060708"),
			d:        25,
			f:        16,
			wantLeaf: 33203212,
			wantFP:   46340,
		},
		{
			name:     "8-byte Jaeger-style id, small d/f",
			traceID:  mustDecodeHex(t, "0102030405060708"),
			d:        4,
			f:        8,
			wantLeaf: 15,
			wantFP:   213,
		},
		{
			name:     "d=1 f=0 boundary",
			traceID:  mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f10"),
			d:        1,
			f:        0,
			wantLeaf: 0,
			wantFP:   0,
		},
		{
			name:     "d=32 f=32 boundary (d+f=64)",
			traceID:  mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f10"),
			d:        32,
			f:        32,
			wantLeaf: 840479272,
			wantFP:   3418735657,
		},
		{
			name:     "1-byte id",
			traceID:  mustDecodeHex(t, "7f"),
			d:        6,
			f:        10,
			wantLeaf: 30,
			wantFP:   860,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaf, fp := Address(tt.traceID, hashSeed, tt.d, tt.f)
			assert.Equal(t, tt.wantLeaf, leaf, "leafIdx")
			assert.Equal(t, tt.wantFP, fp, "fingerprint")
		})
	}

	// The invariant DESIGN.md calls out explicitly (§ Design constraints):
	// a 64-bit Jaeger-style trace ID must hash IDENTICALLY to its 16-byte
	// zero-padded form, because that is the canonical form vparquet stores
	// and the query API parser produces. Padding must happen in exactly
	// one place (pkg/util.PadTraceIDTo16Bytes) upstream of every computed
	// hash; this assertion is what would catch a second, drifted padding
	// implementation being introduced anywhere in the call chain.
	t.Run("8-byte id equals its pre-padded 16-byte form", func(t *testing.T) {
		eightByte := mustDecodeHex(t, "0102030405060708")
		prePadded := mustDecodeHex(t, "00000000000000000102030405060708")
		require.Len(t, prePadded, 16)

		leafFromEight, fpFromEight := Address(eightByte, hashSeed, 25, 16)
		leafFromPadded, fpFromPadded := Address(prePadded, hashSeed, 25, 16)

		assert.Equal(t, leafFromPadded, leafFromEight)
		assert.Equal(t, fpFromPadded, fpFromEight)
	})
}

// TestHashSeed_SeedFingerprint_DomainSeparation locks in AMENDMENT A4: the
// two derivations must differ (so disclosing SeedFingerprint never
// discloses the actual hashing seed) and must each be frozen byte-for-byte
// forever, since every gateway instance must land on the same hashSeed and
// every consumer of a query response / snapshot header must agree on
// SeedFingerprint.
func TestHashSeed_SeedFingerprint_DomainSeparation(t *testing.T) {
	tests := []struct {
		name            string
		seed            string
		wantHashSeed    uint64
		wantFingerprint uint64
	}{
		{"non-empty seed", "test-seed-alpha", 14667966163368906271, 17407430052991024168},
		{"empty seed", "", 16804241149081757544, 9962287286179718960},
		{"single-byte seed", "z", 16125329376712418272, 9603959075697281734},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hs := HashSeed([]byte(tt.seed))
			fp := SeedFingerprint([]byte(tt.seed))

			assert.Equal(t, tt.wantHashSeed, hs)
			assert.Equal(t, tt.wantFingerprint, fp)
			assert.NotEqual(t, hs, fp, "HashSeed and SeedFingerprint must never collide in value, or SeedFingerprint would disclose the effective hashing seed")
		})
	}
}

func TestWithDomainTag_DoesNotAliasCaller(t *testing.T) {
	// Give seed spare capacity so append(seed, tag) WOULD alias/mutate it
	// if withDomainTag used append directly instead of a fresh allocation.
	seed := make([]byte, 4, 16)
	copy(seed, []byte("abcd"))

	tagged := withDomainTag(seed, hashSeedDomainTag)

	assert.Len(t, tagged, 5)
	assert.Equal(t, byte(0x00), tagged[4])
	assert.Equal(t, 4, len(seed), "withDomainTag must not mutate the caller's seed slice")
	assert.Equal(t, "abcd", string(seed))
}

// TestAddress_IndependentBitSlicingCrossCheck re-derives leafIdx and
// fingerprint via division/modulo on a freshly computed xxhash sum,
// deliberately written differently from Address's shift/mask expressions,
// across randomized (but seeded, for reproducibility) inputs. This is the
// second half of the conformance strategy described in
// TestHash_ConformanceVectors's doc comment: a bug in Address's bit
// arithmetic would have to also appear in this independently phrased
// arithmetic to go undetected.
func TestAddress_IndependentBitSlicingCrossCheck(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	hashSeed := HashSeed([]byte("cross-check-seed"))

	for i := 0; i < 200; i++ {
		traceID := make([]byte, 16)
		_, err := rng.Read(traceID)
		require.NoError(t, err)

		d := uint8(1 + rng.Intn(32))          // [1, 32]
		f := uint8(rng.Intn(64 - int(d) + 1)) // keep d+f <= 64

		wantLeaf, wantFP := referenceAddress(t, traceID, hashSeed, d, f)
		gotLeaf, gotFP := Address(traceID, hashSeed, d, f)

		require.Equal(t, wantLeaf, gotLeaf, "d=%d f=%d traceID=%x", d, f, traceID)
		require.Equal(t, wantFP, gotFP, "d=%d f=%d traceID=%x", d, f, traceID)

		// Bound checks use uint64 arithmetic throughout: d and f can each be
		// 32, and uint32(1)<<32 would itself silently wrap to 0 (shifting a
		// 32-bit value by its own bit width), which would make the check
		// vacuous instead of a real bound — this bit the test itself during
		// development, not Address.
		assert.Less(t, uint64(gotLeaf), uint64(1)<<d, "leafIdx must fit in d bits")
		if f > 0 {
			assert.Less(t, uint64(gotFP), uint64(1)<<f, "fingerprint must fit in f bits")
		} else {
			assert.Zero(t, gotFP)
		}
	}
}

// referenceAddress independently recomputes what Address computes, using
// division and modulo instead of shift and mask. traceID must already be
// 16 bytes (the caller pads if needed) so this helper exercises only the
// bit-slicing, not padding.
func referenceAddress(t *testing.T, traceID []byte, hashSeed uint64, d, f uint8) (uint32, uint32) {
	t.Helper()
	require.Len(t, traceID, 16)

	digest := xxhash.NewWithSeed(hashSeed)
	_, err := digest.Write(traceID)
	require.NoError(t, err)
	sum := digest.Sum64()

	leafDivisor := uint64(1) << (64 - uint64(d))
	leaf := uint32(sum / leafDivisor)

	if f == 0 {
		return leaf, 0
	}
	fpDivisor := uint64(1) << (64 - uint64(d) - uint64(f))
	modulus := uint64(1) << uint64(f)
	fp := uint32((sum / fpDivisor) % modulus)
	return leaf, fp
}

// FuzzAddress is seed-corpus-only by default (f.Skip() below), per this
// repo's convention for fuzz tests (modules/blockbuilder/util/id_test.go):
// `go test` never spends time on it. To actually fuzz, comment out the
// f.Skip() line locally and run `go test -fuzz=FuzzAddress` — f.Skip()
// unconditionally short-circuits BOTH the seed-corpus run and -fuzz mode
// (verified directly; it is not a "skip only outside -fuzz" toggle), so it
// must be removed, not just left in place, to get real fuzzing.
func FuzzAddress(f *testing.F) {
	f.Skip()

	f.Add([]byte("0102030405060708090a0b0c0d0e0f10"), uint64(1), uint8(25), uint8(16))
	f.Add([]byte(""), uint64(0), uint8(1), uint8(0))
	f.Add([]byte("0102030405060708"), uint64(0xffffffffffffffff), uint8(32), uint8(32))

	f.Fuzz(func(t *testing.T, traceID []byte, hashSeed uint64, d, f uint8) {
		d = 1 + d%32 // keep in [1, 32]
		f %= 65 - d  // keep d+f <= 64

		leaf, fp := Address(traceID, hashSeed, d, f)

		// uint64 bounds: see the comment in
		// TestAddress_IndependentBitSlicingCrossCheck on why uint32(1)<<32
		// is not a usable bound.
		if uint64(leaf) >= uint64(1)<<d {
			t.Fatalf("leafIdx %d out of range for d=%d", leaf, d)
		}
		if f > 0 && uint64(fp) >= uint64(1)<<f {
			t.Fatalf("fingerprint %d out of range for f=%d", fp, f)
		}
		if f == 0 && fp != 0 {
			t.Fatalf("fingerprint must be 0 when f=0, got %d", fp)
		}
	})
}
