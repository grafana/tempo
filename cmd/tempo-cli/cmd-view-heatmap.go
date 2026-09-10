package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
)

// Same environment variables as vparquet5's own benchmarks (see blockForBenchmarks in
// block_traceql_test.go), so a block already set up for `go test -bench` can be pointed at
// here without repeating the tenant/block-id/path on the command line.
const (
	envVarBenchBlockID = "VP5_BENCH_BLOCKID"
	envVarBenchPath    = "VP5_BENCH_PATH"
	envVarBenchTenant  = "VP5_BENCH_TENANTID"
)

type heatmapCmd struct {
	backendOptions

	TenantID string `arg:"" optional:"" help:"tenant-id within the bucket, defaults to $VP5_BENCH_TENANTID (or \"1\")"`
	BlockID  string `arg:"" optional:"" help:"block ID to visualize, defaults to $VP5_BENCH_BLOCKID"`
}

func (cmd *heatmapCmd) Run(ctx *globalOptions) error {
	cmd.applyBenchEnvDefaults()

	if cmd.BlockID == "" {
		return fmt.Errorf("block id is required: pass it as an argument or set $%s", envVarBenchBlockID)
	}

	blockID, err := uuid.Parse(cmd.BlockID)
	if err != nil {
		return fmt.Errorf("invalid block id: %w", err)
	}

	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	meta, err := r.BlockMeta(context.Background(), blockID, cmd.TenantID)
	if err != nil {
		return fmt.Errorf("failed to load block meta: %w", err)
	}
	if meta.TotalObjects <= 0 {
		return errors.New("block contains no traces")
	}
	if meta.Version != vparquet5.VersionString {
		return fmt.Errorf("heatmap requires a vParquet5 block, got version %q", meta.Version)
	}

	rowGroupEnds, rowGroupByteEnds, err := rowGroupBoundaries(context.Background(), r, meta)
	if err != nil {
		return fmt.Errorf("failed to read row group boundaries: %w", err)
	}

	searchOpts := common.SearchOptions{}
	tempodb.SearchConfig{}.ApplyToOptions(&searchOpts)

	// The block itself is (re)opened fresh per query in runHeatmapFetch, wrapping r in an
	// ioTrackingReader so each fetch's file reads can be attributed to that fetch alone.
	m := newHeatmapModel(r, meta, searchOpts, rowGroupEnds, rowGroupByteEnds)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	m.program = p

	_, err = p.Run()
	return err
}

// rowGroupBoundaries returns, for each row group in the block in order, the cumulative trace
// row count and the cumulative file byte offset through the end of that row group. Row groups
// are the unit tempo compacts and flushes in, so seeing where they fall relative to a query's
// matches is a useful landmark on both heatmaps. This opens the file the same way block.Fetch
// does internally (see backendBlock.openForSearch), but we only need the footer's row group
// metadata, not any column data, so it's a cheap one-time read.
//
// The byte offsets come from each column chunk's own dictionary/data page offset plus its
// TotalCompressedSize (the actual on-disk placement) - not RowGroup.TotalByteSize, which is
// the *uncompressed* size and so runs far larger than the file itself, nor
// ColumnChunk.FileOffset, which points at the ColumnMetaData record rather than the data and
// was observed non-monotonic across row groups on a real block. Checked empirically against a
// real multi-GB block: row groups are written back-to-back with no gaps, right after the
// 4-byte "PAR1" magic header.
func rowGroupBoundaries(ctx context.Context, r backend.Reader, meta *backend.BlockMeta) (rowEnds, byteEnds []int64, err error) {
	rr := vparquet5.NewBackendReaderAt(ctx, r, vparquet5.DataFileName, meta)
	schema, _, _ := vparquet5.SchemaWithDynamicChanges(meta.DedicatedColumns)

	pf, err := parquet.OpenFile(rr, int64(meta.Size_),
		parquet.SkipBloomFilters(true),
		parquet.SkipPageIndex(true),
		parquet.FileSchema(schema),
	)
	if err != nil {
		return nil, nil, err
	}

	rgs := pf.Metadata().RowGroups
	rowEnds = make([]int64, 0, len(rgs))
	byteEnds = make([]int64, 0, len(rgs))

	var cumRows int64
	for _, rg := range rgs {
		cumRows += rg.NumRows
		rowEnds = append(rowEnds, cumRows)

		var end int64
		for _, col := range rg.Columns {
			// ColumnChunk.FileOffset is documented as pointing at the ColumnMetaData
			// record, not the actual data - it's unreliable (observed non-monotonic
			// across row groups on a real block). The dictionary/data page offsets are
			// where the column's bytes actually start.
			start := col.MetaData.DataPageOffset
			if col.MetaData.DictionaryPageOffset > 0 && col.MetaData.DictionaryPageOffset < start {
				start = col.MetaData.DictionaryPageOffset
			}
			if e := start + col.MetaData.TotalCompressedSize; e > end {
				end = e
			}
		}
		byteEnds = append(byteEnds, end)
	}
	return rowEnds, byteEnds, nil
}

// applyBenchEnvDefaults fills in any of tenant-id, block-id, and bucket path that weren't
// passed on the command line from the same VP5_BENCH_* environment variables vparquet5's own
// benchmarks use, so a block already staged for `go test -bench` needs no extra typing here.
// Explicit flags/args always win; env vars only fill in what's left blank.
func (cmd *heatmapCmd) applyBenchEnvDefaults() {
	if cmd.BlockID == "" {
		cmd.BlockID = os.Getenv(envVarBenchBlockID)
	}

	if cmd.TenantID == "" {
		if tenantID, ok := os.LookupEnv(envVarBenchTenant); ok {
			cmd.TenantID = tenantID
		} else {
			cmd.TenantID = "1"
		}
	}

	if cmd.Bucket == "" {
		if path, ok := os.LookupEnv(envVarBenchPath); ok {
			cmd.Bucket = path
			if cmd.Backend == "" {
				cmd.Backend = "local"
			}
		}
	}
}
