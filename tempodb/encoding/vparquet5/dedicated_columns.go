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
		backend.DedicatedColumnTypeInt: {
			"rs.list.element.Resource.DedicatedAttributes.Int01",
			"rs.list.element.Resource.DedicatedAttributes.Int02",
			"rs.list.element.Resource.DedicatedAttributes.Int03",
			"rs.list.element.Resource.DedicatedAttributes.Int04",
			"rs.list.element.Resource.DedicatedAttributes.Int05",
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
		backend.DedicatedColumnTypeInt: {
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.Int01",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.Int02",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.Int03",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.Int04",
			"rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.Int05",
		},
	},
}

type dedicatedColumn struct {
	Type        backend.DedicatedColumnType
	ColumnPath  string
	ColumnIndex int
	IsArray     bool
}

func (dc *dedicatedColumn) readValue(attrs *DedicatedAttributes) *v1.AnyValue {
	var val *v1.AnyValue

	switch dc.Type {
	case backend.DedicatedColumnTypeString:
		switch dc.ColumnIndex {
		case 0:
			val = dedicatedColStrToAnyValue(attrs.String01, dc.IsArray)
		case 1:
			val = dedicatedColStrToAnyValue(attrs.String02, dc.IsArray)
		case 2:
			val = dedicatedColStrToAnyValue(attrs.String03, dc.IsArray)
		case 3:
			val = dedicatedColStrToAnyValue(attrs.String04, dc.IsArray)
		case 4:
			val = dedicatedColStrToAnyValue(attrs.String05, dc.IsArray)
		case 5:
			val = dedicatedColStrToAnyValue(attrs.String06, dc.IsArray)
		case 6:
			val = dedicatedColStrToAnyValue(attrs.String07, dc.IsArray)
		case 7:
			val = dedicatedColStrToAnyValue(attrs.String08, dc.IsArray)
		case 8:
			val = dedicatedColStrToAnyValue(attrs.String09, dc.IsArray)
		case 9:
			val = dedicatedColStrToAnyValue(attrs.String10, dc.IsArray)
		}
	case backend.DedicatedColumnTypeInt:
		switch dc.ColumnIndex {
		case 0:
			val = dedicatedColIntToAnyValue(attrs.Int01, dc.IsArray)
		case 1:
			val = dedicatedColIntToAnyValue(attrs.Int02, dc.IsArray)
		case 2:
			val = dedicatedColIntToAnyValue(attrs.Int03, dc.IsArray)
		case 3:
			val = dedicatedColIntToAnyValue(attrs.Int04, dc.IsArray)
		case 4:
			val = dedicatedColIntToAnyValue(attrs.Int05, dc.IsArray)
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
			attrs.String01, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String01)
		case 1:
			attrs.String02, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String02)
		case 2:
			attrs.String03, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String03)
		case 3:
			attrs.String04, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String04)
		case 4:
			attrs.String05, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String05)
		case 5:
			attrs.String06, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String06)
		case 6:
			attrs.String07, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String07)
		case 7:
			attrs.String08, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String08)
		case 8:
			attrs.String09, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String09)
		case 9:
			attrs.String10, written = anyValueToDedicatedColStr(value, dc.IsArray, attrs.String10)
		}
	case backend.DedicatedColumnTypeInt:
		switch dc.ColumnIndex {
		case 0:
			attrs.Int01, written = anyValueToDedicatedColInt(value, dc.IsArray, attrs.Int01)
		case 1:
			attrs.Int02, written = anyValueToDedicatedColInt(value, dc.IsArray, attrs.Int02)
		case 2:
			attrs.Int03, written = anyValueToDedicatedColInt(value, dc.IsArray, attrs.Int03)
		case 3:
			attrs.Int04, written = anyValueToDedicatedColInt(value, dc.IsArray, attrs.Int04)
		case 4:
			attrs.Int05, written = anyValueToDedicatedColInt(value, dc.IsArray, attrs.Int05)
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

func (dm *dedicatedColumnMapping) items() iter.Seq2[string, dedicatedColumn] {
	return func(yield func(string, dedicatedColumn) bool) {
		for _, k := range dm.keys {
			if !yield(k, dm.mapping[k]) {
				return
			}
		}
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

			var isArray bool
			for _, opt := range c.Options {
				if opt == backend.DedicatedColumnOptionArray {
					isArray = true
					break
				}
			}

			mapping.put(c.Name, dedicatedColumn{
				Type:        c.Type,
				ColumnPath:  spareColumnPaths[i],
				ColumnIndex: i,
				IsArray:     isArray,
			})
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

func anyValueToDedicatedColStr(value *v1.AnyValue, isArray bool, buf []string) ([]string, bool) {
	buf = buf[:0]
	if !isArray {
		value, ok := value.Value.(*v1.AnyValue_StringValue)
		if !ok || value == nil {
			return nil, false
		}
		buf = append(buf, value.StringValue)
	} else {
		value, ok := value.Value.(*v1.AnyValue_ArrayValue)
		if !ok || value == nil {
			return nil, false
		}

		for _, v := range value.ArrayValue.Values {
			v, ok := v.Value.(*v1.AnyValue_StringValue)
			if !ok || v == nil {
				return nil, false
			}
			buf = append(buf, v.StringValue)
		}
	}
	return buf, true
}

func dedicatedColStrToAnyValue(v []string, isArray bool) *v1.AnyValue {
	if !isArray && len(v) == 1 {
		return &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: v[0]}}
	}

	values := make([]*v1.AnyValue, 0, len(v))
	switch len(v) {
	case 0:
		return nil
	case 1:
		values = append(values, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: v[0]}})
	default:
		// allocate v1.Any* values in batches
		allocAny := make([]v1.AnyValue, len(v))
		allocStr := make([]v1.AnyValue_StringValue, len(v))
		for i, s := range v {
			anyS := &allocStr[i]
			anyS.StringValue = s
			anyV := &allocAny[i]
			anyV.Value = anyS

			values = append(values, anyV)
		}
	}
	return &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}}
}

func anyValueToDedicatedColInt(value *v1.AnyValue, isArray bool, buf []int64) ([]int64, bool) {
	buf = buf[:0]
	if !isArray {
		value, ok := value.Value.(*v1.AnyValue_IntValue)
		if !ok || value == nil {
			return nil, false
		}
		buf = append(buf, value.IntValue)
	} else {
		value, ok := value.Value.(*v1.AnyValue_ArrayValue)
		if !ok || value == nil {
			return nil, false
		}

		for _, v := range value.ArrayValue.Values {
			v, ok := v.Value.(*v1.AnyValue_IntValue)
			if !ok || v == nil {
				return nil, false
			}
			buf = append(buf, v.IntValue)
		}
	}
	return buf, true
}

func dedicatedColIntToAnyValue(v []int64, isArray bool) *v1.AnyValue {
	if !isArray && len(v) == 1 {
		return &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: v[0]}}
	}

	values := make([]*v1.AnyValue, 0, len(v))
	switch len(v) {
	case 0:
		return nil
	case 1:
		values = append(values, &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: v[0]}})
	default:
		// allocate v1.Any* values in batches
		allocAny := make([]v1.AnyValue, len(v))
		allocInt := make([]v1.AnyValue_IntValue, len(v))
		for i, n := range v {
			anyS := &allocInt[i]
			anyS.IntValue = n
			anyV := &allocAny[i]
			anyV.Value = anyS

			values = append(values, anyV)
		}
	}
	return &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}}
}
