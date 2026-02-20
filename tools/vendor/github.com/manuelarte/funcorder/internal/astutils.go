package internal

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
)

func FuncCanBeConstructor(n *ast.FuncDecl) bool {
	if !n.Name.IsExported() || n.Recv != nil {
		return false
	}

	if n.Type.Results == nil || len(n.Type.Results.List) == 0 {
		return false
	}

	for _, prefix := range []string{"new", "must"} {
		if strings.HasPrefix(strings.ToLower(n.Name.Name), prefix) &&
			len(n.Name.Name) > len(prefix) { // TODO(ldez): bug if the name is just `New`.
			return true
		}
	}

	return false
}

func FuncIsMethod(n *ast.FuncDecl) (*ast.Ident, bool) {
	if n.Recv == nil {
		return nil, false
	}

	if len(n.Recv.List) != 1 {
		return nil, false
	}

	if recv, ok := GetIdent(n.Recv.List[0].Type); ok {
		return recv, true
	}

	return nil, false
}

func GetIdent(expr ast.Expr) (*ast.Ident, bool) {
	switch exp := expr.(type) {
	case *ast.StarExpr:
		return GetIdent(exp.X)

	case *ast.Ident:
		return exp, true

	default:
		return nil, false
	}
}

// GetStartingPos returns the token starting position of the function
// taking into account if there are comments.
func GetStartingPos(function *ast.FuncDecl) token.Pos {
	startingPos := function.Pos()
	if function.Doc != nil {
		startingPos = function.Doc.Pos()
	}

	return startingPos
}

// NodeToBytes convert the ast.Node in bytes.
func NodeToBytes(fset *token.FileSet, node ast.Node) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// SplitExportedUnexported split functions/methods based on whether they are exported or not.
//
//nolint:nonamedreturns // names serve as documentation
func SplitExportedUnexported(funcDecls []*ast.FuncDecl) (exported, unexported []*ast.FuncDecl) {
	for _, f := range funcDecls {
		if f.Name.IsExported() {
			exported = append(exported, f)
		} else {
			unexported = append(unexported, f)
		}
	}

	return exported, unexported
}
