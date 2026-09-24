package parquetquery

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

type testInt struct {
	N int64 `parquet:",dict"`
}

type mockPredicate struct {
	ret         bool
	valCalled   bool
	pageCalled  bool
	chunkCalled bool
}

type testDictString struct {
	S string `parquet:",dict"`
}

var _ Predicate = (*mockPredicate)(nil)

func newAlwaysTruePredicate() *mockPredicate {
	return &mockPredicate{ret: true}
}

func newAlwaysFalsePredicate() *mockPredicate {
	return &mockPredicate{ret: false}
}

func (p *mockPredicate) String() string               { return "mockPredicate{}" }
func (p *mockPredicate) KeepValue(parquet.Value) bool { p.valCalled = true; return p.ret }

func (p *mockPredicate) KeepPage(parquet.Page) bool { p.pageCalled = true; return p.ret }

func (p *mockPredicate) KeepColumnChunk(*ColumnChunkHelper) bool { p.chunkCalled = true; return p.ret }

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

				require.NoError(t, w.Write(&testDictString{"abc"})) // kept
				require.NoError(t, w.Write(&testDictString{"bcd"})) // kept
				require.NoError(t, w.Write(&testDictString{"cde"})) // skipped
			},
		},
		{
			testName:   "dictionary in the page header allows for skipping a page",
			predicate:  NewSubstringPredicate("x"), // Not present in any values
			keptChunks: 0,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"abc"}))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.testName, func(t *testing.T) {
			testPredicate(t, tC)
		})
	}
}

func TestNewStringInPredicate(t *testing.T) {
	testCases := []predicateTestCase{
		{
			testName: "all chunks/pages/values inspected",
			predicate: func() Predicate {
				return NewStringInPredicate([]string{"abc", "acd"})
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"})) // kept
				require.NoError(t, w.Write(&testDictString{"acd"})) // kept
				require.NoError(t, w.Write(&testDictString{"cde"})) // skipped
			},
		},
		{
			testName: "dictionary in the page header allows for skipping a column chunk",
			predicate: func() Predicate {
				return NewStringInPredicate([]string{"x"})
			}(), // Not present in any values
			keptChunks: 0,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"abc"}))
			},
		},
		{
			testName: "map path (>= byteInMapThreshold needles)",
			predicate: func() Predicate {
				// >= byteInMapThreshold needles forces the map-backed KeepValue path
				return NewStringInPredicate([]string{"abc", "acd", "n0", "n1", "n2", "n3", "n4", "n5"})
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"})) // kept
				require.NoError(t, w.Write(&testDictString{"acd"})) // kept
				require.NoError(t, w.Write(&testDictString{"cde"})) // skipped
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
				require.NoError(t, err)

				return pred
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"})) // kept
				require.NoError(t, w.Write(&testDictString{"acd"})) // kept
				require.NoError(t, w.Write(&testDictString{"cde"})) // skipped
			},
		},
		{
			testName: "dictionary in the page header allows for skipping a column chunk",
			predicate: func() Predicate {
				pred, err := NewRegexInPredicate([]string{"x.*"})
				require.NoError(t, err)

				return pred
			}(), // Not present in any values
			keptChunks: 0,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"abc"}))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.testName, func(t *testing.T) {
			testPredicate(t, tC)
		})
	}
}

func TestNewStringNotInPredicate(t *testing.T) {
	testCases := []predicateTestCase{
		{
			testName: "all chunks/pages/values inspected",
			predicate: func() Predicate {
				return NewStringNotInPredicate([]string{"abc", "acd"})
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"})) // skipped
				require.NoError(t, w.Write(&testDictString{"acd"})) // skipped
				require.NoError(t, w.Write(&testDictString{"cde"})) // kept
				require.NoError(t, w.Write(&testDictString{"xde"})) // kept
			},
		},
		{
			testName: "dictionary in the page header allows for skipping a column chunk",
			predicate: func() Predicate {
				return NewStringNotInPredicate([]string{"xyz"})
			}(), // All values excluded
			keptChunks: 0,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"xyz"}))
				require.NoError(t, w.Write(&testDictString{"xyz"}))
			},
		},
		{
			testName: "map path (>= byteInMapThreshold needles)",
			predicate: func() Predicate {
				// >= byteInMapThreshold needles forces the map-backed KeepValue path
				return NewStringNotInPredicate([]string{"abc", "acd", "n0", "n1", "n2", "n3", "n4", "n5"})
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"})) // skipped
				require.NoError(t, w.Write(&testDictString{"acd"})) // skipped
				require.NoError(t, w.Write(&testDictString{"cde"})) // kept
				require.NoError(t, w.Write(&testDictString{"xde"})) // kept
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
				require.NoError(t, err)

				return pred
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"})) // skipped
				require.NoError(t, w.Write(&testDictString{"acd"})) // skipped
				require.NoError(t, w.Write(&testDictString{"cde"})) // kept
				require.NoError(t, w.Write(&testDictString{"xde"})) // kept
			},
		},
		{
			testName: "dictionary in the page header allows for skipping a column chunk",
			predicate: func() Predicate {
				pred, err := NewRegexNotInPredicate([]string{"x.*"})
				require.NoError(t, err)

				return pred
			}(), // Not present in any values
			keptChunks: 0,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"xyz"}))
				require.NoError(t, w.Write(&testDictString{"xyz"}))
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

	p := InstrumentedPredicate{Pred: tc.predicate}

	i := NewSyncIterator(context.TODO(), r.RowGroups(), 0, SyncIteratorOptPredicate(&p))
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

func BenchmarkRegexInPredicate(b *testing.B) {
	p, err := NewRegexInPredicate([]string{"abc"})
	require.NoError(b, err)

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

func TestIntInPredicate(t *testing.T) {
	testCases := []predicateTestCase{
		{
			testName: "all chunks/pages/values inspected",
			predicate: func() Predicate {
				var p Predicate = NewIntInPredicate([]int64{1, 3})
				return p
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testInt{1}))  // kept
				require.NoError(t, w.Write(&testInt{2}))  // skipped
				require.NoError(t, w.Write(&testInt{3}))  // kept
				require.NoError(t, w.Write(&testInt{10})) // skipped
			},
		},
		{
			testName: "dictionary allows skipping a column chunk when no ints match",
			predicate: func() Predicate {
				var p Predicate = NewIntInPredicate([]int64{0, 4, 100})
				return p
			}(),
			keptChunks: 0,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testInt{1}))
				require.NoError(t, w.Write(&testInt{2}))
				require.NoError(t, w.Write(&testInt{3}))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.testName, func(t *testing.T) {
			testPredicate(t, tC)
		})
	}
}

func TestIntNotInPredicate(t *testing.T) {
	testCases := []predicateTestCase{
		{
			testName: "all chunks/pages/values inspected",
			predicate: func() Predicate {
				var p Predicate = NewIntNotInPredicate([]int64{1, 3})
				return p
			}(),
			keptChunks: 1,
			keptPages:  1,
			keptValues: 2,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testInt{1})) // skipped
				require.NoError(t, w.Write(&testInt{2})) // kept
				require.NoError(t, w.Write(&testInt{3})) // skipped
				require.NoError(t, w.Write(&testInt{4})) // kept
			},
		},
		{
			testName: "dictionary allows skipping a column chunk when all ints excluded",
			predicate: func() Predicate {
				var p Predicate = NewIntNotInPredicate([]int64{7, 8})
				return p
			}(),
			keptChunks: 0,
			keptPages:  0,
			keptValues: 0,
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testInt{7}))
				require.NoError(t, w.Write(&testInt{8}))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.testName, func(t *testing.T) {
			testPredicate(t, tC)
		})
	}
}

const benchByteInLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"

func randStringN(r *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = benchByteInLetters[r.Intn(len(benchByteInLetters))]
	}
	return string(b)
}

// benchByteInData builds the predicate needles plus a 1000-value workload that
// is ~50% hits and ~50% misses, so both the matching and non-matching paths are
// exercised.
func benchByteInData(numNeedles, keyLen int) (needles []string, values []parquet.Value) {
	r := rand.New(rand.NewSource(int64(numNeedles*1000 + keyLen)))

	needles = make([]string, numNeedles)
	for i := range needles {
		needles[i] = randStringN(r, keyLen)
	}

	values = make([]parquet.Value, 1000)
	for i := range values {
		if i%2 == 0 {
			values[i] = parquet.ValueOf(needles[i%numNeedles])
		} else {
			values[i] = parquet.ValueOf(randStringN(r, keyLen))
		}
	}
	return needles, values
}

// BenchmarkByteInPredicate exercises the production constructor, which picks the
// slice or map implementation based on byteInMapThreshold.
func BenchmarkByteInPredicate(b *testing.B) {
	for _, numNeedles := range []int{1, 4, 8, 16, 64, 256} {
		for _, keyLen := range []int{8, 36} {
			needles, values := benchByteInData(numNeedles, keyLen)
			p := NewStringInPredicate(needles)

			b.Run(fmt.Sprintf("needles=%d/keyLen=%d", numNeedles, keyLen), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, v := range values {
						p.KeepValue(v)
					}
				}
			})
		}
	}
}

func BenchmarkByteNotInPredicate(b *testing.B) {
	for _, numNeedles := range []int{1, 4, 8, 16, 64, 256} {
		for _, keyLen := range []int{8, 36} {
			needles, values := benchByteInData(numNeedles, keyLen)
			p := NewStringNotInPredicate(needles)

			b.Run(fmt.Sprintf("needles=%d/keyLen=%d", numNeedles, keyLen), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, v := range values {
						p.KeepValue(v)
					}
				}
			})
		}
	}
}
