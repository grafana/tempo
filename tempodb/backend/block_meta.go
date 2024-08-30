package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	backend_v1 "github.com/grafana/tempo/tempodb/backend/v1"
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

	DefaultDedicatedColumnType  = DedicatedColumnTypeString
	DefaultDedicatedColumnScope = DedicatedColumnScopeSpan

	maxSupportedSpanColumns     = 10
	maxSupportedResourceColumns = 10
)

func DedicatedColumnTypeFromTempopb(t tempopb.DedicatedColumn_Type) (DedicatedColumnType, error) {
	switch t {
	case tempopb.DedicatedColumn_STRING:
		return DedicatedColumnTypeString, nil
	default:
		return "", fmt.Errorf("invalid value for tempopb.DedicatedColumn_Type '%v'", t)
	}
}

func (t DedicatedColumnType) ToTempopb() (tempopb.DedicatedColumn_Type, error) {
	switch t {
	case DedicatedColumnTypeString:
		return tempopb.DedicatedColumn_STRING, nil
	default:
		return 0, fmt.Errorf("invalid value for dedicated column type '%v'", t)
	}
}

func (t DedicatedColumnType) ToStaticType() (traceql.StaticType, error) {
	switch t {
	case DedicatedColumnTypeString:
		return traceql.TypeString, nil
	default:
		return traceql.TypeNil, fmt.Errorf("unsupported dedicated column type '%s'", t)
	}
}

func DedicatedColumnScopeFromTempopb(s tempopb.DedicatedColumn_Scope) (DedicatedColumnScope, error) {
	switch s {
	case tempopb.DedicatedColumn_SPAN:
		return DedicatedColumnScopeSpan, nil
	case tempopb.DedicatedColumn_RESOURCE:
		return DedicatedColumnScopeResource, nil
	default:
		return "", fmt.Errorf("invalid value for tempopb.DedicatedColumn_Scope '%v'", s)
	}
}

func (s DedicatedColumnScope) ToTempopb() (tempopb.DedicatedColumn_Scope, error) {
	switch s {
	case DedicatedColumnScopeSpan:
		return tempopb.DedicatedColumn_SPAN, nil
	case DedicatedColumnScopeResource:
		return tempopb.DedicatedColumn_RESOURCE, nil
	default:
		return 0, fmt.Errorf("invalid value for dedicated column scope '%v'", s)
	}
}

type CompactedBlockMeta struct {
	CompactedTime time.Time `json:"compactedTime"`
	BlockMeta
}

func (b *CompactedBlockMeta) ToBackendV1Proto() (*backend_v1.CompactedBlockMeta, error) {
	bm, err := b.BlockMeta.ToBackendV1Proto()
	if err != nil {
		return nil, err
	}

	return &backend_v1.CompactedBlockMeta{
		BlockMeta:     *bm,
		CompactedTime: b.CompactedTime,
	}, nil
}

func (b *CompactedBlockMeta) FromBackendV1Proto(pb *backend_v1.CompactedBlockMeta) error {
	err := b.BlockMeta.FromBackendV1Proto(&pb.BlockMeta)
	if err != nil {
		return err
	}

	b.CompactedTime = pb.CompactedTime

	return nil
}

const (
	DefaultReplicationFactor          = 0 // Replication factor for blocks from the ingester. This is the default value to indicate RF3.
	MetricsGeneratorReplicationFactor = 1
)

// The BlockMeta data that is stored for each individual block.
type BlockMeta struct {
	// A Version that indicates the block format. This includes specifics of how the indexes and data is stored.
	Version string `json:"format"`
	// BlockID is a unique identifier of the block.
	BlockID uuid.UUID `json:"blockID"`
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
	DedicatedColumns DedicatedColumns `json:"dedicatedColumns,omitempty"`
	// ReplicationFactor is the number of times the data written in this block has been replicated.
	// It's left unset if replication factor is 3. Default is 0 (RF3).
	ReplicationFactor uint8 `json:"replicationFactor,omitempty"`
}

// DedicatedColumn contains the configuration for a single attribute with the given name that should
// be stored in a dedicated column instead of the generic attribute column.
type DedicatedColumn struct {
	// The Scope of the attribute
	Scope DedicatedColumnScope `yaml:"scope" json:"s,omitempty"`
	// The Name of the attribute stored in the dedicated column
	Name string `yaml:"name" json:"n"`
	// The Type of attribute value
	Type DedicatedColumnType `yaml:"type" json:"t,omitempty"`
}

func (dc *DedicatedColumn) MarshalJSON() ([]byte, error) {
	type dcAlias DedicatedColumn // alias required to avoid recursive calls of MarshalJSON

	cpy := (dcAlias)(*dc)
	if cpy.Scope == DefaultDedicatedColumnScope {
		cpy.Scope = ""
	}
	if cpy.Type == DefaultDedicatedColumnType {
		cpy.Type = ""
	}
	return json.Marshal(&cpy)
}

func (dc *DedicatedColumn) UnmarshalJSON(b []byte) error {
	type dcAlias DedicatedColumn // alias required to avoid recursive calls of UnmarshalJSON

	err := json.Unmarshal(b, (*dcAlias)(dc))
	if err != nil {
		return err
	}
	if dc.Scope == "" {
		dc.Scope = DefaultDedicatedColumnScope
	}
	if dc.Type == "" {
		dc.Type = DefaultDedicatedColumnType
	}
	return nil
}

// DedicatedColumns represents a set of configured dedicated columns.
type DedicatedColumns []DedicatedColumn

func (dcs *DedicatedColumns) UnmarshalJSON(b []byte) error {
	// Get the pre-unmarshalled data if available.
	v := getDedicatedColumns(string(b))
	if v != nil {
		*dcs = *v
		return nil
	}

	type dcsAlias DedicatedColumns // alias required to avoid recursive calls of UnmarshalJSON
	err := json.Unmarshal(b, (*dcsAlias)(dcs))
	if err != nil {
		return err
	}

	putDedicatedColumns(string(b), dcs)

	return nil
}

func NewBlockMeta(tenantID string, blockID uuid.UUID, version string, encoding Encoding, dataEncoding string) *BlockMeta {
	return NewBlockMetaWithDedicatedColumns(tenantID, blockID, version, encoding, dataEncoding, nil)
}

func NewBlockMetaWithDedicatedColumns(tenantID string, blockID uuid.UUID, version string, encoding Encoding, dataEncoding string, dc DedicatedColumns) *BlockMeta {
	b := &BlockMeta{
		Version:          version,
		BlockID:          blockID,
		TenantID:         tenantID,
		Encoding:         encoding,
		DataEncoding:     dataEncoding,
		DedicatedColumns: dc,
	}

	return b
}

// ObjectAdded updates the block meta appropriately based on information about an added record
// start/end are unix epoch seconds, when 0 the start and the end are not applied.
func (b *BlockMeta) ObjectAdded(start, end uint32) {
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

	b.TotalObjects++
}

func (b *BlockMeta) DedicatedColumnsHash() uint64 {
	return b.DedicatedColumns.Hash()
}

func (b *BlockMeta) ToBackendV1Proto() (*backend_v1.BlockMeta, error) {
	blockID, err := b.BlockID.MarshalText()
	if err != nil {
		return nil, err
	}

	m := &backend_v1.BlockMeta{
		Version:           b.Version,
		BlockId:           blockID,
		TenantId:          b.TenantID,
		StartTime:         b.StartTime,
		EndTime:           b.EndTime,
		TotalObjects:      int32(b.TotalObjects),
		Size_:             b.Size,
		CompactionLevel:   uint32(b.CompactionLevel),
		Encoding:          int32(b.Encoding),
		IndexPageSize:     b.IndexPageSize,
		TotalRecords:      b.TotalRecords,
		DataEncoding:      b.DataEncoding,
		BloomShardCount:   uint32(b.BloomShardCount),
		FooterSize:        b.FooterSize,
		ReplicationFactor: uint32(b.ReplicationFactor),
	}

	dc, err := b.DedicatedColumns.ToTempopb()
	if err != nil {
		return nil, err
	}
	m.DedicatedColumns = dc

	return m, nil
}

func (b *BlockMeta) FromBackendV1Proto(pb *backend_v1.BlockMeta) error {
	blockID, err := uuid.ParseBytes(pb.BlockId)
	if err != nil {
		return err
	}

	b.Version = pb.Version
	b.BlockID = blockID
	b.TenantID = pb.TenantId
	b.StartTime = pb.StartTime
	b.EndTime = pb.EndTime
	b.TotalObjects = int(pb.TotalObjects)
	b.Size = pb.Size_
	b.CompactionLevel = uint8(pb.CompactionLevel)
	b.Encoding = Encoding(pb.Encoding)
	b.IndexPageSize = pb.IndexPageSize
	b.TotalRecords = pb.TotalRecords
	b.DataEncoding = pb.DataEncoding
	b.BloomShardCount = uint16(pb.BloomShardCount)
	b.FooterSize = pb.FooterSize
	b.ReplicationFactor = uint8(pb.ReplicationFactor)
	dcs, err := DedicatedColumnsFromTempopb(pb.DedicatedColumns)
	if err != nil {
		return err
	}

	if len(dcs) > 0 {
		b.DedicatedColumns = dcs
	}

	return nil
}

func DedicatedColumnsFromTempopb(tempopbCols []*tempopb.DedicatedColumn) (DedicatedColumns, error) {
	cols := make(DedicatedColumns, 0, len(tempopbCols))

	for _, c := range tempopbCols {
		scope, err := DedicatedColumnScopeFromTempopb(c.Scope)
		if err != nil {
			return nil, fmt.Errorf("unable to convert dedicated column '%s': %w", c.Name, err)
		}

		typ, err := DedicatedColumnTypeFromTempopb(c.Type)
		if err != nil {
			return nil, fmt.Errorf("unable to convert dedicated column '%s': %w", c.Name, err)
		}

		cols = append(cols, DedicatedColumn{
			Scope: scope,
			Name:  c.Name,
			Type:  typ,
		})
	}

	return cols, nil
}

func (dcs DedicatedColumns) ToTempopb() ([]*tempopb.DedicatedColumn, error) {
	tempopbCols := make([]*tempopb.DedicatedColumn, 0, len(dcs))

	for _, c := range dcs {
		scope, err := c.Scope.ToTempopb()
		if err != nil {
			return nil, fmt.Errorf("unable to convert dedicated column '%s': %w", c.Name, err)
		}

		typ, err := c.Type.ToTempopb()
		if err != nil {
			return nil, fmt.Errorf("unable to convert dedicated column '%s': %w", c.Name, err)
		}

		tempopbCols = append(tempopbCols, &tempopb.DedicatedColumn{
			Scope: scope,
			Name:  c.Name,
			Type:  typ,
		})
	}

	return tempopbCols, nil
}

func (dcs DedicatedColumns) Validate() error {
	var countSpan, countRes int
	for _, dc := range dcs {
		err := dc.Validate()
		if err != nil {
			return err
		}
		switch dc.Scope {
		case DedicatedColumnScopeSpan:
			countSpan++
		case DedicatedColumnScopeResource:
			countRes++
		}
	}
	if countSpan > maxSupportedSpanColumns {
		return fmt.Errorf("number of dedicated columns with scope 'span' must be <= %d but was %d", maxSupportedSpanColumns, countSpan)
	}
	if countRes > maxSupportedResourceColumns {
		return fmt.Errorf("number of dedicated columns with scope 'resource' must be <= %d but was %d", maxSupportedResourceColumns, countRes)
	}
	return nil
}

func (dc *DedicatedColumn) Validate() error {
	if dc.Name == "" {
		return errors.New("dedicated column invalid: name must not be empty")
	}
	_, err := dc.Type.ToTempopb()
	if err != nil {
		return fmt.Errorf("dedicated column '%s' invalid: %w", dc.Name, err)
	}
	_, err = dc.Scope.ToTempopb()
	if err != nil {
		return fmt.Errorf("dedicated column '%s' invalid: %w", dc.Name, err)
	}
	return nil
}

// separatorByte is a byte that cannot occur in valid UTF-8 sequences
var separatorByte = []byte{255}

// Hash hashes the given dedicated columns configuration
func (dcs DedicatedColumns) Hash() uint64 {
	if len(dcs) == 0 {
		return 0
	}
	h := xxhash.New()
	for _, c := range dcs {
		_, _ = h.WriteString(string(c.Scope))
		_, _ = h.Write(separatorByte)
		_, _ = h.WriteString(c.Name)
		_, _ = h.Write(separatorByte)
		_, _ = h.WriteString(string(c.Type))
	}
	return h.Sum64()
}
