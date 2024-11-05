package inspect

import (
	"fmt"
	"io"
	"strings"

	"github.com/stoewer/parquet-cli/pkg/output"

	"github.com/parquet-go/parquet-go"
)

var schemaHeader = [...]any{
	"Index",
	"Name",
	"Optional",
	"Repeated",
	"Required",
	"Is Leaf",
	"Type",
	"Go Type",
	"Encoding",
	"Compression",
	"Path",
}

type Schema struct {
	pf *parquet.File

	fields []fieldWithPath
	next   int
}

func NewSchema(pf *parquet.File) *Schema {
	return &Schema{pf: pf}
}

func (s *Schema) Text() (string, error) {
	textRaw := s.pf.Schema().String()

	var text strings.Builder
	for _, r := range textRaw {
		if r == '\t' {
			text.WriteString("    ")
		} else {
			text.WriteRune(r)
		}
	}

	return text.String(), nil
}

func (s *Schema) Header() []any {
	return schemaHeader[:]
}

func (s *Schema) NextRow() (output.TableRow, error) {
	if s.fields == nil {
		s.fields = fieldsFromSchema(s.pf.Schema())
	}
	if s.next >= len(s.fields) {
		return nil, fmt.Errorf("no more fields: %w", io.EOF)
	}

	nextField := s.fields[s.next]
	s.next++
	return toSchemaNode(&nextField), nil
}

func (s *Schema) NextSerializable() (any, error) {
	return s.NextRow()
}

func toSchemaNode(n *fieldWithPath) *schemaNode {
	sn := &schemaNode{
		Index:    n.Index,
		Name:     n.Name(),
		Optional: n.Optional(),
		Repeated: n.Repeated(),
		Required: n.Required(),
		IsLeaf:   n.Leaf(),
	}

	if n.Leaf() {
		sn.Type = n.Type().String()
		sn.GoType = n.GoType().String()
		if n.Encoding() != nil {
			sn.Encoding = n.Encoding().String()
		}
		if n.Compression() != nil {
			sn.Compression = n.Compression().String()
		}
	}

	if len(n.Path) > 0 {
		sn.Path = strings.Join(n.Path, ".")
		sn.Name = PathToDisplayName(n.Path)
	}

	return sn
}

type schemaNode struct {
	Index       int    `json:"index,omitempty"`
	Name        string `json:"name"`
	Optional    bool   `json:"optional"`
	Repeated    bool   `json:"repeated"`
	Required    bool   `json:"required"`
	IsLeaf      bool   `json:"is_leaf"`
	Type        string `json:"type,omitempty"`
	GoType      string `json:"go_type,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
	Compression string `json:"compression,omitempty"`
	Path        string `json:"path,omitempty"`
}

func (sn *schemaNode) Cells() []any {
	return []any{
		sn.Index,
		sn.Name,
		sn.Optional,
		sn.Repeated,
		sn.Required,
		sn.IsLeaf,
		sn.Type,
		sn.GoType,
		sn.Encoding,
		sn.Compression,
		sn.Path,
	}
}

type fieldWithPath struct {
	parquet.Field
	Path  []string
	Index int
}

func fieldsFromSchema(schema *parquet.Schema) []fieldWithPath {
	result := make([]fieldWithPath, 0)

	for _, field := range schema.Fields() {
		result = fieldsFromPathRecursive(field, []string{}, result)
	}

	var idx int
	for i := range result {
		if result[i].Leaf() {
			result[i].Index = idx
			idx++
		}
	}

	return result
}

func fieldsFromPathRecursive(field parquet.Field, path []string, result []fieldWithPath) []fieldWithPath {
	cpy := path[:len(path):len(path)]
	path = append(cpy, field.Name())

	result = append(result, fieldWithPath{Field: field, Path: path})

	for _, child := range field.Fields() {
		result = fieldsFromPathRecursive(child, path, result)
	}

	return result
}
