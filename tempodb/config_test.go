package tempodb

import (
	"testing"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestApplyToOptions(t *testing.T) {
	opts := common.SearchOptions{}
	cfg := SearchConfig{}

	// test defaults
	cfg.ApplyToOptions(&opts)
	require.Equal(t, opts.PrefetchTraceCount, DefaultPrefetchTraceCount)
	require.Equal(t, opts.ChunkSizeBytes, uint32(DefaultSearchChunkSizeBytes))
	require.Equal(t, opts.ReadBufferCount, DefaultReadBufferCount)
	require.Equal(t, opts.ReadBufferSize, DefaultReadBufferSize)

	// test parameter fields are left alone
	opts.StartPage = 1
	opts.TotalPages = 2
	opts.MaxBytes = 3
	cfg.ApplyToOptions(&opts)
	require.Equal(t, opts.StartPage, 1)
	require.Equal(t, opts.TotalPages, 2)
	require.Equal(t, opts.MaxBytes, 3)

	// test non defaults
	cfg.ChunkSizeBytes = 4
	cfg.PrefetchTraceCount = 5
	cfg.ReadBufferCount = 6
	cfg.ReadBufferSize = 7
	cfg.ApplyToOptions(&opts)
	require.Equal(t, cfg.ChunkSizeBytes, uint32(4))
	require.Equal(t, cfg.PrefetchTraceCount, 5)
	require.Equal(t, cfg.ReadBufferCount, 6)
	require.Equal(t, cfg.ReadBufferSize, 7)
}
