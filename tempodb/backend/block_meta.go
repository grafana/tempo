package backend

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

// DedicatedColumnType is the type of the values in the dedicated attribute column. Only 'string' is supported.
type DedicatedColumnType string

// DedicatedColumnScope is the scope of the attribute that is stored in a dedicated column. Possible values are
// 'resource' and 'span'.
type DedicatedColumnScope string

// DedicatedColumnOption is a string option applied to a dedicated column
type DedicatedColumnOption string

// DedicatedColumnOptions is a list of options applied to a dedicated column
type DedicatedColumnOptions []DedicatedColumnOption

const (
	DedicatedColumnTypeString DedicatedColumnType = "string"
	DedicatedColumnTypeInt    DedicatedColumnType = "int"

	DedicatedColumnScopeResource DedicatedColumnScope = "resource"
	DedicatedColumnScopeSpan     DedicatedColumnScope = "span"

	DedicatedColumnOptionArray DedicatedColumnOption = "array"

	DefaultDedicatedColumnType  = DedicatedColumnTypeString
	DefaultDedicatedColumnScope = DedicatedColumnScopeSpan
)

var maxSupportedColumns = map[DedicatedColumnType]map[DedicatedColumnScope]int{
	DedicatedColumnTypeString: {DedicatedColumnScopeSpan: 10, DedicatedColumnScopeResource: 10},
	DedicatedColumnTypeInt:    {DedicatedColumnScopeSpan: 5, DedicatedColumnScopeResource: 5},
}

func DedicatedColumnTypeFromTempopb(t tempopb.DedicatedColumn_Type) (DedicatedColumnType, error) {
	switch t {
	case tempopb.DedicatedColumn_STRING:
		return DedicatedColumnTypeString, nil
	case tempopb.DedicatedColumn_INT:
		return DedicatedColumnTypeInt, nil
	default:
		return "", fmt.Errorf("invalid value for tempopb.DedicatedColumn_Type '%v'", t)
	}
}

func (t DedicatedColumnType) ToTempopb() (tempopb.DedicatedColumn_Type, error) {
	switch t {
	case DedicatedColumnTypeString:
		return tempopb.DedicatedColumn_STRING, nil
	case DedicatedColumnTypeInt:
		return tempopb.DedicatedColumn_INT, nil
	default:
		return 0, fmt.Errorf("invalid value for dedicated column type '%v'", t)
	}
}

func (t DedicatedColumnType) ToStaticType() (traceql.StaticType, error) {
	switch t {
	case DedicatedColumnTypeString:
		return traceql.TypeString, nil
	case DedicatedColumnTypeInt:
		return traceql.TypeInt, nil
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

func DedicatedColumnOptionsFromTempopb(o tempopb.DedicatedColumn_Option) DedicatedColumnOptions {
	var options DedicatedColumnOptions
	if o&tempopb.DedicatedColumn_ARRAY != 0 {
		options = append(options, DedicatedColumnOptionArray)
	}
	return options
}

func (o DedicatedColumnOptions) ToTempopb() (tempopb.DedicatedColumn_Option, error) {
	var optionsMask tempopb.DedicatedColumn_Option
	for _, opt := range o {
		switch opt {
		case DedicatedColumnOptionArray:
			optionsMask |= tempopb.DedicatedColumn_ARRAY
		default:
			return 0, fmt.Errorf("invalid value for dedicated column option '%v'", opt)
		}
	}
	return optionsMask, nil
}

const (
	DefaultReplicationFactor          = 0 // Replication factor for blocks from the ingester. This is the default value to indicate RF3.
	MetricsGeneratorReplicationFactor = 1
	LiveStoreReplicationFactor        = MetricsGeneratorReplicationFactor
)

var defaultDedicatedColumns = DedicatedColumns{
	// resource
	{Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString, Name: "k8s.cluster.name"},
	{Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString, Name: "k8s.namespace.name"},
	{Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString, Name: "k8s.pod.name"},
	{Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString, Name: "k8s.container.name"},
	// span
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Name: "http.request.method"},
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt, Name: "http.response.status_code"},
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Name: "url.path"},
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Name: "url.route"},
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Name: "server.address"},
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt, Name: "server.port"},
	// legacy
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Name: "http.method"},
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Name: "http.url"},
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Name: "http.route"},
	{Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt, Name: "http.status_code"},
}

func DefaultDedicatedColumns() DedicatedColumns {
	return slices.Clone(defaultDedicatedColumns)
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
	// The Options applied to the dedicated attribute column
	Options DedicatedColumnOptions `yaml:"options" json:"o,omitempty"`
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
	// get the pre-unmarshalled data if available
	v, ok := getDedicatedColumnsFromCache(b)
	if ok && v != nil {
		*dcs = v
		return nil
	}

	type dcsAlias DedicatedColumns // alias required to avoid recursive calls of UnmarshalJSON
	err := json.Unmarshal(b, (*dcsAlias)(dcs))
	if err != nil {
		return err
	}

	putDedicatedColumnsToCache(b, *dcs)
	return nil
}

func NewBlockMeta(tenantID string, blockID uuid.UUID, version string, encoding Encoding, dataEncoding string) *BlockMeta {
	return NewBlockMetaWithDedicatedColumns(tenantID, blockID, version, encoding, dataEncoding, nil)
}

func NewBlockMetaWithDedicatedColumns(tenantID string, blockID uuid.UUID, version string, encoding Encoding, dataEncoding string, dc DedicatedColumns) *BlockMeta {
	b := &BlockMeta{
		Version:          version,
		BlockID:          UUID(blockID),
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

		options := DedicatedColumnOptionsFromTempopb(c.Options)

		cols = append(cols, DedicatedColumn{
			Scope:   scope,
			Name:    c.Name,
			Type:    typ,
			Options: options,
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

		options, err := c.Options.ToTempopb()
		if err != nil {
			return nil, fmt.Errorf("unable to convert dedicated column '%s': %w", c.Name, err)
		}

		tempopbCols = append(tempopbCols, &tempopb.DedicatedColumn{
			Scope:   scope,
			Name:    c.Name,
			Type:    typ,
			Options: options,
		})
	}

	return tempopbCols, nil
}

func (dcs DedicatedColumns) Validate() error {
	columnCount := map[DedicatedColumnType]map[DedicatedColumnScope]int{}
	nameCount := map[DedicatedColumnScope]map[string]struct{}{}

	for _, dc := range dcs {
		if dc.Name == "" {
			return fmt.Errorf("invalid dedicated attribute columns: empty name")
		}

		// check for duplicate names
		if names, ok := nameCount[dc.Scope]; !ok {
			nameCount[dc.Scope] = map[string]struct{}{dc.Name: {}}
		} else {
			if _, duplicate := names[dc.Name]; duplicate {
				return fmt.Errorf("invalid dedicated attribute columns: duplicate name '%s' for scope '%s'", dc.Name, dc.Scope)
			}
			names[dc.Name] = struct{}{}
		}

		// count columns by type and scope
		if scopes, ok := columnCount[dc.Type]; !ok {
			columnCount[dc.Type] = map[DedicatedColumnScope]int{dc.Scope: 1}
		} else {
			scopes[dc.Scope]++
		}

		for _, opt := range dc.Options {
			if opt != DedicatedColumnOptionArray {
				return fmt.Errorf("invalid dedicated attribute columns: invalid option '%s'", opt)
			}
		}
	}

	// check max number of columns by type and scope
	for typ, scopes := range columnCount {
		for scope, count := range scopes {
			supportedScopes, ok := maxSupportedColumns[typ]
			if !ok {
				return fmt.Errorf("invalid dedicated attribute columns: unsupported dedicated column type '%s'", typ)
			}
			maxCount, ok := supportedScopes[scope]
			if !ok {
				return fmt.Errorf("invalid dedicated attribute columns: unsupported dedicated column scope '%s'", scope)
			}
			if count > maxCount {
				return fmt.Errorf("invalid dedicated attribute columns: number of columns with type '%s' and scope '%s' must be <= %d but was %d", typ, scope, maxCount, count)
			}
		}
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
		for _, opt := range c.Options {
			_, _ = h.Write(separatorByte)
			_, _ = h.WriteString(string(opt))
		}
	}
	return h.Sum64()
}

func (dcs DedicatedColumns) Size() int {
	if len(dcs) == 0 {
		return 0
	}

	b, _ := dcs.Marshal()
	return len(b)
}

func (dcs DedicatedColumns) Marshal() ([]byte, error) {
	if len(dcs) == 0 {
		return nil, nil
	}

	// NOTE: The json bytes interned in a map to avoid re-unmarshalling the same byte slice.
	return json.Marshal(dcs)
}

func (dcs DedicatedColumns) MarshalTo(data []byte) (n int, err error) {
	bb, err := dcs.Marshal()
	if err != nil {
		return 0, err
	}
	copy(data, bb)

	return len(bb), nil
}

func (dcs *DedicatedColumns) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// NOTE: The json bytes interned in a map to avoid re-unmarshalling the same byte slice.
	return json.Unmarshal(data, &dcs)
}

func (b *CompactedBlockMeta) UnmarshalJSON(data []byte) error {
	var msg interface{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return err
	}
	msgMap := msg.(map[string]interface{})

	if v, ok := msgMap["compactedTime"]; ok {
		b.CompactedTime, err = time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return fmt.Errorf("failed to parse time at compactedTime: %w", err)
		}
	}

	if err := json.Unmarshal(data, &b.BlockMeta); err != nil {
		return fmt.Errorf("failed at unmarshal for dedicated columns: %w", err)
	}

	return nil
}
