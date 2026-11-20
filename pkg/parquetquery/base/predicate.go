package base

import (
	"fmt"

	pq "github.com/parquet-go/parquet-go"
)

// Predicate is a pushdown predicate evaluated at the value level (KeepValue) and,
// for chunk/page skipping, at the value-range level (KeepRange). Chunk-, page-,
// dictionary-, and null-level skipping are handled generically by the iterator
// layer (parquetquery's keepColumnChunk / keepPage) using these two methods.
type Predicate interface {
	fmt.Stringer

	KeepValue(pq.Value) bool
	// KeepRange reports whether any value in the inclusive [minV,maxV] range could match.
	KeepRange(minV, maxV pq.Value) bool
	// SupportsColumnIndexPruning reports whether a chunk's column index can prune for this
	// predicate: KeepRange depends on the bounds, or only nulls are kept. When it can't,
	// keepColumnChunk doesn't read the index, which is a separate backend request, and
	// calls KeepRange with null bounds instead.
	SupportsColumnIndexPruning() bool
}

// Sampler thins values during iteration. It is intentionally narrower than
// traceql.Sampler so this package takes no dependency on traceql.
type Sampler interface {
	// Expect reports an accepted page's value count: the population to sample from.
	Expect(count uint64)
	Sample() bool
}

// PredicateStats counts the chunk/page/value decisions the iterator makes while
// applying its predicate. Read them with parquetquery.SyncIterator.Stats; Close exports them as
// metrics.
type PredicateStats struct {
	InspectedColumnChunks int64
	KeptColumnChunks      int64
	InspectedPages        int64
	KeptPages             int64
	InspectedValues       int64
	KeptValues            int64
}
