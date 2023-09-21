package main

import (
	"context"
	"fmt"

	"github.com/dustin/go-humanize"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

type migrateTenantCmd struct {
	SourceConfigFile string `type:"path" required:"" help:"Path to tempo config file for source"`

	SourceTenantID string `arg:"" help:"source tenant-id"`
	DestTenantID   string `arg:"" help:"dest tenant-id"`
}

func (cmd *migrateTenantCmd) Run(opts *globalOptions) error {
	ctx := context.Background()

	readerSource, readerDest, writerDest, err := cmd.setupBackends(opts)
	if err != nil {
		return fmt.Errorf("setting up backends: %w", err)
	}
	defer func() {
		readerSource.Shutdown()
		readerDest.Shutdown()
	}()

	sourceTenantIndex, err := readerSource.TenantIndex(ctx, cmd.SourceTenantID)
	if err != nil {
		return fmt.Errorf("reading source tenant index: %w", err)
	}
	fmt.Printf("Blocks in source: %d, compacted: %d\n", len(sourceTenantIndex.Meta), len(sourceTenantIndex.CompactedMeta))

	// TODO create dest directory if it doesn't exist yet?

	blocksDest, err := readerDest.Blocks(ctx, cmd.DestTenantID)
	if err != nil {
		return err
	}
	fmt.Printf("Blocks in destination: %d\n", len(blocksDest))

	var copiedBlocks, copiedSize uint64

blocks:
	for _, sourceBlockMeta := range sourceTenantIndex.Meta {
		// check for collisions
		for _, uuidDest := range blocksDest {
			if sourceBlockMeta.BlockID == uuidDest {
				fmt.Printf("UUID %s exists in source and destination, skipping block\n", sourceBlockMeta.BlockID)
				continue blocks
			}
		}

		// create a copy with destination tenant ID
		destBlockMeta := *sourceBlockMeta
		destBlockMeta.TenantID = cmd.DestTenantID

		encoder, err := encoding.FromVersion(sourceBlockMeta.Version)
		if err != nil {
			return fmt.Errorf("creating encoder from version: %w", err)
		}

		err = encoder.MigrateBlock(ctx, sourceBlockMeta, &destBlockMeta, readerSource, writerDest)
		if err != nil {
			return fmt.Errorf("copying block: %w", err)
		}

		copiedBlocks++
		copiedSize += sourceBlockMeta.Size
	}

	fmt.Printf("Finished migrating data. Copied %d blocks, %s\n", copiedBlocks, humanize.Bytes(copiedSize))
	return nil
}

func (cmd *migrateTenantCmd) setupBackends(optsDest *globalOptions) (readerSource, readerDest backend.Reader, writerDest backend.Writer, err error) {
	emptyBackendOptions := backendOptions{}

	optsSource := &globalOptions{
		ConfigFile: cmd.SourceConfigFile,
	}
	readerSource, _, _, err = loadBackend(&emptyBackendOptions, optsSource)
	if err != nil {
		return
	}

	readerDest, writerDest, _, err = loadBackend(&emptyBackendOptions, optsDest)
	return
}
