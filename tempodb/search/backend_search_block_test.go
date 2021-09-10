package search

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/stretchr/testify/require"
)

func genSearchData(traceID []byte, i int) [][]byte {
	return [][]byte{(&tempofb.SearchEntryMutable{
		TraceID: traceID,
		Tags: tempofb.SearchDataMap{
			"key" + strconv.Itoa(i): {"value_A_" + strconv.Itoa(i), "value_B_" + strconv.Itoa(i)},
		}}).ToBytes()}
}

func newBackendSearchBlockWithTraces(t testing.TB, traceCount int, enc backend.Encoding, pageSizeBytes int) *BackendSearchBlock {
	//id := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // 16-byte ids required

	f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	b1, err := NewStreamingSearchBlockForFile(f)
	require.NoError(t, err)

	for i := 0; i < traceCount; i++ {
		id := make([]byte, 16)
		binary.LittleEndian.PutUint32(id, uint32(i))
		require.NoError(t, b1.Append(context.Background(), id, genSearchData(id, i)))
	}

	l, err := local.NewBackend(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	blockID := uuid.New()
	tenantID := "fake"
	err = NewBackendSearchBlock(b1, l, blockID, tenantID, enc, pageSizeBytes)
	require.NoError(t, err)

	b2 := OpenBackendSearchBlock(l, blockID, tenantID)
	return b2
}

func TestBackendSearchBlockSearch(t *testing.T) {
	traceCount := 50_000

	b2 := newBackendSearchBlockWithTraces(t, traceCount, backend.EncNone, 0)

	p := NewSearchPipeline(&tempopb.SearchRequest{
		Tags: map[string]string{"key20": "value_B_20"},
	})

	sr := NewResults()

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
	require.Equal(t, 1, len(results))
	require.Equal(t, traceCount, int(sr.TracesInspected()))
}

func BenchmarkBackendSearchBlockSearch(b *testing.B) {
	pageSizesMB := []float32{0.5, 1, 2}

	for _, enc := range backend.SupportedEncoding {
		for _, sz := range pageSizesMB {
			b.Run(fmt.Sprint(enc.String(), "/", sz, "MiB"), func(b *testing.B) {

				b2 := newBackendSearchBlockWithTraces(b, b.N, enc, int(sz*1024*1024))

				// Use secret tag to perform exhaustive search
				p := NewSearchPipeline(&tempopb.SearchRequest{
					Tags: map[string]string{SecretExhaustiveSearchTag: "!"},
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
			})
		}
	}
}
