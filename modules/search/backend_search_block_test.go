package search

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBackendSearchBlockWithTraces(traceCount int, t testing.TB) *BackendSearchBlock {
	id := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} // 16-byte ids required
	searchData := [][]byte{(&tempofb.SearchDataMutable{
		Tags: tempofb.SearchDataMap{
			"key1": {"value10", "value11"},
			"key2": {"value20", "value21"},
			"key3": {"value30", "value31"},
			"key4": {"value40", "value41"},
		}}).ToBytes()}

	f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	b1, err := NewStreamingSearchBlockForFile(f)
	require.NoError(t, err)
	for i := 0; i < traceCount; i++ {
		assert.NoError(t, b1.Append(context.Background(), id, searchData))
	}

	l, err := local.NewBackend(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	blockID := uuid.New()
	tenantID := "fake"
	err = NewBackendSearchBlock(b1, l, blockID, tenantID, backend.EncNone)
	require.NoError(t, err)

	b2 := OpenBackendSearchBlock(l, blockID, tenantID)
	return b2
}

func TestBackendSearchBlockSearch(t *testing.T) {
	traceCount := 50_000

	b2 := newBackendSearchBlockWithTraces(traceCount, t)

	// Matches every trace
	p := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{"key1": "value10"},
	})

	sr := NewSearchResults()

	sr.StartWorker()
	go func() {
		defer sr.FinishWorker()
		err := b2.Search(context.TODO(), p, sr)
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

func BenchmarkBackendSearchBlockSearch(b *testing.B) {
	b2 := newBackendSearchBlockWithTraces(b.N, b)

	// Matches nothing, will perform an exhaustive search.
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
				err := b2.Search(context.TODO(), p, sr)
				require.NoError(b, err)
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("BackendSearchBlock search throughput: %v elapsed %.2f MB = %.2f MiB/s \t %d traces = %.2fM traces/s \n",
		elapsed,
		float64(sr.bytesInspected.Load())/(1024*1024),
		float64(sr.bytesInspected.Load())/(elapsed.Seconds())/(1024*1024),
		sr.TracesInspected(),
		float64(sr.TracesInspected())/(elapsed.Seconds())/1_000_000,
	)
}
