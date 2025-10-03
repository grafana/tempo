package registry

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
)

func TestNewCardinalityClampsInputs(t *testing.T) {
	t.Parallel()

	c := NewCardinality(2, 30*time.Second, 15*time.Second)

	if c.precision != 14 {
		t.Fatalf("expected precision to clamp to 14, got %d", c.precision)
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

	c := NewCardinality(precision, staleTime, sketchDuration)

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

func TestCardinalityAdvanceEvictsStaleData(t *testing.T) {
	t.Parallel()

	c := NewCardinality(14, 15*time.Minute, 5*time.Minute)

	for i := 0; i < 1_000; i++ {
		c.Insert(testHashUint64(uint64(i)))
	}

	if got := c.Estimate(); got == 0 {
		t.Fatalf("expected non-zero estimate after inserts, got %d", got)
	}

	for i := 0; i < c.sketchesLength; i++ {
		c.Advance()
	}

	if got := c.Estimate(); got != 0 {
		t.Fatalf("expected estimate to drop to 0 after evicting stale data, got %d", got)
	}
}

func testHashUint64(v uint64) uint64 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	return xxhash.Sum64(buf[:])
}

var benchmarkEstimate uint64

func BenchmarkCardinalityInsert(b *testing.B) {
	c := NewCardinality(14, 15*time.Minute, 5*time.Minute)
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Insert(uint64(i))
	}
}

func BenchmarkCardinalityEstimate(b *testing.B) {
	c := NewCardinality(14, 15*time.Minute, 5*time.Minute)
	for i := 0; i < 1<<16; i++ {
		c.Insert(uint64(i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	var estimate uint64
	for i := 0; i < b.N; i++ {
		estimate = c.Estimate()
	}
	benchmarkEstimate = estimate
}

func BenchmarkCardinalityAdvance(b *testing.B) {
	c := NewCardinality(14, 15*time.Minute, 5*time.Minute)
	for i := 0; i < c.sketchesLength*64; i++ {
		c.Insert(uint64(i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Advance()
	}

	benchmarkEstimate = c.Estimate()
}
