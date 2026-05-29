package registry

import (
	"encoding/binary"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/require"
)

func TestNewCardinalityClampsInputs(t *testing.T) {
	t.Parallel()

	c := NewCardinality(30*time.Second, 15*time.Second)

	if c.precision != 10 {
		t.Fatalf("expected precision to default to 10, got %d", c.precision)
	}

	if c.sketchDuration != 5*time.Minute {
		t.Fatalf("expected sketchDuration to default to 5m, got %s", c.sketchDuration)
	}

	if c.sketchesLength != 4 {
		t.Fatalf("expected sketchesLength to be 4, got %d", c.sketchesLength)
	}
}

func TestNewCardinalityUsesValidParameters(t *testing.T) {
	t.Parallel()

	const (
		precision      = uint8(10)
		staleTime      = 30 * time.Minute
		sketchDuration = 6 * time.Minute
	)

	c := NewCardinality(staleTime, sketchDuration)

	if c.precision != precision {
		t.Fatalf("expected precision %d, got %d", precision, c.precision)
	}

	if c.sketchDuration != sketchDuration {
		t.Fatalf("expected sketchDuration %s, got %s", sketchDuration, c.sketchDuration)
	}

	expectedSketches := int((staleTime + sketchDuration) / sketchDuration)
	if c.sketchesLength != expectedSketches {
		t.Fatalf("expected sketchesLength %d, got %d", expectedSketches, c.sketchesLength)
	}
}

func TestCardinalityEstimateAccuracy(t *testing.T) {
	t.Parallel()

	c := NewCardinality(15*time.Minute, 5*time.Minute)
	const inserts = 100_000

	for i := 0; i < inserts; i++ {
		c.Insert(testHashUint64(uint64(i)))
	}

	estimate := c.Estimate()
	actual := float64(inserts)
	diff := math.Abs(float64(estimate)-actual) / actual

	if diff > 0.05 {
		t.Fatalf("estimate error too large: got %d, want about %d (diff %.2f%%)", estimate, inserts, diff*100)
	}
}

func TestCardinalityAdvanceEvictsStaleData(t *testing.T) {
	t.Parallel()

	c := NewCardinality(15*time.Minute, 5*time.Minute)

	for i := 0; i < 1_000; i++ {
		c.Insert(testHashUint64(uint64(i)))
	}

	if got := c.Estimate(); got == 0 {
		t.Fatalf("expected non-zero estimate after inserts, got %d", got)
	}

	for i := 0; i < c.sketchesLength; i++ {
		c.mu.Lock()
		c.lastAdvance = c.lastAdvance.Add(-c.sketchDuration)
		c.mu.Unlock()
		c.Advance()
	}

	if got := c.Estimate(); got != 0 {
		t.Fatalf("expected estimate to drop to 0 after evicting stale data, got %d", got)
	}
}

func TestCardinalityAdvanceForcedStep(t *testing.T) {
	t.Parallel()

	c := NewCardinality(15*time.Minute, 5*time.Minute)

	base := time.Now().Add(-c.sketchDuration / 2)

	c.mu.Lock()
	initialCurrent := c.current
	c.lastAdvance = base
	c.mu.Unlock()

	c.Advance()

	c.mu.RLock()
	defer c.mu.RUnlock()

	expectedCurrent := (initialCurrent + 1) % c.sketchesLength
	if c.current != expectedCurrent {
		t.Fatalf("expected current index %d, got %d", expectedCurrent, c.current)
	}

	advanced := c.lastAdvance.Sub(base)
	if advanced != c.sketchDuration {
		t.Fatalf("expected lastAdvance to move by %s, got %s", c.sketchDuration, advanced)
	}
}

func TestCardinalityAdvanceCapsSteps(t *testing.T) {
	t.Parallel()

	c := NewCardinality(15*time.Minute, 5*time.Minute)

	deltaSkips := c.sketchesLength + 5
	base := time.Now().Add(-time.Duration(deltaSkips) * c.sketchDuration)

	c.mu.Lock()
	c.current = 1
	c.lastAdvance = base
	c.mu.Unlock()

	c.Advance()

	c.mu.RLock()
	defer c.mu.RUnlock()

	expectedAdvance := time.Duration(c.sketchesLength) * c.sketchDuration
	advanced := c.lastAdvance.Sub(base)
	if advanced != expectedAdvance {
		t.Fatalf("expected lastAdvance to move by %s, got %s", expectedAdvance, advanced)
	}

	if c.current != 1 {
		t.Fatalf("expected current index to wrap back to 1, got %d", c.current)
	}
}

func TestCardinalityConcurrentInsertEstimate(t *testing.T) {
	t.Parallel()

	const (
		writers       = 8
		perWriter     = 2_000
		estimateIters = 1_000
	)

	c := NewCardinality(15*time.Minute, 5*time.Minute)

	var writerWG sync.WaitGroup
	for w := 0; w < writers; w++ {
		writerWG.Add(1)
		w := w
		go func() {
			defer writerWG.Done()
			base := uint64(w * perWriter)
			for i := 0; i < perWriter; i++ {
				c.Insert(testHashUint64(base + uint64(i)))
			}
		}()
	}

	var readerWG sync.WaitGroup
	const readers = 4
	for r := 0; r < readers; r++ {
		readerWG.Add(1)
		go func() {
			defer readerWG.Done()
			for i := 0; i < estimateIters; i++ {
				_ = c.Estimate()
			}
		}()
	}

	writerWG.Wait()
	readerWG.Wait()

	want := float64(writers * perWriter)
	got := float64(c.Estimate())
	diff := math.Abs(got-want) / want
	if diff > 0.05 {
		t.Fatalf("estimate error too large: got %.0f, want %.0f (diff %.2f%%)", got, want, diff*100)
	}
}

func testHashUint64(v uint64) uint64 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	return xxhash.Sum64(buf[:])
}

func BenchmarkCardinalityInsert(b *testing.B) {
	c := NewCardinality(15*time.Minute, 5*time.Minute)
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Insert(uint64(i))
	}
}

func BenchmarkCardinalityEstimate(b *testing.B) {
	c := NewCardinality(15*time.Minute, 5*time.Minute)
	for i := 0; i < 1<<16; i++ {
		c.Insert(uint64(i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Estimate()
	}
}

func BenchmarkCardinalityAdvance(b *testing.B) {
	c := NewCardinality(15*time.Minute, 5*time.Minute)
	for i := 0; i < c.sketchesLength*64; i++ {
		c.Insert(uint64(i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Advance()
	}
}

// TestCardinalityCacheInvalidatedByInsert pins the cache-invalidation
// contract on the Insert path: after Insert bumps the internal version,
// the next Estimate must observe the new value rather than serve the
// cached estimate computed before the Insert.
//
// A regression that drops the conditional `version++` in Insert (e.g.,
// always bumps, never bumps, or wrong condition) would leave Estimate
// returning stale data. TestCardinalityEstimateAccuracy / Concurrent /
// Advance* only check the final value within a 5% bound, which can
// silently mask cache-staleness.
func TestCardinalityCacheInvalidatedByInsert(t *testing.T) {
	c := NewCardinality(15*time.Minute, 5*time.Minute)
	const insertsBeforeCache = 100

	// Seed an initial population and force the cache to be populated.
	for i := 0; i < insertsBeforeCache; i++ {
		c.Insert(testHashUint64(uint64(i)))
	}
	firstEstimate := c.Estimate()
	require.NotZero(t, firstEstimate, "initial estimate must be non-zero")

	// Add many more distinct values. The cache must invalidate so the next
	// Estimate recomputes — observing the new values, not the stale cache.
	const additional = 1_000
	for i := 0; i < additional; i++ {
		c.Insert(testHashUint64(uint64(insertsBeforeCache + i)))
	}
	secondEstimate := c.Estimate()
	require.Greater(t, secondEstimate, firstEstimate,
		"Estimate after additional Inserts must observe new entries, not the cached value (got first=%d second=%d)",
		firstEstimate, secondEstimate)
}
