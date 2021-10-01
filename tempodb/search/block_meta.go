package search

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
)

type BlockMeta struct {
	Version       string           `json:"version"`
	Encoding      backend.Encoding `json:"v"` // Encoding/compression format
	IndexPageSize uint32           `json:"indexPageSize"`
	IndexRecords  uint32           `json:"indexRecords"`
}

const searchMetaObjectName = "search.meta.json"

func WriteSearchBlockMeta(ctx context.Context, w backend.RawWriter, blockID uuid.UUID, tenantID string, sm *BlockMeta) error {
	metaBytes, err := json.Marshal(sm)
	if err != nil {
		return err
	}

	err = w.Write(ctx, searchMetaObjectName, backend.KeyPathForBlock(blockID, tenantID), bytes.NewReader(metaBytes), int64(len(metaBytes)), false)
	return err
}

func ReadSearchBlockMeta(ctx context.Context, r backend.RawReader, blockID uuid.UUID, tenantID string) (*BlockMeta, error) {
	metaReader, size, err := r.Read(ctx, searchMetaObjectName, backend.KeyPathForBlock(blockID, tenantID), false)
	if err != nil {
		return nil, err
	}

	defer metaReader.Close()
	metaBytes, err := tempo_io.ReadAllWithEstimate(metaReader, size)
	if err != nil {
		return nil, err
	}

	meta := &BlockMeta{}
	err = json.Unmarshal(metaBytes, meta)
	if err != nil {
		return nil, err
	}

	return meta, nil
}
