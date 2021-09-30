package search

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func newStreamingSearchBlockWithTraces(traceCount int, t testing.TB) *StreamingSearchBlock {
	id := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	searchData := [][]byte{(&tempofb.SearchEntryMutable{
		TraceID: id,
		Tags: tempofb.SearchDataMap{
			"key1": {"value10", "value11"},
			"key2": {"value20", "value21"},
			"key3": {"value30", "value31"},
			"key4": {"value40", "value41"},
		}}).ToBytes()}

	f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	sb, err := NewStreamingSearchBlockForFile(f)
	require.NoError(t, err)

	for i := 0; i < traceCount; i++ {
		require.NoError(t, sb.Append(context.Background(), id, searchData))
	}

	return sb
}

func TestStreamingSearchBlockReplay(t *testing.T) {
	traceCount := 100
	sb := newStreamingSearchBlockWithTraces(traceCount, t)
	assert.NotNil(t, sb)

	// grab the wal filename from the block and close the old file handler
	walFile := sb.file.Name()
	assert.NoError(t, sb.Close())

	// create new block from the same wal file
	newSearchBlock, warning, err := newStreamingSearchBlockFromWALReplay(walFile)
	assert.NoError(t, warning)
	assert.NoError(t, err)
	assert.Equal(t, traceCount, len(newSearchBlock.appender.Records()))

	// search the new block
	p := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{"key1": "value10"},
	})

	sr := NewResults()

	sr.StartWorker()
	go func() {
		defer sr.FinishWorker()
		err := newSearchBlock.Search(context.TODO(), p, sr)
		require.NoError(t, err)
	}()
	sr.AllWorkersStarted()

	var results []*tempopb.TraceSearchMetadata
	for r := range sr.Results() {
		results = append(results, r)
	}

	require.Equal(t, traceCount, len(results))
}

func TestStreamingSearchBlockSearchBlock(t *testing.T) {
	traceCount := 10
	sb := newStreamingSearchBlockWithTraces(traceCount, t)

	testCases := []struct {
		name                    string
		req                     map[string]string
		expectedResultLen       int
		expectedBlocksInspected int
		expectedTracesInspected int
		expectedBlocksSkipped   int
	}{
		{
			name:                    "matches every trace",
			req:                     map[string]string{"key1": "value10"},
			expectedResultLen:       traceCount,
			expectedBlocksInspected: 1,
			expectedTracesInspected: traceCount,
		},
		{
			name:                  "skips block",
			req:                   map[string]string{"nomatch": "nomatch"},
			expectedBlocksSkipped: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			p := NewSearchPipeline(&tempopb.SearchRequest{
				Tags: tc.req,
			})

			sr := NewResults()

			sr.StartWorker()
			go func() {
				defer sr.FinishWorker()
				err := sb.Search(context.TODO(), p, sr)
				require.NoError(t, err)
			}()
			sr.AllWorkersStarted()

			var results []*tempopb.TraceSearchMetadata
			for r := range sr.Results() {
				results = append(results, r)
			}
			require.Equal(t, tc.expectedResultLen, len(results))
			require.Equal(t, tc.expectedBlocksInspected, int(sr.BlocksInspected()))
			require.Equal(t, tc.expectedTracesInspected, int(sr.TracesInspected()))
			require.Equal(t, tc.expectedBlocksSkipped, int(sr.BlocksSkipped()))
		})
	}
}

func BenchmarkStreamingSearchBlockSearch(b *testing.B) {

	sb := newStreamingSearchBlockWithTraces(b.N, b)

	p := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{"nomatch": "nomatch"},
	})

	sr := NewResults()

	b.ResetTimer()
	start := time.Now()
	// Search 10x10 because reading the search data is much faster than creating it, but we need
	// to spend at least 1 second to satisfy go bench minimum elapsed time requirement.
	loops := 10
	wg := &sync.WaitGroup{}
	for i := 0; i < loops; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				err := sb.Search(context.TODO(), p, sr)
				require.NoError(b, err)
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("StreamingSearchBlock search throughput: %v elapsed %.2f MB = %.2f MiB/s throughput \n",
		elapsed,
		float64(sr.bytesInspected.Load())/(1024*1024),
		float64(sr.bytesInspected.Load())/(elapsed.Seconds())/(1024*1024))
}
