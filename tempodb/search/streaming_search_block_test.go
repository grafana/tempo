package search

import (
	"context"
	"fmt"
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
	searchData := [][]byte{(&tempofb.SearchDataMutable{
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

func TestStreamingSearchBlockSearch(t *testing.T) {
	traceCount := 10

	sb := newStreamingSearchBlockWithTraces(traceCount, t)

	// Matches every trace
	p := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{"key1": "value10"},
	})

	sr := NewSearchResults()

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
	require.Equal(t, traceCount, len(results))
	require.Equal(t, traceCount, int(sr.TracesInspected()))
}

func BenchmarkStreamingSearchBlockSearch(b *testing.B) {

	sb := newStreamingSearchBlockWithTraces(b.N, b)

	p := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{"nomatch": "nomatch"},
	})

	sr := NewSearchResults()

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
