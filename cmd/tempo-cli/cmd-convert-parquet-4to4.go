package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

type convertParquet4to4 struct {
	In  string `arg:"" help:"The input parquet block to read from."`
	Out string `arg:"" help:"The output folder to write block to." default:"./out" optional:""`
}

func (cmd *convertParquet4to4) Run() error {
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

	// copy block
	blockCfg := &common.BlockConfig{
		BloomFP:             0.99,
		BloomShardSizeBytes: 1024 * 1024,
		Version:             vparquet4.VersionString,
		RowGroupSizeBytes:   100 * 1024 * 1024,
	}

	newMeta := *meta
	newMeta.Version = vparquet4.VersionString
	newMeta.DedicatedColumns = meta.DedicatedColumns

	// create iterator over in file
	iter := &parquetIterator4{
		r: parquet.NewGenericReader[*vparquet4.Trace](pf),
		m: meta,
	}

	fmt.Printf("Creating vParquet4 block in %s\n", filepath.Join(cmd.Out, meta.TenantID, newMeta.BlockID.String()))
	fmt.Printf("Converting rows 0 to %d\n", pf.NumRows())
	outMeta, err := vparquet4.CreateBlock(context.Background(), blockCfg, &newMeta, iter, backend.NewReader(outR), backend.NewWriter(outW))
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

	_, err := i.r.Read(traces)
	if errors.Is(err, io.EOF) {
		return nil, nil, io.EOF
	}
	if err != nil {
		return nil, nil, err
	}

	pqTrace := traces[0]
	pbTrace := vparquet4.ParquetTraceToTempopbTrace(i.m, pqTrace)
	return pqTrace.TraceID, pbTrace, nil
}

func (i *parquetIterator4) Close() {
	_ = i.r.Close()
}
