package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
)

// parquetRewrite is a command that rewrites a parquet block on disk using the latest code for that encoding, and optionally
// a new set of dedicated columns.  This command is useful to test changes of things like encoding, compression, or different
// dedicated columns, or other code changes.
type parquetRewrite struct {
	In               string   `arg:"" help:"The input parquet block to read from."`
	Out              string   `arg:"" help:"The output folder to write block to." default:"./out" optional:""`
	DedicatedColumns []string `arg:"" help:"List of dedicated columns to convert. Overwrites existing dedicated columns" optional:""`
}

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

	blockCfg := &common.BlockConfig{
		BloomFP:             common.DefaultBloomFP,
		BloomShardSizeBytes: common.DefaultBloomShardSizeBytes,
		Version:             enc.Version(),
		RowGroupSizeBytes:   100 * 1024 * 1024,
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
	case vparquet3.VersionString, vparquet4.VersionString:
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
	case vparquet3.VersionString:
		iter = &parquetIterator3{
			r: parquet.NewGenericReader[*vparquet3.Trace](pf),
			m: meta,
		}
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

	_, err := i.r.Read(traces)
	if errors.Is(err, io.EOF) {
		return nil, nil, io.EOF
	}
	if err != nil {
		return nil, nil, err
	}

	pqTrace := traces[0]
	pbTrace := vparquet5.ParquetTraceToTempopbTrace(i.m, pqTrace)
	return pqTrace.TraceID, pbTrace, nil
}

func (i *parquetIterator5) Close() {
	_ = i.r.Close()
}
