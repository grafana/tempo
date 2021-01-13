package encoding

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grafana/tempo/tempodb/backend"
)

const (
	nameObjects     = "data"
	nameIndex       = "index"
	nameBloomPrefix = "bloom-"
)

func bloomName(shard int) string {
	return nameBloomPrefix + strconv.Itoa(shard)
}

func writeBlock(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, index []byte, blooms [][]byte) error {
	// index
	err := w.Write(ctx, nameIndex, meta.BlockID, meta.TenantID, index)
	if err != nil {
		return fmt.Errorf("unexpected error writing index %w", err)
	}

	// bloom
	for i, bloom := range blooms {
		err := w.Write(ctx, bloomName(i), meta.BlockID, meta.TenantID, bloom)
		if err != nil {
			return fmt.Errorf("unexpected error writing bloom-%d %w", i, err)
		}
	}

	// meta
	err = w.WriteBlockMeta(ctx, meta)
	if err != nil {
		return fmt.Errorf("unexpected error writing meta %w", err)
	}

	return nil
}
