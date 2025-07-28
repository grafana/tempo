package v2

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type slowWriter struct {
	backend.Writer
	wait chan struct{}
}

func (w *slowWriter) WriteBlockMeta(ctx context.Context, meta *backend.BlockMeta) error {
	w.wait <- struct{}{} // send a signal to a goroutine
	<-w.wait             // wait for the signal from the goroutine
	return w.Writer.WriteBlockMeta(ctx, meta)
}

func TestWriteBlockMetaWithNoCompactFlag(t *testing.T) {
	tempDir := t.TempDir()

	r, w, _, err := local.New(&local.Config{
		Path: tempDir,
	})
	require.NoError(t, err)

	for _, withNoCompactFlag := range []bool{true, false} {
		t.Run(fmt.Sprintf("withNoCompactFlag=%t", withNoCompactFlag), func(t *testing.T) {
			ctx := t.Context()
			meta := &backend.BlockMeta{
				BlockID:  backend.NewUUID(),
				TenantID: "test-tenant",
			}

			waitChan := make(chan struct{})
			reader := backend.NewReader(r)
			writer := &slowWriter{
				Writer: backend.NewWriter(w),
				wait:   waitChan,
			}

			// Create a StreamingBlock with nocompact flag configured
			cfg := &common.BlockConfig{
				BloomFP:                 0.01,
				BloomShardSizeBytes:     1024,
				IndexDownsampleBytes:    1024 * 1024, // 1MB default
				IndexPageSizeBytes:      250 * 1024,  // 250KB default
				CreateWithNoCompactFlag: withNoCompactFlag,
				Encoding:                backend.EncLZ4_64k,
			}
			streamingBlock, err := NewStreamingBlock(cfg, (uuid.UUID)(meta.BlockID), meta.TenantID, []*backend.BlockMeta{meta}, 100)
			require.NoError(t, err)

			go func() {
				<-waitChan // writing block meta started, stopping it to emulate a slow write
				hasFlag, err := reader.HasNoCompactFlag(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
				require.NoError(t, err)
				assert.Equal(t, withNoCompactFlag, hasFlag, fmt.Sprintf("nocompact flag should be %t in the middle of writing", withNoCompactFlag))
				waitChan <- struct{}{} // we checked flag, proceed with writing
			}()

			// Complete the streamingBlock - this should write nocompact flag first, then all block data
			_, err = streamingBlock.Complete(ctx, nil, writer)
			require.NoError(t, err)

			// Verify nocompact flag remains after successful Complete (flag removal is done at higher level)
			hasFlag, err := reader.HasNoCompactFlag(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
			require.NoError(t, err)
			assert.Equal(t, withNoCompactFlag, hasFlag, fmt.Sprintf("nocompact flag should be %t after successful Complete", withNoCompactFlag))

			// Verify meta.json was written
			blockMeta, err := reader.BlockMeta(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
			require.NoError(t, err)
			assert.Equal(t, meta.BlockID, blockMeta.BlockID)
			assert.Equal(t, meta.TenantID, blockMeta.TenantID)
		})
	}
}
