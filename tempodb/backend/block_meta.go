package backend

import (
	"bytes"
	"time"

	"github.com/google/uuid"
)

type CompactedBlockMeta struct {
	BlockMeta

	CompactedTime time.Time `json:"-"`
}

type BlockMeta struct {
	Version         string    `json:"format"`
	BlockID         uuid.UUID `json:"blockID"`
	MinID           []byte    `json:"minID"`
	MaxID           []byte    `json:"maxID"`
	TenantID        string    `json:"tenantID"`
	StartTime       time.Time `json:"startTime"`
	EndTime         time.Time `json:"endTime"`
	TotalObjects    int       `json:"totalObjects"`
	CompactionLevel uint8     `json:"compactionLevel"`
	Encoding        Encoding  `json:"encoding"`
}

func NewBlockMeta(tenantID string, blockID uuid.UUID) *BlockMeta {
	now := time.Now()
	b := &BlockMeta{
		Version:   "v0", // jpe - think about this
		BlockID:   blockID,
		MinID:     []byte{},
		MaxID:     []byte{},
		TenantID:  tenantID,
		StartTime: now,
		EndTime:   now,
	}

	return b
}

func (b *BlockMeta) ObjectAdded(id []byte) {
	b.EndTime = time.Now()

	if len(b.MinID) == 0 || bytes.Compare(id, b.MinID) == -1 {
		b.MinID = id
	}

	if len(b.MaxID) == 0 || bytes.Compare(id, b.MaxID) == 1 {
		b.MaxID = id
	}

	b.TotalObjects++
}
