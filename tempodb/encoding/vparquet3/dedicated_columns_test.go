package vparquet3

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
			name: "scope span",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String01"},
					"span.two": {Type: "string", ColumnIndex: 1, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String02"},
				},
				keys: []string{"span.one", "span.two"},
			},
		},
		{
			name: "scope resource",
			columns: backend.DedicatedColumns{
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "string"},
				{Scope: "resource", Name: "res.two", Type: "string"},
			},
			scopes: []backend.DedicatedColumnScope{"resource"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"res.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.list.element.Resource.DedicatedAttributes.String01"},
					"res.two": {Type: "string", ColumnIndex: 1, ColumnPath: "rs.list.element.Resource.DedicatedAttributes.String02"},
				},
				keys: []string{"res.one", "res.two"},
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
					"res.one":  {Type: "string", ColumnIndex: 0, ColumnPath: "rs.list.element.Resource.DedicatedAttributes.String01"},
					"res.two":  {Type: "string", ColumnIndex: 1, ColumnPath: "rs.list.element.Resource.DedicatedAttributes.String02"},
					"span.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String01"},
					"span.two": {Type: "string", ColumnIndex: 1, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String02"},
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
					"res.one":  {Type: "string", ColumnIndex: 0, ColumnPath: "rs.list.element.Resource.DedicatedAttributes.String01"},
					"res.two":  {Type: "string", ColumnIndex: 1, ColumnPath: "rs.list.element.Resource.DedicatedAttributes.String02"},
					"span.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String01"},
					"span.two": {Type: "string", ColumnIndex: 1, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String02"},
				},
				keys: []string{"res.one", "res.two", "span.one", "span.two"},
			},
		},
		{
			name: "wrong type",
			columns: backend.DedicatedColumns{
				{Scope: "span", Name: "span.one", Type: "string"},
				{Scope: "resource", Name: "res.one", Type: "string"},
				{Scope: "span", Name: "span.two", Type: "integer"}, // ignored
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one": {Type: "string", ColumnIndex: 0, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String01"},
				},
				keys: []string{"span.one"},
			},
		},
		{
			name: "too many columns",
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
				{Scope: "span", Name: "span.eleven", Type: "string"}, // ignored
			},
			scopes: []backend.DedicatedColumnScope{"span"},
			expectedMapping: dedicatedColumnMapping{
				mapping: map[string]dedicatedColumn{
					"span.one":   {Type: "string", ColumnIndex: 0, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String01"},
					"span.two":   {Type: "string", ColumnIndex: 1, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String02"},
					"span.three": {Type: "string", ColumnIndex: 2, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String03"},
					"span.four":  {Type: "string", ColumnIndex: 3, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String04"},
					"span.five":  {Type: "string", ColumnIndex: 4, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String05"},
					"span.six":   {Type: "string", ColumnIndex: 5, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String06"},
					"span.seven": {Type: "string", ColumnIndex: 6, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String07"},
					"span.eight": {Type: "string", ColumnIndex: 7, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String08"},
					"span.nine":  {Type: "string", ColumnIndex: 8, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String09"},
					"span.ten":   {Type: "string", ColumnIndex: 9, ColumnPath: "rs.list.element.ss.list.element.Spans.list.element.DedicatedAttributes.String10"},
				},
				keys: []string{"span.one", "span.two", "span.three", "span.four", "span.five", "span.six", "span.seven", "span.eight", "span.nine", "span.ten"},
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
	strPtr := func(s string) *string { return &s }
	attrComplete := DedicatedAttributes{
		strPtr("one"), strPtr("two"), strPtr("three"), strPtr("four"), strPtr("five"),
		strPtr("six"), strPtr("seven"), strPtr("eight"), strPtr("nine"), strPtr("ten"),
	}

	tests := []struct {
		name            string
		dedicatedColumn dedicatedColumn
		attr            DedicatedAttributes
		want            *v1.AnyValue
	}{
		{
			name:            "not nil",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 0},
			attr:            attrComplete,
			want:            &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "one"}},
		},
		{
			name:            "nil",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 0},
			attr:            DedicatedAttributes{},
			want:            nil,
		},
		{
			name:            "index too high",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 10},
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
	strPtr := func(s string) *string { return &s }

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
			expectedAttr:    DedicatedAttributes{String05: strPtr("five")},
		},
		{
			name:            "wrong type",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 1},
			value:           &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 2}},
		},
		{
			name:            "index too high",
			dedicatedColumn: dedicatedColumn{Type: "string", ColumnIndex: 10},
			value:           &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "eleven"}},
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
