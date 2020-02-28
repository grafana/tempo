package local

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/pkg/util"
)

type compactor struct {
	rw *readerWriter
}

func NewCompactor(cfg *Config) (backend.Compactor, error) {
	rw, err := newReaderWriter(cfg)
	if err != nil {
		return nil, err
	}

	return &compactor{
		rw: rw,
	}, nil
}

func (c *compactor) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	// move meta file to a new location
	metaFilename := c.rw.metaFileName(blockID, tenantID)
	compactedMetaFilename := util.CompactedMetaFileName(blockID, tenantID)

	return os.Rename(metaFilename, compactedMetaFilename)
}

func (c *compactor) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return fmt.Errorf("empty tenant id")
	}

	if blockID == uuid.Nil {
		return fmt.Errorf("empty block id")
	}

	return os.RemoveAll(util.RootPath(c.rw.cfg.Path, tenantID, blockID))
}
