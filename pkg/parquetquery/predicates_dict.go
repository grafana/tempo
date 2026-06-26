package parquetquery

import pq "github.com/parquet-go/parquet-go"

// DictionaryPredicate is an optional interface implemented by predicates that can
// be evaluated directly against a column-chunk dictionary.
//
// When a column chunk is dictionary-encoded the iterator resolves the predicate
// once over the (small) set of distinct dictionary values into a per-index keep
// bitmap, then matches each row by its integer dictionary index. This avoids
// materializing the value and running a byte comparison for every row, which
// dominates CPU for exact-match string filters (service.name, span:name, etc.).
//
// Implementations must only advertise support when their KeepValue semantics over
// present (non-null) values fully capture matching - i.e. the predicate does not
// depend on null-ness beyond what KeepValue already returns for a present value.
type DictionaryPredicate interface {
	Predicate

	// KeepIndexes resolves the predicate against dict and returns a bitmap of
	// length dict.Len() where entry i is true iff dict.Index(i) satisfies the
	// predicate. Returning nil signals that the predicate cannot be pushed down
	// to the dictionary and the caller must fall back to per-row evaluation.
	KeepIndexes(dict pq.Dictionary) []bool
}

// dictionaryKeepIndexes builds the keep bitmap by evaluating keep against every
// distinct dictionary value once. The dictionary is small relative to the number
// of rows, so this is paid once per column chunk rather than once per row.
func dictionaryKeepIndexes(dict pq.Dictionary, keep func(pq.Value) bool) []bool {
	n := dict.Len()
	out := make([]bool, n)
	for i := 0; i < n; i++ {
		out[i] = keep(dict.Index(int32(i)))
	}
	return out
}

func (p *ByteInPredicate) KeepIndexes(dict pq.Dictionary) []bool {
	return dictionaryKeepIndexes(dict, p.KeepValue)
}

func (p ByteEqualPredicate) KeepIndexes(dict pq.Dictionary) []bool {
	return dictionaryKeepIndexes(dict, p.KeepValue)
}

// regex and substring matching are pure functions of the value and reject nulls,
// so they resolve against the dictionary just like exact matches. This is an even
// bigger win for them because the per-row cost (regex / bytes.Contains) is far
// higher than a byte comparison.
func (p *regexPredicate) KeepIndexes(dict pq.Dictionary) []bool {
	return dictionaryKeepIndexes(dict, p.KeepValue)
}

func (p *SubstringPredicate) KeepIndexes(dict pq.Dictionary) []bool {
	return dictionaryKeepIndexes(dict, p.KeepValue)
}

// KeepIndexes supports dictionary pushdown only when every child is itself
// dictionary-pushable. A nil child means "match all" and a non-dictionary child
// (e.g. regex on a value the dictionary doesn't narrow) would require per-row
// evaluation, so in both cases we return nil to disable the optimization.
func (p *OrPredicate) KeepIndexes(dict pq.Dictionary) []bool {
	for _, child := range p.preds {
		if child == nil {
			return nil
		}
		if _, ok := child.(DictionaryPredicate); !ok {
			return nil
		}
	}
	return dictionaryKeepIndexes(dict, p.KeepValue)
}
