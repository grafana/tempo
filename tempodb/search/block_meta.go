package search

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

type BlockMeta struct {
	Version       string           `json:"version"`
	Encoding      backend.Encoding `json:"encoding"` // Encoding/compression format
	IndexPageSize uint32           `json:"indexPageSize"`
	IndexRecords  uint32           `json:"indexRecords"`
}

const searchMetaObjectName = "search.meta.json"

func WriteSearchBlockMeta(ctx context.Context, w backend.Writer, blockID uuid.UUID, tenantID string, sm *BlockMeta) error {
	metaBytes, err := json.Marshal(sm)
	if err != nil {
		return err
	}

	err = w.Write(ctx, searchMetaObjectName, blockID, tenantID, metaBytes, false)
	return err
}

func ReadSearchBlockMeta(ctx context.Context, r backend.Reader, blockID uuid.UUID, tenantID string) (*BlockMeta, error) {
	metaBytes, err := r.Read(ctx, searchMetaObjectName, blockID, tenantID, false)
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
