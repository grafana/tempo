package vparquet5

import (
	"math/rand"
	"testing"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestCloneSpanForBatchCopiesArrayValues(t *testing.T) {
	ints := []int{1, 2}
	floats := []float64{1.5, 2.5}
	strings := []string{"one", "two"}
	bools := []bool{true, false}
	intAttribute := traceql.NewAttribute("int-array")
	floatAttribute := traceql.NewAttribute("float-array")
	stringAttribute := traceql.NewAttribute("string-array")
	boolAttribute := traceql.NewAttribute("bool-array")
	snapshot := cloneSpanForBatch(&span{spanAttrs: []attrVal{
		{a: intAttribute, s: traceql.NewStaticIntArray(ints)},
		{a: floatAttribute, s: traceql.NewStaticFloatArray(floats)},
		{a: stringAttribute, s: traceql.NewStaticStringArray(strings)},
		{a: boolAttribute, s: traceql.NewStaticBooleanArray(bools)},
	}})
	defer putSpan(snapshot)

	ints[0] = 99
	floats[0] = 99.5
	strings[0] = "changed"
	bools[0] = false

	got, ok := snapshot.AttributeFor(intAttribute)
	require.True(t, ok)
	actualInts, ok := got.IntArray()
	require.True(t, ok)
	require.Equal(t, []int{1, 2}, actualInts)

	got, ok = snapshot.AttributeFor(floatAttribute)
	require.True(t, ok)
	actualFloats, ok := got.FloatArray()
	require.True(t, ok)
	require.Equal(t, []float64{1.5, 2.5}, actualFloats)

	got, ok = snapshot.AttributeFor(stringAttribute)
	require.True(t, ok)
	actualStrings, ok := got.StringArray()
	require.True(t, ok)
	require.Equal(t, []string{"one", "two"}, actualStrings)

	got, ok = snapshot.AttributeFor(boolAttribute)
	require.True(t, ok)
	actualBools, ok := got.BooleanArray()
	require.True(t, ok)
	require.Equal(t, []bool{true, false}, actualBools)
}

func TestCloneStaticForBatchReusesArrayBuffer(t *testing.T) {
	dst := &span{}

	first := cloneStaticForBatch(dst, traceql.NewStaticIntArray([]int{1, 2}))
	firstValues, ok := first.IntArray()
	require.True(t, ok)
	require.Equal(t, []int{1, 2}, firstValues)
	require.Len(t, dst.batchArrayBuffers, 1)

	source := traceql.NewStaticIntArray([]int{3, 4})
	dst.batchArrayBuffers = resetBatchArrayBuffers(dst.batchArrayBuffers)
	second := cloneStaticForBatch(dst, source)
	secondValues, ok := second.IntArray()
	require.True(t, ok)
	require.Equal(t, []int{3, 4}, secondValues)
	require.True(t, &firstValues[0] == &secondValues[0], "released batch buffers must be reused")

	allocs := testing.AllocsPerRun(100, func() {
		dst.batchArrayBuffers = resetBatchArrayBuffers(dst.batchArrayBuffers)
		if cloned := cloneStaticForBatch(dst, source); cloned.Type != traceql.TypeIntArray {
			panic("expected integer array")
		}
	})
	require.Zero(t, allocs)
}

func TestSearchFetchSpansOnly(t *testing.T) {
	var (
		ctx          = t.Context()
		numTraces    = 250
		traces       = make([]*Trace, 0, numTraces)
		wantTraceIdx = rand.Intn(numTraces)
		wantTraceID  = test.ValidTraceID(nil)
		traceIDText  = util.TraceIDToHexString(wantTraceID)
	)

	for i := 0; i < numTraces; i++ {
		if i == wantTraceIdx {
			traces = append(traces, fullyPopulatedTestTrace(wantTraceID))
			continue
		}

		id := test.ValidTraceID(nil)
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTrace(1, id), nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)

	for _, tc := range searchesThatMatch(t, traceIDText) {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.FetchSpans(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "search request:%v", req)
			defer resp.Results.Close()

			found := false
			for {
				span, err := resp.Results.Next(ctx)
				require.NoError(t, err, "search request:%v", req)
				if span == nil {
					break
				}

				// Ensure that every attribute returned is present in the list of conditions.
				span.AllAttributesFunc(func(a traceql.Attribute, _ traceql.Static) {
					if !req.HasAttribute(a) {
						t.Errorf("attribute %v not found in conditions", a)
					}
				})

				traceID, ok := span.AttributeFor(traceql.IntrinsicTraceIDAttribute)
				if !ok {
					continue
				}
				traceIDString := traceID.EncodeToString(false)
				found = (traceIDString == traceIDText)
				if found {
					break
				}
			}
			require.True(t, found, "search request:%v", req)
		})
	}

	for _, tc := range searchesThatDontMatch(t) {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.FetchSpans(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "search request:%v", req)
			defer resp.Results.Close()

			for {
				span, err := resp.Results.Next(ctx)
				require.NoError(t, err, "search request:%v", req)
				if span == nil {
					break
				}
				traceID, ok := span.AttributeFor(traceql.IntrinsicTraceIDAttribute)
				if !ok {
					continue
				}
				traceIDString := traceID.EncodeToString(false)
				require.NotEqual(t, traceIDText, traceIDString, "search request:%v", req)
			}
		})
	}
}

func TestSelectAllFetchSpansOnly(t *testing.T) {
	var (
		ctx             = t.Context()
		numTraces       = 250
		traces          = make([]*Trace, 0, numTraces)
		wantTraceIdx    = rand.Intn(numTraces)
		wantTraceID     = test.ValidTraceID(nil)
		wantTraceIDText = util.TraceIDToHexString(wantTraceID)
		wantTrace       = fullyPopulatedTestTrace(wantTraceID)
		dc              = test.MakeDedicatedColumns()
		dcm             = dedicatedColumnsToColumnMapping(dc)
		opts            = common.DefaultSearchOptions()
	)

	// TODO - This strips unsupported attributes types for now. Revisit when
	// add support for arrays/kvlists in the fetch layer.
	trimForSelectAll(wantTrace)

	for i := 0; i < numTraces; i++ {
		if i == wantTraceIdx {
			traces = append(traces, wantTrace)
			continue
		}

		id := test.ValidTraceID(nil)
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTrace(1, id), nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)

	_, _, eval, req, err := traceql.Compile("{}")
	require.NoError(t, err)

	req.SecondPass = func(inSS *traceql.Spanset) ([]*traceql.Spanset, error) { return eval([]*traceql.Spanset{inSS}) }
	req.SecondPassSelectAll = true

	resp, err := b.FetchSpans(ctx, *req, opts)
	require.NoError(t, err)
	defer resp.Results.Close()

	// This is a dump of all spans in the fully-populated test trace
	// Since this fetch is spans only, we compare one by one.
	// Spans are returned in the same order as they were written.
	wantSS := flattenForSelectAll(wantTrace, dcm)
	found := false

	for {
		// Seek to our desired trace
		sp, err := resp.Results.Next(ctx)
		require.NoError(t, err)
		if sp == nil {
			break
		}
		tid, ok := sp.AttributeFor(traceql.IntrinsicTraceIDAttribute)
		require.True(t, ok, "trace id attribute missing")
		if tid.EncodeToString(false) != wantTraceIDText {
			continue
		}

		found = true

		// Cleanup found data for comparison
		// equal will fail on the rownum mismatches. this is an internal detail to the
		// fetch layer. just wipe them out here
		gotSp := sp.(*span)
		gotSp.cbSpanset = nil
		gotSp.cbSpansetFinal = false

		rn := gotSp.rowNum
		gotSp.rowNum = pq.RowNumber{}

		gotSp.startTimeUnixNanos = 0 // selectall doesn't imply start time
		sortAttrs(gotSp.traceAttrs)
		sortAttrs(gotSp.resourceAttrs)
		sortAttrs(gotSp.spanAttrs)
		sortAttrs(gotSp.instrumentationAttrs)

		// Pop next wanted span from the spanset.
		wantSp := wantSS.Spans[0].(*span)
		wantSS.Spans = wantSS.Spans[1:]

		require.Equal(t, wantSp, gotSp)

		// Restore row number because we are mucking with internal state.
		// This is special
		gotSp.rowNum = rn
	}
	require.True(t, found, "trace was found")
}
