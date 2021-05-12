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
	Version         string    `json:"format"`          // Version indicates the block format version. This includes specifics of how the indexes and data is stored
	BlockID         uuid.UUID `json:"blockID"`         // Unique block id
	MinID           []byte    `json:"minID"`           // Minimum object id stored in this block
	MaxID           []byte    `json:"maxID"`           // Maximum object id stored in this block
	TenantID        string    `json:"tenantID"`        // ID of tehant to which this block belongs
	StartTime       time.Time `json:"startTime"`       // Currently mostly meaningless but roughly matches to the time the first obj was written to this block
	EndTime         time.Time `json:"endTime"`         // Currently mostly meaningless but roughly matches to the time the last obj was written to this block
	TotalObjects    int       `json:"totalObjects"`    // Total objects in this block
	Size            uint64    `json:"size"`            // Total size in bytes of the data object
	CompactionLevel uint8     `json:"compactionLevel"` // Kind of the number of times this block has been compacted
	Encoding        Encoding  `json:"encoding"`        // Encoding/compression format
	IndexPageSize   uint32    `json:"indexPageSize"`   // Size of each index page in bytes
	TotalRecords    uint32    `json:"totalRecords"`    // Total Records stored in the index file
	DataEncoding    string    `json:"dataEncoding"`    // DataEncoding is a string provided externally, but tracked by tempodb that indicates the way the bytes are encoded
	BloomShardCount uint8     `json:"bloomShardCount"` //
}

func NewBlockMeta(tenantID string, blockID uuid.UUID, version string, encoding Encoding, dataEncoding string) *BlockMeta {
	now := time.Now()
	b := &BlockMeta{
		Version:      version,
		BlockID:      blockID,
		MinID:        []byte{},
		MaxID:        []byte{},
		TenantID:     tenantID,
		StartTime:    now,
		EndTime:      now,
		Encoding:     encoding,
		DataEncoding: dataEncoding,
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
