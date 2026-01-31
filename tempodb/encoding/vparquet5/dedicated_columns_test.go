package vparquet5

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/tempodb/backend"
)

func TestDedicatedColumnsToColumnMapping(t *testing.T) {
	tests := []struct {
		name            string
		columns         backend.DedicatedColumns
		scopes          []backend.DedicatedColumnScope
		expectedMapping dedicatedColumnMapping
	}{
		{
			name: "scope span str",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String01"},
					"span.two": {Type: "string", ColumnIndex: 1, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String02"},
				},
				keys: []string{"span.one", "span.two"},
			},
		},
		{
			name: "scope span int",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span.one", Type: "int"},
				{Scope: "resource", Name: "res.one", Type: "int"},
				{Scope: "span", Name: "span.two", Type: "int"},
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one": {Type: "int", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int01"},
					"span.two": {Type: "int", ColumnIndex: 1, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int02"},
				},
				keys: []string{"span.one", "span.two"},
			},
		},
		{
			name: "scope span mix",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
				{Scope: "span", Name: "span.one-int", Type: "int"},
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one":     {Type: "string", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String01"},
					"span.two":     {Type: "string", ColumnIndex: 1, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String02"},
					"span.one-int": {Type: "int", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int01"},
				},
				keys: []string{"span.one", "span.two", "span.one-int"},
			},
		},
		{
			name: "scope resource str",
			columns: backend.DedicatedColumns{
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
				{Scope: "resource", Name: "res.two", Type: "string"},
			},
			scopes: []backend.DedicatedColumnScope{"resource"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"res.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.Resource.DedicatedAttributes.String01"},
					"res.two": {Type: "string", ColumnIndex: 1, ColumnPath: "rs.Resource.DedicatedAttributes.String02"},
				},
				keys: []string{"res.one", "res.two"},
			},
		},
		{
			name: "scope resource int",
			columns: backend.DedicatedColumns{
				{Scope: "resource", Name: "res.one", Type: "int"},
				{Scope: "span", Name: "span.one", Type: "int"},
				{Scope: "span", Name: "span.two", Type: "int"},
				{Scope: "resource", Name: "res.two", Type: "int"},
			},
			scopes: []backend.DedicatedColumnScope{"resource"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"res.one": {Type: "int", ColumnIndex: 0, ColumnPath: "rs.Resource.DedicatedAttributes.Int01"},
					"res.two": {Type: "int", ColumnIndex: 1, ColumnPath: "rs.Resource.DedicatedAttributes.Int02"},
				},
				keys: []string{"res.one", "res.two"},
			},
		},
		{
			name: "scope resource mix",
			columns: backend.DedicatedColumns{
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "resource", Name: "res.one-int", Type: "int"},
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
				{Scope: "resource", Name: "res.two", Type: "string"},
			},
			scopes: []backend.DedicatedColumnScope{"resource"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"res.one":     {Type: "string", ColumnIndex: 0, ColumnPath: "rs.Resource.DedicatedAttributes.String01"},
					"res.two":     {Type: "string", ColumnIndex: 1, ColumnPath: "rs.Resource.DedicatedAttributes.String02"},
					"res.one-int": {Type: "int", ColumnIndex: 0, ColumnPath: "rs.Resource.DedicatedAttributes.Int01"},
				},
				keys: []string{"res.one", "res.one-int", "res.two"},
			},
		},
		{
			name: "all scopes explicit",
			columns: backend.DedicatedColumns{
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
				{Scope: "resource", Name: "res.two", Type: "string"},
			},
			scopes: []backend.DedicatedColumnScope{"resource", "span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"res.one":  {Type: "string", ColumnIndex: 0, ColumnPath: "rs.Resource.DedicatedAttributes.String01"},
					"res.two":  {Type: "string", ColumnIndex: 1, ColumnPath: "rs.Resource.DedicatedAttributes.String02"},
					"span.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String01"},
					"span.two": {Type: "string", ColumnIndex: 1, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String02"},
				},
				keys: []string{"res.one", "res.two", "span.one", "span.two"},
			},
		},
		{
			name: "all scopes implicit",
			columns: backend.DedicatedColumns{
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
				{Scope: "resource", Name: "res.two", Type: "string"},
			},
			scopes: []backend.DedicatedColumnScope{},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"res.one":  {Type: "string", ColumnIndex: 0, ColumnPath: "rs.Resource.DedicatedAttributes.String01"},
					"res.two":  {Type: "string", ColumnIndex: 1, ColumnPath: "rs.Resource.DedicatedAttributes.String02"},
					"span.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String01"},
					"span.two": {Type: "string", ColumnIndex: 1, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String02"},
				},
				keys: []string{"res.one", "res.two", "span.one", "span.two"},
			},
		},
		{
			name: "all scopes mix",
			columns: backend.DedicatedColumns{
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "resource", Name: "res.one-int", Type: "int"},
				{Scope: "resource", Name: "res.two-int", Type: "int"},
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
				{Scope: "span", Name: "span.two-int", Type: "int"},
				{Scope: "resource", Name: "res.two", Type: "string"},
			},
			scopes: []backend.DedicatedColumnScope{"resource", "span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"res.one":      {Type: "string", ColumnIndex: 0, ColumnPath: "rs.Resource.DedicatedAttributes.String01"},
					"res.two":      {Type: "string", ColumnIndex: 1, ColumnPath: "rs.Resource.DedicatedAttributes.String02"},
					"res.one-int":  {Type: "int", ColumnIndex: 0, ColumnPath: "rs.Resource.DedicatedAttributes.Int01"},
					"res.two-int":  {Type: "int", ColumnIndex: 1, ColumnPath: "rs.Resource.DedicatedAttributes.Int02"},
					"span.one":     {Type: "string", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String01"},
					"span.two":     {Type: "string", ColumnIndex: 1, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String02"},
					"span.two-int": {Type: "int", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int01"},
				},
				keys: []string{"res.one", "res.one-int", "res.two-int", "res.two", "span.one", "span.two", "span.two-int"},
			},
		},
		{
			name: "wrong type",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "bool"}, // ignored
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String01"},
				},
				keys: []string{"span.one"},
			},
		},
		{
			name: "too many columns str",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
				{Scope: "span", Name: "span.three", Type: "string"},
				{Scope: "span", Name: "span.four", Type: "string"},
				{Scope: "span", Name: "span.five", Type: "string"},
				{Scope: "span", Name: "span.six", Type: "string"},
				{Scope: "span", Name: "span.seven", Type: "string"},
				{Scope: "span", Name: "span.eight", Type: "string"},
				{Scope: "span", Name: "span.nine", Type: "string"},
				{Scope: "span", Name: "span.ten", Type: "string"},
				{Scope: "span", Name: "span.eleven", Type: "string"},
				{Scope: "span", Name: "span.twelve", Type: "string"},
				{Scope: "span", Name: "span.thirteen", Type: "string"},
				{Scope: "span", Name: "span.fourteen", Type: "string"},
				{Scope: "span", Name: "span.fifteen", Type: "string"},
				{Scope: "span", Name: "span.sixteen", Type: "string"},
				{Scope: "span", Name: "span.seventeen", Type: "string"},
				{Scope: "span", Name: "span.eighteen", Type: "string"},
				{Scope: "span", Name: "span.nineteen", Type: "string"},
				{Scope: "span", Name: "span.twenty", Type: "string"},
				{Scope: "span", Name: "span.twenty-one", Type: "string"}, // ignored
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one":       {Type: "string", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String01"},
					"span.two":       {Type: "string", ColumnIndex: 1, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String02"},
					"span.three":     {Type: "string", ColumnIndex: 2, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String03"},
					"span.four":      {Type: "string", ColumnIndex: 3, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String04"},
					"span.five":      {Type: "string", ColumnIndex: 4, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String05"},
					"span.six":       {Type: "string", ColumnIndex: 5, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String06"},
					"span.seven":     {Type: "string", ColumnIndex: 6, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String07"},
					"span.eight":     {Type: "string", ColumnIndex: 7, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String08"},
					"span.nine":      {Type: "string", ColumnIndex: 8, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String09"},
					"span.ten":       {Type: "string", ColumnIndex: 9, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String10"},
					"span.eleven":    {Type: "string", ColumnIndex: 10, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String11"},
					"span.twelve":    {Type: "string", ColumnIndex: 11, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String12"},
					"span.thirteen":  {Type: "string", ColumnIndex: 12, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String13"},
					"span.fourteen":  {Type: "string", ColumnIndex: 13, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String14"},
					"span.fifteen":   {Type: "string", ColumnIndex: 14, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String15"},
					"span.sixteen":   {Type: "string", ColumnIndex: 15, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String16"},
					"span.seventeen": {Type: "string", ColumnIndex: 16, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String17"},
					"span.eighteen":  {Type: "string", ColumnIndex: 17, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String18"},
					"span.nineteen":  {Type: "string", ColumnIndex: 18, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String19"},
					"span.twenty":    {Type: "string", ColumnIndex: 19, ColumnPath: "rs.ss.Spans.DedicatedAttributes.String20"},
				},
				keys: []string{"span.one", "span.two", "span.three", "span.four", "span.five", "span.six", "span.seven", "span.eight", "span.nine", "span.ten", "span.eleven", "span.twelve", "span.thirteen", "span.fourteen", "span.fifteen", "span.sixteen", "span.seventeen", "span.eighteen", "span.nineteen", "span.twenty"},
			},
		},
		{
			name: "too many columns int",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span.one", Type: "int"},
				{Scope: "span", Name: "span.two", Type: "int"},
				{Scope: "span", Name: "span.three", Type: "int"},
				{Scope: "span", Name: "span.four", Type: "int"},
				{Scope: "span", Name: "span.five", Type: "int"},
				{Scope: "span", Name: "span.six", Type: "int"}, // ignored
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one":   {Type: "int", ColumnIndex: 0, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int01"},
					"span.two":   {Type: "int", ColumnIndex: 1, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int02"},
					"span.three": {Type: "int", ColumnIndex: 2, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int03"},
					"span.four":  {Type: "int", ColumnIndex: 3, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int04"},
					"span.five":  {Type: "int", ColumnIndex: 4, ColumnPath: "rs.ss.Spans.DedicatedAttributes.Int05"},
				},
				keys: []string{"span.one", "span.two", "span.three", "span.four", "span.five"},
			},
		},
	}
	for _, tc := range tests {
		meta := backend.BlockMeta{DedicatedColumns: tc.columns}
		mapping := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, tc.scopes...)
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedMapping, mapping)
		})
	}
}

func TestDedicatedColumn_readValue(t *testing.T) {
	attrComplete := DedicatedAttributes{
		String01: []string{"one"},
		String02: []string{"two"},
		String03: []string{"three", "three_b"},
		String04: []string{"four"},
		String05: []string{"five"},
		String06: []string{"six"},
		String07: []string{"seven"},
		String08: []string{"eight"},
		String09: []string{"nine"},
		String10: []string{"ten"},
		Int01:    []int64{1},
		Int02:    []int64{2},
		Int03:    []int64{3},
		Int04:    []int64{4, 40},
		Int05:    []int64{5},
	}

	tests := []struct {
		name            string
		dedicatedColumn dedicatedColumn
		attr            DedicatedAttributes
		want            *v1.AnyValue
	}{
		{
			name:            "str not nil",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 1},
			attr:            attrComplete,
			want:            &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "two"}},
		},
		{
			name:            "str array",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 2 /*, IsArray: true*/},
			attr:            attrComplete,
			want: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{
				ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{{Value: &v1.AnyValue_StringValue{StringValue: "three"}}, {Value: &v1.AnyValue_StringValue{StringValue: "three_b"}}}},
			}},
		},
		{
			name:            "str nil",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 0},
			attr:            DedicatedAttributes{},
			want:            nil,
		},
		{
			name:            "str index too high",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 10},
			attr:            attrComplete,
			want:            nil,
		},
		{
			name:            "int not nil",
			dedicatedColumn: dedicatedColumn{Type: "int", ColumnIndex: 2},
			attr:            attrComplete,
			want:            &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 3}},
		},
		{
			name:            "int array",
			dedicatedColumn: dedicatedColumn{Type: "int", ColumnIndex: 3 /*, IsArray: true*/},
			attr:            attrComplete,
			want: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{
				ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{{Value: &v1.AnyValue_IntValue{IntValue: 4}}, {Value: &v1.AnyValue_IntValue{IntValue: 40}}}},
			}},
		},
		{
			name:            "str empty",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 2},
			attr:            DedicatedAttributes{},
			want:            nil,
		},
		{
			name:            "int empty",
			dedicatedColumn: dedicatedColumn{Type: "int", ColumnIndex: 3},
			attr:            DedicatedAttributes{},
			want:            nil,
		},
		{
			name:            "int nil",
			dedicatedColumn: dedicatedColumn{Type: "int", ColumnIndex: 1},
			attr:            DedicatedAttributes{},
			want:            nil,
		},
		{
			name:            "int index too high",
			dedicatedColumn: dedicatedColumn{Type: "int", ColumnIndex: 5},
			attr:            attrComplete,
			want:            nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val := tc.dedicatedColumn.readValue(&tc.attr)
			assert.Equal(t, tc.want, val)
		})
	}
}

func TestDedicatedColumn_writeValue(t *testing.T) {
	tests := []struct {
		name            string
		dedicatedColumn dedicatedColumn
		value           *v1.AnyValue
		expectedWritten bool
		expectedAttr    DedicatedAttributes
	}{
		{
			name:            "string",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 4},
			value:           &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "five"}},
			expectedWritten: true,
			expectedAttr:    DedicatedAttributes{String05: []string{"five"}},
		},
		{
			name:            "string array",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 4},
			value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{
				ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{{Value: &v1.AnyValue_StringValue{StringValue: "five"}}, {Value: &v1.AnyValue_StringValue{StringValue: "five_b"}}}},
			}},
			expectedWritten: true,
			expectedAttr:    DedicatedAttributes{String05: []string{"five", "five_b"}},
		},
		{
			name:            "int",
			dedicatedColumn: dedicatedColumn{Type: "int", ColumnIndex: 3},
			value:           &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 11}},
			expectedWritten: true,
			expectedAttr:    DedicatedAttributes{Int04: []int64{11}},
		},
		{
			name:            "int array",
			dedicatedColumn: dedicatedColumn{Type: "int", ColumnIndex: 3},
			value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{
				ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{{Value: &v1.AnyValue_IntValue{IntValue: 11}}, {Value: &v1.AnyValue_IntValue{IntValue: 12}}}},
			}},
			expectedWritten: true,
			expectedAttr:    DedicatedAttributes{Int04: []int64{11, 12}},
		},
		{
			name:            "wrong type",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 1},
			value:           &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 2}},
		},
		{
			name:            "wrong array element type",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 2},
			value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{
				ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{{Value: &v1.AnyValue_IntValue{IntValue: 2}}}},
			}},
		},
		{
			name:            "index too high",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 20},
			value:           &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "twenty"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var attr DedicatedAttributes
			written := tc.dedicatedColumn.writeValue(&attr, tc.value)

			assert.Equal(t, tc.expectedWritten, written)
			assert.Equal(t, tc.expectedAttr, attr)
		})
	}
}

func TestFilterDedicatedColumns(t *testing.T) {
	tests := []struct {
		name     string
		columns  backend.DedicatedColumns
		expected backend.DedicatedColumns
	}{
		{
			name:     "empty columns",
			columns:  backend.DedicatedColumns{},
			expected: backend.DedicatedColumns{},
		},
		{
			name: "empty result span",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span-one", Type: "float"},
			},
			expected: backend.DedicatedColumns{},
		},
		{
			name: "empty result resource",
			columns: backend.DedicatedColumns{
				{Scope: "resource", Name: "res-one", Type: "float"},
				{Scope: "scope", Name: "res-two", Type: "string"},
			},
			expected: backend.DedicatedColumns{},
		},
		{
			name: "filtered result",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span-one", Type: "string"},
				{Scope: "span", Name: "span-two", Type: "string"},
				{Scope: "span", Name: "span-three", Type: "float"},
				{Scope: "resource", Name: "res-one", Type: "string"},
				{Scope: "resource", Name: "res-two", Type: "string"},
				{Scope: "event", Name: "res-three", Type: "string"},
				{Scope: "link", Name: "res-four", Type: "string"},
			},
			expected: backend.DedicatedColumns{
				{Scope: "span", Name: "span-one", Type: "string"},
				{Scope: "span", Name: "span-two", Type: "string"},
				{Scope: "resource", Name: "res-one", Type: "string"},
				{Scope: "resource", Name: "res-two", Type: "string"},
				{Scope: "event", Name: "res-three", Type: "string"},
			},
		},
		{
			name: "non-filtered result",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span-one", Type: "string"},
				{Scope: "span", Name: "span-two", Type: "string"},
				{Scope: "resource", Name: "res-one", Type: "string"},
				{Scope: "resource", Name: "res-two", Type: "string"},
			},
			expected: backend.DedicatedColumns{
				{Scope: "span", Name: "span-one", Type: "string"},
				{Scope: "span", Name: "span-two", Type: "string"},
				{Scope: "resource", Name: "res-one", Type: "string"},
				{Scope: "resource", Name: "res-two", Type: "string"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterDedicatedColumns(tt.columns)
			assert.Equal(t, tt.expected, result)
		})
	}
}
