package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

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

	Fingerprint       string `help:"Trace fingerprints to write: v1 computes them from the traces, none writes none. Required when the input block has fingerprints, which are never copied. A block written with fingerprints is flagged nocompact, because compaction does not recompute them when it merges traces." optional:""`
	FingerprintReport string `help:"After rewriting, read the vParquet5 output block back and write a JSON report of fingerprint measurements to this new file. Works with or without fingerprints, so two rewrites can be compared." optional:""`
}

// defaultRowGroupSizeBytes is the row group size used when --row-group-size-bytes is not
// passed. It matches the ~100MB default used across Tempo (common.BlockConfig).
const defaultRowGroupSizeBytes = 100 * 1024 * 1024

// fingerprintNone is the --fingerprint value that writes no fingerprints.
const fingerprintNone = "none"

func (cmd *parquetRewrite) Run() error {
	cmd.In = getPathToBlockDir(cmd.In)

	meta, err := readBlockMeta(cmd.In)
	if err != nil {
		return err
	}
	options, err := cmd.outputOptions(meta)
	if err != nil {
		return err
	}
	if err := cmd.checkReport(meta); err != nil {
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

	return cmd.writeBlock(enc, meta, options, pf.NumRows(), iter)
}

// writeBlock writes the rewritten block from iter, which reads the inputRows
// rows of the input block, and then the fingerprint report if one is asked
// for. If the rewrite fails, the report records how far it got.
func (cmd *parquetRewrite) writeBlock(enc encoding.VersionedEncoding, meta *backend.BlockMeta, options map[string]string, inputRows int64, iter common.Iterator) error {
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
	blockCfg.ComputeTraceFingerprints = options[vparquet5.TraceSignatureVersionOption] == vparquet5.TraceSignatureVersionV1
	if blockCfg.ComputeTraceFingerprints {
		fmt.Println("Flagging the block nocompact: compaction does not recompute trace fingerprints")
	}

	newMeta := *meta
	newMeta.Version = enc.Version()
	newMeta.DedicatedColumns = dedicatedCols
	newMeta.Options = options

	blockDir := filepath.Join(cmd.Out, meta.TenantID, newMeta.BlockID.String())
	if err := reserveBlockDir(blockDir); err != nil {
		return err
	}
	var reportFile *os.File
	if cmd.FingerprintReport != "" {
		reportFile, err = reserveReport(cmd.FingerprintReport, cmd.In, blockDir)
		if err != nil {
			_ = os.Remove(blockDir)
			return err
		}
		defer reportFile.Close()
	}

	fmt.Printf("Detected encoding %s from block meta\n", meta.Version)
	fmt.Printf("Rewriting %s block in %s\n", enc.Version(), blockDir)
	fmt.Printf("Converting rows 0 to %d\n", inputRows)
	counted := &countingIterator{Iterator: iter, expected: inputRows}
	var before runtime.MemStats
	if reportFile != nil {
		runtime.ReadMemStats(&before)
	}
	start := time.Now()
	outMeta, err := enc.CreateBlock(context.Background(), blockCfg, &newMeta, counted, backend.NewReader(outR), backend.NewWriter(outW))
	rewrite := rewriteStats{duration: time.Since(start), inputRows: inputRows, inputRowsRead: counted.read, inputRowFailed: counted.failed}
	if reportFile != nil {
		rewrite.measureMemory(&before)
	}
	if err != nil {
		// reserveBlockDir created blockDir, so it holds only this run's partial output.
		if rmErr := os.RemoveAll(blockDir); rmErr != nil {
			err = errors.Join(err, fmt.Errorf("removing the partial output block: %w", rmErr))
		}
		if reportFile == nil {
			return err
		}
		report := newFingerprintReport(newMeta.BlockID.String(), rewrite)
		report.fail(phaseRewrite, err)
		if writeErr := report.write(reportFile); writeErr != nil {
			return errors.Join(err, fmt.Errorf("writing fingerprint report: %w", writeErr))
		}
		return err
	}

	fmt.Printf("Successfully created block with size=%d and footerSize=%d in %s\n", outMeta.Size_, outMeta.FooterSize, rewrite.duration)

	if reportFile == nil {
		return nil
	}
	report := newFingerprintReport(outMeta.BlockID.String(), rewrite)
	measureErr := report.measure(blockDir, outMeta)
	if err := report.write(reportFile); err != nil {
		return errors.Join(measureErr, fmt.Errorf("writing fingerprint report: %w", err))
	}
	if measureErr != nil {
		return fmt.Errorf("measuring fingerprints: %w", measureErr)
	}
	return nil
}

// reserveBlockDir creates the output block directory, which must not exist.
// Writing over a block, possibly the input itself, would truncate its files
// while they are still needed.
func reserveBlockDir(dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("output block %s already exists: remove it or choose another output folder", dir)
		}
		return err
	}
	return nil
}

// reserveReport creates the report file before the rewrite, so a report that
// cannot be written fails early. It must be a new file outside blockDirs,
// where the rewrite could replace it or it could replace a block file.
func reserveReport(path string, blockDirs ...string) (*os.File, error) {
	parent, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("--fingerprint-report: %w", err)
	}
	for _, dir := range blockDirs {
		info, err := os.Stat(dir)
		if err != nil {
			return nil, err
		}
		if os.SameFile(parent, info) {
			return nil, fmt.Errorf("--fingerprint-report %s is in block directory %s", path, dir)
		}
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("--fingerprint-report: %w", err)
	}
	return f, nil
}

// countingIterator passes on the traces of an input with expected rows and
// counts them. It fails if the input ends early or has more rows, so a short
// read is never written as a complete block.
type countingIterator struct {
	common.Iterator
	expected int64
	read     int64
	failed   bool
}

func (c *countingIterator) Next(ctx context.Context) (common.ID, *tempopb.Trace, error) {
	id, tr, err := c.Iterator.Next(ctx)
	switch {
	case errors.Is(err, io.EOF) || (err == nil && tr == nil):
		if c.read != c.expected {
			return nil, nil, fmt.Errorf("the input ended after %d of %d rows", c.read, c.expected)
		}
	case err != nil:
		c.failed = true
	case c.read == c.expected:
		return nil, nil, fmt.Errorf("the input has more than %d rows", c.expected)
	default:
		c.read++
	}
	return id, tr, err
}

// reservedFileNames are files whose presence alone changes how Tempo treats a
// block. For example, meta.deleted.json marks it for deletion.
var reservedFileNames = []string{backend.MetaName, backend.CompactedMetaName, backend.DeletedMetaName, backend.NoCompactFileName}

// checkReport fails before the rewrite if --fingerprint-report cannot be
// used: the report reads the output with the vParquet5 schema, and its file
// must not be named like a file that changes how Tempo treats a block.
func (cmd *parquetRewrite) checkReport(meta *backend.BlockMeta) error {
	if cmd.FingerprintReport == "" {
		return nil
	}
	if meta.Version != vparquet5.VersionString {
		return fmt.Errorf("--fingerprint-report needs a %s block, got %s", vparquet5.VersionString, meta.Version)
	}
	name := filepath.Base(cmd.FingerprintReport)
	if slices.ContainsFunc(reservedFileNames, func(reserved string) bool { return strings.EqualFold(reserved, name) }) {
		return fmt.Errorf("--fingerprint-report cannot be named %s, a block file that Tempo acts on", name)
	}
	return nil
}

// outputOptions returns the block options of the rewritten block. Its
// fingerprint version comes only from --fingerprint, so a rewrite never
// carries over or relabels the input block's fingerprints. An input with
// fingerprints needs an explicit --fingerprint, so they are never dropped
// by accident, and without one the options of an unsigned input are kept
// as they are.
func (cmd *parquetRewrite) outputOptions(meta *backend.BlockMeta) (map[string]string, error) {
	inputVersion := meta.Options[vparquet5.TraceSignatureVersionOption]
	options := maps.Clone(meta.Options)

	switch cmd.Fingerprint {
	case "":
		if inputVersion != vparquet5.TraceSignatureVersionNone {
			return nil, fmt.Errorf("the input block has trace fingerprints (version %s): pass --fingerprint=%s to recompute them or --fingerprint=%s to drop them",
				inputVersion, vparquet5.TraceSignatureVersionV1, fingerprintNone)
		}
	case fingerprintNone:
		delete(options, vparquet5.TraceSignatureVersionOption)
	case vparquet5.TraceSignatureVersionV1:
		if meta.Version != vparquet5.VersionString {
			return nil, fmt.Errorf("trace fingerprints need a %s block, got %s", vparquet5.VersionString, meta.Version)
		}
		if options == nil {
			options = map[string]string{}
		}
		options[vparquet5.TraceSignatureVersionOption] = vparquet5.TraceSignatureVersionV1
	default:
		return nil, fmt.Errorf("unsupported --fingerprint %q, supported: %s, %s", cmd.Fingerprint, vparquet5.TraceSignatureVersionV1, fingerprintNone)
	}
	return options, nil
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
		schema         *parquet.Schema
		fileOptions    []parquet.FileOption
		readerOptions  []parquet.ReaderOption
		unknownVersion bool
	)

	switch meta.Version {
	case vparquet4.VersionString:
	case vparquet5.VersionString:
		schema, _, readerOptions = vparquet5.SchemaWithDynamicChanges(vparquet5.TraceSchemaConfig{DedicatedColumns: meta.DedicatedColumns, Options: meta.Options})
		version := meta.Options[vparquet5.TraceSignatureVersionOption]
		unknownVersion = version != vparquet5.TraceSignatureVersionNone && version != vparquet5.TraceSignatureVersionV1
		// With its own schema, the file reads only the columns of schema.
		// parquet.FileSchema would make it read and then discard the
		// Similarity columns of an unknown version, which may not decode.
		if !unknownVersion {
			fileOptions = []parquet.FileOption{parquet.FileSchema(schema)}
		}
	default:
		_ = in.Close()
		return nil, nil, nil, fmt.Errorf("unsupported block version %q", meta.Version)
	}

	pf, err := parquet.OpenFile(in, inStat.Size(), fileOptions...)
	if err != nil {
		_ = in.Close()
		return nil, nil, nil, err
	}
	if schema != nil {
		if err := checkColumns(pf, schema, unknownVersion); err != nil {
			_ = in.Close()
			return nil, nil, nil, fmt.Errorf("input block %s: %w", blockPath, err)
		}
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

// checkColumns returns an error unless every row group of pf has the leaf
// columns of schema, in order. With ignoreSimilarity, Similarity columns
// after them are also allowed, as an input labeled with a fingerprint version
// this code does not know has them; the caller must then read pf without
// them. parquet.FileSchema checks none of this, and reading a file with a
// schema it was not written with can panic.
func checkColumns(pf *parquet.File, schema *parquet.Schema, ignoreSimilarity bool) error {
	want := schema.Columns()
	ignore := ignoreSimilarity && !slices.ContainsFunc(want, isSimilarityColumn)
	for i, rg := range pf.Metadata().RowGroups {
		cols := rg.Columns
		for ignore && len(cols) > len(want) && isSimilarityColumn(cols[len(cols)-1].MetaData.PathInSchema) {
			cols = cols[:len(cols)-1]
		}
		if len(cols) != len(want) {
			return fmt.Errorf("row group %d has %d columns, but the block meta describes %d", i, len(cols), len(want))
		}
		for j, col := range cols {
			if !slices.Equal(col.MetaData.PathInSchema, want[j]) {
				return fmt.Errorf("row group %d column %d is %s, but the block meta describes %s",
					i, j, strings.Join(col.MetaData.PathInSchema, "."), strings.Join(want[j], "."))
			}
		}
	}
	return nil
}

func isSimilarityColumn(path []string) bool {
	return len(path) > 0 && path[0] == "Similarity"
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
