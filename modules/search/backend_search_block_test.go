package search

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkBackendSearchBlockSearch(b *testing.B) {
	ctx := context.TODO()
	//n := 1_000_000

	id := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	searchData := [][]byte{(&tempofb.SearchDataMutable{
		Tags: tempofb.SearchDataMap{
			"key1": {"value10", "value11"},
			"key2": {"value20", "value21"},
			"key3": {"value30", "value31"},
			"key4": {"value40", "value41"},
		}}).ToBytes()}

	f, err := os.OpenFile(path.Join(b.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(b, err)

	b1, err := NewStreamingSearchBlockForFile(f)
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		assert.NoError(b, b1.Append(ctx, id, searchData))
	}

	l, err := local.NewBackend(&local.Config{
		Path: b.TempDir(),
	})
	require.NoError(b, err)

	blockID := uuid.New()
	tenantID := "fake"
	bytesFlushed, err := NewBackendSearchBlock(b1, l, blockID, tenantID)
	require.NoError(b, err)

	b2 := OpenBackendSearchBlock(l, blockID, tenantID)

	p := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{"nomatch": "nomatch"},
	})

	b.ResetTimer()
	start := time.Now()
	// Search 10x because this is really fast but creating the test data is slow
	// and it helps the benchmark reach consensus faster.
	loops := 10
	for i := 0; i < loops; i++ {
		_, err = b2.Search(ctx, p)
		require.NoError(b, err)
	}
	elapsed := time.Since(start)

	fmt.Printf("BackendSearchBlock search throughput: %v elapsed %.2f MB = %.2f MiB/s throughput \n",
		elapsed,
		float64(bytesFlushed*loops)/(1024*1024),
		float64(bytesFlushed*loops)/(elapsed.Seconds())/(1024*1024))
}
