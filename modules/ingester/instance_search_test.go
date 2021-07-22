package ingester

import (
	"bytes"
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checkEqual(t *testing.T, ids [][]byte, sr *tempopb.SearchResponse) {
	for _, meta := range sr.Traces {
		parsedTraceID, err := util.HexStringToTraceID(meta.TraceID)
		assert.NoError(t, err)

		present := false
		for _, id := range ids {
			if bytes.Equal(parsedTraceID, id) {
				present = true
			}
		}
		assert.True(t, present)
	}
}

func TestInstanceSearch(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	i, err := newInstance("fake", limiter, ingester.store, ingester.local)
	assert.NoError(t, err, "unexpected error creating new instance")

	numTraces := 500
	searchAnnotatedFractionDenominator := 100
	ids := [][]byte{}

	// add dummy search data
	var tagKey = "foo"
	var tagValue = "bar"

	for j := 0; j < numTraces; j++ {
		id := make([]byte, 16)
		rand.Read(id)

		trace := test.MakeTrace(10, id)
		model.SortTrace(trace)
		traceBytes, err := trace.Marshal()
		require.NoError(t, err)

		// annotate just a fraction of traces with search data
		var searchData []byte
		if j%searchAnnotatedFractionDenominator == 0 {
			data := &tempofb.SearchDataMutable{}
			data.TraceID = id
			data.AddTag(tagKey, tagValue)
			searchData = data.ToBytes()

			// these are the only ids we want to test against
			ids = append(ids, id)
		}

		// searchData will be nil if not
		err = i.PushBytes(context.Background(), id, traceBytes, searchData)
		require.NoError(t, err)

		assert.Equal(t, int(i.traceCount.Load()), len(i.traces))
	}

	var req = &tempopb.SearchRequest{
		Tags: map[string]string{},
	}
	req.Tags[tagKey] = tagValue

	sr, err := i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, numTraces/searchAnnotatedFractionDenominator)
	// todo: test that returned results are in sorted time order, create order of id's beforehand
	checkEqual(t, ids, sr)

	// Test after appending to WAL
	err = i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	assert.Equal(t, int(i.traceCount.Load()), len(i.traces))

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, numTraces/searchAnnotatedFractionDenominator)
	checkEqual(t, ids, sr)

	// Test after cutting new headblock
	blockID, err := i.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)
	assert.NotEqual(t, blockID, uuid.Nil)

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, numTraces/searchAnnotatedFractionDenominator)
	checkEqual(t, ids, sr)

	// Test after completing a block
	err = i.CompleteBlock(blockID)
	require.NoError(t, err)

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, numTraces/searchAnnotatedFractionDenominator)
	checkEqual(t, ids, sr)

	err = ingester.stopping(nil)
	require.NoError(t, err)

	// create new ingester.  this should replay wal!
	ingester, _, _ = defaultIngester(t, tempDir)

	i, ok := ingester.getInstanceByID("fake")
	assert.True(t, ok)

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	// note: search is experimental and removed on every startup. Verify no search results now
	assert.Len(t, sr.Traces, 0)
}

func TestInstanceSearchNoData(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	i, err := newInstance("fake", limiter, ingester.store, ingester.local)
	assert.NoError(t, err, "unexpected error creating new instance")

	var req = &tempopb.SearchRequest{
		Tags: map[string]string{},
	}

	sr, err := i.Search(context.Background(), req)
	assert.NoError(t, err)
	require.Len(t, sr.Traces, 0)
}
