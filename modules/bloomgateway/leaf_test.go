package bloomgateway

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertSorted fails the test unless l's entries are in (fp, handle) order,
// the invariant every method on Leaf relies on internally (lowerBound's
// binary search is meaningless otherwise).
func assertSorted(t *testing.T, l *Leaf) {
	t.Helper()
	for i := 1; i < l.Len(); i++ {
		prevFP, prevH := l.fps[i-1], l.handles[i-1]
		fp, h := l.fps[i], l.handles[i]
		less := prevFP < fp || (prevFP == fp && prevH < h)
		assert.True(t, less, "entries not strictly increasing at index %d: (%d,%d) then (%d,%d)", i, prevFP, prevH, fp, h)
	}
}

func TestLeaf_NewLeafIsEmpty(t *testing.T) {
	l := NewLeaf()
	assert.Equal(t, 0, l.Len())
	assert.Nil(t, l.Lookup(0))
}

func TestLeaf_InsertIfAbsent_Idempotent(t *testing.T) {
	l := NewLeaf()

	inserted := l.InsertIfAbsent(42, Handle(7))
	assert.True(t, inserted)
	assert.Equal(t, 1, l.Len())

	// Re-inserting the exact same (fp, handle) pair must be a no-op — this
	// is what makes redelivery of the same AddChunk idempotent.
	inserted = l.InsertIfAbsent(42, Handle(7))
	assert.False(t, inserted)
	assert.Equal(t, 1, l.Len())

	// A different handle at the same fp is a genuinely new entry (a
	// legitimate collision or multi-block duplication), not absorbed by
	// insert-if-absent.
	inserted = l.InsertIfAbsent(42, Handle(8))
	assert.True(t, inserted)
	assert.Equal(t, 2, l.Len())
}

func TestLeaf_InsertIfAbsent_MaintainsSortOrderUnderRandomInserts(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	l := NewLeaf()

	const n = 2000
	inserts := 0
	for i := 0; i < n; i++ {
		fp := uint16(rng.Intn(64)) // small fp range forces frequent collisions
		h := Handle(rng.Intn(64))
		if l.InsertIfAbsent(fp, h) {
			inserts++
		}
	}

	assert.Equal(t, inserts, l.Len())
	assertSorted(t, l)
}

func TestLeaf_Lookup_ReturnsAllCollidingHandles(t *testing.T) {
	l := NewLeaf()
	require.True(t, l.InsertIfAbsent(5, Handle(1)))
	require.True(t, l.InsertIfAbsent(5, Handle(2)))
	require.True(t, l.InsertIfAbsent(5, Handle(3)))
	require.True(t, l.InsertIfAbsent(6, Handle(4))) // different fp, must not leak into the fp=5 lookup

	got := l.Lookup(5)
	assert.ElementsMatch(t, []Handle{1, 2, 3}, got, "Lookup must return every handle colliding on fp, not just the first")

	got = l.Lookup(6)
	assert.Equal(t, []Handle{4}, got)
}

func TestLeaf_Lookup_NoMatchReturnsNil(t *testing.T) {
	l := NewLeaf()
	require.True(t, l.InsertIfAbsent(5, Handle(1)))
	require.True(t, l.InsertIfAbsent(9, Handle(2)))

	assert.Nil(t, l.Lookup(7), "a fingerprint between two present entries must miss")
	assert.Nil(t, l.Lookup(0), "a fingerprint before every present entry must miss")
	assert.Nil(t, l.Lookup(65535), "a fingerprint after every present entry must miss")
}

func TestLeaf_Lookup_EmptyLeafAlwaysMisses(t *testing.T) {
	l := NewLeaf()
	assert.Nil(t, l.Lookup(0))
	assert.Nil(t, l.Lookup(65535))
}

func TestLeaf_RemoveWhere_RemovesExactlyMatchingEntriesAndKeepsOrder(t *testing.T) {
	l := NewLeaf()
	// handles 1..10, each at a distinct fp equal to the handle value so the
	// expected surviving order is easy to state.
	for h := 1; h <= 10; h++ {
		require.True(t, l.InsertIfAbsent(uint16(h), Handle(h)))
	}

	// Drop even handles.
	removed := l.RemoveWhere(func(h Handle) bool { return h%2 != 0 })
	assert.Equal(t, 5, removed)
	assert.Equal(t, 5, l.Len())
	assertSorted(t, l)

	var remaining []Handle
	remaining = append(remaining, l.handles...)
	assert.Equal(t, []Handle{1, 3, 5, 7, 9}, remaining, "surviving entries must keep their original relative order")
}

func TestLeaf_RemoveWhere_KeepAllIsNoOp(t *testing.T) {
	l := NewLeaf()
	for h := 1; h <= 5; h++ {
		require.True(t, l.InsertIfAbsent(uint16(h), Handle(h)))
	}

	removed := l.RemoveWhere(func(Handle) bool { return true })
	assert.Equal(t, 0, removed)
	assert.Equal(t, 5, l.Len())
}

func TestLeaf_RemoveWhere_RemoveAllEmptiesTheLeaf(t *testing.T) {
	l := NewLeaf()
	for h := 1; h <= 5; h++ {
		require.True(t, l.InsertIfAbsent(uint16(h), Handle(h)))
	}

	removed := l.RemoveWhere(func(Handle) bool { return false })
	assert.Equal(t, 5, removed)
	assert.Equal(t, 0, l.Len())
	assert.Nil(t, l.Lookup(1))
}

func TestLeaf_Clone_IsTrueDeepCopy(t *testing.T) {
	l := NewLeaf()
	require.True(t, l.InsertIfAbsent(1, Handle(10)))
	require.True(t, l.InsertIfAbsent(2, Handle(20)))

	clone := l.Clone()
	assert.Equal(t, l.Lookup(1), clone.Lookup(1))
	assert.Equal(t, l.Lookup(2), clone.Lookup(2))

	// Mutating the clone must not affect the original.
	clone.InsertIfAbsent(3, Handle(30))
	assert.Equal(t, 3, clone.Len())
	assert.Equal(t, 2, l.Len(), "mutating the clone must not affect the original")
	assert.Nil(t, l.Lookup(3))

	// Mutating the original must not affect the (already-taken) clone.
	l.InsertIfAbsent(4, Handle(40))
	assert.Equal(t, 3, l.Len())
	assert.Equal(t, 3, clone.Len(), "mutating the original after Clone must not affect the clone")
	assert.Nil(t, clone.Lookup(4))
}

func TestLeaf_Clone_OfEmptyLeaf(t *testing.T) {
	l := NewLeaf()
	clone := l.Clone()
	assert.Equal(t, 0, clone.Len())
	clone.InsertIfAbsent(1, Handle(1))
	assert.Equal(t, 0, l.Len())
}

// TestLeaf_AgainstReferenceModel is a randomized model-based test: every
// InsertIfAbsent/RemoveWhere call is mirrored against a plain
// map[[2]uint32]bool ("reference") of (fp, handle) pairs, and after each
// step the leaf's full contents (via Lookup over every fp actually used)
// must match the reference exactly. This exercises interleaved insert/
// remove sequences well beyond the other, more targeted cases above.
func TestLeaf_AgainstReferenceModel(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	l := NewLeaf()
	reference := map[[2]uint32]bool{} // [2]uint32{fp, handle} -> present

	const fpRange = 32
	const handleRange = 8

	for step := 0; step < 500; step++ {
		if rng.Intn(3) == 0 && len(reference) > 0 {
			// Remove a random handle value from every fp it appears at,
			// exactly like the sweep's set-membership predicate.
			targetHandle := Handle(rng.Intn(handleRange))
			l.RemoveWhere(func(h Handle) bool { return h != targetHandle })
			for k := range reference {
				if k[1] == uint32(targetHandle) {
					delete(reference, k)
				}
			}
		} else {
			fp := uint16(rng.Intn(fpRange))
			h := Handle(rng.Intn(handleRange))
			inserted := l.InsertIfAbsent(fp, h)
			key := [2]uint32{uint32(fp), uint32(h)}
			assert.Equal(t, !reference[key], inserted, "InsertIfAbsent's return value must match whether the pair was already present")
			reference[key] = true
		}
	}

	assertSorted(t, l)

	gotByFP := map[uint16][]Handle{}
	for fp := uint16(0); fp < fpRange; fp++ {
		if got := l.Lookup(fp); got != nil {
			gotByFP[fp] = got
		}
	}

	wantByFP := map[uint16][]Handle{}
	for k := range reference {
		fp, h := uint16(k[0]), Handle(k[1])
		wantByFP[fp] = append(wantByFP[fp], h)
	}

	require.Equal(t, len(wantByFP), len(gotByFP))
	for fp, want := range wantByFP {
		got := gotByFP[fp]
		sortHandles(want)
		sortHandles(got)
		assert.Equal(t, want, got, "fp=%d", fp)
	}
}

func sortHandles(hs []Handle) {
	sort.Slice(hs, func(i, j int) bool { return hs[i] < hs[j] })
}
