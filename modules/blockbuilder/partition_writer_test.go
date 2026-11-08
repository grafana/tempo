package blockbuilder

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/wal"
)

func getPartitionWriter(t *testing.T, concurrency int) *writer {
	var (
		logger        = log.NewNopLogger()
		blockCfg      = BlockConfig{}
		tmpDir        = t.TempDir()
		partition     = uint64(1)
		startOffset   = uint64(1)
		startTime     = time.Now()
		cycleDuration = time.Minute
		slackDuration = time.Minute
	)
	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
		IngestionSlack: 3 * time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)

	return newPartitionSectionWriter(logger, partition, startOffset, startTime, cycleDuration, slackDuration, concurrency, blockCfg, &mockOverrides{}, w, encoding.DefaultEncoding())
}

func TestPushBytes(t *testing.T) {
	pw := getPartitionWriter(t, 4)

	tenant := "test-tenant"
	traceID := generateTraceID(t)
	now := time.Now()
	startTime := uint64(now.UnixNano())
	endTime := uint64(now.Add(time.Second).UnixNano())
	req := test.MakePushBytesRequest(t, 1, traceID, startTime, endTime)

	err := pw.pushBytes(now, tenant, req)
	require.NoError(t, err)
}

func TestWriterFlushConcurrency(t *testing.T) {
	for _, concurrency := range []int{1, 2, 4, 6} {
		t.Run(strconv.Itoa(concurrency), func(t *testing.T) {
			pw := getPartitionWriter(t, concurrency)
			now := time.Now()
			for i := 0; i < 10; i++ {
				req := test.MakePushBytesRequest(t, 1, generateTraceID(t), uint64(now.UnixNano()), uint64(now.Add(time.Second).UnixNano()))
				require.NoError(t, pw.pushBytes(now, strconv.Itoa(i), req))
			}

			synctest.Test(t, func(t *testing.T) {
				flushErr := errors.New("block metadata unavailable")
				release := make(chan struct{})
				unblock := sync.OnceFunc(func() { close(release) })
				defer unblock()
				var calls atomic.Int32
				reader := &flushConcurrencyReader{
					blockMeta: func() error {
						calls.Add(1)
						<-release
						return flushErr
					},
				}
				done := make(chan error, 1)
				go func() {
					done <- pw.flush(context.Background(), reader, nil, nil)
				}()

				// Every started flush is blocked in BlockMeta, so no more than
				// the configured number of tenants should have been started.
				synctest.Wait()
				require.Equal(t, int32(concurrency), calls.Load())

				// Release the flushes and verify that backend errors still propagate.
				unblock()
				require.ErrorIs(t, <-done, flushErr)
			})
		})
	}
}

var _ tempodb.Reader = (*flushConcurrencyReader)(nil)

type flushConcurrencyReader struct {
	tempodb.Reader
	blockMeta func() error
}

func (r *flushConcurrencyReader) BlockMeta(context.Context, string, backend.UUID) (*backend.BlockMeta, *backend.CompactedBlockMeta, error) {
	return nil, nil, r.blockMeta()
}
