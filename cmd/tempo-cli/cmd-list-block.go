package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

type listBlockCmd struct {
	backendOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`
	BlockID  string `arg:"" help:"block ID to list"`
	Scan     bool   `help:"scan contents of block for duplicate trace IDs and other info (warning, can be intense)"`
}

func (cmd *listBlockCmd) Run(ctx *globalOptions) error {
	r, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	return dumpBlock(r, c, cmd.TenantID, time.Hour, cmd.BlockID, cmd.Scan)
}

func dumpBlock(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration, blockID string, scan bool) error {
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
	fmt.Println("Data Size     : ", humanize.Bytes(unifiedMeta.size))
	fmt.Println("Encoding      : ", unifiedMeta.encoding)
	fmt.Println("Level         : ", unifiedMeta.compactionLevel)
	fmt.Println("Window        : ", unifiedMeta.window)
	fmt.Println("Start         : ", unifiedMeta.start)
	fmt.Println("End           : ", unifiedMeta.end)
	fmt.Println("Duration      : ", fmt.Sprint(unifiedMeta.end.Sub(unifiedMeta.start).Round(time.Second)))
	fmt.Println("Age           : ", fmt.Sprint(time.Since(unifiedMeta.end).Round(time.Second)))

	if scan {
		fmt.Println("Scanning block contents.  Press CRTL+C to quit ...")

		en, err := tempodb_backend.ParseEncoding(unifiedMeta.encoding)
		if err != nil {
			return err
		}

		block, err := encoding.NewBackendBlock(&tempodb_backend.BlockMeta{
			Encoding: en,
			Version:  unifiedMeta.version,
			TenantID: tenantID,
			BlockID:  id,
		}, r)
		if err != nil {
			return err
		}

		iter, err := block.Iterator(uint32(2 * 1024 * 1024))
		if err != nil {
			return err
		}
		defer iter.Close()

		// Scanning stats
		i := 0
		dupe := 0
		maxObjSize := 0
		minObjSize := 0

		printStats := func() {
			fmt.Println()
			fmt.Println("Scanning results:")
			fmt.Println("Objects scanned : ", i)
			fmt.Println("Duplicates      : ", dupe)
			fmt.Println("Smallest object : ", humanize.Bytes(uint64(minObjSize)))
			fmt.Println("Largest object  : ", humanize.Bytes(uint64(maxObjSize)))
		}

		// Print stats on ctrl+c
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			printStats()
			os.Exit(0)
		}()

		prevID := make([]byte, 16)
		for {
			objID, obj, err := iter.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			if len(obj) > maxObjSize {
				maxObjSize = len(obj)
			}

			if len(obj) < minObjSize || minObjSize == 0 {
				minObjSize = len(obj)
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

		printStats()
	}

	return nil
}
