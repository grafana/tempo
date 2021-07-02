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
	"github.com/stretchr/testify/require"
)

func BenchmarkBackendSearchBlockSearch(b *testing.B) {
	ctx := context.TODO()
	//n := 1_000_000

	id := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	searchData := [][]byte{tempofb.SearchDataBytesFromValues(id, tempofb.SearchDataMap{
		"key1": {"value10", "value11"},
		"key2": {"value20", "value21"},
		"key3": {"value30", "value31"},
		"key4": {"value40", "value41"},
	}, 0, 0)}

	f, err := os.OpenFile(path.Join(b.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(b, err)

	b1, err := NewStreamingSearchBlockForFile(f)
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		b1.Append(ctx, id, searchData)
	}

	r, w, _, err := local.New(&local.Config{
		Path: b.TempDir(),
	})
	require.NoError(b, err)

	blockID := uuid.New()
	tenantID := "fake"
	bytesFlushed, err := NewBackendSearchBlock(b1, w, blockID, tenantID)
	require.NoError(b, err)

	b2 := OpenBackendSearchBlock(r, blockID, tenantID)

	p := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{"nomatch": "nomatch"},
	})

	b.ResetTimer()
	start := time.Now()
	//for i := 0; i < b.N; i++ {
	_, err = b2.Search(ctx, p)
	require.NoError(b, err)
	//}
	elapsed := time.Since(start)
	fmt.Println("BackendSearchBlock search throughput:", float64(bytesFlushed)/(elapsed.Seconds())/(1024*1024), "MiB/s", elapsed.Seconds(), "s elapsed", "bytesFlushed:", float64(bytesFlushed)/(1024*1024), "MB", b.N, "records")
}
