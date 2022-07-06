package search

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/stretchr/testify/require"
)

const testTenantID = "fake"

func genSearchData(traceID []byte, i int) [][]byte {
	return [][]byte{(&tempofb.SearchEntryMutable{
		TraceID: traceID,
		Tags: tempofb.NewSearchDataMapWithData(map[string][]string{
			"key" + strconv.Itoa(i): {"value_A_" + strconv.Itoa(i), "value_B_" + strconv.Itoa(i)},
		})}).ToBytes()}
}

func newBackendSearchBlockWithTraces(t testing.TB, traceCount int, enc backend.Encoding, pageSizeBytes int) *BackendSearchBlock {
	f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	b1, err := NewStreamingSearchBlockForFile(f, uuid.New(), enc)
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
	err = NewBackendSearchBlock(b1, backend.NewWriter(l), blockID, testTenantID, enc, pageSizeBytes)
	require.NoError(t, err)

	b2 := OpenBackendSearchBlock(blockID, testTenantID, backend.NewReader(l))
	return b2
}

func TestBackendSearchBlockSearch(t *testing.T) {
	ctx := context.Background()

	for _, enc := range backend.SupportedEncoding {
		t.Run(enc.String(), func(t *testing.T) {

			id, wantTr, _, _, meta, searchesThatMatch, searchesThatDontMatch, tags, tagValues := trace.SearchTestSuite()

			// Create backend search block with the test trace
			data := trace.ExtractSearchData(wantTr, id, func(s string) bool { return true })

			f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
			require.NoError(t, err)

			b1, err := NewStreamingSearchBlockForFile(f, uuid.New(), enc)
			require.NoError(t, err)

			require.NoError(t, b1.Append(ctx, id, [][]byte{data}))

			l, err := local.NewBackend(&local.Config{
				Path: t.TempDir(),
			})
			require.NoError(t, err)

			blockID := uuid.New()
			err = NewBackendSearchBlock(b1, backend.NewWriter(l), blockID, testTenantID, enc, 0)
			require.NoError(t, err)

			b2 := OpenBackendSearchBlock(blockID, testTenantID, backend.NewReader(l))

			// Perform test suite

			for _, req := range searchesThatMatch {
				resp := search(t, b2, req)
				require.Equal(t, 1, len(resp.Traces), "search request:", req)
				require.Equal(t, meta, resp.Traces[0])
			}

			for _, req := range searchesThatDontMatch {
				resp := search(t, b2, req)
				require.Equal(t, 0, len(resp.Traces), "search request:", req)
			}

			var gotTags []string
			b2.Tags(ctx, func(k string) { gotTags = append(gotTags, k) })
			sort.Strings(gotTags)
			require.Equal(t, tags, gotTags)

			for k, v := range tagValues {
				var gotValues []string
				b1.TagValues(ctx, k, func(s string) { gotValues = append(gotValues, s) })
				sort.Strings(gotValues)
				require.Equal(t, v, gotValues)
			}
		})
	}
}

func search(t *testing.T, block SearchableBlock, req *tempopb.SearchRequest) *tempopb.SearchResponse {
	p := NewSearchPipeline(req)

	sr := NewResults()

	sr.StartWorker()
	go func() {
		defer sr.FinishWorker()
		err := block.Search(context.TODO(), p, sr)
		require.NoError(t, err)
	}()
	sr.AllWorkersStarted()

	resp := &tempopb.SearchResponse{}
	for r := range sr.Results() {
		resp.Traces = append(resp.Traces, r)
	}
	return resp
}

func TestBackendSearchBlockFinalSize(t *testing.T) {
	traceCount := 10000
	pageSizesMB := []float32{1}

	f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	b1, err := NewStreamingSearchBlockForFile(f, uuid.New(), backend.EncNone)
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

	for _, enc := range backend.SupportedEncoding {
		for _, sz := range pageSizesMB {

			err := NewBackendSearchBlock(b1, backend.NewWriter(l), blockID, testTenantID, enc, int(sz*1024*1024))
			require.NoError(t, err)

			_, len, err := l.Read(context.TODO(), "search", backend.KeyPathForBlock(blockID, testTenantID), false)
			require.NoError(t, err)

			fmt.Printf("BackendSearchBlock/%s/%.1fMiB, %d traces = %d bytes, %.2f bytes per trace \n", enc.String(), sz, traceCount, len, float32(len)/float32(traceCount))

		}
	}
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
