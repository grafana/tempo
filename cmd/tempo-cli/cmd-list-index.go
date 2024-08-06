package main

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	v2 "github.com/grafana/tempo/v2/tempodb/encoding/v2"
)

type listIndexCmd struct {
	backendOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`
	BlockID  string `arg:"" help:"block ID to list"`
}

func (cmd *listIndexCmd) Run(ctx *globalOptions) error {
	blockID, err := uuid.Parse(cmd.BlockID)
	if err != nil {
		return err
	}

	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	meta, err := r.BlockMeta(context.TODO(), blockID, cmd.TenantID)
	if err != nil {
		return err
	}

	if meta.Version != v2.VersionString {
		return fmt.Errorf("unsupported block version: %s", meta.Version)
	}

	b, err := v2.NewBackendBlock(meta, r)
	if err != nil {
		return err
	}

	reader, err := b.NewIndexReader()
	if err != nil {
		return err
	}

	var minRecord *v2.Record
	var maxRecord *v2.Record
	count := 0

	for i := 0; ; i++ {
		record, err := reader.At(context.TODO(), i)
		if err != nil {
			return err
		}

		if record == nil {
			break
		}

		count++

		if minRecord == nil || record.Length < minRecord.Length {
			minRecord = record
		}

		if maxRecord == nil || record.Length > maxRecord.Length {
			maxRecord = record
		}
	}

	fmt.Println("Index entries:", count)

	if minRecord != nil {
		fmt.Printf("Min record: ID:%s Start:%v Length:%v\n", hex.EncodeToString(minRecord.ID), minRecord.Start, minRecord.Length)
	}

	if maxRecord != nil {
		fmt.Printf("Max record: ID:%s Start:%v Length:%v\n", hex.EncodeToString(maxRecord.ID), maxRecord.Start, maxRecord.Length)
	}

	return nil
}
