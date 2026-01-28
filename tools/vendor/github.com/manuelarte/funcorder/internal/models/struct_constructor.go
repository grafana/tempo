package models

import (
	"go/ast"

	"github.com/manuelarte/funcorder/internal/astutils"
)

type StructConstructor struct {
	constructor  *ast.FuncDecl
	structReturn *ast.Ident
}

func NewStructConstructor(funcDec *ast.FuncDecl) (StructConstructor, bool) {
	if !astutils.FuncCanBeConstructor(funcDec) {
		return StructConstructor{}, false
	}

	expr := funcDec.Type.Results.List[0].Type

	returnType, ok := astutils.GetIdent(expr)
	if !ok {
		return StructConstructor{}, false
	}

	return StructConstructor{
		constructor:  funcDec,
		structReturn: returnType,
	}, true
}

// GetStructReturn Return the struct linked to this "constructor".
func (sc StructConstructor) GetStructReturn() *ast.Ident {
	return sc.structReturn
}

func (sc StructConstructor) GetConstructor() *ast.FuncDecl {
	return sc.constructor
}
