package pkg

import (
	"go/ast"
	"go/token"
)

func inferExprType(expr ast.Expr) ast.Expr {
	switch e := expr.(type) {
	case *ast.ArrayType, *ast.StructType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.ChanType:
		return e
	case *ast.ParenExpr:
		return inferExprType(e.X)
	case *ast.SliceExpr:
		return inferExprType(e.X)
	case *ast.TypeAssertExpr:
		return inferExprType(e.Type)
	case *ast.CompositeLit:
		return inferExprType(e.Type)
	case *ast.Ellipsis:
		return &ast.ArrayType{Elt: e.Elt}
	case *ast.FuncLit:
		return &ast.FuncType{Results: e.Type.Results}
	case *ast.BasicLit:
		return inferBasicType(e)
	case *ast.BinaryExpr:
		return inferBinaryType(e)
	case *ast.StarExpr:
		return inferStarType(e)
	case *ast.UnaryExpr:
		return inferUnaryType(e)
	case *ast.CallExpr:
		return inferCallType(e)
	case *ast.IndexExpr:
		return inferIndexType(e)
	case *ast.IndexListExpr:
		return inferIndexListType(e)
	case *ast.SelectorExpr:
		return inferSelectorType(e)
	case *ast.Ident:
		return inferIdentType(e)
	default:
		return nil
	}
}

func inferBasicType(basic *ast.BasicLit) ast.Expr {
	switch basic.Kind {
	case token.INT:
		return ast.NewIdent("int")
	case token.FLOAT:
		return ast.NewIdent("float64")
	case token.IMAG:
		return ast.NewIdent("imag")
	case token.CHAR:
		return ast.NewIdent("char")
	case token.STRING:
		return ast.NewIdent("string")
	default:
		return nil
	}
}

func inferBinaryType(binary *ast.BinaryExpr) ast.Expr {
	switch binary.Op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return ast.NewIdent("bool")
	default:
		if x := inferExprType(binary.X); x != nil {
			return x
		}
		return inferExprType(binary.Y)
	}
}

func inferStarType(star *ast.StarExpr) ast.Expr {
	switch x := inferExprType(star.X).(type) {
	case nil:
		return nil
	case *ast.StarExpr:
		return inferExprType(x.X)
	default:
		return &ast.StarExpr{X: x}
	}
}

func inferUnaryType(unary *ast.UnaryExpr) ast.Expr {
	if x := inferExprType(unary.X); x != nil {
		switch unary.Op {
		case token.AND:
			return &ast.StarExpr{X: x}
		case token.ARROW:
			if ct, ok := x.(*ast.ChanType); ok {
				return inferExprType(ct.Value)
			}
			return x
		default:
			return x
		}
	}
	return nil
}

func inferCallType(call *ast.CallExpr) ast.Expr {
	if id, ok := call.Fun.(*ast.Ident); ok && id.Obj == nil {
		switch id.Name {
		case "len", "cap", "copy":
			return ast.NewIdent("int")
		case "real", "imag":
			return ast.NewIdent("float64")
		case "complex":
			return ast.NewIdent("complex64")
		case "recover":
			return ast.NewIdent("any")
		case "make", "min", "max":
			if len(call.Args) > 0 {
				return inferExprType(call.Args[0])
			}
		case "new":
			if len(call.Args) > 0 {
				if arg := inferExprType(call.Args[0]); arg != nil {
					return &ast.StarExpr{X: arg}
				}
			}
		case "append":
			if len(call.Args) > 0 {
				if arg := inferExprType(call.Args[0]); arg != nil {
					return arg
				}
				return &ast.ArrayType{}
			}
		}
	}

	fun := inferExprType(call.Fun)
	if ft, ok := fun.(*ast.FuncType); ok && len(ft.Results.List) > 0 {
		return inferExprType(ft.Results.List[0].Type)
	}
	return fun
}

func inferIndexType(index *ast.IndexExpr) ast.Expr {
	if selector, ok := index.X.(*ast.SelectorExpr); ok && selector.Sel != nil && selector.Sel.Name == "Seq" {
		if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == "iter" {
			return &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{{
					Names: []*ast.Ident{{Name: "yield"}},
					Type: &ast.FuncType{
						Params:  &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("V")}}},
						Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}},
					},
				}}},
			}
		}
	}

	switch x := inferExprType(index.X).(type) {
	case *ast.ArrayType:
		return inferExprType(x.Elt)
	case *ast.MapType:
		return inferExprType(x.Value)
	default:
		return x
	}
}

func inferIndexListType(index *ast.IndexListExpr) ast.Expr {
	if selector, ok := index.X.(*ast.SelectorExpr); ok && selector.Sel != nil && selector.Sel.Name == "Seq2" {
		if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == "iter" {
			return &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{{
					Names: []*ast.Ident{{Name: "yield"}},
					Type: &ast.FuncType{
						Params: &ast.FieldList{List: []*ast.Field{
							{Type: ast.NewIdent("K")},
							{Type: ast.NewIdent("V")},
						}},
						Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}},
					},
				}}},
			}
		}
	}

	x := inferExprType(index.X)
	if at, ok := x.(*ast.ArrayType); ok {
		return inferExprType(at.Elt)
	}
	return x
}

func inferSelectorType(sel *ast.SelectorExpr) ast.Expr {
	x := inferExprType(sel.X)
	if se, ok := x.(*ast.StarExpr); ok {
		x = se.X
	}
	switch x := x.(type) {
	case *ast.StructType:
		for _, field := range x.Fields.List {
			for _, name := range field.Names {
				if name.Name == sel.Sel.Name {
					return inferExprType(field.Type)
				}
			}
		}
	case *ast.InterfaceType:
		for _, method := range x.Methods.List {
			for _, name := range method.Names {
				if name.Name == sel.Sel.Name {
					return inferExprType(method.Type)
				}
			}
		}
	}
	return nil
}

func inferIdentType(ident *ast.Ident) ast.Expr {
	if ident.Obj == nil {
		switch ident.Name {
		case "bool", "byte", "comparable", "error", "rune", "string", "any",
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"float32", "float64", "complex64", "complex128":
			return ident
		case "nil":
			return ast.NewIdent("any")
		case "true", "false":
			return ast.NewIdent("bool")
		case "iota":
			return ast.NewIdent("int")
		}
	} else {
		switch decl := ident.Obj.Decl.(type) {
		case *ast.Field:
			return inferExprType(decl.Type)
		case *ast.FuncDecl:
			return inferExprType(decl.Type)
		case *ast.TypeSpec:
			// abort when recursive pointer type detected
			t := decl.Type
			for {
				if star, ok := t.(*ast.StarExpr); ok {
					t = star.X
				} else if t == ident {
					return nil
				} else {
					break
				}
			}
			return inferExprType(decl.Type)
		case *ast.ValueSpec:
			return inferValueType(decl, ident.Name)
		case *ast.AssignStmt:
			return inferAssignType(decl, ident.Name)
		}
	}
	return nil
}

func inferValueType(value *ast.ValueSpec, name string) ast.Expr {
	if value.Type != nil {
		return inferExprType(value.Type)
	}

	index := -1
	for i := range value.Names {
		if value.Names[i].Name == name {
			index = i
		}
	}
	if index < 0 {
		return nil
	}

	if len(value.Names) == len(value.Values) {
		return inferExprType(value.Values[index])
	}

	return inferAssignMultiType(value.Values[0], index)
}

func inferAssignType(assign *ast.AssignStmt, name string) ast.Expr {
	index := -1
	for i := range assign.Lhs {
		if id, ok := assign.Lhs[i].(*ast.Ident); ok && id.Name == name {
			index = i
		}
	}
	if index < 0 {
		return nil
	}

	if len(assign.Rhs) == 1 {
		if ue, ok := assign.Rhs[0].(*ast.UnaryExpr); ok && ue.Op == token.RANGE {
			switch rhs := inferExprType(assign.Rhs[0]).(type) {
			case *ast.ArrayType:
				switch index {
				case 0:
					return ast.NewIdent("int")
				case 1:
					return inferExprType(rhs.Elt)
				}
			case *ast.MapType:
				switch index {
				case 0:
					return inferExprType(rhs.Key)
				case 1:
					return inferExprType(rhs.Value)
				}
			case *ast.Ident:
				if rhs.Name == "string" {
					switch index {
					case 0:
						return ast.NewIdent("int")
					case 1:
						return ast.NewIdent("rune")
					}
				}
			case *ast.ChanType:
				if index == 0 {
					return inferExprType(rhs.Value)
				}
			}
		}
	}

	if len(assign.Lhs) == len(assign.Rhs) {
		return inferExprType(assign.Rhs[index])
	}

	return inferAssignMultiType(assign.Rhs[0], index)
}

func inferAssignMultiType(rhs ast.Expr, index int) ast.Expr {
	switch rhs := rhs.(type) {
	case *ast.TypeAssertExpr:
		switch index {
		case 0:
			return inferExprType(rhs.Type)
		case 1:
			return ast.NewIdent("bool")
		}
	case *ast.CallExpr:
		if fun, ok := inferExprType(rhs.Fun).(*ast.FuncType); ok {
			for _, res := range fun.Results.List {
				for range res.Names {
					if index == 0 {
						return inferExprType(res.Type)
					}
					index--
				}
			}
		}
	case *ast.IndexExpr:
		if mt, ok := inferExprType(rhs.X).(*ast.MapType); ok {
			switch index {
			case 0:
				return inferExprType(mt.Value)
			case 1:
				return ast.NewIdent("bool")
			}
		}
	case *ast.UnaryExpr:
		if ct, ok := inferExprType(rhs.X).(*ast.ChanType); ok {
			switch index {
			case 0:
				return inferExprType(ct.Value)
			case 1:
				return ast.NewIdent("bool")
			}
		}
	}

	return nil
}
