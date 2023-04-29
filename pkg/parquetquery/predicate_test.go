package parquetquery

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Predicate = (*mockPredicate)(nil)

type mockPredicate struct {
	ret         bool
	valCalled   bool
	pageCalled  bool
	chunkCalled bool
}

func newAlwaysTruePredicate() *mockPredicate {
	return &mockPredicate{ret: true}
}

func newAlwaysFalsePredicate() *mockPredicate {
	return &mockPredicate{ret: false}
}

func (p *mockPredicate) String() string                           { return "mockPredicate{}" }
func (p *mockPredicate) KeepValue(parquet.Value) bool             { p.valCalled = true; return p.ret }
func (p *mockPredicate) KeepPage(parquet.Page) bool               { p.pageCalled = true; return p.ret }
func (p *mockPredicate) KeepColumnChunk(parquet.ColumnChunk) bool { p.chunkCalled = true; return p.ret }

type predicateTestCase struct {
	testName   string
	writeData  func(w *parquet.Writer) //nolint:all
	keptChunks int
	keptPages  int
	keptValues int
	predicate  Predicate
}

func TestSubstringPredicate(t *testing.T) {
	testCases := []predicateTestCase{
		{
			testName:   "all chunks/pages/values inspected",
			predicate:  NewSubstringPredicate("b"),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				type String struct {
					S string `parquet:",dict"`
				}
				require.NoError(t, w.Write(&String{"abc"})) // kept
				require.NoError(t, w.Write(&String{"bcd"})) // kept
				require.NoError(t, w.Write(&String{"cde"})) // skipped
			},
		},
		{
			testName:   "dictionary in the page header allows for skipping a page",
			predicate:  NewSubstringPredicate("x"), // Not present in any values
			keptChunks: 1,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				type dictString struct {
					S string `parquet:",dict"`
				}
				require.NoError(t, w.Write(&dictString{"abc"}))
				require.NoError(t, w.Write(&dictString{"abc"}))
				require.NoError(t, w.Write(&dictString{"abc"}))
				require.NoError(t, w.Write(&dictString{"abc"}))
				require.NoError(t, w.Write(&dictString{"abc"}))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.testName, func(t *testing.T) {
			testPredicate(t, tC)
		})
	}
}

func TestNewRegexInPredicate(t *testing.T) {
	testCases := []predicateTestCase{
		{
			testName: "all chunks/pages/values inspected",
			predicate: func() Predicate {
				pred, err := NewRegexInPredicate([]string{"a.*"})
				assert.NoError(t, err)
				return pred
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				type String struct {
					S string `parquet:",dict"`
				}
				require.NoError(t, w.Write(&String{"abc"})) // kept
				require.NoError(t, w.Write(&String{"acd"})) // kept
				require.NoError(t, w.Write(&String{"cde"})) // skipped
			},
		},
		{
			testName: "dictionary in the page header allows for skipping a page",
			predicate: func() Predicate {
				pred, err := NewRegexInPredicate([]string{"x.*"})
				assert.NoError(t, err)
				return pred
			}(), // Not present in any values
			keptChunks: 1,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				type dictString struct {
					S string `parquet:",dict"`
				}
				require.NoError(t, w.Write(&dictString{"abc"}))
				require.NoError(t, w.Write(&dictString{"abc"}))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.testName, func(t *testing.T) {
			testPredicate(t, tC)
		})
	}
}

func TestNewRegexNotInPredicate(t *testing.T) {
	testCases := []predicateTestCase{
		{
			testName: "all chunks/pages/values inspected",
			predicate: func() Predicate {
				pred, err := NewRegexNotInPredicate([]string{"a.*"})
				assert.NoError(t, err)
				return pred
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				type String struct {
					S string `parquet:",dict"`
				}
				require.NoError(t, w.Write(&String{"abc"})) // skipped
				require.NoError(t, w.Write(&String{"acd"})) // skipped
				require.NoError(t, w.Write(&String{"cde"})) // kept
				require.NoError(t, w.Write(&String{"xde"})) // kept
			},
		},
		{
			testName: "dictionary in the page header allows for skipping a page",
			predicate: func() Predicate {
				pred, err := NewRegexNotInPredicate([]string{"x.*"})
				assert.NoError(t, err)
				return pred
			}(), // Not present in any values
			keptChunks: 1,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				type dictString struct {
					S string `parquet:",dict"`
				}
				require.NoError(t, w.Write(&dictString{"xyz"}))
				require.NoError(t, w.Write(&dictString{"xyz"}))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.testName, func(t *testing.T) {
			testPredicate(t, tC)
		})
	}
}

// TestOrPredicateCallsKeepColumnChunk ensures that the OrPredicate calls
// KeepColumnChunk on all of its children. This is important because the
// Dictionary predicates rely on KeepColumnChunk always being called at the
// beginning of a row group to reset their page.
func TestOrPredicateCallsKeepColumnChunk(t *testing.T) {
	tcs := []struct {
		preds []*mockPredicate
	}{
		{},
		{
			preds: []*mockPredicate{
				newAlwaysTruePredicate(),
			},
		},
		{
			preds: []*mockPredicate{
				newAlwaysFalsePredicate(),
			},
		},
		{
			preds: []*mockPredicate{
				newAlwaysFalsePredicate(),
				newAlwaysTruePredicate(),
			},
		},
		{
			preds: []*mockPredicate{
				newAlwaysTruePredicate(),
				newAlwaysFalsePredicate(),
			},
		},
	}

	for _, tc := range tcs {
		preds := make([]Predicate, 0, len(tc.preds)+1)
		for _, pred := range tc.preds {
			preds = append(preds, pred)
		}

		recordPred := &mockPredicate{}
		preds = append(preds, recordPred)

		p := NewOrPredicate(preds...)
		p.KeepColumnChunk(nil)

		for _, pred := range preds {
			require.True(t, pred.(*mockPredicate).chunkCalled)
		}
	}
}

// testPredicate by writing data and then iterating the column.
// The data model must contain a single column.
func testPredicate(t *testing.T, tc predicateTestCase) {
	t.Helper()
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
	for {
		res, err := i.Next()
		require.NoError(t, err)
		if res == nil {
			break
		}
	}

	require.Equal(t, tc.keptChunks, int(p.KeptColumnChunks), "keptChunks")
	require.Equal(t, tc.keptPages, int(p.KeptPages), "keptPages")
	require.Equal(t, tc.keptValues, int(p.KeptValues), "keptValues")
}

func BenchmarkSubstringPredicate(b *testing.B) {
	p := NewSubstringPredicate("abc")

	s := make([]parquet.Value, 1000)
	for i := 0; i < 1000; i++ {
		s[i] = parquet.ValueOf(uuid.New().String())
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, ss := range s {
			p.KeepValue(ss)
		}
	}
}

func BenchmarkStringInPredicate(b *testing.B) {
	p := NewStringInPredicate([]string{"abc"})

	s := make([]parquet.Value, 1000)
	for i := 0; i < 1000; i++ {
		s[i] = parquet.ValueOf(uuid.New().String())
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, ss := range s {
			p.KeepValue(ss)
		}
	}
}
