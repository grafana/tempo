package parquetquery

import (
	"testing"

	pq "github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

// byteArrayDict builds a real parquet byte-array dictionary containing the given
// values in order, so tests exercise the same Dictionary implementation used at
// query time.
func byteArrayDict(t testing.TB, values ...string) pq.Dictionary {
	t.Helper()
	dict := pq.ByteArrayType.NewDictionary(0, 0, pq.ByteArrayType.NewValues(nil, nil))
	vals := make([]pq.Value, len(values))
	for i, v := range values {
		vals[i] = pq.ByteArrayValue([]byte(v))
	}
	idx := make([]int32, len(vals))
	dict.Insert(idx, vals)
	return dict
}

func TestByteInPredicateKeepIndexes(t *testing.T) {
	dict := byteArrayDict(t, "alpha", "bravo", "charlie", "delta")

	p := NewStringInPredicate([]string{"bravo", "delta", "missing"})
	dp, ok := p.(DictionaryPredicate)
	require.True(t, ok, "ByteInPredicate should implement DictionaryPredicate")

	require.Equal(t, []bool{false, true, false, true}, dp.KeepIndexes(dict))
}

func TestByteEqualPredicateKeepIndexes(t *testing.T) {
	dict := byteArrayDict(t, "alpha", "bravo", "charlie")

	p := NewByteEqualPredicate([]byte("charlie"))
	dp, ok := any(p).(DictionaryPredicate)
	require.True(t, ok, "ByteEqualPredicate should implement DictionaryPredicate")

	require.Equal(t, []bool{false, false, true}, dp.KeepIndexes(dict))
}

func TestOrPredicateKeepIndexes(t *testing.T) {
	dict := byteArrayDict(t, "alpha", "bravo", "charlie", "delta")

	// OR of exact matches -> union of matching indexes
	p := NewOrPredicate(
		NewByteEqualPredicate([]byte("alpha")),
		NewByteEqualPredicate([]byte("delta")),
	)
	require.Equal(t, []bool{true, false, false, true}, p.KeepIndexes(dict))
}

func TestOrPredicateKeepIndexesUnsupportedChild(t *testing.T) {
	dict := byteArrayDict(t, "alpha", "bravo")

	// A child that is not dictionary-pushable (NOT IN keeps nulls, so it does not
	// implement DictionaryPredicate) disables the optimization, signalled by a nil
	// bitmap so the caller falls back to per-row evaluation.
	p := NewOrPredicate(
		NewByteEqualPredicate([]byte("alpha")),
		NewStringNotInPredicate([]string{"bravo"}),
	)
	require.Nil(t, p.KeepIndexes(dict))
}
