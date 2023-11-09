package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	//"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
	"github.com/parquet-go/parquet-go"
)

type convertParquet3to4 struct {
	In  string `arg:"" help:"The input parquet file to read from."`
	Out string `arg:"" help:"The output folder to write to."`
	//DedicatedColumns []string `arg:"" help:"List of dedicated columns to convert"`
}

func (cmd *convertParquet3to4) Run() error {
	// open the in file
	ctx := context.Background()

	in, err := os.Open(cmd.In)
	if err != nil {
		return err
	}
	defer in.Close()

	inStat, err := in.Stat()
	if err != nil {
		return err
	}

	pf, err := parquet.OpenFile(in, inStat.Size())
	if err != nil {
		return err
	}

	// create out block
	if cmd.Out == "" {
		cmd.Out = "./out"
	}
	outR, outW, _, err := local.New(&local.Config{
		Path: cmd.Out,
	})
	if err != nil {
		return err
	}

	// dedicatedCols := make([]backend.DedicatedColumn, 0, len(cmd.DedicatedColumns))
	// for _, col := range cmd.DedicatedColumns {
	// 	att, err := traceql.ParseIdentifier(col)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	scope := backend.DedicatedColumnScopeSpan
	// 	if att.Scope == traceql.AttributeScopeResource {
	// 		scope = backend.DedicatedColumnScopeResource
	// 	}

	// 	fmt.Println("scope", scope, "name", att.Name)

	// 	dedicatedCols = append(dedicatedCols, backend.DedicatedColumn{
	// 		Scope: scope,
	// 		Name:  att.Name,
	// 		Type:  backend.DedicatedColumnTypeString,
	// 	})
	// }

	blockCfg := &common.BlockConfig{
		BloomFP:             0.99,
		BloomShardSizeBytes: 1024 * 1024,
		Version:             vparquet4.VersionString,
		RowGroupSizeBytes:   100 * 1024 * 1024,
		//	DedicatedColumns:    dedicatedCols,
	}
	meta := &backend.BlockMeta{
		Version:      vparquet4.VersionString,
		BlockID:      uuid.New(),
		TenantID:     "test",
		TotalObjects: 1000000, // required for bloom filter calculations
		//	DedicatedColumns: dedicatedCols,
	}

	// create iterator over in file
	iter := &parquetIterator3{
		r: parquet.NewGenericReader[*vparquet3.Trace](pf),
		m: meta,
	}

	_, err = vparquet4.CreateBlock(ctx, blockCfg, meta, iter, backend.NewReader(outR), backend.NewWriter(outW))
	if err != nil {
		return err
	}

	return nil
}

type parquetIterator3 struct {
	r *parquet.GenericReader[*vparquet3.Trace]
	i int
	m *backend.BlockMeta
}

func (i *parquetIterator3) Next(_ context.Context) (common.ID, *tempopb.Trace, error) {
	traces := make([]*vparquet3.Trace, 1)

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
	pbTrace := vparquet3.ParquetTraceToTempopbTrace(i.m, pqTrace)
	return pqTrace.TraceID, pbTrace, nil
}

func (i *parquetIterator3) Close() {
	_ = i.r.Close()
}
