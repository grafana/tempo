package backend

import (
	"bytes"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

// DedicatedColumnType is the type of the values in the dedicated attribute column. Only 'string' is supported.
type DedicatedColumnType string

// DedicatedColumnScope is the scope of the attribute that is stored in a dedicated column. Possible values are
// 'resource' and 'span'.
type DedicatedColumnScope string

const (
	DedicatedColumnTypeString DedicatedColumnType = "string"

	DedicatedColumnScopeResource DedicatedColumnScope = "resource"
	DedicatedColumnScopeSpan     DedicatedColumnScope = "span"
)

func DedicatedColumnTypeFromTempopb(t tempopb.DedicatedColumn_Type) (DedicatedColumnType, error) {
	switch t {
	case tempopb.DedicatedColumn_STRING:
		return DedicatedColumnTypeString, nil
	default:
		return "", errors.Errorf("invalid value for tempopb.DedicatedColumn_Type '%v'", t)
	}
}

func (t DedicatedColumnType) ToTempopb() (tempopb.DedicatedColumn_Type, error) {
	switch t {
	case DedicatedColumnTypeString:
		return tempopb.DedicatedColumn_STRING, nil
	default:
		return 0, errors.Errorf("invalid value for DedicatedColumnType '%v'", t)
	}
}

func (t DedicatedColumnType) ToStaticType() (traceql.StaticType, error) {
	switch t {
	case DedicatedColumnTypeString:
		return traceql.TypeString, nil
	default:
		return traceql.TypeNil, errors.Errorf("unsupported dedicated column type '%s'", t)
	}
}

func DedicatedColumnScopeFromTempopb(s tempopb.DedicatedColumn_Scope) (DedicatedColumnScope, error) {
	switch s {
	case tempopb.DedicatedColumn_SPAN:
		return DedicatedColumnScopeSpan, nil
	case tempopb.DedicatedColumn_RESOURCE:
		return DedicatedColumnScopeResource, nil
	default:
		return "", errors.Errorf("invalid value for tempopb.DedicatedColumn_Scope '%v'", s)
	}
}

func (s DedicatedColumnScope) ToTempopb() (tempopb.DedicatedColumn_Scope, error) {
	switch s {
	case DedicatedColumnScopeSpan:
		return tempopb.DedicatedColumn_SPAN, nil
	case DedicatedColumnScopeResource:
		return tempopb.DedicatedColumn_RESOURCE, nil
	default:
		return 0, errors.Errorf("invalid value for DedicatedColumnScope '%v'", s)
	}
}

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
	// DedicatedColumns configuration for attributes (used by vParquet3)
	DedicatedColumns []DedicatedColumn `json:"dedicatedColumns,omitempty"`
}

// DedicatedColumn contains the configuration for a single attribute with the given name that should
// be stored in a dedicated column instead of the generic attribute column.
type DedicatedColumn struct {
	// The Scope of the attribute
	Scope DedicatedColumnScope `yaml:"scope" json:"scope"`
	// The Name of the attribute stored in the dedicated column
	Name string `yaml:"name" json:"name"`
	// The Type of attribute value
	Type DedicatedColumnType `yaml:"type" json:"type"`
}

func NewBlockMeta(tenantID string, blockID uuid.UUID, version string, encoding Encoding, dataEncoding string) *BlockMeta {
	return NewBlockMetaWithDedicatedColumns(tenantID, blockID, version, encoding, dataEncoding, nil)
}

func NewBlockMetaWithDedicatedColumns(tenantID string, blockID uuid.UUID, version string, encoding Encoding, dataEncoding string, dc []DedicatedColumn) *BlockMeta {
	b := &BlockMeta{
		Version:          version,
		BlockID:          blockID,
		MinID:            []byte{},
		MaxID:            []byte{},
		TenantID:         tenantID,
		Encoding:         encoding,
		DataEncoding:     dataEncoding,
		DedicatedColumns: dc,
	}

	return b
}

// ObjectAdded updates the block meta appropriately based on information about an added record
// start/end are unix epoch seconds
func (b *BlockMeta) ObjectAdded(id []byte, start, end uint32) {
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

// TODO: Find a way to not compute the hash every time
//  Storing it bring some issues:
//    * It's not clear how to keep BlockMeta.DedicatedColumns and the hash in sync.
//    * BlockMeta can be initialized directly, without going through NewBlockMetaWithDedicatedColumns
//      (e.g. when loading from disk). We can do a lazy init, but we'd have little control over when
//      the hash is computed and that the field is not accessed directly.
//    * We could make the field private and add a method to access it, but that doesn't work well with JSON tags.

func (b *BlockMeta) DedicatedColumnsHash() uint64 { return HashColumns(b.DedicatedColumns) }

func DedicateColumnsFromTempopb(tempopbCols []*tempopb.DedicatedColumn) ([]DedicatedColumn, error) {
	cols := make([]DedicatedColumn, 0, len(tempopbCols))

	for _, c := range tempopbCols {
		scope, err := DedicatedColumnScopeFromTempopb(c.Scope)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert dedicated column '%s'", c.Name)
		}

		typ, err := DedicatedColumnTypeFromTempopb(c.Type)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert dedicated column '%s'", c.Name)
		}

		cols = append(cols, DedicatedColumn{
			Scope: scope,
			Name:  c.Name,
			Type:  typ,
		})
	}

	return cols, nil
}

func DedicateColumnsToTempopb(metaCols []DedicatedColumn) ([]*tempopb.DedicatedColumn, error) {
	tempopbCols := make([]*tempopb.DedicatedColumn, 0, len(metaCols))

	for _, c := range metaCols {
		scope, err := c.Scope.ToTempopb()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert dedicated column '%s'", c.Name)
		}

		typ, err := c.Type.ToTempopb()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert dedicated column '%s'", c.Name)
		}

		tempopbCols = append(tempopbCols, &tempopb.DedicatedColumn{
			Scope: scope,
			Name:  c.Name,
			Type:  typ,
		})
	}

	return tempopbCols, nil
}

// separatorByte is a byte that cannot occur in valid UTF-8 sequences
var separatorByte = []byte{255}

// HashColumns hashes the given dedicated columns configuration
func HashColumns(d []DedicatedColumn) uint64 {
	if len(d) == 0 {
		return 0
	}
	h := xxhash.New()
	for _, c := range d {
		_, _ = h.WriteString(string(c.Scope))
		_, _ = h.Write(separatorByte)
		_, _ = h.WriteString(c.Name)
		_, _ = h.Write(separatorByte)
		_, _ = h.WriteString(string(c.Type))
	}
	return h.Sum64()
}
