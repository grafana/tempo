package benchmark

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"path/filepath"
	"runtime/debug"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/prometheus/common/version"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/pkg/util"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// dataFileName is the parquet file in every vParquet block.
const dataFileName = "data.parquet"

// TraceIDsAll asks for every trace ID in the block rather than a sample.
const TraceIDsAll = -1

// ProfileOptions control what is measured.
type ProfileOptions struct {
	// NumTraceIDs is how many present IDs to sample: 0 for none, or TraceIDsAll
	// to defer enumeration to run time. One absent ID is derived per present
	// ID, so the hit and miss paths are measured over the same number of
	// samples.
	NumTraceIDs int
}

// LoadLocalBlock opens the block at path, which must be a directory laid out as
// <bucket>/<tenant-id>/<block-id> because that is how backend.KeyPathForBlock
// addresses a block.
func LoadLocalBlock(ctx context.Context, path string) (*backend.BlockMeta, backend.Reader, error) {
	meta, raw, err := openLocalBlock(ctx, path)
	if err != nil {
		return nil, nil, err
	}
	return meta, backend.NewReader(raw), nil
}

// openLocalBlock returns the raw reader too, for callers that need to wrap it.
func openLocalBlock(ctx context.Context, path string) (*backend.BlockMeta, backend.RawReader, error) {
	path = filepath.Clean(path)

	blockDir, tenantID := filepath.Base(path), filepath.Base(filepath.Dir(path))
	bucket := filepath.Dir(filepath.Dir(path))

	blockID, err := uuid.Parse(blockDir)
	if err != nil {
		return nil, nil, fmt.Errorf("%q is not a block directory: its name must be a block ID: %w", path, err)
	}
	if tenantID == "." || tenantID == string(filepath.Separator) {
		return nil, nil, fmt.Errorf("%q has no tenant directory above the block", path)
	}

	rawR, _, _, err := local.New(&local.Config{Path: bucket})
	if err != nil {
		return nil, nil, err
	}

	meta, err := backend.NewReader(rawR).BlockMeta(ctx, blockID, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("reading block meta for %s in tenant %s: %w", blockID, tenantID, err)
	}
	return meta, rawR, nil
}

// ProfileBlock measures the block. The read cost is paid once here, and every
// variant of an experiment then works from the same profile.
func ProfileBlock(ctx context.Context, meta *backend.BlockMeta, r backend.Reader, o ProfileOptions) (*BlockProfile, error) {
	if meta == nil {
		return nil, errors.New("block metadata is required")
	}

	pf, err := openParquetFile(ctx, meta, r)
	if err != nil {
		return nil, err
	}

	blk, err := encoding.OpenBlock(meta, r)
	if err != nil {
		return nil, fmt.Errorf("opening block: %w", err)
	}

	traceIDs, err := profileTraceIDs(ctx, blk, pf, o.NumTraceIDs)
	if err != nil {
		return nil, err
	}

	p := &BlockProfile{
		SchemaVersion: ProfileSchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		GeneratedBy:   buildInfo(),
		Block:         meta,
		RowGroups:     len(pf.RowGroups()),
		TraceIDs:      traceIDs,
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("produced an invalid profile: %w", err)
	}
	return p, nil
}

// openParquetFile reads the footer of the block's data file. Its row-group
// count is authoritative: meta.TotalRecords can over-count, because
// streamingBlock.Complete increments it and then flushes, so a block whose rows
// divide evenly into row groups records one more than it has.
func openParquetFile(ctx context.Context, meta *backend.BlockMeta, r backend.Reader) (*parquet.File, error) {
	pf, err := parquet.OpenFile(&blockReaderAt{ctx: ctx, r: r, meta: meta}, int64(meta.Size_))
	if err != nil {
		return nil, fmt.Errorf("opening %s for block %s: %w", dataFileName, meta.BlockID, err)
	}
	if len(pf.RowGroups()) == 0 {
		return nil, fmt.Errorf("block %s has no row groups", meta.BlockID)
	}
	return pf, nil
}

func rowGroupCount(ctx context.Context, meta *backend.BlockMeta, r backend.Reader) (int, error) {
	pf, err := openParquetFile(ctx, meta, r)
	if err != nil {
		return 0, err
	}
	return len(pf.RowGroups()), nil
}

// blockReaderAt reads data.parquet through the backend, so the footer can be
// opened without importing a specific block encoding.
type blockReaderAt struct {
	ctx  context.Context
	r    backend.Reader
	meta *backend.BlockMeta
}

func (b *blockReaderAt) ReadAt(p []byte, off int64) (int, error) {
	err := b.r.ReadRange(b.ctx, dataFileName, uuid.UUID(b.meta.BlockID), b.meta.TenantID, uint64(off), p, nil)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// profileTraceIDs samples present trace IDs and derives an absent one next to
// each of them.
func profileTraceIDs(ctx context.Context, blk common.BackendBlock, pf *parquet.File, num int) (TraceIDProfile, error) {
	if num == 0 {
		return TraceIDProfile{Mode: TraceIDModeSample}, nil
	}
	if num == TraceIDsAll {
		// Embedding every ID would make the profile as large as the block's ID
		// column, so record the intent and let the runner enumerate both the
		// present IDs and their absent counterparts.
		return TraceIDProfile{Mode: TraceIDModeAll}, nil
	}

	present, absent, err := sampleTraceIDs(ctx, blk, pf, num)
	if err != nil {
		return TraceIDProfile{}, err
	}
	return TraceIDProfile{Mode: TraceIDModeSample, Present: present, Absent: absent}, nil
}

// sampleTraceIDs takes num present IDs at an even stride over every row of the
// block, and pairs each with an absent ID.
//
// Striding over all rows costs a full scan, but taking the head of each row
// group instead would leave a trace-by-ID benchmark reading only the opening
// pages of each row group — well under 1% of a large block's pages, on a path
// whose cost is dominated by page reads. The scan is paid once per block and
// every variant then reuses the sample, which needs no seed because the block's
// row order is fixed.
//
// Each absent ID is the midpoint between a sampled ID and the one that follows
// it in the block. Because a row group holds a contiguous range of the trace ID
// sort key, and the scan reads every row of it, those two IDs are adjacent in
// the whole block: no trace can lie between them, so the midpoint is absent by
// construction rather than by a lookup. That keeps profiling off the bloom
// filters, and spreads the absent IDs over the block's ID range, which matters
// because the bloom shard a lookup reads is a hash of the whole ID.
func sampleTraceIDs(ctx context.Context, blk common.BackendBlock, pf *parquet.File, num int) (present, absent []string, err error) {
	var total int64
	for _, rg := range pf.RowGroups() {
		total += rg.NumRows()
	}
	if total == 0 {
		return nil, nil, errors.New("block has no rows")
	}

	stride := max(total/int64(num), 1)

	present = make([]string, 0, num)
	absent = make([]string, 0, num)

	// An ID can only be paired once its successor is known, and a row group's
	// last ID is followed by the next group's first: row groups hold
	// contiguous ranges of the sort key, so that pair is adjacent too.
	var (
		prev        []byte
		prevOnStep  bool
		row         int64
		prevGroupID int
	)
	for rg, group := range pf.RowGroups() {
		ids, err := listTraceIDs(ctx, blk, group, rg)
		if err != nil {
			return nil, nil, fmt.Errorf("listing trace IDs in row group %d: %w", rg, err)
		}
		slices.SortFunc(ids, bytes.Compare)

		// The midpoints below are only absent if this row group's IDs all sort
		// above the previous one's, so check rather than assume it.
		if len(ids) > 0 && prev != nil && bytes.Compare(prev, ids[0]) >= 0 {
			return nil, nil, fmt.Errorf("row group %d overlaps row group %d, cannot derive absent IDs", rg, prevGroupID)
		}
		prevGroupID = rg

		for _, id := range ids {
			if prevOnStep {
				if mid := midpointTraceID(prev, id); !bytes.Equal(mid, prev) {
					present = append(present, util.PadTraceIDString(util.TraceIDToHexString(prev)))
					absent = append(absent, util.PadTraceIDString(util.TraceIDToHexString(mid)))
					if len(present) == num {
						return present, absent, nil
					}
				}
			}
			prev, prevOnStep = id, row%stride == 0
			row++
		}
	}

	if len(present) == 0 {
		return nil, nil, errors.New("found no trace IDs in the block")
	}
	// Fewer IDs than asked for is a property of the block, not an error.
	return present, absent, nil
}

// listTraceIDs returns every trace ID in one row group.
//
// The time range spans everything representable rather than the block's own
// window: a block's declared start and end can clip traces whose spans fall in
// its ingestion slack, and a partial list would break the adjacency the absent
// IDs rely on.
func listTraceIDs(ctx context.Context, blk common.BackendBlock, group parquet.RowGroup, rowGroup int) ([][]byte, error) {
	opts := common.DefaultSearchOptions()
	opts.StartPage, opts.TotalPages = rowGroup, 1

	fetcher := traceql.NewSpansetFetcherWrapperBoth(
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return blk.Fetch(ctx, req, opts)
		},
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			return blk.FetchSpans(ctx, req, opts)
		},
	)

	req := &tempopb.SearchRequest{
		Query: "{}",
		Limit: uint32(group.NumRows()),
		Start: 1,
		End:   math.MaxUint32,
	}

	resp, err := traceql.NewEngine().ExecuteSearch(ctx, req, fetcher)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	ids := make([][]byte, 0, len(resp.Traces))
	for _, tr := range resp.Traces {
		if tr.TraceID == "" {
			continue
		}
		id, err := util.HexStringToTraceID(tr.TraceID)
		if err != nil {
			return nil, fmt.Errorf("decoding trace ID %q: %w", tr.TraceID, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// midpointTraceID returns the ID halfway between a and b, read as big-endian
// integers.
func midpointTraceID(a, b []byte) []byte {
	mid := new(big.Int).Add(new(big.Int).SetBytes(a), new(big.Int).SetBytes(b))
	mid.Rsh(mid, 1)

	out := make([]byte, len(a))
	mid.FillBytes(out)
	return out
}

// buildInfo falls back to the VCS stamp because the ldflags that populate
// prometheus/common/version are only set by the Makefile.
func buildInfo() BuildInfo {
	info := BuildInfo{TempoVersion: version.Version, GitSHA: version.Revision}
	if info.TempoVersion != "" && info.GitSHA != "" {
		return info
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	for _, setting := range bi.Settings {
		if setting.Key == "vcs.revision" && info.GitSHA == "" {
			info.GitSHA = setting.Value
		}
	}
	return info
}
