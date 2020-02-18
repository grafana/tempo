package friggdb

import (
	"bytes"
	"time"

	"github.com/google/uuid"
)

type searchableBlockMeta struct {
	Version string    `json:"format"`
	BlockID uuid.UUID `json:"blockID"`
	MinID   ID        `json:"minID"`
	MaxID   ID        `json:"maxID"`
}

type blockMeta struct {
	searchableBlockMeta
	TenantID     string    `json:"tenantID"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	CreationTime time.Time `json:"creationTime"`
}

func newBlockMeta(tenantID string, blockID uuid.UUID) *blockMeta {
	now := time.Now()
	b := &blockMeta{
		searchableBlockMeta: searchableBlockMeta{
			Version: "v0",
			BlockID: blockID,
			MinID:   []byte{},
			MaxID:   []byte{},
		},
		TenantID:     tenantID,
		StartTime:    now,
		EndTime:      now,
		CreationTime: now,
	}

	return b
}

func (b *blockMeta) objectAdded(id ID) {
	b.EndTime = time.Now()

	if len(b.MinID) == 0 || bytes.Compare(id, b.MinID) == -1 {
		b.MinID = id
	}

	if len(b.MaxID) == 0 || bytes.Compare(id, b.MaxID) == 1 {
		b.MaxID = id
	}
}
