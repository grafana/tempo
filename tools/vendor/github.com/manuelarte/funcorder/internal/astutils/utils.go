package astutils

import (
	"go/ast"
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
