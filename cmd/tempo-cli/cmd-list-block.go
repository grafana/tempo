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

type lsBlockCmd struct {
	backendOptions

	TenantID   string `arg:"" help:"tenant-id within the bucket"`
	BlockID    string `arg:"" help:"block ID to list"`
	CheckDupes bool   `help:"check contents of block for duplicate trace IDs (warning, can be intense)"`
}

func (cmd *lsBlockCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions)
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

	objects, lvl, window, start, end := blockStats(meta, compactedMeta, windowRange)

	fmt.Println("ID            : ", id)
	fmt.Println("Total Objects : ", objects)
	fmt.Println("Level         : ", lvl)
	fmt.Println("Window        : ", window)
	fmt.Println("Start         : ", start)
	fmt.Println("End           : ", end)

	if checkDupes {
		fmt.Println("Searching for dupes ...")

		iter, err := encoding.NewBackendIterator(tenantID, id, 10*1024*1024, r)
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
