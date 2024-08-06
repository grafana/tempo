package vparquet4

import (
	v1 "github.com/grafana/tempo/v2/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/v2/tempodb/backend"
)

// Column paths for spare dedicated attribute columns
var DedicatedResourceColumnPaths = map[backend.DedicatedColumnScope]map[backend.DedicatedColumnType][]string{
	backend.DedicatedColumnScopeResource: {
		backend.DedicatedColumnTypeString: {
			"rs.list.element.Resource.DedicatedAttributes.String01",
			"rs.list.element.Resource.DedicatedAttributes.String02",
			"rs.list.element.Resource.DedicatedAttributes.String03",
			"rs.list.element.Resource.DedicatedAttributes.String04",
			"rs.list.element.Resource.DedicatedAttributes.String05",
			"rs.list.element.Resource.DedicatedAttributes.String06",
			"rs.list.element.Resource.DedicatedAttributes.String07",
			"rs.list.element.Resource.DedicatedAttributes.String08",
			"rs.list.element.Resource.DedicatedAttributes.String09",
			"rs.list.element.Resource.DedicatedAttributes.String10",
		},
	},
	backend.DedicatedColumnScopeSpan: {
		backend.DedicatedColumnTypeString: {
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String01",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String02",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String03",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String04",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String05",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String06",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String07",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String08",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String09",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String10",
		},
	},
}

type dedicatedColumn struct {
	Type        backend.DedicatedColumnType
	ColumnPath  string
	ColumnIndex int
}

func (dc *dedicatedColumn) readValue(attrs *DedicatedAttributes) *v1.AnyValue {
	switch dc.Type {
	case backend.DedicatedColumnTypeString:
		var strVal *string
		switch dc.ColumnIndex {
		case 0:
			strVal = attrs.String01
		case 1:
			strVal = attrs.String02
		case 2:
			strVal = attrs.String03
		case 3:
			strVal = attrs.String04
		case 4:
			strVal = attrs.String05
		case 5:
			strVal = attrs.String06
		case 6:
			strVal = attrs.String07
		case 7:
			strVal = attrs.String08
		case 8:
			strVal = attrs.String09
		case 9:
			strVal = attrs.String10
		}
		if strVal == nil {
			return nil
		}
		return &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: *strVal}}
	default:
		return nil
	}
}

func (dc *dedicatedColumn) writeValue(attrs *DedicatedAttributes, value *v1.AnyValue) bool {
	switch dc.Type {
	case backend.DedicatedColumnTypeString:
		strVal, ok := value.Value.(*v1.AnyValue_StringValue)
		if !ok {
			return false
		}
		switch dc.ColumnIndex {
		case 0:
			attrs.String01 = &strVal.StringValue
		case 1:
			attrs.String02 = &strVal.StringValue
		case 2:
			attrs.String03 = &strVal.StringValue
		case 3:
			attrs.String04 = &strVal.StringValue
		case 4:
			attrs.String05 = &strVal.StringValue
		case 5:
			attrs.String06 = &strVal.StringValue
		case 6:
			attrs.String07 = &strVal.StringValue
		case 7:
			attrs.String08 = &strVal.StringValue
		case 8:
			attrs.String09 = &strVal.StringValue
		case 9:
			attrs.String10 = &strVal.StringValue
		default:
			return false
		}
	default:
		return false
	}
	return true
}

func newDedicatedColumnMapping(size int) dedicatedColumnMapping {
	return dedicatedColumnMapping{
		mapping: make(map[string]dedicatedColumn, size),
		keys:    make([]string, 0, size),
	}
}

// dedicatedColumnMapping maps the attribute names to dedicated columns while preserving the
// order of dedicated columns
type dedicatedColumnMapping struct {
	mapping map[string]dedicatedColumn
	keys    []string
}

func (dm *dedicatedColumnMapping) put(attr string, col dedicatedColumn) {
	dm.mapping[attr] = col
	dm.keys = append(dm.keys, attr)
}

func (dm *dedicatedColumnMapping) get(attr string) (dedicatedColumn, bool) {
	col, ok := dm.mapping[attr]
	return col, ok
}

func (dm *dedicatedColumnMapping) forEach(callback func(attr string, column dedicatedColumn)) {
	for _, k := range dm.keys {
		callback(k, dm.mapping[k])
	}
}

var allScopes = []backend.DedicatedColumnScope{backend.DedicatedColumnScopeResource, backend.DedicatedColumnScopeSpan}

// dedicatedColumnsToColumnMapping returns mapping from attribute names to spare columns for a give
// block meta and scope.
func dedicatedColumnsToColumnMapping(dedicatedColumns backend.DedicatedColumns, scopes ...backend.DedicatedColumnScope) dedicatedColumnMapping {
	if len(scopes) == 0 {
		scopes = allScopes
	}

	mapping := newDedicatedColumnMapping(len(dedicatedColumns))

	for _, scope := range scopes {
		spareColumnsByType, ok := DedicatedResourceColumnPaths[scope]
		if !ok {
			continue
		}

		indexByType := map[backend.DedicatedColumnType]int{}
		for _, c := range dedicatedColumns {
			if c.Scope != scope {
				continue
			}
			spareColumnPaths, exists := spareColumnsByType[c.Type]
			if !exists {
				continue
			}

			i := indexByType[c.Type]
			if i >= len(spareColumnPaths) {
				continue // skip if there are not enough spare columns
			}

			mapping.put(c.Name, dedicatedColumn{
				Type:        c.Type,
				ColumnPath:  spareColumnPaths[i],
				ColumnIndex: i,
			})
			indexByType[c.Type]++
		}
	}

	return mapping
}
