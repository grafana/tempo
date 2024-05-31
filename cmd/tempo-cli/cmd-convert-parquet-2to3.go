package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
)

type convertParquet2to3 struct {
	In               string   `arg:"" help:"The input parquet file to read from."`
	Out              string   `arg:"" help:"The output folder to write block to." default:"./out" optional:""`
	DedicatedColumns []string `arg:"" help:"List of dedicated columns to convert." optional:""`
}

func (cmd *convertParquet2to3) Run() error {
	cmd.In = getPathToBlockDir(cmd.In)

	// open the in file
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

	// create out block
	outR, outW, _, err := local.New(&local.Config{
		Path: cmd.Out,
	})
	if err != nil {
		return err
	}

	dedicatedCols := make([]backend.DedicatedColumn, 0, len(cmd.DedicatedColumns))
	for _, col := range cmd.DedicatedColumns {
		att, err := traceql.ParseIdentifier(col)
		if err != nil {
			return err
		}

		scope := backend.DedicatedColumnScopeSpan
		if att.Scope == traceql.AttributeScopeResource {
			scope = backend.DedicatedColumnScopeResource
		}

		fmt.Println("scope", scope, "name", att.Name)

		dedicatedCols = append(dedicatedCols, backend.DedicatedColumn{
			Scope: scope,
			Name:  att.Name,
			Type:  backend.DedicatedColumnTypeString,
		})
	}

	blockCfg := &common.BlockConfig{
		BloomFP:             0.99,
		BloomShardSizeBytes: 1024 * 1024,
		Version:             vparquet3.VersionString,
		RowGroupSizeBytes:   100 * 1024 * 1024,
		DedicatedColumns:    dedicatedCols,
	}

	newMeta := *meta
	newMeta.Version = vparquet3.VersionString
	newMeta.DedicatedColumns = dedicatedCols

	// create iterator over in file
	iter := &parquetIterator2{
		r: parquet.NewGenericReader[*vparquet2.Trace](pf),
	}

	_, err = vparquet3.CreateBlock(context.Background(), blockCfg, &newMeta, iter, backend.NewReader(outR), backend.NewWriter(outW))
	if err != nil {
		return err
	}

	return nil
}

type parquetIterator2 struct {
	r *parquet.GenericReader[*vparquet2.Trace]
	i int
}

func (i *parquetIterator2) Next(_ context.Context) (common.ID, *tempopb.Trace, error) {
	traces := make([]*vparquet2.Trace, 1)

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
	pbTrace := vparquet2.ParquetTraceToTempopbTrace(pqTrace)
	return pqTrace.TraceID, pbTrace, nil
}

func (i *parquetIterator2) Close() {
	_ = i.r.Close()
}
