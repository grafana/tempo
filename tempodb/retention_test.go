package tempodb

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

func TestRetention(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	blockID := uuid.New()

	wal := w.WAL()
	assert.NoError(t, err)

	head, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err)

	complete, err := w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	err = w.WriteBlock(context.Background(), complete)
	assert.NoError(t, err)

	rw := r.(*readerWriter)
	// poll
	checkBlocklists(t, blockID, 1, 0, rw)

	// retention should mark it compacted
	r.(*readerWriter).doRetention()
	checkBlocklists(t, blockID, 0, 1, rw)

	// retention again should clear it
	r.(*readerWriter).doRetention()
	checkBlocklists(t, blockID, 0, 0, rw)
}

func TestBlockRetentionOverride(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	overrides := &mockOverrides{}

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Hour,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, overrides)

	cutTestBlocks(t, w, testTenantID, 10, 10)

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	// Retention = 1 hour, does nothing
	overrides.blockRetention = time.Hour
	r.(*readerWriter).doRetention()
	rw.pollBlocklist()
	assert.Equal(t, 10, len(rw.blocklist(testTenantID)))

	// Retention = 0, use default, still does nothing
	overrides.blockRetention = time.Minute
	r.(*readerWriter).doRetention()
	rw.pollBlocklist()
	assert.Equal(t, 10, len(rw.blocklist(testTenantID)))

	// Retention = 1ns, deletes everything
	overrides.blockRetention = time.Nanosecond
	r.(*readerWriter).doRetention()
	rw.pollBlocklist()
	assert.Equal(t, 0, len(rw.blocklist(testTenantID)))
}
