package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet5"
)

type convertParquet4to5 struct {
	In               string   `arg:"" help:"The input parquet block to read from."`
	Out              string   `arg:"" help:"The output folder to write block to." default:"./out" optional:""`
	DedicatedColumns []string `arg:"" help:"List of dedicated columns to convert. Overwrites existing dedicated columns" optional:""`
}

func (cmd *convertParquet4to5) Run() error {
	cmd.In = getPathToBlockDir(cmd.In)

	// open the input parquet file
	in, pf, err := openParquetFile(cmd.In)
	if err != nil {
		return err
	}
	defer in.Close()

	// open the input metadata file
	meta, err := readBlockMeta(cmd.In)
	if err != nil {
		return err
	}

	// create output block
	outR, outW, _, err := local.New(&local.Config{
		Path: cmd.Out,
	})
	if err != nil {
		return err
	}

	// calculate dedicated columns
	var dedicatedCols backend.DedicatedColumns

	if len(cmd.DedicatedColumns) > 0 {
		dedicatedCols = make(backend.DedicatedColumns, 0, len(cmd.DedicatedColumns))

		for _, col := range cmd.DedicatedColumns {

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
				return err
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
				return fmt.Errorf("dedicated columns must be scoped: %s", att.Scope)
			}

			fmt.Printf("add dedicated column scope=%s type=%s name=%s\n", scope, typ, att.Name)

			dedicatedCols = append(dedicatedCols, backend.DedicatedColumn{
				Scope:   scope,
				Name:    att.Name,
				Type:    typ,
				Options: options,
			})
		}
	} else {
		dedicatedCols = meta.DedicatedColumns
	}

	// copy block
	blockCfg := &common.BlockConfig{
		BloomFP:             0.99,
		BloomShardSizeBytes: 1024 * 1024,
		Version:             vparquet5.VersionString,
		RowGroupSizeBytes:   100 * 1024 * 1024,
	}

	newMeta := *meta
	newMeta.Version = vparquet5.VersionString
	newMeta.DedicatedColumns = dedicatedCols

	// create iterator over in file
	iter := &parquetIterator4{
		r: parquet.NewGenericReader[*vparquet4.Trace](pf),
		m: meta,
	}

	fmt.Printf("Creating vParquet5 block in %s\n", filepath.Join(cmd.Out, meta.TenantID, newMeta.BlockID.String()))
	fmt.Printf("Converting rows 0 to %d\n", pf.NumRows())
	outMeta, err := vparquet5.CreateBlock(context.Background(), blockCfg, &newMeta, iter, backend.NewReader(outR), backend.NewWriter(outW))
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created block with size=%d and footerSize=%d\n", outMeta.Size_, outMeta.FooterSize)
	return nil
}

type parquetIterator4 struct {
	r *parquet.GenericReader[*vparquet4.Trace]
	m *backend.BlockMeta
	i int
}

func (i *parquetIterator4) Next(_ context.Context) (common.ID, *tempopb.Trace, error) {
	traces := []*vparquet4.Trace{{}}

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
	pbTrace := vparquet4.ParquetTraceToTempopbTrace(i.m, pqTrace)
	return pqTrace.TraceID, pbTrace, nil
}

func (i *parquetIterator4) Close() {
	_ = i.r.Close()
}

func getPathToBlockDir(path string) string {
	if filepath.Base(path) == "data.parquet" {
		return filepath.Dir(path)
	}
	return path
}

func openParquetFile(blockPath string) (*os.File, *parquet.File, error) {
	inFile := filepath.Join(blockPath, "data.parquet")
	in, err := os.Open(inFile)
	if err != nil {
		return nil, nil, err
	}

	inStat, err := in.Stat()
	if err != nil {
		return nil, nil, err
	}

	pf, err := parquet.OpenFile(in, inStat.Size())
	if err != nil {
		return nil, nil, err
	}

	return in, pf, nil
}

func readBlockMeta(blockPath string) (*backend.BlockMeta, error) {
	metaFile := filepath.Join(blockPath, "meta.json")
	inMeta, err := os.Open(metaFile)
	if err != nil {
		return nil, err
	}
	defer inMeta.Close()

	var meta backend.BlockMeta
	err = json.NewDecoder(inMeta).Decode(&meta)
	if err != nil {
		return nil, err
	}

	return &meta, nil
}
