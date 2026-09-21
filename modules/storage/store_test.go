package storage

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/services"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

func TestStopTerminatesWithPollingEnabled(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), s))

	// query-frontend and querier enable polling with a context they never cancel, so the
	// store has to end the poller itself or Reader.Shutdown never returns
	s.EnablePolling(context.Background(), nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, services.StopAndAwaitTerminated(ctx, s))
}

func TestStopTerminatesWithoutPolling(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), s))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, services.StopAndAwaitTerminated(ctx, s))
}

func newTestStore(t *testing.T) Store {
	tempDir := t.TempDir()

	s, err := NewStore(Config{
		Trace: tempodb.Config{
			Backend: backend.Local,
			Local:   &local.Config{Path: path.Join(tempDir, "traces")},
			Block: &common.BlockConfig{
				BloomFP:             .01,
				BloomShardSizeBytes: 100_000,
				Version:             encoding.DefaultEncoding().Version(),
			},
			WAL:           &wal.Config{Filepath: path.Join(tempDir, "wal")},
			BlocklistPoll: 100 * time.Millisecond,
			Search: &tempodb.SearchConfig{
				ChunkSizeBytes:      1_000_000,
				ReadBufferCount:     8,
				ReadBufferSizeBytes: 4 * 1024 * 1024,
			},
		},
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	return s
}
