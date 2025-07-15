package v2

import (
	"context"
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

	ctx := context.Background()
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

	go func() {
		<-waitChan // writing block meta started, stopping it to emulate a slow write
		hasFlag, err := reader.HasNoCompactFlag(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
		require.NoError(t, err)
		assert.True(t, hasFlag, "nocompact flag should be true in the middle of writing")
		waitChan <- struct{}{} // we checked flag, proceed with writing
	}()

	// Create test data
	indexBytes := []byte("test-index")
	bloom := common.NewBloom(0.01, 100, 100)
	bloom.Add([]byte("test-trace-id"))

	// Write block meta - this should include nocompact flag first, then remove it
	err = writeBlockMeta(ctx, writer, meta, indexBytes, bloom)
	require.NoError(t, err)

	// Verify nocompact flag was removed after successful write
	hasFlag, err := reader.HasNoCompactFlag(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
	require.NoError(t, err)
	assert.False(t, hasFlag, "nocompact flag should be removed after successful writeBlockMeta")

	// Verify meta.json was written
	blockMeta, err := reader.BlockMeta(ctx, (uuid.UUID)(meta.BlockID), meta.TenantID)
	require.NoError(t, err)
	assert.Equal(t, meta.BlockID, blockMeta.BlockID)
	assert.Equal(t, meta.TenantID, blockMeta.TenantID)
}
