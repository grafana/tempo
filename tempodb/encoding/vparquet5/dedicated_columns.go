package vparquet5

import (
	"iter"

	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/tempodb/backend"
)

// DedicatedResourceColumnPaths makes paths for spare dedicated attribute columns available
var DedicatedResourceColumnPaths = map[backend.DedicatedColumnScope]map[backend.DedicatedColumnType][]string{
	backend.DedicatedColumnScopeResource: {
		backend.DedicatedColumnTypeString: {
			"rs.Resource.DedicatedAttributes.String01",
			"rs.Resource.DedicatedAttributes.String02",
			"rs.Resource.DedicatedAttributes.String03",
			"rs.Resource.DedicatedAttributes.String04",
			"rs.Resource.DedicatedAttributes.String05",
			"rs.Resource.DedicatedAttributes.String06",
			"rs.Resource.DedicatedAttributes.String07",
			"rs.Resource.DedicatedAttributes.String08",
			"rs.Resource.DedicatedAttributes.String09",
			"rs.Resource.DedicatedAttributes.String10",
			"rs.Resource.DedicatedAttributes.String11",
			"rs.Resource.DedicatedAttributes.String12",
			"rs.Resource.DedicatedAttributes.String13",
			"rs.Resource.DedicatedAttributes.String14",
			"rs.Resource.DedicatedAttributes.String15",
			"rs.Resource.DedicatedAttributes.String16",
			"rs.Resource.DedicatedAttributes.String17",
			"rs.Resource.DedicatedAttributes.String18",
			"rs.Resource.DedicatedAttributes.String19",
			"rs.Resource.DedicatedAttributes.String20",
		},
		backend.DedicatedColumnTypeInt: {
			"rs.Resource.DedicatedAttributes.Int01",
			"rs.Resource.DedicatedAttributes.Int02",
			"rs.Resource.DedicatedAttributes.Int03",
			"rs.Resource.DedicatedAttributes.Int04",
			"rs.Resource.DedicatedAttributes.Int05",
		},
	},
	backend.DedicatedColumnScopeSpan: {
		backend.DedicatedColumnTypeString: {
			"rs.ss.Spans.DedicatedAttributes.String01",
			"rs.ss.Spans.DedicatedAttributes.String02",
			"rs.ss.Spans.DedicatedAttributes.String03",
			"rs.ss.Spans.DedicatedAttributes.String04",
			"rs.ss.Spans.DedicatedAttributes.String05",
			"rs.ss.Spans.DedicatedAttributes.String06",
			"rs.ss.Spans.DedicatedAttributes.String07",
			"rs.ss.Spans.DedicatedAttributes.String08",
			"rs.ss.Spans.DedicatedAttributes.String09",
			"rs.ss.Spans.DedicatedAttributes.String10",
			"rs.ss.Spans.DedicatedAttributes.String11",
			"rs.ss.Spans.DedicatedAttributes.String12",
			"rs.ss.Spans.DedicatedAttributes.String13",
			"rs.ss.Spans.DedicatedAttributes.String14",
			"rs.ss.Spans.DedicatedAttributes.String15",
			"rs.ss.Spans.DedicatedAttributes.String16",
			"rs.ss.Spans.DedicatedAttributes.String17",
			"rs.ss.Spans.DedicatedAttributes.String18",
			"rs.ss.Spans.DedicatedAttributes.String19",
			"rs.ss.Spans.DedicatedAttributes.String20",
		},
		backend.DedicatedColumnTypeInt: {
			"rs.ss.Spans.DedicatedAttributes.Int01",
			"rs.ss.Spans.DedicatedAttributes.Int02",
			"rs.ss.Spans.DedicatedAttributes.Int03",
			"rs.ss.Spans.DedicatedAttributes.Int04",
			"rs.ss.Spans.DedicatedAttributes.Int05",
		},
	},
	backend.DedicatedColumnScopeEvent: {
		backend.DedicatedColumnTypeString: {
			"rs.ss.Spans.Events.DedicatedAttributes.String01",
			"rs.ss.Spans.Events.DedicatedAttributes.String02",
			"rs.ss.Spans.Events.DedicatedAttributes.String03",
			"rs.ss.Spans.Events.DedicatedAttributes.String04",
			"rs.ss.Spans.Events.DedicatedAttributes.String05",
			"rs.ss.Spans.Events.DedicatedAttributes.String06",
			"rs.ss.Spans.Events.DedicatedAttributes.String07",
			"rs.ss.Spans.Events.DedicatedAttributes.String08",
			"rs.ss.Spans.Events.DedicatedAttributes.String09",
			"rs.ss.Spans.Events.DedicatedAttributes.String10",
			"rs.ss.Spans.Events.DedicatedAttributes.String11",
			"rs.ss.Spans.Events.DedicatedAttributes.String12",
			"rs.ss.Spans.Events.DedicatedAttributes.String13",
			"rs.ss.Spans.Events.DedicatedAttributes.String14",
			"rs.ss.Spans.Events.DedicatedAttributes.String15",
			"rs.ss.Spans.Events.DedicatedAttributes.String16",
			"rs.ss.Spans.Events.DedicatedAttributes.String17",
			"rs.ss.Spans.Events.DedicatedAttributes.String18",
			"rs.ss.Spans.Events.DedicatedAttributes.String19",
			"rs.ss.Spans.Events.DedicatedAttributes.String20",
		},
		backend.DedicatedColumnTypeInt: {
			"rs.ss.Spans.Events.DedicatedAttributes.Int01",
			"rs.ss.Spans.Events.DedicatedAttributes.Int02",
			"rs.ss.Spans.Events.DedicatedAttributes.Int03",
			"rs.ss.Spans.Events.DedicatedAttributes.Int04",
			"rs.ss.Spans.Events.DedicatedAttributes.Int05",
		},
	},
}

type dedicatedColumn struct {
	Type        backend.DedicatedColumnType
	ColumnPath  string
	ColumnIndex int
	// IsArray     bool
	IsBlob bool
}

func (dc *dedicatedColumn) readValue(attrs *DedicatedAttributes) *v1.AnyValue {
	var val *v1.AnyValue

	switch dc.Type {
	case backend.DedicatedColumnTypeString:
		switch dc.ColumnIndex {
		case 0:
			val = dedicatedColStrToAnyValue(attrs.String01)
		case 1:
			val = dedicatedColStrToAnyValue(attrs.String02)
		case 2:
			val = dedicatedColStrToAnyValue(attrs.String03)
		case 3:
			val = dedicatedColStrToAnyValue(attrs.String04)
		case 4:
			val = dedicatedColStrToAnyValue(attrs.String05)
		case 5:
			val = dedicatedColStrToAnyValue(attrs.String06)
		case 6:
			val = dedicatedColStrToAnyValue(attrs.String07)
		case 7:
			val = dedicatedColStrToAnyValue(attrs.String08)
		case 8:
			val = dedicatedColStrToAnyValue(attrs.String09)
		case 9:
			val = dedicatedColStrToAnyValue(attrs.String10)
		case 10:
			val = dedicatedColStrToAnyValue(attrs.String11)
		case 11:
			val = dedicatedColStrToAnyValue(attrs.String12)
		case 12:
			val = dedicatedColStrToAnyValue(attrs.String13)
		case 13:
			val = dedicatedColStrToAnyValue(attrs.String14)
		case 14:
			val = dedicatedColStrToAnyValue(attrs.String15)
		case 15:
			val = dedicatedColStrToAnyValue(attrs.String16)
		case 16:
			val = dedicatedColStrToAnyValue(attrs.String17)
		case 17:
			val = dedicatedColStrToAnyValue(attrs.String18)
		case 18:
			val = dedicatedColStrToAnyValue(attrs.String19)
		case 19:
			val = dedicatedColStrToAnyValue(attrs.String20)
		}
	case backend.DedicatedColumnTypeInt:
		switch dc.ColumnIndex {
		case 0:
			val = dedicatedColIntToAnyValue(attrs.Int01)
		case 1:
			val = dedicatedColIntToAnyValue(attrs.Int02)
		case 2:
			val = dedicatedColIntToAnyValue(attrs.Int03)
		case 3:
			val = dedicatedColIntToAnyValue(attrs.Int04)
		case 4:
			val = dedicatedColIntToAnyValue(attrs.Int05)
		}
	}

	return val
}

func (dc *dedicatedColumn) writeValue(attrs *DedicatedAttributes, value *v1.AnyValue) bool {
	var written bool

	switch dc.Type {
	case backend.DedicatedColumnTypeString:
		switch dc.ColumnIndex {
		case 0:
			attrs.String01, written = anyValueToDedicatedColStr(value, attrs.String01)
		case 1:
			attrs.String02, written = anyValueToDedicatedColStr(value, attrs.String02)
		case 2:
			attrs.String03, written = anyValueToDedicatedColStr(value, attrs.String03)
		case 3:
			attrs.String04, written = anyValueToDedicatedColStr(value, attrs.String04)
		case 4:
			attrs.String05, written = anyValueToDedicatedColStr(value, attrs.String05)
		case 5:
			attrs.String06, written = anyValueToDedicatedColStr(value, attrs.String06)
		case 6:
			attrs.String07, written = anyValueToDedicatedColStr(value, attrs.String07)
		case 7:
			attrs.String08, written = anyValueToDedicatedColStr(value, attrs.String08)
		case 8:
			attrs.String09, written = anyValueToDedicatedColStr(value, attrs.String09)
		case 9:
			attrs.String10, written = anyValueToDedicatedColStr(value, attrs.String10)
		case 10:
			attrs.String11, written = anyValueToDedicatedColStr(value, attrs.String11)
		case 11:
			attrs.String12, written = anyValueToDedicatedColStr(value, attrs.String12)
		case 12:
			attrs.String13, written = anyValueToDedicatedColStr(value, attrs.String13)
		case 13:
			attrs.String14, written = anyValueToDedicatedColStr(value, attrs.String14)
		case 14:
			attrs.String15, written = anyValueToDedicatedColStr(value, attrs.String15)
		case 15:
			attrs.String16, written = anyValueToDedicatedColStr(value, attrs.String16)
		case 16:
			attrs.String17, written = anyValueToDedicatedColStr(value, attrs.String17)
		case 17:
			attrs.String18, written = anyValueToDedicatedColStr(value, attrs.String18)
		case 18:
			attrs.String19, written = anyValueToDedicatedColStr(value, attrs.String19)
		case 19:
			attrs.String20, written = anyValueToDedicatedColStr(value, attrs.String20)
		}
	case backend.DedicatedColumnTypeInt:
		switch dc.ColumnIndex {
		case 0:
			attrs.Int01, written = anyValueToDedicatedColInt(value, attrs.Int01)
		case 1:
			attrs.Int02, written = anyValueToDedicatedColInt(value, attrs.Int02)
		case 2:
			attrs.Int03, written = anyValueToDedicatedColInt(value, attrs.Int03)
		case 3:
			attrs.Int04, written = anyValueToDedicatedColInt(value, attrs.Int04)
		case 4:
			attrs.Int05, written = anyValueToDedicatedColInt(value, attrs.Int05)
		}
	}
	return written
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

func (dm *dedicatedColumnMapping) usesPath(path string) bool {
	for _, col := range dm.mapping {
		if col.ColumnPath == path {
			return true
		}
	}
	return false
}

func (dm *dedicatedColumnMapping) items() iter.Seq2[string, dedicatedColumn] {
	return func(yield func(string, dedicatedColumn) bool) {
		for _, k := range dm.keys {
			if !yield(k, dm.mapping[k]) {
				return
			}
		}
	}
}

func (dm *dedicatedColumnMapping) len() int {
	return len(dm.keys)
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

			dc := dedicatedColumn{
				Type:        c.Type,
				ColumnPath:  spareColumnPaths[i],
				ColumnIndex: i,
			}

			for _, opt := range c.Options {
				switch opt {
				case backend.DedicatedColumnOptionArray:
					// dc.IsArray = true
				case backend.DedicatedColumnOptionBlob:
					dc.IsBlob = true
				}
			}

			mapping.put(c.Name, dc)
			indexByType[c.Type]++
		}
	}

	return mapping
}

func filterDedicatedColumns(columns backend.DedicatedColumns) backend.DedicatedColumns {
	filtered := make(backend.DedicatedColumns, 0, len(columns))
	for _, c := range columns {
		if isIgnoredDedicatedColumn(&c) {
			continue
		}
		filtered = append(filtered, c)
	}
	return filtered
}

func isIgnoredDedicatedColumn(dc *backend.DedicatedColumn) bool {
	if _, found := DedicatedResourceColumnPaths[dc.Scope][dc.Type]; !found {
		return true // unsupported scope or type
	}
	return false
}

func anyValueToDedicatedColStr(value *v1.AnyValue, buf []string) ([]string, bool) {
	buf = buf[:0]
	switch value := value.Value.(type) {
	case *v1.AnyValue_StringValue:
		buf = append(buf, value.StringValue)
	case *v1.AnyValue_ArrayValue:
		for _, v := range value.ArrayValue.Values {
			switch v := v.Value.(type) {
			case *v1.AnyValue_StringValue:
				buf = append(buf, v.StringValue)
			default:
				// Mixed array types, not supported
				return nil, false
			}
		}
	default:
		// Wrong type, not supported
		return nil, false
	}

	return buf, true
}

func dedicatedColStrToAnyValue(v []string) *v1.AnyValue {
	switch len(v) {
	case 0:
		return nil

	case 1:
		return &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: v[0]}}

	default:
		var (
			// Build array.
			// All pointers allocated together for performance.
			values   = make([]*v1.AnyValue, 0, len(v))
			allocAny = make([]v1.AnyValue, len(v))
			allocStr = make([]v1.AnyValue_StringValue, len(v))
		)
		for i, s := range v {
			anyS := &allocStr[i]
			anyS.StringValue = s
			anyV := &allocAny[i]
			anyV.Value = anyS
			values = append(values, anyV)
		}
		return &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}}
	}
}

func anyValueToDedicatedColInt(value *v1.AnyValue, buf []int64) ([]int64, bool) {
	buf = buf[:0]
	switch value := value.Value.(type) {
	case *v1.AnyValue_IntValue:
		buf = append(buf, value.IntValue)
	case *v1.AnyValue_ArrayValue:
		for _, v := range value.ArrayValue.Values {
			switch v := v.Value.(type) {
			case *v1.AnyValue_IntValue:
				buf = append(buf, v.IntValue)
			default:
				// Mixed array types, not supported
				return nil, false
			}
		}
	default:
		// Wrong type, not supported
		return nil, false
	}

	return buf, true
}

func dedicatedColIntToAnyValue(v []int64) *v1.AnyValue {
	switch len(v) {
	case 0:
		return nil

	case 1:
		return &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: v[0]}}

	default:
		var (
			// Build array.
			// All pointers allocated together for performance.
			values   = make([]*v1.AnyValue, 0, len(v))
			allocAny = make([]v1.AnyValue, len(v))
			allocInt = make([]v1.AnyValue_IntValue, len(v))
		)
		for i, n := range v {
			anyS := &allocInt[i]
			anyS.IntValue = n
			anyV := &allocAny[i]
			anyV.Value = anyS
			values = append(values, anyV)
		}

		return &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}}
	}
}
