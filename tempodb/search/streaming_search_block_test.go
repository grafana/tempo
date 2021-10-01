package search

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// newStreamingSearchBlockWithTraces returns (tmpDir path, *StreamingSearchBlock)
func newStreamingSearchBlockWithTraces(t testing.TB, traceCount int, enc backend.Encoding) (string, *StreamingSearchBlock) {
	tmpDir := t.TempDir()

	// create search sub-directory
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "search"), os.ModePerm))

	id := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	searchData := [][]byte{(&tempofb.SearchEntryMutable{
		TraceID: id,
		Tags: tempofb.SearchDataMap{
			"key1": {"value10", "value11"},
			"key2": {"value20", "value21"},
			"key3": {"value30", "value31"},
			"key4": {"value40", "value41"},
		}}).ToBytes()}

	f, err := os.OpenFile(path.Join(tmpDir, "search", fmt.Sprintf("1c505e8b-26cd-4621-ba7d-792bb55282d5:single-tenant:v2:%s:", enc.String())), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	sb, err := NewStreamingSearchBlockForFile(f, "v2", enc)
	require.NoError(t, err)

	for i := 0; i < traceCount; i++ {
		require.NoError(t, sb.Append(context.Background(), id, searchData))
	}

	return tmpDir, sb
}

func TestStreamingSearchBlockReplay(t *testing.T) {
	traceCount := 100

	for _, enc := range backend.SupportedEncoding {
		t.Run(enc.String(), func(t *testing.T) {
			walDir, sb := newStreamingSearchBlockWithTraces(t, traceCount, enc)
			assert.NotNil(t, sb)

			// close the old file handler
			assert.NoError(t, sb.Close())

			// create new block from the same wal file
			blocks, err := RescanBlocks(walDir)
			assert.NoError(t, err)
			require.Len(t, blocks, 1)
			assert.Equal(t, traceCount, len(blocks[0].appender.Records()))

			// search the new block
			p := NewSearchPipeline(&tempopb.SearchRequest{
				Tags: map[string]string{"key1": "value10"},
			})

			sr := NewResults()

			sr.StartWorker()
			go func() {
				defer sr.FinishWorker()
				err := blocks[0].Search(context.TODO(), p, sr)
				require.NoError(t, err)
			}()
			sr.AllWorkersStarted()

			var results []*tempopb.TraceSearchMetadata
			for r := range sr.Results() {
				results = append(results, r)
			}

			require.Equal(t, traceCount, len(results))
		})
	}
}

func TestStreamingSearchBlockSearchBlock(t *testing.T) {
	traceCount := 10
	_, sb := newStreamingSearchBlockWithTraces(t, traceCount, backend.EncNone)

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

	for _, enc := range backend.SupportedEncoding {
		b.Run(enc.String(), func(b *testing.B) {
			_, sb := newStreamingSearchBlockWithTraces(b, b.N, enc)

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
		})
	}
}
