package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

func TestDropTraceCmd(t *testing.T) {
	testCase := func(t *testing.T, blocksNum int, tracesNum int, deleteEvery int) {
		cmd := dropTracesCmd{
			backendOptions: backendOptions{
				Backend: "local",
				Bucket:  t.TempDir(),
			},
			TenantID:  "single-tenant",
			DropTrace: true,
		}
		generateTestBlocks(t, cmd.backendOptions.Bucket, cmd.TenantID, blocksNum, tracesNum)

		before := getAllTraceIDs(t, cmd.backendOptions.Bucket, cmd.TenantID)

		var expectedAfter, toRemove []string
		for i, traceID := range before {
			if i%deleteEvery == 0 {
				toRemove = append(toRemove, traceID)
			} else {
				expectedAfter = append(expectedAfter, traceID)
			}
		}
		cmd.TraceIDs = strings.Join(toRemove, ",")

		err := cmd.Run(&globalOptions{})
		require.NoError(t, err)

		after := getAllTraceIDs(t, cmd.backendOptions.Bucket, cmd.TenantID)

		require.ElementsMatch(t, after, expectedAfter)
	}

	testCase(t, 1, 10, 3)
	testCase(t, 2, 5, 3)
	testCase(t, 2, 5, 1)
}

func generateTestBlocks(t *testing.T, tempDir string, tenantID string, blockCount int, traceCount int) {
	t.Helper()

	rawR, rawW, _, err := local.New(&local.Config{
		Path: tempDir,
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	ctx := context.Background()

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
	}

	for bn := 0; bn < blockCount; bn++ {
		traces := newTestTraces(traceCount)
		iter := &testIterator{traces: traces}
		meta := backend.NewBlockMeta(tenantID, uuid.New(), vparquet4.VersionString, backend.EncNone, "")
		meta.TotalObjects = int64(len(iter.traces))
		_, err := vparquet4.CreateBlock(ctx, cfg, meta, iter, r, w)
		require.NoError(t, err)
	}
}

func getAllTraceIDs(t *testing.T, dir string, tenant string) []string {
	t.Helper()

	rawR, _, _, err := local.New(&local.Config{
		Path: dir,
	})
	require.NoError(t, err)

	reader := backend.NewReader(rawR)
	ctx := context.Background()

	tenants, err := reader.Tenants(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{tenant}, tenants)

	blocks, _, err := reader.Blocks(ctx, tenant)
	require.NoError(t, err)

	var traceIDs []string
	for _, block := range blocks {
		meta, err := reader.BlockMeta(ctx, block, tenant)
		require.NoError(t, err)
		rr := vparquet4.NewBackendReaderAt(ctx, reader, vparquet4.DataFileName, meta)
		br := tempo_io.NewBufferedReaderAt(rr, int64(meta.Size_), 2*1024*1024, 64)
		parquetSchema := parquet.SchemaOf(&vparquet4.Trace{})
		o := []parquet.FileOption{
			parquet.SkipBloomFilters(true),
			parquet.SkipPageIndex(true),
			parquet.FileSchema(parquetSchema),
			parquet.FileReadMode(parquet.ReadModeAsync),
		}
		pf, err := parquet.OpenFile(br, int64(meta.Size_), o...)
		require.NoError(t, err)
		r := parquet.NewReader(pf, parquetSchema)
		defer func() {
			err := r.Close()
			require.NoError(t, err)
		}()
		traceIDIndex, _ := parquetquery.GetColumnIndexByPath(pf, vparquet4.TraceIDColumnName)
		require.GreaterOrEqual(t, traceIDIndex, 0)
		defer func() {
			err := r.Close()
			require.NoError(t, err)
		}()

		for read := int64(0); read < r.NumRows(); {
			rows := make([]parquet.Row, r.NumRows())
			n, err := r.ReadRows(rows)
			if !errors.Is(err, io.EOF) {
				require.NoError(t, err)
			}
			require.Greater(t, n, 0)
			rows = rows[:n]
			read += int64(n)

			getTraceID := func(row parquet.Row) common.ID {
				for _, v := range row {
					if v.Column() == traceIDIndex {
						return v.ByteArray()
					}
				}

				return nil
			}

			for _, row := range rows {
				traceID := getTraceID(row)
				traceIDs = append(traceIDs, util.TraceIDToHexString(traceID))
			}
		}

		// Ensure that we read all rows
		_, err = r.ReadRows([]parquet.Row{{}})
		require.ErrorIs(t, err, io.EOF)
	}

	return traceIDs
}

type testTrace struct {
	traceID common.ID
	trace   *tempopb.Trace
}

type testIterator struct {
	traces []testTrace
}

func newTestTraces(traceCount int) []testTrace {
	traces := make([]testTrace, 0, traceCount)

	for i := 0; i < traceCount; i++ {
		traceID := test.ValidTraceID(nil)
		trace := test.MakeTraceWithTags(traceID, "megaservice", int64(i))
		traces = append(traces, testTrace{traceID: traceID, trace: trace})
	}

	sort.Slice(traces, func(i, j int) bool {
		return bytes.Compare(traces[i].traceID, traces[j].traceID) == -1
	})

	return traces
}

func (i *testIterator) Next(context.Context) (common.ID, *tempopb.Trace, error) {
	if len(i.traces) == 0 {
		return nil, nil, io.EOF
	}
	tr := i.traces[0]
	i.traces = i.traces[1:]
	return tr.traceID, tr.trace, nil
}

func (i *testIterator) Close() {
}
