package main

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

type viewIndexCmd struct {
	backendOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`
	BlockID  string `arg:"" help:"block ID to list"`
}

func (cmd *viewIndexCmd) Run(ctx *globalOptions) error {
	blockID, err := uuid.Parse(cmd.BlockID)
	if err != nil {
		return err
	}

	r, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	meta, err := backend.ReadBlockMeta(context.TODO(), r, blockID, cmd.TenantID)
	if err != nil {
		return err
	}

	b, err := encoding.NewBackendBlock(meta, r)
	if err != nil {
		return err
	}

	reader, err := b.NewIndexReader()
	if err != nil {
		return err
	}

	pageSize := 20

	for i := 0; ; i++ {
		record, err := reader.At(context.TODO(), i)
		if err != nil {
			return err
		}

		if record == nil {
			return nil
		}

		fmt.Printf("Index entry: %10v     ID: %s     Start: %10v     Length: %10v\n", i, hex.EncodeToString(record.ID), record.Start, record.Length)

		if (i+1)%pageSize == 0 {
			fmt.Printf("Press enter to continue\r")
			fmt.Scanln()
		}
	}
}
