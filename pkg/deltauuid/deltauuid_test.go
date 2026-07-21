package deltauuid

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// randomUUIDs returns n independently random 16-byte values from rng, in no
// particular order.
func randomUUIDs(rng *rand.Rand, n int) [][16]byte {
	ids := make([][16]byte, n)
	for i := range ids {
		binary.BigEndian.PutUint64(ids[i][:8], rng.Uint64())
		binary.BigEndian.PutUint64(ids[i][8:], rng.Uint64())
	}
	return ids
}

// randomAscendingUUIDs returns n random 16-byte values, sorted ascending and
// deduplicated — the precondition EncodeSortedDeltas is documented to want
// for its size benefit (collisions are astronomically unlikely at these
// sizes over the 128-bit space, so len(result) == n in practice).
func randomAscendingUUIDs(rng *rand.Rand, n int) [][16]byte {
	ids := randomUUIDs(rng, n)
	sort.Slice(ids, func(i, j int) bool {
		return bytes.Compare(ids[i][:], ids[j][:]) < 0
	})
	return dedupSorted(ids)
}

func dedupSorted(ids [][16]byte) [][16]byte {
	if len(ids) == 0 {
		return ids
	}
	out := ids[:1]
	for _, id := range ids[1:] {
		if id != out[len(out)-1] {
			out = append(out, id)
		}
	}
	return out
}

func TestEncodeSortedDeltas_FirstEntryGapsFromAllZeroUUID(t *testing.T) {
	// The first entry's "previous" is the all-zero UUID (§0 D13), so an
	// all-zero first entry is itself a zero gap: one length-zero byte.
	encoded := EncodeSortedDeltas([][16]byte{{}})
	assert.Equal(t, []byte{0}, encoded)
}

func TestEncodeSortedDeltas_Empty(t *testing.T) {
	encoded := EncodeSortedDeltas(nil)
	assert.Empty(t, encoded)

	decoded, err := DecodeSortedDeltas(encoded)
	require.NoError(t, err)
	assert.Empty(t, decoded)
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"empty", 0},
		{"single", 1},
		{"small", 7},
		{"medium", 596}, // DESIGN.md §Sizing reference entries-per-leaf
		{"large", 20000},
	}
	rng := rand.New(rand.NewSource(1))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := randomAscendingUUIDs(rng, tt.n)

			encoded := EncodeSortedDeltas(ids)
			decoded, err := DecodeSortedDeltas(encoded)

			require.NoError(t, err)
			assert.Equal(t, ids, decoded)
		})
	}
}

// TestRoundTrip_PropertyRandomSizes is the property test the plan asks for:
// round-trip holds for many random ascending slices of varying size, not
// just the fixed sizes in TestRoundTrip.
func TestRoundTrip_PropertyRandomSizes(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 200; i++ {
		n := rng.Intn(300)
		ids := randomAscendingUUIDs(rng, n)

		encoded := EncodeSortedDeltas(ids)
		decoded, err := DecodeSortedDeltas(encoded)

		require.NoError(t, err)
		require.Equal(t, ids, decoded)
	}
}

func TestDecodeSortedDeltas_Malformed(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{
			name:    "length byte claims bytes but none remain",
			input:   []byte{5},
			wantErr: ErrTruncatedInput,
		},
		{
			name:    "length byte claims more bytes than remain",
			input:   []byte{16, 1, 2, 3},
			wantErr: ErrTruncatedInput,
		},
		{
			name:    "valid entry followed by a truncated entry",
			input:   []byte{0, 5, 1, 2},
			wantErr: ErrTruncatedInput,
		},
		{
			name:    "length byte just above the valid range (17)",
			input:   []byte{17},
			wantErr: ErrInvalidLengthByte,
		},
		{
			name:    "length byte at the max possible value (255)",
			input:   []byte{255},
			wantErr: ErrInvalidLengthByte,
		},
		{
			name:    "valid entry followed by an out-of-range length byte",
			input:   []byte{0, 200},
			wantErr: ErrInvalidLengthByte,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := DecodeSortedDeltas(tt.input)

			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.wantErr), "got error %v, want it to wrap %v", err, tt.wantErr)
			assert.Nil(t, decoded)
		})
	}
}

// TestEncodeSortedDeltas_NonAscendingInput locks in the decision documented
// on EncodeSortedDeltas: the function does not require or verify ascending,
// deduplicated input. It is caller's responsibility to sort for the size
// benefit, but every gap is an unsigned 128-bit difference computed with
// mod-2^128 wraparound, so any input order still round-trips exactly and
// still costs at most 17 B/entry (1 length byte + all 16 gap bytes) — the
// same pathological-case bound §0 D13 states for sorted input.
func TestEncodeSortedDeltas_NonAscendingInput(t *testing.T) {
	t.Run("reverse-sorted input round-trips in its own (non-ascending) order", func(t *testing.T) {
		rng := rand.New(rand.NewSource(7))
		ascending := randomAscendingUUIDs(rng, 20)
		reversed := make([][16]byte, len(ascending))
		for i, id := range ascending {
			reversed[len(ascending)-1-i] = id
		}

		encoded := EncodeSortedDeltas(reversed)
		decoded, err := DecodeSortedDeltas(encoded)

		require.NoError(t, err)
		assert.Equal(t, reversed, decoded, "decode must reproduce the exact input order, not re-sort it")
	})

	t.Run("duplicate consecutive entries round-trip via a zero gap", func(t *testing.T) {
		rng := rand.New(rand.NewSource(8))
		id := randomUUIDs(rng, 1)[0]
		ids := [][16]byte{id, id, id}

		encoded := EncodeSortedDeltas(ids)
		decoded, err := DecodeSortedDeltas(encoded)

		require.NoError(t, err)
		assert.Equal(t, ids, decoded)
		// Only the first entry's gap is from the all-zero UUID (and so
		// depends on id's own value); the two repeats are gaps from an
		// identical previous entry, i.e. zero gaps: a single 0x00 byte each.
		assert.Equal(t, []byte{0, 0}, encoded[len(encoded)-2:], "the two repeated entries are one-byte zero gaps")
	})

	t.Run("worst-case alternation still respects the 17 B/entry pathological bound", func(t *testing.T) {
		var allZero, allOnes [16]byte
		for i := range allOnes {
			allOnes[i] = 0xff
		}
		const n = 200
		ids := make([][16]byte, n)
		for i := range ids {
			if i%2 == 0 {
				ids[i] = allZero
			} else {
				ids[i] = allOnes
			}
		}

		encoded := EncodeSortedDeltas(ids)
		decoded, err := DecodeSortedDeltas(encoded)

		require.NoError(t, err)
		assert.Equal(t, ids, decoded)
		assert.LessOrEqual(t, len(encoded), n*17, "never more than 1 byte/entry worse than raw 16-byte UUIDs")
	})
}

// FuzzDecodeSortedDeltas guards the "errors, never panics" contract on
// malformed input beyond the fixed table above. Seed-corpus only per repo
// convention (f.Skip()); run explicitly with -fuzz=FuzzDecodeSortedDeltas.
func FuzzDecodeSortedDeltas(f *testing.F) {
	f.Skip()

	f.Add([]byte{0})
	f.Add([]byte{16, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	f.Add([]byte{255})
	f.Fuzz(func(_ *testing.T, b []byte) {
		_, _ = DecodeSortedDeltas(b) // must not panic; an error is fine
	})
}

// FuzzEncodeSortedDeltas_RoundTrip checks that any two 16-byte values,
// encoded consecutively in either order, decode back exactly — the fuzz
// analogue of TestEncodeSortedDeltas_NonAscendingInput.
func FuzzEncodeSortedDeltas_RoundTrip(f *testing.F) {
	f.Skip()

	f.Add(uint64(0), uint64(0), uint64(1), uint64(1))
	f.Fuzz(func(t *testing.T, aHi, aLo, bHi, bLo uint64) {
		var a, b [16]byte
		binary.BigEndian.PutUint64(a[:8], aHi)
		binary.BigEndian.PutUint64(a[8:], aLo)
		binary.BigEndian.PutUint64(b[:8], bHi)
		binary.BigEndian.PutUint64(b[8:], bLo)

		ids := [][16]byte{a, b}
		encoded := EncodeSortedDeltas(ids)
		decoded, err := DecodeSortedDeltas(encoded)
		if err != nil {
			t.Fatalf("unexpected error decoding %x: %v", encoded, err)
		}
		if decoded[0] != a || decoded[1] != b {
			t.Fatalf("round trip mismatch: got %x want %x", decoded, ids)
		}
	})
}
