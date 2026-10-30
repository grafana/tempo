package parquetquery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/encoding"
	"github.com/stretchr/testify/require"
)

type seekAttr struct {
	Key    string   `parquet:",dict"`
	Str    *string  `parquet:",optional"`
	Int    *int64   `parquet:",optional"`
	Double *float64 `parquet:",optional"`
	Bool   *bool    `parquet:",optional"`
}

type seekSpan struct {
	Attrs []seekAttr
}

// seekRow has columns of each physical type, required, optional and repeated at one
// or two levels, plain and dictionary-encoded.
type seekRow struct {
	ID      int64
	Flag    bool
	I32     *int32   `parquet:",optional"`
	I64     *int64   `parquet:",optional"`
	U32     *uint32  `parquet:",optional"`
	U64     *uint64  `parquet:",optional"`
	Float   *float32 `parquet:",optional"`
	Double  *float64 `parquet:",optional"`
	Bool    *bool    `parquet:",optional"`
	Str     *string  `parquet:",optional"`
	DictStr *string  `parquet:",dict,optional"`
	DictI64 *int64   `parquet:",dict,optional"`
	Fixed   [4]byte
	Fixed16 [16]byte
	Ints    []int64
	Attrs   []seekAttr
	Spans   []seekSpan
}

const seekTestRows = 6_000

func seekTestRowsData() []seekRow {
	rnd := rand.New(rand.NewPCG(1, 2))
	ptr := func(v string) *string {
		if rnd.IntN(4) == 0 {
			return nil
		}
		return &v
	}
	some := func() bool { return rnd.IntN(4) != 0 }
	words := []string{"", "api", "db", "shop", "web", "a longer value to spread the pages"}
	attr := func() seekAttr {
		a := seekAttr{Key: words[rnd.IntN(len(words))], Str: ptr(words[rnd.IntN(len(words))])}
		if some() {
			v := rnd.Int64() - math.MaxInt64/2
			a.Int = &v
		}
		if some() {
			v := rnd.NormFloat64()
			a.Double = &v
		}
		if some() {
			v := rnd.IntN(2) == 0
			a.Bool = &v
		}
		return a
	}

	rows := make([]seekRow, seekTestRows)
	for i := range rows {
		r := &rows[i]
		r.ID = int64(i)
		r.Flag = rnd.IntN(3) == 0
		if some() {
			v := int32(rnd.Uint32()) // negative half the time
			r.I32 = &v
		}
		if some() {
			v := rnd.Int64() - math.MaxInt64/2
			r.I64 = &v
		}
		if some() {
			v := rnd.Uint32() // above math.MaxInt32 half the time
			r.U32 = &v
		}
		if some() {
			v := rnd.Uint64()
			r.U64 = &v
		}
		if some() {
			v := float32(rnd.NormFloat64())
			r.Float = &v
		}
		if some() {
			v := rnd.NormFloat64()
			r.Double = &v
		}
		if some() {
			v := rnd.IntN(2) == 0
			r.Bool = &v
		}
		if some() {
			v := rnd.Int64N(20) - 10
			r.DictI64 = &v
		}
		r.Str = ptr(fmt.Sprintf("%s-%d", words[rnd.IntN(len(words))], rnd.IntN(50)))
		r.DictStr = ptr(words[rnd.IntN(len(words))])
		r.Fixed = [4]byte{byte(i), byte(i >> 8), 1, 2}
		r.Fixed16[rnd.IntN(16)] = byte(i)
		for range rnd.IntN(4) {
			r.Ints = append(r.Ints, rnd.Int64N(1000))
		}
		for range rnd.IntN(4) {
			r.Attrs = append(r.Attrs, attr())
		}
		for range rnd.IntN(3) {
			s := seekSpan{}
			for range rnd.IntN(3) {
				s.Attrs = append(s.Attrs, attr())
			}
			r.Spans = append(r.Spans, s)
		}
	}
	return rows
}

// seekTestSources returns the same rows written with large pages (a page holds more
// than a long seek), small pages (seeks cross pages), and v1 and v2 data pages.
func seekTestSources(t *testing.T) map[string][]parquet.RowGroup {
	t.Helper()
	rows := seekTestRowsData()
	write := func(opts ...parquet.WriterOption) []parquet.RowGroup {
		buf := new(bytes.Buffer)
		opts = append(opts, parquet.MaxRowsPerRowGroup(seekTestRows/2))
		w := parquet.NewGenericWriter[seekRow](buf, opts...)
		_, err := w.Write(rows)
		require.NoError(t, err)
		require.NoError(t, w.Close())
		f, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)
		require.Len(t, f.RowGroups(), 2)
		return f.RowGroups()
	}
	return map[string][]parquet.RowGroup{
		"large-pages-v2": write(parquet.PageBufferSize(1 << 20)),
		"large-pages-v1": write(parquet.PageBufferSize(1<<20), parquet.DataPageVersion(1)),
		"small-pages-v2": write(parquet.PageBufferSize(2 << 10)),
		"small-pages-v1": write(parquet.PageBufferSize(2<<10), parquet.DataPageVersion(1)),
	}
}

// hashPredicate keeps about a third of the values, chosen by a hash of their content,
// and nulls when keepNull is set.
type hashPredicate struct{ keepNull bool }

func (p hashPredicate) String() string { return fmt.Sprintf("hash(keepNull=%t)", p.keepNull) }

func (p hashPredicate) KeepValue(v parquet.Value) bool {
	if v.IsNull() {
		return p.keepNull
	}
	h := fnv.New64a()
	switch v.Kind() {
	case parquet.ByteArray, parquet.FixedLenByteArray:
		_, _ = h.Write(v.ByteArray())
	default:
		_, _ = fmt.Fprint(h, v.Uint64())
	}
	return h.Sum64()%3 == 0
}

func (p hashPredicate) KeepRange(_, _ parquet.Value) bool { return true }

func (p hashPredicate) SupportsColumnIndexPruning() bool { return false }

// seekResult is a result with its value broken down, so results from different pages
// compare equal.
type seekResult struct {
	rn        RowNumber
	kind      parquet.Kind
	null      bool
	rep, def  int
	column    int
	bits      uint64
	byteArray string
}

func makeSeekResult(res *IteratorResult) seekResult {
	return makeSeekResultOf(res.RowNumber, res.Entries[0].Value)
}

func makeSeekResultOf(rn RowNumber, v parquet.Value) seekResult {
	r := seekResult{
		rn:     rn,
		kind:   v.Kind(),
		null:   v.IsNull(),
		rep:    v.RepetitionLevel(),
		def:    v.DefinitionLevel(),
		column: v.Column(),
	}
	switch {
	case r.null:
	case r.kind == parquet.ByteArray || r.kind == parquet.FixedLenByteArray:
		r.byteArray = string(v.ByteArray())
	default:
		r.bits = v.Uint64()
	}
	return r
}

// seekStep is a call to Next, or to SeekTo a row, or a row's child at definition
// level 1.
type seekStep struct {
	next   bool
	row    int32 // the target row, or -1 to seek stride rows past the last result
	stride int32
	child  int32 // -1 seeks to the row itself
}

// seekPlans returns sequences of steps, each to run on a new iterator: seeks of every
// length from the last result, some to a row's child and some replaced by Next; seeks
// around the first row of each page; and seeks to the first and last rows and past
// the end.
func seekPlans(seed uint64, repeated bool, pageStarts []int32) [][]seekStep {
	rnd := rand.New(rand.NewPCG(seed, 7))
	strides := []int32{1, 2, 3, 40, 999, 1_000, 1_001, 1_500, 2_500}
	var plans [][]seekStep
	for range 8 {
		var plan []seekStep
		for range 40 {
			step := seekStep{row: -1, stride: strides[rnd.IntN(len(strides))], child: -1}
			switch rnd.IntN(10) {
			case 0:
				step.next = true
			case 1:
				if repeated {
					step.child = int32(rnd.IntN(3))
				}
			}
			plan = append(plan, step)
		}
		plans = append(plans, plan)
	}

	var plan []seekStep
	for _, start := range pageStarts {
		for _, row := range []int32{start - 1, start, start + 1} {
			plan = append(plan, seekStep{row: max(row, 0), child: -1})
		}
	}
	plans = append(plans, plan)

	return append(plans,
		[]seekStep{{row: 0, child: -1}, {row: seekTestRows - 1, child: -1}, {row: seekTestRows, child: -1}},
		[]seekStep{{row: seekTestRows + 5_000, child: -1}},
	)
}

// pageStarts returns the first row of every page of column, from the offset index.
func pageStarts(t *testing.T, rgs []parquet.RowGroup, column int) []int32 {
	t.Helper()
	var starts []int32
	var base int64
	for _, rg := range rgs {
		if oi, err := rg.ColumnChunks()[column].OffsetIndex(); err == nil {
			for i := range oi.NumPages() {
				starts = append(starts, int32(base+oi.FirstRowIndex(i)))
			}
		}
		base += rg.NumRows()
	}
	return starts
}

// seekTarget returns the row number and definition level step seeks to, with last the
// root row of the previous result.
func seekTarget(step seekStep, last int32) (RowNumber, int) {
	to := EmptyRowNumber()
	to[0] = step.row
	if step.row < 0 {
		to[0] = last + step.stride
	}
	if step.child >= 0 {
		to[1] = step.child
		return to, 1
	}
	return to, 0
}

// runSeekSteps runs steps on it until it is exhausted, and returns each result, with a
// null row number for the end.
func runSeekSteps(t *testing.T, it *SyncIterator, steps []seekStep) []seekResult {
	t.Helper()
	var results []seekResult
	last := int32(0)
	for i, step := range steps {
		var res *IteratorResult
		var err error
		if step.next {
			res, err = it.Next()
		} else {
			res, err = it.SeekTo(seekTarget(step, last))
		}
		require.NoError(t, err, "step %d", i)
		if res == nil {
			return append(results, seekResult{rn: EmptyRowNumber()})
		}
		results = append(results, makeSeekResult(res))
		last = res.RowNumber[0]
	}
	return results
}

// oracleSlot is a value of a column with its row number, as the page's value reader
// reads it.
type oracleSlot struct {
	rn    RowNumber
	value parquet.Value
}

// valueReaderSlots reads every value of column through the pages' value readers
// (pq.Page.Values), which SyncIterator doesn't use, numbering rows as SyncIterator does.
func valueReaderSlots(t *testing.T, rgs []parquet.RowGroup, column, maxDefinitionLevel int) []oracleSlot {
	t.Helper()
	var slots []oracleSlot
	rn := EmptyRowNumber()
	buf := make([]parquet.Value, 100)
	for _, rg := range rgs {
		pages := rg.ColumnChunks()[column].Pages()
		for {
			pg, err := pages.ReadPage()
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			values := pg.Values()
			for {
				n, err := values.ReadValues(buf)
				for _, v := range buf[:n] {
					rn.Next(v.RepetitionLevel(), v.DefinitionLevel(), maxDefinitionLevel)
					slots = append(slots, oracleSlot{rn: rn, value: v.Clone()})
				}
				if errors.Is(err, io.EOF) || n == 0 {
					break
				}
				require.NoError(t, err)
			}
			parquet.Release(pg)
		}
		require.NoError(t, pages.Close())
	}
	return slots
}

// runOracle runs steps on slots the way SyncIterator runs them: Next returns the next
// value the filter and then the sampler keep, and SeekTo drops the values before its
// target first, without asking either.
func runOracle(slots []oracleSlot, steps []seekStep, filter Predicate, sampler Sampler) []seekResult {
	var results []seekResult
	pos, last := 0, int32(0)
	for _, step := range steps {
		if !step.next {
			to, definitionLevel := seekTarget(step, last)
			for pos < len(slots) && CompareRowNumbers(definitionLevel, slots[pos].rn, to) < 0 {
				pos++
			}
		}
		found := false
		for pos < len(slots) && !found {
			s := slots[pos]
			pos++
			if filter != nil && !filter.KeepValue(s.value) {
				continue
			}
			if sampler != nil && !sampler.Sample() {
				continue
			}
			results = append(results, makeSeekResultOf(s.rn, s.value))
			last = s.rn[0]
			found = true
		}
		if !found {
			return append(results, seekResult{rn: EmptyRowNumber()})
		}
	}
	return results
}

// TestPageReaderMatchesValueReader: reading and seeking through pageReader returns
// exactly the values the pages' own value readers read, for every column kind, with
// and without a filter or sampler, across page boundaries and past the end.
func TestPageReaderMatchesValueReader(t *testing.T) {
	columns := []string{
		"ID", "Flag", "I32", "I64", "U32", "U64", "Float", "Double", "Bool", "Str", "DictStr", "DictI64",
		"Fixed", "Fixed16", "Ints", "Attrs.Key", "Attrs.Str", "Attrs.Int", "Attrs.Double", "Attrs.Bool",
		"Spans.Attrs.Str", "Spans.Attrs.Int", "Spans.Attrs.Bool",
	}
	filters := []struct {
		name    string
		filter  func() Predicate
		sampled bool
	}{
		{"nofilter", func() Predicate { return nil }, false},
		{"hash", func() Predicate { return hashPredicate{} }, false},
		{"hash-nulls", func() Predicate { return hashPredicate{keepNull: true} }, false},
		{"skipnils", func() Predicate { return NewSkipNilsPredicate() }, false},
		{"hash-sampled", func() Predicate { return hashPredicate{keepNull: true} }, true},
	}

	schema := parquet.SchemaOf(seekRow{})
	for name, rgs := range seekTestSources(t) {
		encodings := map[bool]bool{} // dictionary-encoded or plain pages seen
		for _, col := range columns {
			leaf, ok := schema.Lookup(strings.Split(col, ".")...)
			require.True(t, ok, col)
			starts := pageStarts(t, rgs, leaf.ColumnIndex)
			slots := valueReaderSlots(t, rgs, leaf.ColumnIndex, leaf.MaxDefinitionLevel)
			dict := dictionaryEncoded(t, rgs, leaf.ColumnIndex)
			require.Equal(t, strings.HasPrefix(col, "Dict") || col == "Attrs.Key", dict, "%s/%s encoding", name, col)
			encodings[dict] = true
			for fi, f := range filters {
				t.Run(fmt.Sprintf("%s/%s/%s", name, col, f.name), func(t *testing.T) {
					for pi, steps := range seekPlans(uint64(leaf.ColumnIndex*10+fi), leaf.MaxRepetitionLevel > 0, starts) {
						opts := []SyncIteratorOpt{
							SyncIteratorOptSelectAs("v"), SyncIteratorOptMaxDefinitionLevel(leaf.MaxDefinitionLevel),
							SyncIteratorOptPredicate(f.filter()),
						}
						var gotSampler, wantSampler Sampler
						if f.sampled {
							gotSampler, wantSampler = &testSampler{keepEvery: 2}, &testSampler{keepEvery: 2}
							opts = append(opts, SyncIteratorOptSampler(gotSampler))
						}
						it := NewSyncIterator(context.TODO(), rgs, leaf.ColumnIndex, opts...)
						got := runSeekSteps(t, it, steps)
						it.Close()
						want := runOracle(slots, steps, f.filter(), wantSampler)
						require.Equal(t, want, got, "plan %d", pi)
						if f.sampled {
							require.Equal(t, wantSampler.(*testSampler).seen, gotSampler.(*testSampler).seen, "plan %d", pi)
						}
					}
				})
			}
		}
		require.True(t, encodings[true] && encodings[false], "%s: both dictionary-encoded and plain pages", name)
	}
}

// TestPageReaderMatchesScan: seeking returns the first value at or after the target
// that a scan by Next returns.
func TestPageReaderMatchesScan(t *testing.T) {
	rgs := seekTestSources(t)["large-pages-v2"]
	schema := parquet.SchemaOf(seekRow{})
	for _, path := range []string{"I64", "Bool", "Str", "DictStr", "Attrs.Key", "Attrs.Int", "Spans.Attrs.Str"} {
		for _, filter := range []Predicate{nil, hashPredicate{}, hashPredicate{keepNull: true}} {
			t.Run(fmt.Sprintf("%s/%v", path, filter), func(t *testing.T) {
				leaf, ok := schema.Lookup(strings.Split(path, ".")...)
				require.True(t, ok)
				newIter := func() *SyncIterator {
					return NewSyncIterator(context.TODO(), rgs, leaf.ColumnIndex, SyncIteratorOptSelectAs("v"),
						SyncIteratorOptMaxDefinitionLevel(leaf.MaxDefinitionLevel), SyncIteratorOptPredicate(filter))
				}

				scan := newIter()
				defer scan.Close()
				var all []seekResult
				for {
					res, err := scan.Next()
					require.NoError(t, err)
					if res == nil {
						break
					}
					all = append(all, makeSeekResult(res))
				}

				it := newIter()
				defer it.Close()
				pos := 0
				for _, to := range []int32{0, 1, 1_002, 1_003, 2_500, 2_999, 3_000, 3_001, 4_500, 5_999, 6_000} {
					rn := EmptyRowNumber()
					rn[0] = to
					res, err := it.SeekTo(rn, 0)
					require.NoError(t, err)
					for pos < len(all) && all[pos].rn[0] < to {
						pos++
					}
					if pos == len(all) {
						require.Nil(t, res, "seek to %d", to)
						continue
					}
					require.NotNil(t, res, "seek to %d", to)
					require.Equal(t, all[pos], makeSeekResult(res), "seek to %d", to)
					pos++
				}
			})
		}
	}
}

// TestNilSyncIteratorSeekTo: a seek returns the first row at or after the target that
// lacks the key, as a scan does (see TestNilIterator), within a page and across pages.
func TestNilSyncIteratorSeekTo(t *testing.T) {
	const count = 30_000
	pf := createNilIterTestFile(t, count)
	idx, _, _ := GetColumnIndexByPath(pf, "attrs.list.element.Key")
	it := NewNilSyncIterator(context.TODO(), pf.RowGroups(), idx, SyncIteratorOptSelectAs("v"),
		SyncIteratorOptPredicate(NewIncludeNilStringEqualPredicate([]byte("special-key"))), SyncIteratorOptMaxDefinitionLevel(1))
	defer it.Close()

	// Every 3rd row has the key, so the others are returned.
	for _, to := range []int32{1, 2, 3, 2_000, 2_001, 9_000, 14_999, 15_000, 29_998, 29_999} {
		rn := EmptyRowNumber()
		rn[0] = to
		res, err := it.SeekTo(rn, 0)
		require.NoError(t, err)
		want := to
		if want%3 == 0 {
			want++
		}
		require.NotNil(t, res, "seek to %d", to)
		require.Equal(t, RowNumber{want, -1, -1, -1, -1, -1, -1, -1}, TruncateRowNumber(0, res.RowNumber), "seek to %d", to)
	}
	res, err := it.Next()
	require.NoError(t, err)
	require.Nil(t, res)
}

// TestPageReaderLoad: load reads a page only with the column's max definition level.
func TestPageReaderLoad(t *testing.T) {
	rgs := seekTestSources(t)["large-pages-v2"]
	leaf, ok := parquet.SchemaOf(seekRow{}).Lookup("ID")
	require.True(t, ok)
	pages := rgs[0].ColumnChunks()[leaf.ColumnIndex].Pages()
	defer pages.Close()
	pg, err := pages.ReadPage()
	require.NoError(t, err)
	defer parquet.Release(pg)

	r := pageReader{maxDef: 1}
	require.ErrorContains(t, r.load(pg), "max definition level is 1")
	require.NoError(t, r.setChunk(rgs[0].ColumnChunks()[leaf.ColumnIndex]))
	require.NoError(t, r.load(pg))
	require.Equal(t, int(pg.NumValues()), r.numSlots)
}

// TestSkipToRow compares skipToRow with a slot by slot walk.
func TestSkipToRow(t *testing.T) {
	rnd := rand.New(rand.NewPCG(3, 4))
	for _, repeated := range []bool{false, true} {
		for range 200 {
			numSlots := rnd.IntN(2_000)
			var defLevels, repLevels []byte
			const maxDef = 2
			for i := range numSlots {
				defLevels = append(defLevels, byte(rnd.IntN(maxDef+1)))
				if repeated {
					rep := byte(0)
					if i > 0 && rnd.IntN(5) != 0 {
						rep = byte(1 + rnd.IntN(2))
					}
					repLevels = append(repLevels, rep)
				}
			}
			slot := rnd.IntN(numSlots + 1)
			cur := int32(-1) // the row before the page
			index := 0
			for s := range slot {
				if !repeated || repLevels[s] == 0 {
					cur++
				}
				if defLevels[s] == maxDef {
					index++
				}
			}
			to := cur + int32(rnd.IntN(300))

			// Walk slot by slot to the first slot of row to.
			wantSlot, wantIndex, row := -1, index, cur
			for s := slot; s < numSlots; s++ {
				if !repeated || repLevels[s] == 0 {
					if row++; row == to {
						wantSlot = s
						break
					}
				}
				if defLevels[s] == maxDef {
					wantIndex++
				}
			}

			r := pageReader{maxDef: maxDef, defLevels: defLevels, repLevels: repLevels, numSlots: numSlots, slot: slot, index: index}
			moved := r.skipToRow(cur, to)
			if wantSlot < 0 || to <= cur+1 { // not on the page, or the next row: left to next
				require.False(t, moved)
				require.Equal(t, slot, r.slot)
				require.Equal(t, index, r.index)
				continue
			}
			require.True(t, moved, "slot %d cur %d to %d", slot, cur, to)
			require.Equal(t, wantSlot, r.slot)
			require.Equal(t, wantIndex, r.index)
		}
	}
}

// TestCheckValueCount: a page must hold one value (or dictionary index) per present slot.
func TestCheckValueCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		data encoding.Values
		want int
		ok   bool
	}{
		{"int64", encoding.Int64Values([]int64{1, 2, 3}), 3, true},
		{"int64 short", encoding.Int64Values([]int64{1, 2}), 3, false},
		{"int32 indexes long", encoding.Int32Values([]int32{0, 1, 0, 1}), 3, false},
		{"bytes", encoding.ByteArrayValues([]byte("abc"), []uint32{0, 1, 3}), 2, true},
		{"bytes short", encoding.ByteArrayValues([]byte("abc"), []uint32{0, 3}), 2, false},
		{"fixed", encoding.FixedLenByteArrayValues([]byte("abcd"), 2), 2, true},
		{"fixed ragged", encoding.FixedLenByteArrayValues([]byte("abcde"), 2), 2, false},
		{"bool last byte", encoding.BooleanValues([]byte{0xff, 0x01}), 9, true},
		{"bool extra byte", encoding.BooleanValues([]byte{0xff, 0x01, 0x00}), 9, false},
		{"bool short", encoding.BooleanValues([]byte{0xff}), 9, false},
	} {
		err := checkValueCount(tc.data, tc.want)
		if tc.ok {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}

// dictionaryEncoded reports whether every page of column is dictionary-encoded; it fails
// the test if only some are.
func dictionaryEncoded(t *testing.T, rgs []parquet.RowGroup, column int) bool {
	t.Helper()
	dict, plain := 0, 0
	for _, rg := range rgs {
		pages := rg.ColumnChunks()[column].Pages()
		for {
			pg, err := pages.ReadPage()
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			if pg.Dictionary() != nil {
				dict++
			} else {
				plain++
			}
			parquet.Release(pg)
		}
		require.NoError(t, pages.Close())
	}
	require.True(t, dict == 0 || plain == 0, "column %d mixes %d dictionary and %d plain pages", column, dict, plain)
	return dict > 0
}
