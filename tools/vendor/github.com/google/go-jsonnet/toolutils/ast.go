// Package toolutils includes several utilities handy for use in code analysis tools
package toolutils

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
)

// Children returns all children of a node. It supports ASTs before and after desugaring.
func Children(node ast.Node) []ast.Node {
	return parser.Children(node)
}
