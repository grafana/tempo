package search

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
)

type SearchBlockMeta struct {
	Version       string `json:"version"`
	IndexPageSize uint32 `json:"indexPageSize"`
	IndexRecords  uint32 `json:"indexRecords"`
}

const searchMetaObjectName = "search.meta.json"

func WriteSearchBlockMeta(ctx context.Context, w backend.RawWriter, blockID uuid.UUID, tenantID string, sm *SearchBlockMeta) error {
	metaJson, err := json.Marshal(sm)
	if err != nil {
		return err
	}

	err = w.Write(ctx, searchMetaObjectName, backend.KeyPathForBlock(blockID, tenantID), bytes.NewReader(metaJson), int64(len(metaJson)), false)
	return err
}

func ReadSearchBlockMeta(ctx context.Context, r backend.RawReader, blockID uuid.UUID, tenantID string) (*SearchBlockMeta, error) {
	metaReader, size, err := r.Read(ctx, searchMetaObjectName, backend.KeyPathForBlock(blockID, tenantID), false)
	if err != nil {
		return nil, err
	}

	defer metaReader.Close()
	metaBytes, err := tempo_io.ReadAllWithEstimate(metaReader, size)
	if err != nil {
		return nil, err
	}

	meta := &SearchBlockMeta{}
	err = json.Unmarshal(metaBytes, meta)
	if err != nil {
		return nil, err
	}

	return meta, nil
}
