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

// The BlockMeta data that is stored for each individual block.
type BlockMeta struct {
	// A Version that indicates the block format. This includes specifics of how the indexes and data is stored.
	Version string `json:"format"`
	// BlockID is a unique identifier of the block.
	BlockID uuid.UUID `json:"blockID"`
	// MinID is the smallest object id stored in this block.
	MinID []byte `json:"minID"`
	// MaxID is the largest object id stored in this block.
	MaxID []byte `json:"maxID"`
	// A TenantID that defines the tenant to which this block belongs.
	TenantID string `json:"tenantID"`
	// StartTime roughly matches when the first obj was written to this block. It is used to determine block.
	// age for different purposes (caching, etc)
	StartTime time.Time `json:"startTime"`
	// EndTime roughly matches to the time the last obj was written to this block. Is currently mostly meaningless.
	EndTime time.Time `json:"endTime"`
	// TotalObjects counts the number of objects in this block.
	TotalObjects int `json:"totalObjects"`
	// The Size in bytes of the block.
	Size uint64 `json:"size"`
	// CompactionLevel defines the number of times this block has been compacted.
	CompactionLevel uint8 `json:"compactionLevel"`
	// Encoding and compression format (used only in v2)
	Encoding Encoding `json:"encoding"`
	// IndexPageSize holds the size of each index page in bytes (used only in v2)
	IndexPageSize uint32 `json:"indexPageSize"`
	// TotalRecords holds the total Records stored in the index file (used only in v2)
	TotalRecords uint32 `json:"totalRecords"`
	// DataEncoding is tracked by tempodb and indicates the way the bytes are encoded.
	DataEncoding string `json:"dataEncoding"`
	// BloomShardCount represents the number of bloom filter shards.
	BloomShardCount uint16 `json:"bloomShards"`
	// FooterSize contains the size of the footer in bytes (used by parquet)
	FooterSize uint32 `json:"footerSize"`
	// DedicatedColumns configuration for attributes (used by parquet)
	DedicatedColumns []DedicatedColumn `json:"dedicatedColumns,omitempty"`
}

// DedicatedColumn contains the configuration for a single attribute with the given name that should
// be stored in a dedicated column instead of the generic attribute column.
type DedicatedColumn struct {
	// The Scope of the attribute: can be 'resource' or 'span'
	Scope string `json:"scope"`
	// The Name of the attribute stored in the dedicated column
	Name string `json:"name"`
	// The Type of attribute value: only 'string' supported
	Type string `json:"type"`
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
// start/end are unix epoch seconds
func (b *BlockMeta) ObjectAdded(id []byte, start uint32, end uint32) {

	if start > 0 {
		startTime := time.Unix(int64(start), 0)
		if b.StartTime.IsZero() || startTime.Before(b.StartTime) {
			b.StartTime = startTime
		}
	}

	if end > 0 {
		endTime := time.Unix(int64(end), 0)
		if b.EndTime.IsZero() || endTime.After(b.EndTime) {
			b.EndTime = endTime
		}
	}

	if len(b.MinID) == 0 || bytes.Compare(id, b.MinID) == -1 {
		b.MinID = id
	}
	if len(b.MaxID) == 0 || bytes.Compare(id, b.MaxID) == 1 {
		b.MaxID = id
	}

	b.TotalObjects++
}
