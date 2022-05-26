package parquetquery

import (
	"bytes"
	"context"
	"testing"

	"github.com/segmentio/parquet-go"
	"github.com/stretchr/testify/require"
)

func TestSubstringPredicate(t *testing.T) {

	// Normal case - all chunks/pages/values inspected
	testPredicate(t, predicateTestCase{
		predicate:  NewSubstringPredicate("b"),
		keptChunks: 1,
		keptPages:  1,
		keptValues: 2,
		writeData: func(w *parquet.Writer) {
			type String struct {
				S string `parquet:",dict"`
			}
			w.Write(&String{"abc"}) // kept
			w.Write(&String{"bcd"}) // kept
			w.Write(&String{"cde"}) // skipped
		},
	})

	// Dictionary in the page header allows for skipping a page
	testPredicate(t, predicateTestCase{
		predicate:  NewSubstringPredicate("x"), // Not present in any values
		keptChunks: 1,
		keptPages:  0,
		keptValues: 0,
		writeData: func(w *parquet.Writer) {
			type dictString struct {
				S string `parquet:",dict"`
			}
			w.Write(&dictString{"abc"})
			w.Write(&dictString{"bcd"})
			w.Write(&dictString{"cde"})
		},
	})
}

type predicateTestCase struct {
	writeData  func(w *parquet.Writer)
	keptChunks int
	keptPages  int
	keptValues int
	predicate  Predicate
}

// testPredicate by writing data and then iterating the column.  The data model
// must contain a single column.
func testPredicate(t *testing.T, tc predicateTestCase) {
	buf := new(bytes.Buffer)
	w := parquet.NewWriter(buf)
	tc.writeData(w)
	w.Flush()
	w.Close()

	file := bytes.NewReader(buf.Bytes())
	r, err := parquet.OpenFile(file, int64(buf.Len()))
	require.NoError(t, err)

	p := InstrumentedPredicate{pred: tc.predicate}

	i := NewColumnIterator(context.TODO(), r.RowGroups(), 0, "test", 100, &p, "")
	for i.Next() != nil {
	}

	require.Equal(t, tc.keptChunks, p.KeptColumnChunks, "keptChunks")
	require.Equal(t, tc.keptPages, p.KeptPages, "keptPages")
	require.Equal(t, tc.keptValues, p.KeptValues, "keptValues")
}

func BenchmarkSubstringPredicate(b *testing.B) {
	v := parquet.ValueOf("abcdefghijklmnopqsrtuvwxyz")
	p := NewSubstringPredicate("JKL")

	for i := 0; i < b.N; i++ {
		p.KeepValue(v)
	}
}
