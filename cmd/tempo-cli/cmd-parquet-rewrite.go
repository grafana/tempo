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

	// Optional parameters. The row group size falls back to the Tempo default (100MB)
	// with a warning; the bloom filter parameters are inherited from the input block.
	RowGroupSizeBytes   int     `help:"Target row group size in bytes. Unset uses the 100MB Tempo default." optional:""`
	BloomFP             float64 `help:"Bloom filter false positive rate. Unset inherits the input block's rate." optional:""`
	BloomShardSizeBytes int     `help:"Bloom filter shard size in bytes. Unset inherits the input block's shard size." optional:""`
}

// defaultRowGroupSizeBytes is the row group size used when --row-group-size-bytes is not
// passed. It matches the ~100MB default used across Tempo (common.BlockConfig).
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

	blockCfg, err := cmd.blockConfig(enc.Version(), meta)
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

// blockConfig assembles the common.BlockConfig used to create the new block.
func (cmd *parquetRewrite) blockConfig(version string, meta *backend.BlockMeta) (*common.BlockConfig, error) {
	cfg := &common.BlockConfig{
		BloomFP:             cmd.BloomFP,
		BloomShardSizeBytes: cmd.BloomShardSizeBytes,
		Version:             version,
		RowGroupSizeBytes:   cmd.RowGroupSizeBytes,
	}

	if cfg.RowGroupSizeBytes == 0 {
		cfg.RowGroupSizeBytes = defaultRowGroupSizeBytes
		fmt.Printf("Warning: no row group size specified, using the %dMB default. Pass --row-group-size-bytes to set an explicit row group size.\n", defaultRowGroupSizeBytes/(1024*1024))
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

// inheritedBloomParams recovers the bloom filter false positive rate and shard size from an existing block directory.
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

const bloomMaxShardCount = 1000 // mirrors common.maxShardCount

// reconstructBloomFP inverts the vendored bloom.EstimateParameters, which is
//
//	m = ceil(-1 * n * ln(p) / ln(2)^2)
//	k = ceil(ln(2) * m / n)
//
// returning a rate that reproduces an existing block's bloom filters:
// same shard size, same hash count k, same shard count.
func reconstructBloomFP(n, shardBits, shardCount, k uint64) float64 {
	if n == 0 || k == 0 || shardBits == 0 {
		// Cannot invert; fall back to a rate that at least matches the hash count.
		return math.Pow(2, -float64(k))
	}

	m := float64(uint64(float64(n) * float64(k) / math.Ln2)) // <= k*n/ln(2)
	if shardCount > 0 && shardCount < bloomMaxShardCount {
		m = math.Min(m, float64(shardCount*shardBits))
	}

	// Inverse of m = ceil(-1 * n * ln(p) / ln(2)^2).
	return math.Exp(-math.Ln2 * math.Ln2 * (m - 0.5) / float64(n))
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
