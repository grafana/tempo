package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

type listBlockCmd struct {
	backendOptions

	TenantID   string `arg:"" help:"tenant-id within the bucket"`
	BlockID    string `arg:"" help:"block ID to list"`
	CheckDupes bool   `help:"check contents of block for duplicate trace IDs (warning, can be intense)"`
}

func (cmd *listBlockCmd) Run(ctx *globalOptions) error {
	r, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	return dumpBlock(r, c, cmd.TenantID, time.Hour, cmd.BlockID, cmd.CheckDupes)
}

func dumpBlock(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration, blockID string, checkDupes bool) error {
	id := uuid.MustParse(blockID)

	meta, err := r.BlockMeta(context.TODO(), id, tenantID)
	if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
		return err
	}

	compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
	if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
		return err
	}

	if meta == nil && compactedMeta == nil {
		fmt.Println("Unable to load any meta for block", blockID)
		return nil
	}

	unifiedMeta := getMeta(meta, compactedMeta, windowRange)

	fmt.Println("ID            : ", unifiedMeta.id)
	fmt.Println("Version       : ", unifiedMeta.version)
	fmt.Println("Total Objects : ", unifiedMeta.objects)
	fmt.Println("Level         : ", unifiedMeta.compactionLevel)
	fmt.Println("Window        : ", unifiedMeta.window)
	fmt.Println("Start         : ", unifiedMeta.start)
	fmt.Println("End           : ", unifiedMeta.end)

	if checkDupes {
		fmt.Println("Searching for dupes ...")

		block, err := encoding.NewBackendBlock(&tempodb_backend.BlockMeta{
			Version:  unifiedMeta.version,
			TenantID: tenantID,
			BlockID:  id,
		}, r)
		if err != nil {
			return err
		}

		iter, err := block.Iterator(10 * 1024 * 1024)
		if err != nil {
			return err
		}

		i := 0
		dupe := 0
		prevID := make([]byte, 16)
		for {
			objID, _, err := iter.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			if bytes.Equal(objID, prevID) {
				dupe++
			}

			copy(prevID, objID)
			i++
			if i%100000 == 0 {
				fmt.Println("Record: ", i)
			}
		}

		fmt.Println("total: ", i)
		fmt.Println("dupes: ", dupe)
	}

	return nil
}
