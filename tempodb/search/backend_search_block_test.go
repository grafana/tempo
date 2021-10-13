package search

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/google/uuid"
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
		Tags: tempofb.SearchDataMap{
			"key" + strconv.Itoa(i): {"value_A_" + strconv.Itoa(i), "value_B_" + strconv.Itoa(i)},
		}}).ToBytes()}
}

func newBackendSearchBlockWithTraces(t testing.TB, traceCount int, enc backend.Encoding, pageSizeBytes int) *BackendSearchBlock {
	f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	b1, err := NewStreamingSearchBlockForFile(f, bufio.NewWriter(f), "v2", enc)
	require.NoError(t, err)

	for i := 0; i < traceCount; i++ {
		id := make([]byte, 16)
		binary.LittleEndian.PutUint32(id, uint32(i))
		require.NoError(t, b1.Append(context.Background(), id, genSearchData(id, i)))
	}
	err = b1.FlushBuffer()
	require.NoError(t, err)

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
	traceCount := 10_000

	for _, enc := range backend.SupportedEncoding {
		t.Run(enc.String(), func(t *testing.T) {

			b2 := newBackendSearchBlockWithTraces(t, traceCount, enc, 0)

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
		})
	}
}

func TestBackendSearchBlockFinalSize(t *testing.T) {
	traceCount := 10000
	pageSizesMB := []float32{1}

	f, err := os.OpenFile(path.Join(t.TempDir(), "searchdata"), os.O_CREATE|os.O_RDWR, 0644)
	require.NoError(t, err)

	b1, err := NewStreamingSearchBlockForFile(f, bufio.NewWriter(f), "v2", backend.EncNone)
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
