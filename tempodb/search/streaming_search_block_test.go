package search

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// newStreamingSearchBlockWithTraces returns (tmpDir path, *StreamingSearchBlock)
func newStreamingSearchBlockWithTraces(t testing.TB, traceCount int, enc backend.Encoding) (string, *StreamingSearchBlock) {
	tmpDir := t.TempDir()

	// create search sub-directory
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "search"), os.ModePerm))

	f, err := os.OpenFile(path.Join(tmpDir, "search", fmt.Sprintf("1c505e8b-26cd-4621-ba7d-792bb55282d5:single-tenant:v2:%s:", enc.String())), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	sb, err := NewStreamingSearchBlockForFile(f, uuid.New(), "v2", enc)
	require.NoError(t, err)

	for i := 0; i < traceCount; i++ {
		id := []byte{0, 0, 0, 0, 0, 0, 0, 0}

		// ensure unique ids
		binary.LittleEndian.PutUint32(id, uint32(i))

		searchData := [][]byte{(&tempofb.SearchEntryMutable{
			TraceID: id,
			Tags: tempofb.SearchDataMap1{
				"key1": {"value10", "value11"},
				"key2": {"value20", "value21"},
				"key3": {"value30", "value31"},
				"key4": {"value40", "value41"},
			}}).ToBytes()}

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

func TestStreamingSearchBlockIteratorDedupes(t *testing.T) {
	traceCount := 1_000

	testCases := []struct {
		name                 string
		searchDataGenerator  func(traceID []byte, i int) [][]byte
		searchTags           map[string]string
		expectedLenResults   int
		expectedLenInspected int
	}{
		{
			name:                 "distinct traces",
			searchDataGenerator:  genSearchData,
			searchTags:           map[string]string{"key10": "value_A_10", "key20": "value_B_20"},
			expectedLenResults:   1,
			expectedLenInspected: 1,
		},
		{
			name: "empty traces",
			searchDataGenerator: func(traceID []byte, i int) [][]byte {
				return [][]byte{}
			},
			searchTags:           map[string]string{"key10": "value_A_10"},
			expectedLenResults:   0,
			expectedLenInspected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
			require.NoError(t, err)

			b1, err := NewStreamingSearchBlockForFile(f, uuid.New(), "v2", backend.EncNone)
			require.NoError(t, err)

			id := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}
			for i := 0; i < traceCount; i++ {
				require.NoError(t, b1.Append(context.Background(), id, tc.searchDataGenerator(id, i)))
			}

			iter, err := b1.Iterator()
			require.NoError(t, err)

			var results []common.ID
			for {
				id, _, err := iter.Next(context.TODO())
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				results = append(results, id)
			}

			require.Equal(t, tc.expectedLenResults, len(results))
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
