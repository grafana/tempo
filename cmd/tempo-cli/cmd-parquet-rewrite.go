package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet5"
)

// parquetRewrite is a command that rewrites a parquet block on disk using the latest code for that encoding, and optionally
// a new set of dedicated columns.  This command is useful to test changes of things like encoding, compression, or different
// dedicated columns, or other code changes.
//
// Block parameters that are not passed on the command line are inherited from the input
// block, so that a rewrite without arguments reproduces the original block as closely
// as possible.
type parquetRewrite struct {
	In               string   `arg:"" help:"The input parquet block to read from."`
	Out              string   `arg:"" help:"The output folder to write block to." default:"./out" optional:""`
	DedicatedColumns []string `arg:"" help:"List of dedicated columns to convert. Overwrites existing dedicated columns" optional:""`

	// Parameters below are optional. Unset (zero) values are inherited from the input block.
	RowGroupSizeBytes   int     `help:"Target row group size in bytes. Unset inherits the input block's row group size." optional:""`
	BloomFP             float64 `help:"Bloom filter false positive rate. Unset inherits the input block's rate." optional:""`
	BloomShardSizeBytes int     `help:"Bloom filter shard size in bytes. Unset inherits the input block's shard size." optional:""`
}

// defaultRowGroupSizeBytes is the fallback row group size when the input parquet
// file has no row groups to inherit from. It matches the ~100MB default that
// common.BlockConfig and the rest of Tempo use.
const defaultRowGroupSizeBytes = 100 * 1024 * 1024

func (cmd *parquetRewrite) Run() error {
	cmd.In = getPathToBlockDir(cmd.In)

	meta, err := readBlockMeta(cmd.In)
	if err != nil {
		return err
	}

	enc, err := encoding.FromVersionForWrites(meta.Version)
	if err != nil {
		return fmt.Errorf("detect encoding from block meta: %w", err)
	}

	in, pf, iter, err := openParquetForRewrite(cmd.In, meta)
	if err != nil {
		return err
	}
	defer in.Close()
	defer iter.Close()

	outR, outW, _, err := local.New(&local.Config{
		Path: cmd.Out,
	})
	if err != nil {
		return err
	}

	dedicatedCols, err := parseDedicatedColumns(cmd.DedicatedColumns, meta.DedicatedColumns)
	if err != nil {
		return err
	}

	blockCfg, err := cmd.blockConfig(enc.Version(), meta, pf)
	if err != nil {
		return err
	}

	newMeta := *meta
	newMeta.Version = enc.Version()
	newMeta.DedicatedColumns = dedicatedCols

	fmt.Printf("Detected encoding %s from block meta\n", meta.Version)
	fmt.Printf("Rewriting %s block in %s\n", enc.Version(), filepath.Join(cmd.Out, meta.TenantID, newMeta.BlockID.String()))
	fmt.Printf("Converting rows 0 to %d\n", pf.NumRows())
	outMeta, err := enc.CreateBlock(context.Background(), blockCfg, &newMeta, iter, backend.NewReader(outR), backend.NewWriter(outW))
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created block with size=%d and footerSize=%d\n", outMeta.Size_, outMeta.FooterSize)
	return nil
}

// blockConfig assembles the common.BlockConfig used to create the new block. Every
// parameter that is not set on the command line is inherited from the input block so
// that the rewrite reproduces the original block's properties as closely as possible:
// the row group size from the input parquet file, and the bloom filter parameters from
// the input block's bloom shard header.
func (cmd *parquetRewrite) blockConfig(version string, meta *backend.BlockMeta, pf *parquet.File) (*common.BlockConfig, error) {
	cfg := &common.BlockConfig{
		BloomFP:             cmd.BloomFP,
		BloomShardSizeBytes: cmd.BloomShardSizeBytes,
		Version:             version,
		RowGroupSizeBytes:   cmd.RowGroupSizeBytes,
	}

	if cfg.RowGroupSizeBytes == 0 {
		cfg.RowGroupSizeBytes = inheritedRowGroupSizeBytes(pf)
		fmt.Printf("Inheriting row group size %d bytes from the input block\n", cfg.RowGroupSizeBytes)
	}

	if cfg.BloomFP == 0 || cfg.BloomShardSizeBytes == 0 {
		fp, shardSizeBytes, err := inheritedBloomParams(cmd.In, meta)
		if err != nil {
			return nil, fmt.Errorf("inheriting bloom filter parameters, pass --bloom-fp and --bloom-shard-size-bytes to override: %w", err)
		}
		if cfg.BloomFP == 0 {
			cfg.BloomFP = fp
			fmt.Printf("Inheriting bloom filter false positive rate %v from the input block\n", cfg.BloomFP)
		}
		if cfg.BloomShardSizeBytes == 0 {
			cfg.BloomShardSizeBytes = shardSizeBytes
			fmt.Printf("Inheriting bloom filter shard size %d bytes from the input block\n", cfg.BloomShardSizeBytes)
		}
	}

	if cfg.RowGroupSizeBytes <= 0 {
		return nil, fmt.Errorf("row group size must be positive, got %d", cfg.RowGroupSizeBytes)
	}
	if err := common.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("validating block config: %w", err)
	}

	return cfg, nil
}

// inheritedRowGroupSizeBytes returns the row group size that best matches the given
// input parquet file. Row groups are cut once the estimated buffered bytes exceed the
// configured size, so the largest row group in the file is the closest surviving record
// of what that size was. Falls back to the Tempo default if the file has no row groups.
func inheritedRowGroupSizeBytes(pf *parquet.File) int {
	var size int64
	for _, rg := range pf.Metadata().RowGroups {
		size = max(size, rg.TotalByteSize)
	}
	if size <= 0 {
		return defaultRowGroupSizeBytes
	}
	return int(size)
}

// inheritedBloomParams recovers the bloom filter false positive rate and shard size
// from an existing block directory. The header of every bloom shard (written by
// bloom.BloomFilter.WriteTo) stores m, the number of bits in the shard, and k, the
// number of hash functions. The shard size is m/8 bytes. The false positive rate is
// not stored anywhere, so it is reconstructed from m, k, the shard count and the
// block's object count with reconstructBloomFP.
func inheritedBloomParams(blockPath string, meta *backend.BlockMeta) (fp float64, shardSizeBytes int, err error) {
	f, err := os.Open(filepath.Join(blockPath, common.BloomName(0)))
	if err != nil {
		return 0, 0, fmt.Errorf("opening bloom shard: %w", err)
	}
	defer f.Close()

	var header [16]byte
	if _, err = io.ReadFull(f, header[:]); err != nil {
		return 0, 0, fmt.Errorf("reading bloom shard header: %w", err)
	}

	m := binary.BigEndian.Uint64(header[0:8])
	k := binary.BigEndian.Uint64(header[8:16])
	if m == 0 || m%8 != 0 || k == 0 {
		return 0, 0, fmt.Errorf("invalid bloom shard header: m=%d k=%d", m, k)
	}

	return reconstructBloomFP(uint64(meta.TotalObjects), m, uint64(meta.BloomShardCount), k), int(m / 8), nil
}

// reconstructBloomFP inverts the vendored bloom.EstimateParameters, which is
//
//	m = ceil(-1 * n * ln(p) / ln(2)^2)
//	k = ceil(ln(2) * m / n)
//
// Given the per-shard bit count m, the hash count k, the shard count and the object
// count n of an existing block, it returns the false positive rate that reproduces
// the block's hash count and shard count. EstimateParameters is not invertible in
// general (many rates map to the same k), so this picks the largest bit count m'
// that fits the block's shards and verifies it by running EstimateParameters forward.
// Falls back to 2^-k if no rate reproduces the block exactly.
const bloomMaxShardCount = 1000 // mirrors common.maxShardCount

func ceilDiv(a, b uint64) uint64 {
	return (a + b - 1) / b
}

func reconstructBloomFP(n, mPerShard, shardCount, k uint64) float64 {
	if n == 0 || k == 0 || mPerShard == 0 {
		return math.Pow(2, -float64(k))
	}

	// Largest bit count that can still produce k hashes: ceil(ln(2)*m/n) <= k.
	mKBound := uint64(float64(n) * float64(k) / math.Ln2)

	// Search down from the block's own shard boundary. A shard count at the cap
	// says nothing about m (the original m was larger), so start at the k bound.
	start := mKBound
	if shardCount != bloomMaxShardCount {
		start = min(start, mPerShard*shardCount)
	}

	for m := start; m > 0; m-- {
		fp := math.Exp(-math.Ln2 * math.Ln2 * float64(m) / float64(n))
		m2, k2 := bloom.EstimateParameters(uint(n), fp)
		if k2 == uint(k) && min(ceilDiv(uint64(m2), mPerShard), bloomMaxShardCount) == shardCount {
			return fp
		}
	}

	return math.Pow(2, -float64(k))
}

func parseDedicatedColumns(fromCLI []string, fromMeta backend.DedicatedColumns) (backend.DedicatedColumns, error) {
	if len(fromCLI) == 0 {
		return fromMeta, nil
	}

	dedicatedCols := make(backend.DedicatedColumns, 0, len(fromCLI))
	for _, col := range fromCLI {
		var (
			typ     = backend.DedicatedColumnTypeString
			options = backend.DedicatedColumnOptions{}
		)

		col, blob := strings.CutPrefix(col, "blob/")
		if blob {
			options = append(options, backend.DedicatedColumnOptionBlob)
		}

		col, isInt := strings.CutPrefix(col, "int/")
		if isInt {
			typ = backend.DedicatedColumnTypeInt
		}

		att, err := traceql.ParseIdentifier(col)
		if err != nil {
			return nil, err
		}

		var scope backend.DedicatedColumnScope
		switch att.Scope {
		case traceql.AttributeScopeSpan:
			scope = backend.DedicatedColumnScopeSpan
		case traceql.AttributeScopeResource:
			scope = backend.DedicatedColumnScopeResource
		case traceql.AttributeScopeEvent:
			scope = backend.DedicatedColumnScopeEvent
		default:
			return nil, fmt.Errorf("dedicated columns must be scoped: %s", att.Scope)
		}

		fmt.Printf("add dedicated column scope=%s type=%s name=%s\n", scope, typ, att.Name)

		dedicatedCols = append(dedicatedCols, backend.DedicatedColumn{
			Scope:   scope,
			Name:    att.Name,
			Type:    typ,
			Options: options,
		})
	}

	return dedicatedCols, nil
}

func openParquetForRewrite(blockPath string, meta *backend.BlockMeta) (*os.File, *parquet.File, common.Iterator, error) {
	inFile := filepath.Join(blockPath, "data.parquet")
	in, err := os.Open(inFile)
	if err != nil {
		return nil, nil, nil, err
	}

	inStat, err := in.Stat()
	if err != nil {
		_ = in.Close()
		return nil, nil, nil, err
	}

	var (
		fileOptions   []parquet.FileOption
		readerOptions []parquet.ReaderOption
	)

	switch meta.Version {
	case vparquet4.VersionString:
	case vparquet5.VersionString:
		schema, _, ro := vparquet5.SchemaWithDynamicChanges(meta.DedicatedColumns)
		readerOptions = ro
		fileOptions = []parquet.FileOption{parquet.FileSchema(schema)}
	default:
		_ = in.Close()
		return nil, nil, nil, fmt.Errorf("unsupported block version %q", meta.Version)
	}

	pf, err := parquet.OpenFile(in, inStat.Size(), fileOptions...)
	if err != nil {
		_ = in.Close()
		return nil, nil, nil, err
	}

	var iter common.Iterator
	switch meta.Version {
	case vparquet4.VersionString:
		iter = &parquetIterator4{
			r: parquet.NewGenericReader[*vparquet4.Trace](pf),
			m: meta,
		}
	case vparquet5.VersionString:
		iter = &parquetIterator5{
			r: parquet.NewGenericReader[*vparquet5.Trace](pf, readerOptions...),
			m: meta,
		}
	}

	return in, pf, iter, nil
}

type parquetIterator5 struct {
	r *parquet.GenericReader[*vparquet5.Trace]
	m *backend.BlockMeta
	i int
}

func (i *parquetIterator5) Next(_ context.Context) (common.ID, *tempopb.Trace, error) {
	traces := []*vparquet5.Trace{{}}

	i.i++
	if i.i%1000 == 0 {
		fmt.Println(i.i)
	}

	n, err := i.r.Read(traces)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, err
	}
	if n == 0 {
		return nil, nil, io.EOF
	}

	pqTrace := traces[0]
	pbTrace := vparquet5.ParquetTraceToTempopbTrace(i.m, pqTrace)

	return pqTrace.TraceID, pbTrace, nil
}

func (i *parquetIterator5) Close() {
	_ = i.r.Close()
}
