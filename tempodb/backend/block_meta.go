package backend

import (
	"bytes"
	"time"

	"github.com/google/uuid"
)

type CompactedBlockMeta struct {
	BlockMeta

	CompactedTime time.Time `json:"compactedTime"`
}

type BlockMeta struct {
	Version         string    `json:"format"`          // Version indicates the block format version. This includes specifics of how the indexes and data is stored
	BlockID         uuid.UUID `json:"blockID"`         // Unique block id
	MinID           []byte    `json:"minID"`           // Minimum object id stored in this block
	MaxID           []byte    `json:"maxID"`           // Maximum object id stored in this block
	TenantID        string    `json:"tenantID"`        // ID of tehant to which this block belongs
	StartTime       time.Time `json:"startTime"`       // Roughly matches when the first obj was written to this block. Used to determine block age for different purposes (cacheing, etc)
	EndTime         time.Time `json:"endTime"`         // Currently mostly meaningless but roughly matches to the time the last obj was written to this block
	TotalObjects    int       `json:"totalObjects"`    // Total objects in this block
	Size            uint64    `json:"size"`            // Total size in bytes of the data object
	CompactionLevel uint8     `json:"compactionLevel"` // Kind of the number of times this block has been compacted
	Encoding        Encoding  `json:"encoding"`        // Encoding/compression format
	IndexPageSize   uint32    `json:"indexPageSize"`   // Size of each index page in bytes
	TotalRecords    uint32    `json:"totalRecords"`    // Total Records stored in the index file
	DataEncoding    string    `json:"dataEncoding"`    // DataEncoding is a string provided externally, but tracked by tempodb that indicates the way the bytes are encoded
	BloomShardCount uint16    `json:"bloomShards"`     // Number of bloom filter shards
}

func NewBlockMeta(tenantID string, blockID uuid.UUID, version string, encoding Encoding, dataEncoding string) *BlockMeta {
	b := &BlockMeta{
		Version:      version,
		BlockID:      blockID,
		MinID:        []byte{},
		MaxID:        []byte{},
		TenantID:     tenantID,
		Encoding:     encoding,
		DataEncoding: dataEncoding,
	}

	return b
}

// ObjectAdded updates the block meta appropriately based on information about an added record
//  start/end are unix epoch seconds
func (b *BlockMeta) ObjectAdded(id []byte, start uint32, end uint32) {
	startTime := time.Unix(int64(start), 0)
	endTime := time.Unix(int64(end), 0)

	if b.StartTime.IsZero() || startTime.Before(b.StartTime) {
		b.StartTime = startTime
	}
	if b.EndTime.IsZero() || endTime.After(b.EndTime) {
		b.EndTime = endTime
	}

	if len(b.MinID) == 0 || bytes.Compare(id, b.MinID) == -1 {
		b.MinID = id
	}
	if len(b.MaxID) == 0 || bytes.Compare(id, b.MaxID) == 1 {
		b.MaxID = id
	}

	b.TotalObjects++
}
