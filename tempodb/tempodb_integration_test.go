package tempodb

import (
	"context"
	"fmt"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	proto "github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func TestIntegrationCompleteBlock(t *testing.T) {
	for _, from := range encoding.AllEncodings() {
		for _, to := range encoding.AllEncodings() {
			t.Run(fmt.Sprintf("%s->%s", from.Version(), to.Version()), func(t *testing.T) {
				testCompleteBlock(t, from.Version(), to.Version())
			})
		}
	}
}

func testCompleteBlock(t *testing.T, from, to string) {
	_, w, _, _ := testConfig(t, backend.EncLZ4_256k, time.Minute, func(c *Config) {
		c.Block.Version = from // temporarily set config to from while we create the wal, so it makes blocks in the "from" format
	})

	wal := w.WAL()
	rw := w.(*readerWriter)
	rw.cfg.Block.Version = to // now set it back so we cut blocks in the "to" format

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	require.NoError(t, err, "unexpected error creating block")
	require.Equal(t, block.BlockMeta().Version, from)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	numMsgs := 100
	reqs := make([]*tempopb.Trace, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := test.ValidTraceID(nil)
		req := test.MakeTrace(rand.Int()%10, id)
		trace.SortTrace(req)
		writeTraceToWal(t, block, dec, id, req, 0, 0)
		reqs = append(reqs, req)
		ids = append(ids, id)
	}
	require.NoError(t, block.Flush())

	complete, err := w.CompleteBlock(context.Background(), block)
	require.NoError(t, err, "unexpected error completing block")
	require.Equal(t, complete.BlockMeta().Version, to)

	for i, id := range ids {
		found, err := complete.FindTraceByID(context.TODO(), id, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, found)
		trace.SortTrace(found)
		require.True(t, proto.Equal(found, reqs[i]))
	}
}

func TestIntegrationCompleteBlockHonorsStartStopTimes(t *testing.T) {
	testEncodings := []string{v2.VersionString, vparquet.VersionString}
	for _, enc := range testEncodings {
		t.Run(enc, func(t *testing.T) {
			testCompleteBlockHonorsStartStopTimes(t, enc)
		})
	}
}

func testCompleteBlockHonorsStartStopTimes(t *testing.T, targetBlockVersion string) {

	tempDir := t.TempDir()

	_, w, _, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              targetBlockVersion,
			Encoding:             backend.EncNone,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			IngestionSlack: time.Minute,
			Filepath:       path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()

	now := time.Now().Unix()
	oneHourAgo := time.Now().Add(-1 * time.Hour).Unix()
	oneHour := time.Now().Add(time.Hour).Unix()

	block, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	require.NoError(t, err, "unexpected error creating block")

	// Write a trace from 1 hour ago.
	// The wal slack time will adjust it to 1 minute ago
	id := test.ValidTraceID(nil)
	req := test.MakeTrace(10, id)
	writeTraceToWal(t, block, dec, id, req, uint32(oneHourAgo), uint32(oneHour))

	complete, err := w.CompleteBlock(context.Background(), block)
	require.NoError(t, err, "unexpected error completing block")

	// Verify the block time was constrained to the slack time.
	require.Equal(t, now, complete.BlockMeta().StartTime.Unix())
	require.Equal(t, now, complete.BlockMeta().EndTime.Unix())
}
